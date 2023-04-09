use std::collections::{HashMap, HashSet};
use std::fmt::{self, Display, Formatter};

use log::debug;

use crate::ast::{SemExpr, SemExprBinaryOp, SemExprUnaryOp, SemValue, AST};
use crate::vm::{CaptureType, ContainerType, Instruction, Program, Value};

#[derive(Debug)]
pub enum Error {
    NotFound(String),
    Semantic(String),
    Type(String),
}

impl Display for Error {
    fn fmt(&self, f: &mut Formatter) -> fmt::Result {
        write!(f, "Compiler Error")?;
        match self {
            Error::NotFound(msg) => write!(f, "[NotFound]: {}", msg),
            Error::Semantic(msg) => write!(f, "[Semantic]: {}", msg),
            Error::Type(msg) => write!(f, "[Type]: {}", msg),
        }
    }
}

#[derive(Debug, Clone)]
pub struct Config {
    optimize: u8,
    enable_captures: bool,
    enable_sem_actions: bool,
}

impl Default for Config {
    fn default() -> Self {
        Self::o0()
    }
}

impl Config {
    /// o0 disables all optimizations
    pub fn o0() -> Self {
        Self {
            optimize: 0,
            enable_captures: true,
            enable_sem_actions: true,
        }
    }

    /// o1 enables some optimizations: `failtwice`, `partialcommit`,
    /// `backcommit`, `testchar` and `testany`
    pub fn o1() -> Self {
        Self {
            optimize: 1,
            enable_captures: true,
            enable_sem_actions: true,
        }
    }

    /// take an existing configuration and disables captures and imply
    /// that semantic actions will also be disabled.
    pub fn disable_captures(mut self) -> Self {
        self.enable_captures = false;
        self.disable_sem_actions()
    }

    /// take an existing configuration and disables the compilation of
    /// semantic action expressions
    pub fn disable_sem_actions(mut self) -> Self {
        self.enable_sem_actions = false;
        self
    }
}

#[derive(Debug, Clone)]
pub struct Compiler {
    // Enable configuring the compiler to some extent
    config: Config,
    // The index of the last instruction written the `code` vector
    cursor: usize,
    // Vector where the compiler writes down the instructions
    code: Vec<Instruction>,
    // Storage for unique (interned) strings
    strings: Vec<String>,
    // Map from strings to their position in the `strings` vector
    strings_map: HashMap<String, usize>,
    // Map from set of production string ids to the set of metadata
    // about the production
    funcs: HashMap<usize, usize>,
    // Map from set of positions of the first instruction of rules to
    // the position of their index in the strings map
    identifiers: HashMap<usize, usize>,
    // Map from call site addresses to production names that keeps
    // calls that need to be patched because they occurred syntaticaly
    // before the definition of the production
    addrs: HashMap<usize /* addr */, usize /* string id */>,
    // Map from the set of labels to the set of messages for error
    // reporting
    labels: HashMap<usize, usize>,
    // Set of all label IDs
    label_ids: HashSet<usize>,
    // Map from label IDs to tuples with two things: address of the
    // recovery expression and its precedence level
    recovery: HashMap<usize, (usize, usize)>,
    // Used for printing out debugging messages with the of the
    // structure the call stack the compiler is traversing
    indent_level: usize,
    // Map from the set of names of functions to the boolean defining
    // if the function is left recursive or not
    left_rec: HashMap<String, bool>,
    /// Keeps track of how many levels of recursion there are when
    /// compiling a SemExpr.  That is needed to find out if a call to
    /// `unwrapped()` is from the root of the sub-expression tree.
    sem_expr_recursion_level: usize,
    // Set with all semantic action rule names read by the first pass.
    // That allows the code generation pass to make decisions about a
    // production depending on the fact it has or not a
    sem_action_names: HashSet<String>,
    // Stack of maps of semantic action parameter names to their
    // position in the value stack.  A new map is pushed before
    // compiling the expression of a semantic action and the map is
    // popped when the semantic action expression is done being
    // compiled.
    sem_action_params: Vec<HashMap<usize, usize>>,
}

