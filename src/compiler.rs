use std::collections::HashMap;

use log::debug;

use crate::{ast::AST, vm};

const DEFAULT_CALL_PRECEDENCE: usize = 1;

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

#[derive(Debug)]
pub struct Config {
    optimize: u8,
    default_call_precedence: usize,
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
            default_call_precedence: DEFAULT_CALL_PRECEDENCE,
        }
    }

    /// o1 enables some optimizations: `failtwice`, `partialcommit`,
    /// `backcommit`, `testchar` and `testany`
    pub fn o1() -> Self {
        Self {
            optimize: 1,
            default_call_precedence: DEFAULT_CALL_PRECEDENCE,
        }
    }

    /// with_disabled_precedence returns a new instance of Config with
    /// all the fields copied over from the current instance, with the
    /// exception of `default_call_precedence` that will always be
    /// zeroed here.
    pub fn with_disabled_precedence(&self) -> Self {
        Self {
            optimize: self.optimize,
            default_call_precedence: 0,
        }
    }
}

#[derive(Debug)]
pub struct Compiler {
    // Enable configuring the compiler to some extent
    config: Config,
    // The index of the last instruction written the `code` vector
    cursor: usize,
    // Vector where the compiler writes down the instructions
    code: Vec<vm::Instruction>,
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
    // Map from the set of label IDs to the set with the first address
    // of the label's respective recovery expression
    recovery: HashMap<usize, usize>,
    // Used for printing out debugging messages with the of the
    // structure the call stack the compiler is traversing
    indent_level: usize,
}

