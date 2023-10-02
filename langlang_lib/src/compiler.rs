use std::collections::{HashMap, HashSet};

use crate::vm::{ContainerType, Instruction, Program};
use crate::wsrewrite::WhiteSpaceHandlerInjector;

use langlang_syntax::ast;
use langlang_syntax::visitor::Visitor;

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
    emit_wsh: bool,
}

impl Default for Config {
    fn default() -> Self {
        Self::o1()
    }
}

impl Config {
    /// o0 disables all optimizations
    pub fn o0() -> Self {
        Self {
            optimize: 0,
            emit_wsh: true,
        }
    }

    /// o1 enables some optimizations: `failtwice`, `partialcommit`,
    /// `backcommit`, `testchar` and `testany`
    pub fn o1() -> Self {
        Self {
            optimize: 1,
            emit_wsh: true,
        }
    }

    /// Generate a new Config instance disabling the flag that wraps
    /// generating code that handle whitespaces automatically
    pub fn disable_injecting_whitespace_handling(&self) -> Self {
        Self {
            optimize: self.optimize,
            emit_wsh: false,
        }
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
    // Map from the set of names of functions to the boolean defining
    // if the function is left recursive or not
    left_rec: HashMap<String, bool>,
    // depth of the use of the lex ('#') operator
    lex_level: usize,
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
            left_rec: HashMap::new(),
            lex_level: 0,
        }
    }

    /// compile a Grammar in its AST form into a program executable by
    /// the virtual machine
    pub fn compile(&mut self, grammar: &ast::Grammar, main: &str) -> Result<Program, Error> {
        DetectLeftRec::default().run(grammar, &mut self.left_rec)?;
        self.code_gen(grammar);
        self.backpatch_callsites()?;
        self.map_recovery_exprs()?;
        self.pick_main(main);

        Ok(Program::new(
            self.identifiers.clone(),
            self.labels.clone(),
            self.recovery.clone(),
            self.strings.clone(),
            self.code.clone(),
        ))
    }

    /// First tries decides if whitespace handling will be emitted, if
    /// so, rewrites the AST to.  Then traverse the ast to generate
    /// the bytecode into the internal code vector.
    fn code_gen(&mut self, grammar: &ast::Grammar) {
        if !self.config.emit_wsh {
            self.visit_grammar(grammar);
            return;
        }
        let g = WhiteSpaceHandlerInjector::default().run(grammar);
        self.visit_grammar(&g);
    }

    /// Try to find string `s` within the table of interned strings.
    /// Return its ID if it is found.  If the string `s` doesn't exist
    /// within the interned table yet, it's inserted and the index
    /// where it was inserted becomes its ID.
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
        Ok(())
    }

    /// walk through all the collected label IDs, if any production
    /// name matches, set that production as the recovery expression
    /// for the label
    fn map_recovery_exprs(&mut self) -> Result<(), Error> {
        for label_id in self.label_ids.iter() {
            if let Some(addr) = self.funcs.get(label_id) {
                let n = &self.strings[self.identifiers[addr]];
                let k = usize::from(self.left_rec[n]);
                self.recovery.insert(*label_id, (*addr, k));
            }
        }
        Ok(())
    }

    /// Find the address of the production `main` and write a call
    /// instruction pointing to such address at the first entry of the
    /// code vector.
    fn pick_main(&mut self, main: &str) {
        let id = self.push_string(main);
        let addr = self.funcs[&id];
        // Mark Ps as left recursive if the detector marked it as such
        let lr = if self.left_rec.get(main).is_some() && self.left_rec[main] {
            1
        } else {
            0
        };
        self.code[0] = match self.code[0] {
            Instruction::Call(..) => Instruction::Call(addr, lr),
            Instruction::CallB(..) => Instruction::CallB(addr, lr),
            _ => unreachable!(),
        }
    }

    /// Generate bytecode for both ZeroOrMore and OneOrMore
    fn compile_seq<'ast>(
        &mut self,
        prefix: Option<&'ast ast::Expression>,
        expr: &'ast ast::Expression,
    ) {
        // For when emitting code for OneOrMore
        if let Some(n) = prefix {
            self.visit_expression(n);
        }

        let pos = self.cursor;
        self.emit(Instruction::Choice(0));
        self.visit_expression(expr);
        self.emit(Instruction::CapCommit);

        let size = self.cursor - pos;
        self.code[pos] = Instruction::Choice(size + 1);
        match self.config.optimize {
            1 => self.emit(Instruction::PartialCommit(size - 1)),
            _ => self.emit(Instruction::CommitB(size)),
        }
    }

    /// Push `instruction` into the internal code vector and increment
    /// the cursor that points at the next instruction
    fn emit(&mut self, instruction: Instruction) {
        self.code.push(instruction);
        self.cursor += 1;
    }
}