impl Compiler {
    /// Return a new instance of the Compiler with default values and
    /// a Config instance attached to it
    pub fn new(config: Config) -> Self {
        Compiler {
            config,
            cursor: 0,
            code: vec![],
            strings: vec![],
            strings_map: HashMap::new(),
            identifiers: HashMap::new(),
            funcs: HashMap::new(),
            addrs: HashMap::new(),
            labels: HashMap::new(),
            label_ids: HashSet::new(),
            recovery: HashMap::new(),
            indent_level: 0,
            left_rec: HashMap::new(),
            sem_expr_recursion_level: 0,
            sem_action_names: HashSet::new(),
            sem_action_params: vec![],
        }
    }

    /// Access the output of the compilation process.  Call this
    /// method after calling `compile_str()`.
    pub fn compile(&mut self, ast: AST) -> Result<Program, Error> {
        self.read_traversal(&ast)?;
        self.compile_node(ast)?;
        self.backpatch_callsites()?;
        self.pick_main()?;
        self.manual_recovery()?;

        Ok(Program::new(
            self.identifiers.clone(),
            self.labels.clone(),
            self.recovery.clone(),
            self.strings.clone(),
            self.code.clone(),
        ))
    }

    fn read_traversal(&mut self, ast: &AST) -> Result<(), Error> {
        let mut first_pass = FirstPass::default();
        first_pass.run(ast)?;
        self.sem_action_names = first_pass.output.sem_actions;
        self.left_rec = first_pass.output.left_rec;
        Ok(())
    }

    /// Try to find the string `s` within the table of interned
    /// strings, and return its ID if it is found.  If the string `s`
    /// doesn't exist within the interned table yet, it's inserted and
    /// the index where it was inserted becomes its ID.
    fn push_string(&mut self, s: &str) -> usize {
        let strid = self.strings.len();
        if let Some(id) = self.strings_map.get(s) {
            return *id;
        }
        self.strings.push(s.to_string());
        self.strings_map.insert(s.to_string(), strid);
        strid
    }

    /// Iterate over the set of addresses of call sites of forward
    /// rule declarations and re-emit the `Call` opcode with the right
    /// offset that could not be figured out in the first pass of the
    /// compilation.
    fn backpatch_callsites(&mut self) -> Result<(), Error> {
        for (addr, id) in &self.addrs {
            match self.funcs.get(id) {
                Some(func_addr) => {
                    self.code[*addr] = match self.code[*addr] {
                        Instruction::Call(_, precedence, ref capture)
                        | Instruction::CallB(_, precedence, ref capture) => {
                            if func_addr > addr {
                                Instruction::Call(func_addr - addr, precedence, capture.clone())
                            } else {
                                Instruction::CallB(addr - func_addr, precedence, capture.clone())
                            }
                        }

                        // rewrite jumps from the end of a production
                        // to the beginning if its semantic action
                        // expression
                        Instruction::Jump(_) => Instruction::Jump(*func_addr),

                        _ => unreachable!(),
                    };
                }
                None => {
                    let name = self.strings[*id].clone();
                    return Err(Error::NotFound(format!(
                        "Production {:?} doesnt exist",
                        name
                    )));
                }
            }
        }
        Ok(())
    }

    /// Picks the first production declared in the grammar and rewrite
    /// the invariant header that starts with a Call instruction to
    /// point it at this first production.  This is needed to skip
    /// through the semantic actions in case there are any declared
    /// before a production.  This also decides if that first Call
    /// will be a recursive call by using data read in the first pass.
    fn pick_main(&mut self) -> Result<(), Error> {
        // point invariant first call to the first production declared
        if let Some(identifier) = self
            .identifiers
            .iter()
            .filter_map(|(k, v)| {
                if self.strings[*v].starts_with("SEM_") {
                    None
                } else {
                    Some(k)
                }
            })
            .min()
        {
            let name = &self.strings[self.identifiers[identifier]];
            let left_rec = usize::from(self.left_rec[name]);
            let captype = self.default_capture_type();
            self.code[0] = match self.code[0] {
                Instruction::Call(..) => Instruction::Call(*identifier, left_rec, captype),
                _ => unreachable!(),
            };
        }
        Ok(())
    }