impl Compiler {
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
            recovery: HashMap::new(),
            indent_level: 0,
        }
    }

    /// Access the output of the compilation process.  Call this
    /// method after calling `compile_str()`.
    pub fn compile(&mut self, ast: AST) -> Result<vm::Program, Error> {
        self.compile_node(ast)?;
        self.backpatch_callsites()?;

        Ok(vm::Program::new(
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
    fn push_string(&mut self, s: String) -> usize {
        let strid = self.strings.len();
        if let Some(id) = self.strings_map.get(&s) {
            return *id;
        }
        self.strings.push(s.clone());
        self.strings_map.insert(s, strid);
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
                        vm::Instruction::Call(_, precedence)
                        | vm::Instruction::CallB(_, precedence) => {
                            if func_addr > addr {
                                vm::Instruction::Call(func_addr - addr, precedence)
                            } else {
                                vm::Instruction::CallB(addr - func_addr, precedence)
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

    fn compile_node(&mut self, node: AST) -> Result<(), Error> {
        match node {
            AST::Grammar(rules) => {
                self.emit(vm::Instruction::Call(
                    2,
                    self.config.default_call_precedence,
                ));
                self.emit(vm::Instruction::Halt);
                for r in rules {
                    self.compile_node(r)?;
                }
                Ok(())
            }
            AST::Definition(name, expr) => {
                let addr = self.cursor;
                let strid = self.push_string(name.clone());
                self.identifiers.insert(addr, strid);
                self.compile_node(*expr)?;
                self.emit(vm::Instruction::Return);
                self.funcs.insert(strid, addr);
                Ok(())
            }
            AST::LabelDefinition(name, message) => {
                let name_id = self.push_string(name);
                let message_id = self.push_string(message);
                self.labels.insert(name_id, message_id);
                Ok(())
            }
            AST::Label(name, element) => {
                let label_id = self.push_string(name);
                let pos = self.cursor;
                self.emit(vm::Instruction::Choice(0));
                self.compile_node(*element)?;
                self.code[pos] = vm::Instruction::Choice(self.cursor - pos + 1);
                self.emit(vm::Instruction::Commit(2));
                self.emit(vm::Instruction::Throw(label_id));
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
                let pos = self.cursor;
                self.emit(vm::Instruction::Choice(0));
                self.compile_node(*op)?;
                let size = self.cursor - pos;
                self.code[pos] = vm::Instruction::Choice(size + 1);
                self.emit(vm::Instruction::Commit(1));
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
                    self.emit(vm::Instruction::Choice(0));
                    self.compile_node(choice)?;
                    self.code[pos] = vm::Instruction::Choice(self.cursor - pos + 1);
                    commits.push(self.cursor);
                    self.emit(vm::Instruction::Commit(0));
                }

                for commit in commits {
                    self.code[commit] = vm::Instruction::Commit(self.cursor - commit);
                }

                Ok(())
            }
            AST::Not(expr) => {
                let pos = self.cursor;
                match self.config.optimize {
                    1 => {
                        self.emit(vm::Instruction::ChoiceP(0));
                        self.compile_node(*expr)?;
                        self.code[pos] = vm::Instruction::ChoiceP(self.cursor - pos + 1);
                        self.emit(vm::Instruction::FailTwice);
                    }
                    _ => {
                        self.emit(vm::Instruction::ChoiceP(0));
                        self.compile_node(*expr)?;
                        self.code[pos] = vm::Instruction::ChoiceP(self.cursor - pos + 2);
                        self.emit(vm::Instruction::Commit(1));
                        self.emit(vm::Instruction::Fail);
                    }
                }
                Ok(())
            }
            AST::And(expr) => {
                match self.config.optimize {
                    1 => {
                        let pos0 = self.cursor;
                        self.emit(vm::Instruction::ChoiceP(0));
                        self.compile_node(*expr)?;
                        let pos1 = self.cursor;
                        self.code[pos0] = vm::Instruction::ChoiceP(pos1 - pos0);
                        self.emit(vm::Instruction::BackCommit(0));
                        self.emit(vm::Instruction::Fail);
                        self.code[pos1] = vm::Instruction::BackCommit(self.cursor - pos1);
                    }
                    _ => self.compile_node(AST::Not(Box::new(AST::Not(expr))))?,
                }
                Ok(())
            }
            AST::ZeroOrMore(expr) => {
                let pos = self.cursor;
                self.emit(vm::Instruction::Choice(0));
                self.compile_node(*expr)?;
                let size = self.cursor - pos;
                self.code[pos] = vm::Instruction::Choice(size + 1);
                match self.config.optimize {
                    1 => self.emit(vm::Instruction::PartialCommit(size - 1)),
                    _ => self.emit(vm::Instruction::CommitB(size)),
                }
                Ok(())
            }
            AST::OneOrMore(expr) => {
                let e = *expr;
                self.compile_node(e.clone())?;
                self.compile_node(AST::ZeroOrMore(Box::new(e)))?;
                Ok(())
            }
            AST::Identifier(name) => {
                let id = self.push_string(name);
                match self.funcs.get(&id) {
                    Some(func_addr) => {
                        let addr = self.cursor - func_addr;
                        self.emit(vm::Instruction::CallB(
                            addr,
                            self.config.default_call_precedence,
                        ));
                    }
                    None => {
                        self.addrs.insert(self.cursor, id);
                        self.emit(vm::Instruction::Call(
                            0,
                            self.config.default_call_precedence,
                        ));
                    }
                }
                Ok(())
            }
            AST::Precedence(n, precedence) => {
                let pos = self.cursor;
                self.compile_node(*n)?;
                // rewrite the above node with the precedence level
                self.code[pos] = match self.code[pos] {
                    vm::Instruction::Call(addr, _) => vm::Instruction::Call(addr, precedence),
                    vm::Instruction::CallB(addr, _) => vm::Instruction::CallB(addr, precedence),
                    _ => {
                        return Err(Error::Semantic(format!(
                            "Precedence suffix should only be used at Identifiers",
                        )))
                    }
                };
                Ok(())
            }
            AST::Range(a, b) => {
                self.emit(vm::Instruction::Span(a, b));
                Ok(())
            }
            AST::Str(s) => {
                let id = self.push_string(s);
                self.emit(vm::Instruction::Str(id));
                Ok(())
            }
            AST::Char(c) => {
                self.emit(vm::Instruction::Char(c));
                Ok(())
            }
            AST::Any => {
                self.emit(vm::Instruction::Any);
                Ok(())
            }
            AST::Empty => Ok(()),
        }
    }

    fn emit(&mut self, instruction: vm::Instruction) {
        self.prt(format!("emit {:?} {:?}", self.cursor, instruction).as_str());
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