impl<'ast> Visitor<'ast> for Compiler {
    fn visit_grammar(&mut self, n: &'ast ast::Grammar) {
        self.emit(Instruction::Call(2, 0));
        self.emit(Instruction::Halt);
        for d in &n.definition_names {
            self.visit_definition(&n.definitions[d]);
        }
    }

    fn visit_definition(&mut self, n: &'ast ast::Definition) {
        let addr = self.cursor;
        let strid = self.push_string(&n.name);
        self.identifiers.insert(addr, strid);
        self.visit_expression(&n.expr);
        self.emit(Instruction::Return);
        self.funcs.insert(strid, addr);
    }

    fn visit_choice(&mut self, n: &'ast ast::Choice) {
        let (mut i, last_choice) = (0, n.items.len() - 1);
        let mut commits = vec![];
        for choice in &n.items {
            if i == last_choice {
                self.visit_expression(choice);
                break;
            }
            i += 1;
            let pos = self.cursor;
            self.emit(Instruction::Choice(0));
            self.visit_expression(choice);
            self.code[pos] = Instruction::Choice(self.cursor - pos + 1);
            commits.push(self.cursor);
            self.emit(Instruction::Commit(0));
        }
        for commit in commits {
            self.code[commit] = Instruction::Commit(self.cursor - commit);
        }
    }

    fn visit_lex(&mut self, n: &'ast ast::Lex) {
        self.lex_level += 1;
        self.visit_expression(&n.expr);
        self.lex_level -= 1;
    }

    fn visit_and(&mut self, n: &'ast ast::And) {
        match self.config.optimize {
            1 => {
                let pos0 = self.cursor;
                self.emit(Instruction::ChoiceP(0));
                self.visit_expression(&n.expr);
                let pos1 = self.cursor;
                self.code[pos0] = Instruction::ChoiceP(pos1 - pos0);
                self.emit(Instruction::BackCommit(0));
                self.emit(Instruction::Fail);
                self.code[pos1] = Instruction::BackCommit(self.cursor - pos1);
            }
            _ => {
                let not = ast::Not::new(
                    n.span.clone(),
                    Box::new(ast::Not::new_expr(
                        n.span.clone(),
                        Box::new((*n.expr).clone()),
                    )),
                );
                self.visit_not(&not);
            }
        }
    }

    fn visit_not(&mut self, n: &'ast ast::Not) {
        let pos = self.cursor;
        match self.config.optimize {
            1 => {
                self.emit(Instruction::ChoiceP(0));
                self.visit_expression(&n.expr);
                self.code[pos] = Instruction::ChoiceP(self.cursor - pos + 1);
                self.emit(Instruction::FailTwice);
            }
            _ => {
                self.emit(Instruction::ChoiceP(0));
                self.visit_expression(&n.expr);
                self.code[pos] = Instruction::ChoiceP(self.cursor - pos + 2);
                self.emit(Instruction::Commit(1));
                self.emit(Instruction::Fail);
            }
        }
    }

    fn visit_optional(&mut self, n: &'ast ast::Optional) {
        let pos = self.cursor;
        self.emit(Instruction::Choice(0));
        self.visit_expression(&n.expr);
        let size = self.cursor - pos;
        self.code[pos] = Instruction::Choice(size + 1);
        self.emit(Instruction::Commit(1));
    }

    fn visit_zero_or_more(&mut self, n: &'ast ast::ZeroOrMore) {
        self.compile_seq(None, &n.expr);
    }

    fn visit_one_or_more(&mut self, n: &'ast ast::OneOrMore) {
        self.compile_seq(Some(&n.expr), &n.expr);
    }

    fn visit_precedence(&mut self, n: &'ast ast::Precedence) {
        let pos = self.cursor;
        self.visit_expression(&n.expr);
        // rewrite the above node with the precedence level
        self.code[pos] = match self.code[pos] {
            Instruction::Call(addr, _) => Instruction::Call(addr, n.precedence),
            Instruction::CallB(addr, _) => Instruction::CallB(addr, n.precedence),
            _ => unreachable!("Precedence only works on Identifiers"),
        };
    }

    fn visit_label(&mut self, n: &'ast ast::Label) {
        let label_id = self.push_string(&n.label);
        let pos = self.cursor;
        self.label_ids.insert(label_id);
        self.emit(Instruction::Choice(0));
        self.visit_expression(&n.expr);
        self.code[pos] = Instruction::Choice(self.cursor - pos + 1);
        self.emit(Instruction::Commit(2));
        self.emit(Instruction::Throw(label_id));
    }