    /// walk through all the collected label IDs, if any production
    /// name matches, set that production as the recovery expression
    /// for the label
    fn manual_recovery(&mut self) -> Result<(), Error> {
        for label_id in self.label_ids.iter() {
            if let Some(addr) = self.funcs.get(label_id) {
                let n = &self.strings[self.identifiers[addr]];
                let k = usize::from(self.left_rec[n]);
                self.recovery.insert(*label_id, (*addr, k));
            }
        }
        Ok(())
    }

    /// Take an AST node and emit node into the private code vector
    fn compile_node(&mut self, node: AST) -> Result<(), Error> {
        match node {
            AST::Grammar(rules) => {
                let captype = self.default_capture_type();
                // This call is rewritten by backpatch_callsites()
                self.emit(Instruction::Call(1, 0, captype));
                self.emit(Instruction::Halt);
                for r in rules {
                    self.compile_node(r)?;
                }
                Ok(())
            }
            AST::Definition(name, expr) => {
                let addr = self.cursor;
                let strid = self.push_string(&name);
                self.identifiers.insert(addr, strid);
                self.compile_node(*expr)?;
                self.funcs.insert(strid, addr);

                // When semantic actions aren't meant to be used
                let captype = self.default_capture_type();
                if !self.config.enable_sem_actions || self.sem_action_names.get(&name).is_none() {
                    self.emit(Instruction::Return(captype));
                    return Ok(());
                }

                // If a production is associated with a semantic
                // action, its last instruction will be a Jump to the
                // first instruction of the semantic action, and not a
                // return.  The return will be the last instruction of
                // the semantic action.  The infrastructure for
                // tracking call sites is reused here to track jumps
                // from the end of the production to its semantic
                // action expressions.
                let mangled = format!("SEM_{}", name);
                let strid = self.push_string(&mangled);
                let addr = match self.funcs.get(&strid) {
                    Some(addr) => *addr,
                    None => {
                        // register position to be backpatched before
                        // returning a placeholder
                        self.addrs.insert(self.cursor, strid);
                        0
                    }
                };
                self.emit(Instruction::Jump(addr));
                Ok(())
            }
            AST::SemanticAction(name, args, semexpr) => {
                if !self.config.enable_sem_actions {
                    return Ok(());
                }
                let addr = self.cursor;
                let mut params = HashMap::new();
                for (i, arg) in args.iter().enumerate() {
                    match arg {
                        SemValue::Identifier(s) => {
                            let argid = self.push_string(s);
                            params.insert(argid, i);
                        }
                        SemValue::String(s) => {
                            let argid = self.push_string(s);
                            params.insert(argid, i);
                        }
                        _ => unreachable!(),
                    }
                }

                let mangled = format!("SEM_{}", name);
                let strid = self.push_string(&mangled);
                self.identifiers.insert(addr, strid);
                self.funcs.insert(strid, addr);

                self.sem_action_params.push(params);
                // self.emit(Instruction::CapPush);
                if let Some(expr) = validate_unwrapped(&semexpr)? {
                    // implementation of the builtin `unwrapped()`, which
                    // changes the capture type of the emitted Return.
                    self.compile_sem_expr(expr)?;
                    self.emit(Instruction::Return(CaptureType::Unwrapped));
                } else {
                    // regular compilation of non builtin semantic actions
                    self.compile_sem_expr(*semexpr)?;
                    self.emit(Instruction::Return(self.default_capture_type()));
                }
                // self.emit(Instruction::CapPop);
                self.sem_action_params.pop();
                Ok(())
            }
            AST::LabelDefinition(name, message) => {
                let name_id = self.push_string(&name);
                let message_id = self.push_string(&message);
                self.labels.insert(name_id, message_id);
                Ok(())
            }
            AST::Label(name, element) => {
                let label_id = self.push_string(&name);
                let pos = self.cursor;
                self.label_ids.insert(label_id);
                self.emit(Instruction::Choice(0));
                self.compile_node(*element)?;
                self.code[pos] = Instruction::Choice(self.cursor - pos + 1);
                self.emit(Instruction::Commit(2));
                self.emit(Instruction::Throw(label_id));
                Ok(())
            }
            AST::Sequence(seq) => {
                self.indent("Seq");
                for s in seq.into_iter() {
                    self.compile_node(s)?;
                }
                self.dedent("Seq");
                Ok(())
            }
            AST::Optional(op) => {
                self.emit(Instruction::CapPush);
                let pos = self.cursor;
                self.emit(Instruction::Choice(0));
                self.compile_node(*op)?;
                let size = self.cursor - pos;
                self.code[pos] = Instruction::Choice(size + 1);
                self.emit(Instruction::Commit(1));
                self.emit(Instruction::CapCommit);
                self.emit(Instruction::CapPop);
                Ok(())
            }
            AST::Choice(choices) => {
                let (mut i, last_choice) = (0, choices.len() - 1);
                let mut commits = vec![];
                for choice in choices {
                    if i == last_choice {
                        self.compile_node(choice)?;
                        break;
                    }
                    i += 1;
                    let pos = self.cursor;
                    self.emit(Instruction::Choice(0));
                    self.compile_node(choice)?;
                    self.code[pos] = Instruction::Choice(self.cursor - pos + 1);
                    commits.push(self.cursor);
                    self.emit(Instruction::Commit(0));
                }
                for commit in commits {
                    self.code[commit] = Instruction::Commit(self.cursor - commit);
                }
                Ok(())
            }
            AST::Not(expr) => {
                let pos = self.cursor;
                match self.config.optimize {
                    1 => {
                        self.emit(Instruction::ChoiceP(0));
                        self.compile_node(*expr)?;
                        self.code[pos] = Instruction::ChoiceP(self.cursor - pos + 1);
                        self.emit(Instruction::FailTwice);
                    }
                    _ => {
                        self.emit(Instruction::ChoiceP(0));
                        self.compile_node(*expr)?;
                        self.code[pos] = Instruction::ChoiceP(self.cursor - pos + 2);
                        self.emit(Instruction::Commit(1));
                        self.emit(Instruction::Fail);
                    }
                }
                Ok(())
            }
            AST::And(expr) => {
                match self.config.optimize {
                    1 => {
                        let pos0 = self.cursor;
                        self.emit(Instruction::ChoiceP(0));
                        self.compile_node(*expr)?;
                        let pos1 = self.cursor;
                        self.code[pos0] = Instruction::ChoiceP(pos1 - pos0);
                        self.emit(Instruction::BackCommit(0));
                        self.emit(Instruction::Fail);
                        self.code[pos1] = Instruction::BackCommit(self.cursor - pos1);
                    }
                    _ => self.compile_node(AST::Not(Box::new(AST::Not(expr))))?,
                }
                Ok(())
            }
            AST::ZeroOrMore(expr) => self.compile_seq(None, *expr),
            AST::OneOrMore(expr) => self.compile_seq(Some(*expr.clone()), *expr),
            AST::Identifier(name) => {
                let precedence = match self.left_rec.get(&name) {
                    Some(v) => usize::from(*v),
                    None => {
                        return Err(Error::NotFound(format!(
                            "Rule {:#?} not found in grammar",
                            name
                        )))
                    }
                };
                let cap = self.default_capture_type();
                let id = self.push_string(&name);
                match self.funcs.get(&id) {
                    Some(func_addr) => {
                        let addr = self.cursor - func_addr;
                        self.emit(Instruction::CallB(addr, precedence, cap));
                    }
                    None => {
                        self.addrs.insert(self.cursor, id);
                        self.emit(Instruction::Call(0, precedence, cap));
                    }
                }
                Ok(())
            }
            AST::Precedence(n, precedence) => {
                let pos = self.cursor;
                self.compile_node(*n)?;
                // rewrite the above node with the precedence level
                self.code[pos] = match self.code[pos] {
                    Instruction::Call(addr, _, ref cap) => {
                        Instruction::Call(addr, precedence, cap.clone())
                    }
                    Instruction::CallB(addr, _, ref cap) => {
                        Instruction::CallB(addr, precedence, cap.clone())
                    }
                    _ => {
                        return Err(Error::Semantic(
                            "Precedence suffix should only be used at Identifiers".to_string(),
                        ))
                    }
                };
                Ok(())
            }
            AST::Node(name, items) => {
                self.emit(Instruction::Open);
                self.compile_node(AST::String(name))?;
                for i in items {
                    self.compile_node(i)?;
                }
                self.emit(Instruction::Close(ContainerType::Node));
                Ok(())
            }
            AST::List(items) => {
                self.emit(Instruction::Open);
                for i in items {
                    self.compile_node(i)?;
                }
                self.emit(Instruction::Close(ContainerType::List));
                Ok(())
            }
            AST::Range(a, b) => {
                self.emit(Instruction::Span(a, b));
                Ok(())
            }
            AST::String(s) => {
                let id = self.push_string(&s);
                self.emit(Instruction::String(id));
                Ok(())
            }
            AST::Char(c) => {
                self.emit(Instruction::Char(c));
                Ok(())
            }
            AST::Any => {
                self.emit(Instruction::Any);
                Ok(())
            }
            AST::Empty => Ok(()),
        }
    }

