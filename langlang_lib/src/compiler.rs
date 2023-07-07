use std::collections::{HashMap, HashSet};

use log::debug;

use crate::ast::AST;
use crate::vm::{ContainerType, Instruction, Program};

#[derive(Debug)]
pub enum Error {
    NotFound(String),
    Semantic(String),
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        write!(f, "Compiler Error")?;
        match self {
            Error::NotFound(msg) => write!(f, "[NotFound]: {}", msg),
            Error::Semantic(msg) => write!(f, "[Semantic]: {}", msg),
        }
    }
}

#[derive(Debug, Clone)]
pub struct Config {
    optimize: u8,
}

impl Default for Config {
    fn default() -> Self {
        Self::o0()
    }
}

impl Config {
    /// o0 disables all optimizations
    pub fn o0() -> Self {
        Self { optimize: 0 }
    }

    /// o1 enables some optimizations: `failtwice`, `partialcommit`,
    /// `backcommit`, `testchar` and `testany`
    pub fn o1() -> Self {
        Self { optimize: 1 }
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
        }
    }

    /// Access the output of the compilation process.  Call this
    /// method after calling `compile_str()`.
    pub fn compile(&mut self, ast: AST) -> Result<Program, Error> {
        DetectLeftRec::default().run(&ast, &mut self.left_rec)?;
        self.compile_node(ast)?;
        self.backpatch_callsites()?;
        self.manual_recovery()?;

        Ok(Program::new(
            self.identifiers.clone(),
            self.labels.clone(),
            self.recovery.clone(),
            self.strings.clone(),
            self.code.clone(),
        ))
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
                        Instruction::Call(_, precedence) | Instruction::CallB(_, precedence) => {
                            if func_addr > addr {
                                Instruction::Call(func_addr - addr, precedence)
                            } else {
                                Instruction::CallB(addr - func_addr, precedence)
                            }
                        }
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
        let main = &self.strings[self.identifiers[&2_usize]];
        if self.left_rec[main] {
            self.code[0] = match self.code[0] {
                Instruction::Call(..) => Instruction::Call(2, 1),
                Instruction::CallB(..) => Instruction::CallB(2, 1),
                _ => unreachable!(),
            }
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
                self.emit(Instruction::Call(2, 0));
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
                self.emit(Instruction::Return);
                self.funcs.insert(strid, addr);
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
                let id = self.push_string(&name);
                match self.funcs.get(&id) {
                    Some(func_addr) => {
                        let addr = self.cursor - func_addr;
                        self.emit(Instruction::CallB(addr, precedence));
                    }
                    None => {
                        self.addrs.insert(self.cursor, id);
                        self.emit(Instruction::Call(0, precedence));
                    }
                }
                Ok(())
            }
            AST::Precedence(n, precedence) => {
                let pos = self.cursor;
                self.compile_node(*n)?;
                // rewrite the above node with the precedence level
                self.code[pos] = match self.code[pos] {
                    Instruction::Call(addr, _) => Instruction::Call(addr, precedence),
                    Instruction::CallB(addr, _) => Instruction::CallB(addr, precedence),
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

impl Default for Compiler {
    fn default() -> Self {
        Self::new(Config::default())
    }
}

#[derive(Default)]
struct DetectLeftRec<'a> {
    stack: Vec<&'a str>,
}

impl<'a> DetectLeftRec<'a> {
    fn run(&mut self, node: &'a AST, found: &mut HashMap<String, bool>) -> Result<(), Error> {
        let mut rules: HashMap<&'a String, &'a AST> = HashMap::new();
        match node {
            AST::Grammar(definitions) => {
                for definition in definitions {
                    match definition {
                        AST::LabelDefinition(..) => {}
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
            found.insert(name.to_string(), is_lr);
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

    fn assert_detectlr(input: &str, expected: HashMap<String, bool>) {
        let node = parser::Parser::new(input).parse().unwrap();
        let mut dlr = DetectLeftRec::default();
        let mut found = HashMap::new();
        dlr.run(&node, &mut found).unwrap();
        assert_eq!(found, expected);
    }

    #[test]
    fn detect_left_recursion_not_lr() {
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
    }

    #[test]
    fn detect_left_recursion_direct_lr() {
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
    }

    #[test]
    fn detect_left_recursion_indirect_lr() {
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
    }

    #[test]
    fn detect_left_recursion_mutual() {
        // Mutual recursion
        assert_detectlr(
            "A <- B '+' A / B
             B <- B '-n' / 'n'",
            HashMap::from([("A".to_string(), true), ("B".to_string(), true)]),
        );
    }

    #[test]
    fn detect_left_recursion_wrapping_precedence() {
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
}