    fn visit_list(&mut self, n: &'ast ast::List) {
        self.emit(Instruction::Open);
        for i in &n.items {
            self.visit_expression(i);
        }
        self.emit(Instruction::Close(ContainerType::List));
    }

    fn visit_node(&mut self, n: &'ast ast::Node) {
        self.emit(Instruction::Open);
        let id = self.push_string(&n.name);
        self.emit(Instruction::String(id));
        self.visit_expression(&n.expr);
        self.emit(Instruction::Close(ContainerType::Node));
    }

    fn visit_identifier(&mut self, n: &'ast ast::Identifier) {
        let precedence = match self.left_rec.get(&n.name) {
            Some(v) => usize::from(*v),
            None => 0,
        };
        let id = self.push_string(&n.name);
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
    }

    fn visit_string(&mut self, n: &'ast ast::String) {
        let id = self.push_string(&n.value);
        self.emit(Instruction::String(id));
    }

    fn visit_class(&mut self, n: &'ast ast::Class) {
        let choice = ast::Choice::new(
            n.span.clone(),
            n.literals
                .iter()
                .map(|i| ast::Expression::Literal(i.clone()))
                .collect(),
        );
        self.visit_choice(&choice);
    }

    fn visit_range(&mut self, n: &'ast ast::Range) {
        self.emit(Instruction::Span(n.start, n.end));
    }

    fn visit_char(&mut self, n: &'ast ast::Char) {
        self.emit(Instruction::Char(n.value));
    }

    fn visit_any(&mut self, _: &'ast ast::Any) {
        self.emit(Instruction::Any);
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
    fn run(
        &mut self,
        node: &'a ast::Grammar,
        found: &mut HashMap<String, bool>,
    ) -> Result<(), Error> {
        let mut rules: HashMap<&'a String, &'a ast::Expression> = HashMap::new();

        for (name, d) in &node.definitions {
            rules.insert(name, &d.expr);
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
        expr: &'a ast::Expression,
        rules: &HashMap<&'a String, &'a ast::Expression>,
    ) -> Result<bool, Error> {
        match expr {
            ast::Expression::Identifier(n) => {
                // for detecting mutual recursion
                if !self.stack.is_empty() && self.stack[self.stack.len() - 1] == n.name {
                    return Ok(true);
                }
                if n.name != name {
                    self.stack.push(&n.name);
                    let r = match rules.get(&n.name) {
                        Some(rule) => self.is_left_recursive(name, rule, rules)?,
                        None => {
                            return Err(Error::Semantic(format!(
                                "Rule {:#?} not found in grammar",
                                n.name
                            )))
                        }
                    };
                    self.stack.pop();
                    return Ok(r);
                }
                Ok(true)
            }
            ast::Expression::Choice(n) => {
                for c in &n.items {
                    if self.is_left_recursive(name, c, rules)? {
                        return Ok(true);
                    }
                }
                Ok(false)
            }
            ast::Expression::Sequence(seq) => {
                let mut i = 0;
                while i < seq.items.len() && is_empty_possible(&seq.items[i]) {
                    i += 1;
                }
                if i < seq.items.len() {
                    return self.is_left_recursive(name, &seq.items[i], rules);
                }
                Ok(false)
            }
            ast::Expression::Precedence(n) => self.is_left_recursive(name, &n.expr, rules),
            _ => Ok(false),
        }
    }
}

fn is_empty_possible(node: &ast::Expression) -> bool {
    matches!(
        node,
        ast::Expression::ZeroOrMore(..) | ast::Expression::Optional(..)
    )
}

pub fn expand(grammar: &ast::Grammar) -> ast::Grammar {
    let defs = grammar.definitions.values().map(expand_def).collect();
    let def_names = grammar.definition_names.to_vec();
    let imports = grammar.imports.to_vec();
    ast::Grammar::new(grammar.span.clone(), imports, def_names, defs)
}

fn expand_def(def: &ast::Definition) -> (String, ast::Definition) {
    (
        def.name.clone(),
        ast::Definition::new(
            def.span.clone(),
            def.name.clone(),
            ast::Node::new_expr(
                def.span.clone(),
                def.name.clone(),
                Box::new(def.expr.clone()),
            ),
        ),
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use langlang_syntax::parser;

    fn assert_detectlr(input: &str, expected: HashMap<String, bool>) {
        let mut p = parser::Parser::new(input);
        let node = p.parse_grammar().unwrap();
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
            "A <- 'foo' / 'bar' / A",
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