    fn compile_seq(&mut self, prefix: Option<AST>, expr: AST) -> Result<(), Error> {
        self.emit(Instruction::CapPush);

        // For when emitting code for OneOrMore
        if let Some(n) = prefix {
            self.compile_node(n)?;
        }
        let pos = self.cursor;
        self.emit(Instruction::Choice(0));
        self.compile_node(expr)?;
        self.emit(Instruction::CapCommit);

        let size = self.cursor - pos;
        self.code[pos] = Instruction::Choice(size + 1);
        match self.config.optimize {
            1 => self.emit(Instruction::PartialCommit(size - 1)),
            _ => self.emit(Instruction::CommitB(size)),
        }

        self.emit(Instruction::CapCommit);
        self.emit(Instruction::CapPop);
        Ok(())
    }

    fn compile_sem_expr(&mut self, v: SemExpr) -> Result<(), Error> {
        self.sem_expr_recursion_level += 1;
        match v {
            SemExpr::Value(v) => self.compile_sem_value(v)?,
            SemExpr::BinaryOp(op, a, b) => self.compile_sem_bin_op(op, *a, *b)?,
            SemExpr::UnaryOp(op, expr) => self.compile_sem_un_op(op, *expr)?,
            SemExpr::Call(name, params) => self.compile_sem_call(name, params)?,
        }
        self.sem_expr_recursion_level -= 1;
        Ok(())
    }

    fn compile_sem_call(&mut self, name: String, params: Vec<SemExpr>) -> Result<(), Error> {
        let arity = params.len();
        let strid = self.push_string(&name);

        // unwrapped() is a builtin provided by the compiler, that
        // doesn't even exist within the VM utilized exclusively to
        // configure the return's capture type of a production as
        // unwrapped. So it doesn't make any sense to be called
        // anywhere else but at the root of the semantic action's
        // expression tree.
        if self.sem_expr_recursion_level > 1 && &name == "unwrapped" {
            return Err(Error::Semantic(
                "unwrapped() can only be called as a top level expression".to_string(),
            ));
        }
        for p in params {
            self.compile_sem_expr(p)?;
        }
        self.emit(Instruction::SemCallPrim(strid, arity));
        Ok(())
    }

    fn compile_sem_bin_op(
        &mut self,
        op: SemExprBinaryOp,
        left: SemExpr,
        right: SemExpr,
    ) -> Result<(), Error> {
        self.compile_sem_expr(left)?;
        self.compile_sem_expr(right)?;
        match op {
            SemExprBinaryOp::Addition => self.emit(Instruction::SemAdd),
            SemExprBinaryOp::Subtraction => self.emit(Instruction::SemSub),
            SemExprBinaryOp::Division => self.emit(Instruction::SemDiv),
            SemExprBinaryOp::Multiplication => self.emit(Instruction::SemMul),
        }
        Ok(())
    }

    // First emits code for the expression `expr` so it leaves a value
    // on the stack, then emits the unary operator that will then pop
    // the value first pushed by expr and then push the result of the
    // unary operation
    fn compile_sem_un_op(&mut self, op: SemExprUnaryOp, expr: SemExpr) -> Result<(), Error> {
        self.compile_sem_expr(expr)?;
        match op {
            SemExprUnaryOp::Positive => self.emit(Instruction::SemPositive),
            SemExprUnaryOp::Negative => self.emit(Instruction::SemNegative),
        }
        Ok(())
    }

    /// Push a literal value into the virtual machine's stack
    fn compile_sem_value(&mut self, v: SemValue) -> Result<(), Error> {
        match v {
            SemValue::Char(c) => {
                let value = Value::Char(c);
                self.emit(Instruction::SemPushVal(value));
            }
            SemValue::Bool(n) => {
                let value = Value::Bool(n);
                self.emit(Instruction::SemPushVal(value));
            }
            SemValue::I32(n) => {
                let value = Value::I32(n);
                self.emit(Instruction::SemPushVal(value));
            }
            SemValue::U32(n) => {
                let value = Value::U32(n);
                self.emit(Instruction::SemPushVal(value));
            }
            SemValue::I64(n) => {
                let value = Value::I64(n);
                self.emit(Instruction::SemPushVal(value));
            }
            SemValue::U64(n) => {
                let value = Value::U64(n);
                self.emit(Instruction::SemPushVal(value));
            }
            SemValue::F32(n) => {
                let value = Value::F32(n);
                self.emit(Instruction::SemPushVal(value));
            }
            SemValue::F64(n) => {
                let value = Value::F64(n);
                self.emit(Instruction::SemPushVal(value));
            }
            SemValue::String(s) => {
                let value = Value::String(s);
                self.emit(Instruction::SemPushVal(value));
            }
            SemValue::Variable(v) => {
                self.emit(Instruction::SemPushVar(v));
            }
            SemValue::Identifier(id) => {
                let argid = self.push_string(&id);
                let params = &self.sem_action_params[self.sem_action_params.len() - 1];
                match params.get(&argid) {
                    None => return Err(Error::NotFound(format!("Name {} not defined", id))),
                    Some(i) => self.emit(Instruction::SemPushVar(*i)),
                }
            }
            SemValue::List(items) => {
                let len = items.len();
                for i in items {
                    self.compile_sem_expr(i)?;
                }
                self.emit(Instruction::SemPushList(len))
            }
        }
        Ok(())
    }

    fn default_capture_type(&self) -> CaptureType {
        if self.config.enable_captures {
            return CaptureType::Wrapped;
        }
        CaptureType::Disabled
    }

    /// Push `instruction` into the internal code vector and increment
    /// the cursor that points at the next instruction
    fn emit(&mut self, instruction: Instruction) {
        self.prt(&format!("emit {:?} {:?}", self.cursor, instruction));
        self.code.push(instruction);
        self.cursor += 1;
    }

    // Debugging helpers

    fn prt(&mut self, msg: &str) {
        debug!("{:indent$}{}", "", msg, indent = self.indent_level);
    }

    fn indent(&mut self, msg: &str) {
        debug!("{:width$}Open {}", "", msg, width = self.indent_level);
        self.indent_level += 2;
    }

    fn dedent(&mut self, msg: &str) {
        self.indent_level -= 2;
        debug!("{:width$}Close {}", "", msg, width = self.indent_level);
    }
}

/// This identifies and unpacks the parameter out of an `unwrapped(P)`
/// call.  If `semexpr` is a call to unwrapped with a single
/// parameter, it will return the sub-expression in its parameter
/// list.  If `semexpr` is a call to unwrapped with any other arity,
/// it throws an error, and if it isn't a call or it's a call to any
/// other builtin, don't return anything.
fn validate_unwrapped(semexpr: &SemExpr) -> Result<Option<SemExpr>, Error> {
    match semexpr {
        SemExpr::Call(name, params) if name.as_str() == "unwrapped" && params.len() == 1 => {
            Ok(Some(params[0].clone()))
        }
        SemExpr::Call(name, params) if name.as_str() == "unwrapped" => Err(Error::Semantic(
            format!("unwrapped() takes 1 param, {} given", params.len()),
        )),
        _ => Ok(None),
    }
}

impl Default for Compiler {
    fn default() -> Self {
        Self::new(Config::default())
    }
}

#[derive(Default)]
struct FirstPassOutput {
    sem_actions: HashSet<String>,
    left_rec: HashMap<String, bool>,
}

/// FirstPass collects the name of all semantic action expressions and
/// detects which productions contain left recursive definitions.
#[derive(Default)]
struct FirstPass<'a> {
    stack: Vec<&'a str>,
    output: FirstPassOutput,
}

impl<'a> FirstPass<'a> {
    fn run(&mut self, node: &'a AST) -> Result<(), Error> {
        let mut rules: HashMap<&'a String, &'a AST> = HashMap::new();
        match node {
            AST::Grammar(definitions) => {
                for definition in definitions {
                    match definition {
                        AST::LabelDefinition(..) => {}
                        AST::SemanticAction(n, _args, _expr) => {
                            self.output.sem_actions.insert(n.clone());
                        }
                        AST::Definition(n, expr) => {
                            rules.insert(n, expr);
                        }
                        r => {
                            return Err(Error::Semantic(format!(
                                "Expected Definition rule, not {:#?}",
                                r
                            )))
                        }
                    }
                }
            }
            r => {
                return Err(Error::Semantic(format!(
                    "Expected Grammar rule, not {:#?}",
                    r
                )))
            }
        }
        for (name, expr) in &rules {
            let is_lr = self.is_left_recursive(name, expr, &rules)?;
            self.output.left_rec.insert(name.to_string(), is_lr);
        }
        Ok(())
    }

    fn is_left_recursive(
        &mut self,
        name: &'a str,
        expr: &'a AST,
        rules: &HashMap<&'a String, &'a AST>,
    ) -> Result<bool, Error> {
        match expr {
            AST::Identifier(n) => {
                // for detecting mutual recursion
                if !self.stack.is_empty() && self.stack[self.stack.len() - 1] == n {
                    return Ok(true);
                }
                if n != name {
                    self.stack.push(n);
                    let r = match rules.get(&n) {
                        Some(rule) => self.is_left_recursive(name, rule, rules)?,
                        None => {
                            return Err(Error::Semantic(format!(
                                "Rule {:#?} not found in grammar",
                                n
                            )))
                        }
                    };
                    self.stack.pop();
                    return Ok(r);
                }
                Ok(true)
            }
            AST::Choice(choices) => {
                for c in choices {
                    if self.is_left_recursive(name, c, rules)? {
                        return Ok(true);
                    }
                }
                Ok(false)
            }
            AST::Sequence(seq) => {
                let mut i = 0;
                while i < seq.len() && is_empty_possible(&seq[i]) {
                    i += 1;
                }
                if i < seq.len() {
                    return self.is_left_recursive(name, &seq[i], rules);
                }
                Ok(false)
            }
            AST::Precedence(n, _) => self.is_left_recursive(name, n, rules),
            _ => Ok(false),
        }
    }
}

fn is_empty_possible(node: &AST) -> bool {
    matches!(node, AST::ZeroOrMore(..) | AST::Optional(..))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::parser;

    #[test]
    fn detect_left_recursion() {
        // input is consumed before A calls itself, so not lr
        assert_detectlr("A <- 'foo' A", HashMap::from([("A".to_string(), false)]));
        assert_detectlr("A <- 'foo'+ A", HashMap::from([("A".to_string(), false)]));
        assert_detectlr(
            "A <- 'a' A / 'b' A",
            HashMap::from([("A".to_string(), false)]),
        );
        assert_detectlr(
            "A <- B
             B <- 'x' A",
            HashMap::from([("A".to_string(), false), ("B".to_string(), false)]),
        );
        assert_detectlr(
            "A <- B
             B <- C
             C <- 'x' A",
            HashMap::from([
                ("A".to_string(), false),
                ("B".to_string(), false),
                ("C".to_string(), false),
            ]),
        );
        // Direct left recursion
        assert_detectlr("A <- A", HashMap::from([("A".to_string(), true)]));
        // Direct left recursion: the first expression in both cases
        // can return successfully without consuming input
        assert_detectlr("A <- 'foo'? A", HashMap::from([("A".to_string(), true)]));
        assert_detectlr("A <- 'foo'* A", HashMap::from([("A".to_string(), true)]));
        // Direct left recursion: no branches of a top level choice
        // can start with a left recursive call
        assert_detectlr("A <- 'foo' / A", HashMap::from([("A".to_string(), true)]));
        assert_detectlr(
            "A <- 'foo' / 'bar'/ A",
            HashMap::from([("A".to_string(), true)]),
        );
        assert_detectlr(
            "A <- 'foo' / 'bar' / 'baz'? A",
            HashMap::from([("A".to_string(), true)]),
        );
        // Indirect left recursion
        assert_detectlr(
            "A <- B / 'x'
             B <- A",
            HashMap::from([("A".to_string(), true), ("B".to_string(), true)]),
        );
        assert_detectlr(
            "A <- 'x' / B
             B <- A",
            HashMap::from([("A".to_string(), true), ("B".to_string(), true)]),
        );
        assert_detectlr(
            "A <- B 'x'
             B <- 'x'? A",
            HashMap::from([("A".to_string(), true), ("B".to_string(), true)]),
        );
        assert_detectlr(
            "A <- B 'x'
             B <- C 'y'
             C <- D 'z'
             D <- E 'a'
             E <- 'x'* A",
            HashMap::from([
                ("A".to_string(), true),
                ("B".to_string(), true),
                ("C".to_string(), true),
                ("D".to_string(), true),
                ("E".to_string(), true),
            ]),
        );
        // Mutual recursion
        assert_detectlr(
            "A <- B '+' A / B
             B <- B '-n' / 'n'",
            HashMap::from([("A".to_string(), true), ("B".to_string(), true)]),
        );
        // With wrapping precedence
        assert_detectlr(
            "
            E <- E¹ '+' E²
               / E¹ '-' E²
               / 'n'
            ",
            HashMap::from([("E".to_string(), true)]),
        );
    }

    fn assert_detectlr(input: &str, expected: HashMap<String, bool>) {
        let node = parser::Parser::new(input).parse().unwrap();
        let mut dlr = FirstPass::default();
        dlr.run(&node).unwrap();
        assert_eq!(dlr.output.left_rec, expected);
    }
}
