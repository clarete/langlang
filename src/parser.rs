use log::debug;
use std::boxed::Box;
use std::collections::HashMap;

use crate::vm;

#[derive(Debug)]
pub struct Location {
    // how many characters have been seen since the begining of
    // parsing
    cursor: usize,
    // how many end-of-line sequences seen since the begining of
    // parsing
    line: usize,
    // how many characters seen since the begining of the line
    column: usize,
}

#[derive(Clone, Debug)]
pub enum AST {
    Grammar(Vec<AST>),
    Definition(String, Box<AST>),
    LabelDefinition(String, String),
    Sequence(Vec<AST>),
    Choice(Vec<AST>),
    Not(Box<AST>),
    Optional(Box<AST>),
    ZeroOrMore(Box<AST>),
    OneOrMore(Box<AST>),
    Identifier(String),
    Str(String),
    Range(char, char),
    Char(char),
    Label(String, Box<AST>),
    Any,
    Empty,
}

#[derive(Debug)]
pub struct Fun {
    name: String,
    addr: usize,
    size: usize,
}

#[derive(Clone, Debug)]
pub enum Token {
    Deferred(usize),
    StringID(usize),
}

#[derive(Debug)]
pub struct Compiler {
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
    funcs: HashMap<usize, Fun>,
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
    // Map from the set of addresses to the set of vectors of
    // terminals that should be expected in case the expression under
    // the address fails parsing.  This is similar to `final_follows`,
    // except that this map can contain deferred addresses that still
    // need to be resolved during backpatching.
    follows: HashMap<usize, Vec<Token>>,
    // Same type of map as `follows`, the difference is that follows
    // map can contain deferred addresses, and this set can't.  All
    // addresses here are final.
    final_follows: HashMap<usize, Vec<usize>>,
    // Stack of sets of firsts, are used to build the follows sets
    firsts: Vec<Vec<Token>>,
    // Map of the sets of string IDs of rule names to the sets of
    // first tokens that appear in a rule
    rules_firsts: HashMap<usize, Vec<Token>>,
    // Used for printing out debugging messages with the of the
    // structure the call stack the compiler is traversing
    indent_level: usize,
}

impl Compiler {
    pub fn new() -> Self {
        Compiler {
            cursor: 0,
            code: vec![],
            strings: vec![],
            strings_map: HashMap::new(),
            identifiers: HashMap::new(),
            funcs: HashMap::new(),
            addrs: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            follows: HashMap::new(),
            final_follows: HashMap::new(),
            firsts: vec![],
            rules_firsts: HashMap::new(),
            indent_level: 0,
        }
    }

    /// Takes a PEG string and runs the compilation process, by
    /// parsing the input string, traversing the output grammar to
    /// emit the code, and then backpatching both call sites and
    /// follows sets deferred during the code main generation pass.
    pub fn compile_str(&mut self, s: &str) -> Result<(), Error> {
        let mut p = Parser::new(s);
        self.compile(p.parse_grammar()?)?;
        self.backpatch_callsites()?;
        self.backpatch_follows();
        Ok(())
    }

    /// Access the output of the compilation process.  Call this
    /// method after calling `compile_str()`.
    pub fn program(self) -> vm::Program {
        vm::Program::new(
            self.identifiers,
            self.final_follows,
            self.labels,
            self.recovery,
            self.strings,
            self.code,
        )
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
                Some(func) => {
                    if func.addr > *addr {
                        self.code[*addr] = vm::Instruction::Call(func.addr - addr, 0);
                    } else {
                        self.code[*addr] = vm::Instruction::CallB(addr - func.addr, 0);
                    }
                }
                None => {
                    let name = self.strings[*id].clone();
                    return Err(Error::CompileError(format!(
                        "Production {:?} doesnt exist",
                        name
                    )));
                }
            }
        }
        Ok(())
    }

    fn backpatch_follows(&mut self) {
        for (pos, tokens) in &self.follows {
            let mut addrs = vec![];
            for token in tokens {
                addrs.extend(self.resolve_token(token))
            }
            addrs.dedup();
            self.final_follows.insert(*pos, addrs);
        }
    }

    fn resolve_token(&self, token: &Token) -> Vec<usize> {
        match token {
            Token::StringID(id) => vec![*id],
            Token::Deferred(id) => {
                let mut v = vec![];
                for first in self.rules_firsts[id].clone() {
                    v.extend(self.resolve_token(&first))
                }
                v
            }
        }
    }

    /// Traverse AST node, emit bytecode and find first/follows sets
    ///
    /// The first part of the work done by this method is to emit
    /// bytecode for the Parsing Expression Grammars virtual machine.
    /// This work is heavily based on the article "A Parsing Machine
    /// for PEGs" by S. Medeiros, et al.
    ///
    /// The second challenge that this method tackles is building the
    /// follows sets.  Learn more about these sets in the article,
    /// also by S. Medeiros, named "Syntax Error Recovery in Parsing
    /// Expression Grammars".  But here's the rough idea:
    ///
    ///   follows(G[ε])       = {ε}                   ; empty
    ///   follows(G[a])       = {ε}                   ; terminal
    ///   follows(G[p1 / p2]) = {ε}                   ; ordered choice
    ///   follows(G[A])       = first(G[P(A)])        ; non-terminal
    ///   follows(G[p1 p2])   = first(p2)             ; sequence
    ///   follows(G[p*])      = {ε}                   ; repetition
    ///   follows(G[!p])      =
    ///
    ///   first(G[ε])       = {ε}                     ; empty
    ///   first(G[a])       = {a}                     ; terminal
    ///   first(G[A])       = first(G[P(A)])          ; non-terminal
    ///   first(G[p1 p2])   = first(p1)               ; sequence
    ///   first(G[p1 / p2]) = first(p1) ++ first(p2)  ; ordered choice
    ///   first(G[p*])      = first(p)                ; repetition
    ///   first(G[!p])      =
    ///
    /// The output of the compilation can be accessed via the
    /// `program()` method.
    fn compile(&mut self, node: AST) -> Result<(), Error> {
        match node {
            AST::Grammar(rules) => {
                self.emit(vm::Instruction::Call(2, 0));
                self.emit(vm::Instruction::Halt);
                for r in rules {
                    self.compile(r)?;
                }
                Ok(())
            }
            AST::Definition(name, expr) => {
                let addr = self.cursor;
                let strid = self.push_string(name.clone());
                self.identifiers.insert(addr, strid);

                let n = format!("Definition {:?}", name);
                self.pushfff(n.as_str());
                self.compile(*expr)?;
                self.emit(vm::Instruction::Return);

                let firsts = self.popfff(n.as_str());
                self.rules_firsts.insert(strid, firsts);
                self.funcs.insert(
                    strid,
                    Fun {
                        name,
                        addr,
                        size: self.cursor - addr,
                    },
                );
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
                self.compile(*element)?;
                self.code[pos] = vm::Instruction::Choice(self.cursor - pos + 1);
                self.emit(vm::Instruction::Commit(2));
                self.emit(vm::Instruction::Throw(label_id));
                Ok(())
            }
            AST::Sequence(seq) => {
                self.indent("Seq");
                for (i, s) in seq.into_iter().enumerate() {
                    let pos = self.cursor;
                    self.pushfff("SeqItem");
                    self.compile(s)?;
                    let firsts = self.popfff("SeqItem");
                    if i == 0 {
                        for f in firsts.clone() {
                            self.add_first(f);
                        }
                    }
                    self.follows.insert(pos, firsts);
                }
                self.dedent("Seq");
                Ok(())
            }
            AST::Optional(op) => {
                let pos = self.cursor;
                self.emit(vm::Instruction::Choice(0));
                self.compile(*op)?;
                let size = self.cursor - pos;
                self.code[pos] = vm::Instruction::Choice(size + 1);
                self.emit(vm::Instruction::Commit(1));
                Ok(())
            }
            AST::Choice(choices) => {
                let (mut i, last_choice) = (0, choices.len() - 1);
                let mut commits = vec![];

                self.pushfff("Choice");

                for choice in choices {
                    if i == last_choice {
                        self.compile(choice)?;
                        break;
                    }
                    i += 1;
                    let pos = self.cursor;
                    self.emit(vm::Instruction::Choice(0));
                    self.compile(choice)?;
                    self.code[pos] = vm::Instruction::Choice(self.cursor - pos + 1);
                    commits.push(self.cursor);
                    self.emit(vm::Instruction::Commit(0));
                }

                for commit in commits {
                    self.code[commit] = vm::Instruction::Commit(self.cursor - commit);
                }

                let firsts = self.popfff("Choice");

                for f in firsts {
                    self.add_first(f);
                }

                Ok(())
            }
            AST::Not(expr) => {
                let pos = self.cursor;
                self.emit(vm::Instruction::ChoiceP(0));
                self.compile(*expr)?;
                self.code[pos] = vm::Instruction::ChoiceP(self.cursor - pos + 2);
                self.emit(vm::Instruction::Commit(1));
                self.emit(vm::Instruction::Fail);
                Ok(())
            }
            AST::ZeroOrMore(expr) => {
                let pos = self.cursor;
                self.emit(vm::Instruction::Choice(0));
                self.compile(*expr)?;
                let size = self.cursor - pos;
                self.code[pos] = vm::Instruction::Choice(size + 1);
                self.emit(vm::Instruction::CommitB(size));
                Ok(())
            }
            AST::OneOrMore(expr) => {
                let e = *expr;
                self.compile(e.clone())?;
                let pos = self.cursor;
                self.emit(vm::Instruction::Choice(0));
                self.compile(e)?;
                self.code[pos] = vm::Instruction::Choice(self.cursor - pos + 1);
                self.emit(vm::Instruction::CommitB(self.cursor - pos));
                Ok(())
            }
            AST::Identifier(name) => {
                let id = self.push_string(name);
                match self.funcs.get(&id) {
                    Some(func) => {
                        let addr = self.cursor - func.addr;
                        for f in self.rules_firsts[&id].clone() {
                            self.add_first(f);
                        }
                        self.emit(vm::Instruction::CallB(addr, 0));
                    }
                    None => {
                        self.add_first(Token::Deferred(id));
                        self.addrs.insert(self.cursor, id);
                        self.emit(vm::Instruction::Call(0, 0));
                    }
                }
                self.emit(vm::Instruction::Capture);
                Ok(())
            }
            AST::Range(a, b) => {
                self.emit(vm::Instruction::Span(a, b));
                self.emit(vm::Instruction::Capture);
                Ok(())
            }
            AST::Str(s) => {
                let id = self.push_string(s);
                self.emit(vm::Instruction::Str(id));
                self.emit(vm::Instruction::Capture);
                self.add_first(Token::StringID(id));
                Ok(())
            }
            AST::Char(c) => {
                self.emit(vm::Instruction::Char(c));
                self.emit(vm::Instruction::Capture);
                Ok(())
            }
            AST::Any => {
                self.emit(vm::Instruction::Any);
                self.emit(vm::Instruction::Capture);
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

    // Helpers for building the "firsts" sets

    fn pushfff(&mut self, msg: &str) {
        self.indent(msg);
        self.firsts.push(vec![]);
    }

    fn popfff(&mut self, msg: &str) -> Vec<Token> {
        self.dedent(msg);
        self.firsts.pop().unwrap()
    }

    fn add_first(&mut self, first: Token) {
        let l = self.firsts.len();
        if l > 0 {
            self.prt_token("first", &first);
            self.firsts[l - 1].push(first);
        }
    }

    // Debugging helpers

    fn prt(&mut self, msg: &str) {
        debug!("{:indent$}{}", "", msg, indent = self.indent_level);
    }

    fn prt_token(&mut self, msg: &str, token: &Token) {
        debug!(
            "{:width$}{} {}",
            "",
            msg,
            match token {
                Token::Deferred(id) => format!("Deferred {:#?} {:?}", id, self.strings[*id]),
                Token::StringID(id) => format!("StringID {:#?} {:?}", id, self.strings[*id]),
            },
            width = self.indent_level
        );
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

/// The first stage of the StandardAlgorithm
///
/// This traversal collects following information:
///
/// 1. all the AST nodes with matchers for recognizable terminals
/// (`Char`, `Str`, and `Range`).  That's used for building the
/// `eatToken` expression.
///
/// 2. the names of rules that match exclusively white space
/// characters or call out identifiers of other rules that match
/// exclusively white spaces. e.g.:
///
/// ```
/// _   <- ws*
/// ws  <- eol / sp
/// eol <- '\n' / '\r\n' / '\r'
/// sp  <- [ \t]
/// ```
///
/// All the rules above would appear in the result of this function as
/// space rules.  The rules `_` and `ws` are examples of rules that
/// contain identifiers, but are still considered space rules because
/// the identifiers they call out to are rules that only match space
/// characters (`eol` and `sp`).
///
/// 3. the names of rules that contain expressions that match
/// syntatical structure. Notice that space rules don't ever appear on
/// this list. e.g.:
///
/// ```
/// T <- D "+" D / D "-" D
/// D <- [0-9]+ _
/// _ <- [ \t]*
/// ```
///
/// In the example above, the rule `T` would be not a lexical rule,
/// but `D` and `_` wouldn't.  Although `D` does contain an identifier
/// to another rule, is considered to be a lexical a because the
/// identifier it contains points to a rule that is a space rule.
fn stage1(node: &AST) -> (Vec<AST>, Vec<String>, Vec<String>) {
    #[derive(Clone, Debug, PartialEq)]
    enum Status {
        Yes,
        No,
        Maybe,
    }
    struct _TraverseEnv {
        // stack for Identifiers that haven't had their status resolved
        unknown_ids_stk: Vec<Vec<String>>,

        // state for finding rules that match whitespace chars only
        rule_is_space: HashMap<String, Status>,
        unknown_space_ids: HashMap<String, Vec<String>>,
        space_rules: Vec<String>,

        // state for finding lexical rules
        rule_is_lexical: HashMap<String, Status>,
        unknown_lexical_ids: HashMap<String, Vec<String>>,
        lexical_rules: Vec<String>,

        // state for eatToken
        lexical_tokens: Vec<AST>,
    }
    fn _traverse(
        env: &mut _TraverseEnv,
        node: &AST,
    ) -> (Status /* is_space */, Status /* is_lex */) {
        match node {
            // doesn't matter
            AST::LabelDefinition(..) => (Status::No, Status::No),
            // not spaces
            AST::Any | AST::Empty => (Status::No, Status::Yes),
            // identifiers are a special case for space rules
            AST::Identifier(id) => match env.rule_is_space.get(id) {
                // if it's a known identifier that points to a space rule, that implies
                // that it's a lexical rule too
                Some(Status::Yes) => (Status::Yes, Status::Yes),
                // if it's known to not point to a space rule, also forward the space status
                Some(Status::No) => (Status::No, Status::No),
                // otherwise, add the identifier to the ones that must be checked later
                Some(Status::Maybe) | None => {
                    env.unknown_ids_stk.last_mut().unwrap().push(id.clone());
                    (Status::Maybe, Status::Maybe)
                }
            },
            // forwards
            AST::Not(e) => _traverse(env, e),
            AST::Label(_, e) => _traverse(env, e),
            AST::Optional(e) | AST::ZeroOrMore(e) | AST::OneOrMore(e) => _traverse(env, e),
            // specialized
            AST::Grammar(ev) => {
                ev.into_iter().for_each(|e| {
                    _traverse(env, e);
                });

                // Find all symbols that are themselves rules with only spaces
                loop {
                    let mut has_change = false;

                    // find definitions that are still worth looking into
                    let mut maybe_rules = env.rule_is_space.clone();
                    maybe_rules.retain(|_, v| *v == Status::Maybe);

                    for rule in maybe_rules.keys() {
                        // if all the unknown IDs were already resolved and it has been resolved
                        // into a space rule, then we mark the `rule` itself as a space rule.
                        if env.unknown_space_ids[rule]
                            .iter()
                            .all(|id| env.rule_is_space[id] == Status::Yes)
                        {
                            env.rule_is_space.insert(rule.to_string(), Status::Yes);
                            // knowing for sure that a rule only matches spaces, also confirms
                            // that it is a lexical rule
                            env.rule_is_lexical.insert(rule.to_string(), Status::Yes);
                            has_change = true;
                        }
                    }
                    if !has_change {
                        break;
                    }
                }

                loop {
                    let mut has_change = false;

                    // find all rules that we're still uncertain about
                    let mut maybe_rules = env.rule_is_lexical.clone();
                    maybe_rules.retain(|_, v| *v == Status::Maybe);

                    for rule in maybe_rules.keys() {
                        if env.unknown_lexical_ids[rule]
                            .iter()
                            .all(|id| env.rule_is_space[id] == Status::Yes)
                        {
                            env.rule_is_lexical.insert(rule.to_string(), Status::Yes);
                            has_change = true;
                        }
                    }
                    if !has_change {
                        break;
                    }
                }

                env.lexical_rules.append(
                    &mut env
                        .rule_is_lexical
                        .iter()
                        .filter(|(_, v)| **v == Status::Maybe)
                        .map(|(k, _)| k.clone())
                        .collect(),
                );

                (Status::No, Status::No)
            }
            AST::Definition(def, expr) => {
                // collect identifiers of rules that we're unsure if they're space rules or not
                env.unknown_ids_stk.push(vec![]);
                let (is_space, mut is_lex) = _traverse(env, expr);
                let unknown_identifiers = env.unknown_ids_stk.pop().unwrap_or(vec![]);

                env.rule_is_space.insert(def.clone(), is_space.clone());
                match is_space {
                    Status::No => {}
                    Status::Yes => {
                        env.space_rules.push(def.clone());
                        // if a rule contains only spaces, we can tell right away that it is a
                        // lexical rule.  This allows us to save some work when patching.
                        is_lex = Status::Yes;
                    }
                    Status::Maybe => {
                        env.unknown_space_ids
                            .insert(def.clone(), unknown_identifiers.clone());
                    }
                }

                env.rule_is_lexical.insert(def.clone(), is_lex.clone());
                match is_lex {
                    Status::No => {}
                    Status::Yes => {
                        env.lexical_rules.push(def.clone());
                    }
                    Status::Maybe => {
                        env.unknown_lexical_ids
                            .insert(def.clone(), unknown_identifiers);
                    }
                }

                (is_space, is_lex)
            }
            AST::Sequence(exprs) | AST::Choice(exprs) => {
                let (mut all_spaces, mut all_lex) = (Status::Yes, Status::Yes);
                for expr in exprs {
                    let (is_space, is_lex) = _traverse(env, expr);
                    all_spaces = match (is_space, all_spaces) {
                        (Status::Yes, Status::Yes) => Status::Yes,
                        (Status::Yes | Status::Maybe, Status::Yes | Status::Maybe) => Status::Maybe,
                        _ => Status::No,
                    };
                    all_lex = match (is_lex, all_lex) {
                        (Status::Yes, Status::Yes) => Status::Yes,
                        (Status::No, Status::No) => Status::No,
                        (Status::Maybe, _) | (_, Status::Maybe) => Status::Maybe,
                        (Status::Yes, Status::No) | (Status::No, Status::Yes) => Status::Yes,
                    };
                }
                (all_spaces, all_lex)
            }
            AST::Range(a, b) => {
                env.lexical_tokens.push(AST::Range(*a, *b));
                let is_space = if char::is_whitespace(*a) && char::is_whitespace(*b) {
                    Status::Yes
                } else {
                    Status::No
                };
                (is_space, Status::Yes)
            }
            AST::Str(st) => {
                env.lexical_tokens.push(AST::Str(st.clone()));
                let is_space = if st.chars().all(|s| char::is_whitespace(s)) {
                    Status::Yes
                } else {
                    Status::No
                };
                (is_space, Status::Yes)
            }
            AST::Char(c) => {
                env.lexical_tokens.push(AST::Char(*c));
                let is_space = if char::is_whitespace(*c) {
                    Status::Yes
                } else {
                    Status::No
                };
                (is_space, Status::Yes)
            }
        }
    }

    let base_env = &mut _TraverseEnv {
        unknown_ids_stk: vec![],
        rule_is_space: HashMap::new(),
        unknown_space_ids: HashMap::new(),
        space_rules: vec![],
        rule_is_lexical: HashMap::new(),
        unknown_lexical_ids: HashMap::new(),
        lexical_rules: vec![],
        lexical_tokens: vec![],
    };

    _traverse(base_env, node);

    let space_rules: Vec<String> = base_env
        .rule_is_space
        .iter()
        .filter(|(_, v)| **v == Status::Yes)
        .map(|(k, _)| k.clone())
        .collect();

    // Collect only rules that contain only lexical matches.  That includes all the space rules.
    let lexical_rules: Vec<String> = base_env
        .lexical_rules
        .iter()
        .filter(|r| !space_rules.contains(r))
        .filter(|r| base_env.rule_is_lexical[*r] != Status::Yes)
        .map(|r| r.clone())
        .collect();

    (base_env.lexical_tokens.clone(), space_rules, lexical_rules)
}

#[derive(Debug)]
pub enum Error {
    BacktrackError(usize, String),
    CompileError(String),
    // ParseError(String),
}

impl std::error::Error for Error {}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        match self {
            Error::BacktrackError(i, m) => write!(f, "Syntax Error: {}: {}", i, m),
            Error::CompileError(m) => write!(f, "Compile Error: {}", m),
        }
    }
}

pub struct Parser {
    cursor: usize,
    ffp: usize,
    source: Vec<char>,
}

type ParseFn<T> = fn(&mut Parser) -> Result<T, Error>;

impl Parser {
    pub fn new(s: &str) -> Self {
        return Parser {
            cursor: 0,
            ffp: 0,
            source: s.chars().collect(),
        };
    }

    // GR: Grammar <- Spacing (Definition / LabelDefinition)+ EndOfFile
    pub fn parse_grammar(&mut self) -> Result<AST, Error> {
        self.parse_spacing()?;
        let defs = self.one_or_more(|p| {
            p.choice(vec![|p| p.parse_label_definition(), |p| {
                p.parse_definition()
            }])
        })?;
        self.parse_eof()?;
        Ok(AST::Grammar(defs))
    }

    // GR: Definition <- Identifier LEFTARROW Expression
    fn parse_definition(&mut self) -> Result<AST, Error> {
        let id = self.parse_identifier()?;
        self.expect('<')?;
        self.expect('-')?;
        self.parse_spacing()?;
        let expr = self.parse_expression()?;
        Ok(AST::Definition(id, Box::new(expr)))
    }

    // GR: LabelDefinition <- LABEL Identifier EQ Literal
    fn parse_label_definition(&mut self) -> Result<AST, Error> {
        self.expect_str("label")?;
        self.parse_spacing()?;
        let label = self.parse_identifier()?;
        self.expect('=')?;
        self.parse_spacing()?;
        let literal = self.parse_literal()?;
        Ok(AST::LabelDefinition(label, literal))
    }

    // GR: Expression <- Sequence (SLASH Sequence)*
    fn parse_expression(&mut self) -> Result<AST, Error> {
        let first = self.parse_sequence()?;
        let mut choices = vec![first];
        choices.append(&mut self.zero_or_more(|p| {
            p.expect('/')?;
            p.parse_spacing()?;
            p.parse_sequence()
        })?);
        Ok(if choices.len() == 1 {
            choices.remove(0)
        } else {
            AST::Choice(choices)
        })
    }

    // GR: Sequence <- Prefix*
    fn parse_sequence(&mut self) -> Result<AST, Error> {
        let seq = self.zero_or_more(|p| p.parse_prefix())?;
        Ok(AST::Sequence(if seq.is_empty() {
            vec![AST::Empty]
        } else {
            seq
        }))
    }

    // GR: Prefix <- (AND / NOT)? Labeled
    fn parse_prefix(&mut self) -> Result<AST, Error> {
        let prefix = self.choice(vec![
            |p| {
                p.expect_str("&")?;
                p.parse_spacing()?;
                Ok("&")
            },
            |p| {
                p.expect_str("!")?;
                p.parse_spacing()?;
                Ok("!")
            },
            |_| Ok(""),
        ]);
        let labeled = self.parse_labeled()?;
        Ok(match prefix {
            Ok("&") => AST::Not(Box::new(AST::Not(Box::new(labeled)))),
            Ok("!") => AST::Not(Box::new(labeled)),
            _ => labeled,
        })
    }

    // GR: Labeled <- Suffix Label?
    fn parse_labeled(&mut self) -> Result<AST, Error> {
        let suffix = self.parse_suffix()?;
        Ok(match self.parse_label() {
            Ok(label) => AST::Label(label, Box::new(suffix)),
            _ => suffix,
        })
    }

    // GR: Label   <- [^⇑] Identifier
    fn parse_label(&mut self) -> Result<String, Error> {
        self.choice(vec![|p| p.expect_str("^"), |p| p.expect_str("⇑")])?;
        self.parse_identifier()
    }

    // GR: Suffix  <- Primary (QUESTION / STAR / PLUS)?
    fn parse_suffix(&mut self) -> Result<AST, Error> {
        let primary = self.parse_primary()?;
        let suffix = self.choice(vec![
            |p| {
                p.expect_str("?")?;
                p.parse_spacing()?;
                Ok("?")
            },
            |p| {
                p.expect_str("*")?;
                p.parse_spacing()?;
                Ok("*")
            },
            |p| {
                p.expect_str("+")?;
                p.parse_spacing()?;
                Ok("+")
            },
            |_| Ok(""),
        ]);
        Ok(match suffix {
            Ok("?") => AST::Optional(Box::new(primary)),
            Ok("*") => AST::ZeroOrMore(Box::new(primary)),
            Ok("+") => AST::OneOrMore(Box::new(primary)),
            _ => primary,
        })
    }

    // GR: Primary <- Identifier !(LEFTARROW / (Identifier EQ))
    // GR:          / OPEN Expression CLOSE
    // GR:          / Literal / Class / DOT
    fn parse_primary(&mut self) -> Result<AST, Error> {
        self.choice(vec![
            |p| {
                let id = p.parse_identifier()?;
                p.not(|p| {
                    p.expect('<')?;
                    p.expect('-')?;
                    p.parse_spacing()
                })?;
                p.not(|p| {
                    p.parse_identifier()?;
                    p.expect('=')?;
                    p.parse_spacing()
                })?;
                Ok(AST::Identifier(id))
            },
            |p| {
                p.expect('(')?;
                p.parse_spacing()?;
                let expr = p.parse_expression()?;
                p.expect(')')?;
                p.parse_spacing()?;
                Ok(expr)
            },
            |p| Ok(AST::Str(p.parse_literal()?)),
            |p| Ok(AST::Choice(p.parse_class()?)),
            |p| {
                p.parse_dot()?;
                Ok(AST::Any)
            },
        ])
    }

    // GR: Identifier <- IdentStart IdentCont* Spacing
    // GR: IdentStart <- [a-zA-Z_]
    // GR: IdentCont <- IdentStart / [0-9]
    fn parse_identifier(&mut self) -> Result<String, Error> {
        let ident_start = self.choice(vec![
            |p| p.expect_range('a', 'z'),
            |p| p.expect_range('A', 'Z'),
            |p| p.expect('_'),
        ])?;
        let ident_cont = self.zero_or_more(|p| {
            p.choice(vec![
                |p| p.expect_range('a', 'z'),
                |p| p.expect_range('A', 'Z'),
                |p| p.expect_range('0', '9'),
                |p| p.expect('_'),
            ])
        })?;
        self.parse_spacing()?;
        let cont_str: String = ident_cont.into_iter().collect();
        let id = format!("{}{}", ident_start, cont_str);
        Ok(id)
    }

    // GR: Literal <- [’] (![’]Char)* [’] Spacing
    // GR:          / ["] (!["]Char)* ["] Spacing
    fn parse_literal(&mut self) -> Result<String, Error> {
        self.choice(vec![|p| p.parse_simple_quote(), |p| p.parse_double_quote()])
    }

    fn parse_simple_quote(&mut self) -> Result<String, Error> {
        self.expect('\'')?;
        let r = self
            .zero_or_more(|p| {
                p.not(|p| p.expect('\''))?;
                p.parse_char()
            })?
            .into_iter()
            .collect();
        self.expect('\'')?;
        self.parse_spacing()?;
        Ok(r)
    }

    // TODO: duplicated the above code as I can't pass the quote as a
    // parameter to a more generic function. The `zero_or_more` parser
    // and all the other parsers expect a function pointer, not a
    // closure.
    fn parse_double_quote(&mut self) -> Result<String, Error> {
        self.expect('"')?;
        let r = self
            .zero_or_more(|p| {
                p.not(|p| p.expect('"'))?;
                p.parse_char()
            })?
            .into_iter()
            .collect();
        self.expect('"')?;
        self.parse_spacing()?;
        Ok(r)
    }

    // GR: Class <- ’[’ (!’]’Range)* ’]’ Spacing
    fn parse_class(&mut self) -> Result<Vec<AST>, Error> {
        self.expect('[')?;
        let output = self.zero_or_more::<AST>(|p| {
            p.not(|pp| pp.expect(']'))?;
            p.parse_range()
        });
        self.expect(']')?;
        self.parse_spacing()?;
        output
    }

    // GR: Range <- Char ’-’ Char / Char
    fn parse_range(&mut self) -> Result<AST, Error> {
        self.choice(vec![
            |p| {
                let left = p.parse_char()?;
                p.expect('-')?;
                Ok(AST::Range(left, p.parse_char()?))
            },
            |p| Ok(AST::Char(p.parse_char()?)),
        ])
    }

    // GR: Char <- ’\\’ [nrt’"\[\]\\]
    // GR:       / ’\\’ [0-2][0-7][0-7]
    // GR:       / ’\\’ [0-7][0-7]?
    // GR:       / !’\\’ .
    fn parse_char(&mut self) -> Result<char, Error> {
        self.choice(vec![|p| p.parse_char_escaped(), |p| {
            p.parse_char_non_escaped()
        }])
    }

    // ’\\’ [nrt’"\[\]\\]
    fn parse_char_escaped(&mut self) -> Result<char, Error> {
        self.expect('\\')?;
        self.choice(vec![
            |p| {
                p.expect('n')?;
                Ok('\n')
            },
            |p| {
                p.expect('r')?;
                Ok('\r')
            },
            |p| {
                p.expect('t')?;
                Ok('\t')
            },
            |p| {
                p.expect('\'')?;
                Ok('\'')
            },
            |p| {
                p.expect('"')?;
                Ok('\"')
            },
            |p| {
                p.expect(']')?;
                Ok(']')
            },
            |p| {
                p.expect('[')?;
                Ok('[')
            },
            |p| {
                p.expect('\\')?;
                Ok('\\')
            },
            |p| {
                p.expect('\'')?;
                Ok('\'')
            },
            |p| {
                p.expect('"')?;
                Ok('"')
            },
        ])
    }

    // !’\\’ .
    fn parse_char_non_escaped(&mut self) -> Result<char, Error> {
        self.not(|p| p.expect('\\'))?;
        self.any()
    }

    // GR: DOT <- '.' Spacing
    fn parse_dot(&mut self) -> Result<char, Error> {
        let r = self.expect('.')?;
        self.parse_spacing()?;
        Ok(r)
    }

    // GR: Spacing <- (Space/ Comment)*
    fn parse_spacing(&mut self) -> Result<(), Error> {
        self.zero_or_more(|p| p.choice(vec![|p| p.parse_space(), |p| p.parse_comment()]))?;
        Ok(())
    }

    // GR: Comment <- ’#’ (!EndOfLine.)* EndOfLine
    fn parse_comment(&mut self) -> Result<(), Error> {
        self.expect('#')?;
        self.zero_or_more(|p| {
            p.not(|p| p.parse_eol())?;
            p.any()
        })?;
        self.parse_eol()
    }

    // GR: Space <- ’ ’ / ’\t’ / EndOfLine
    fn parse_space(&mut self) -> Result<(), Error> {
        self.choice(vec![
            |p| {
                p.expect(' ')?;
                Ok(())
            },
            |p| {
                p.expect('\t')?;
                Ok(())
            },
            |p| p.parse_eol(),
        ])
    }

    // EndOfLine <- ’\r\n’ / ’\n’ / ’\r’
    fn parse_eol(&mut self) -> Result<(), Error> {
        self.choice(vec![
            |p| {
                p.expect('\r')?;
                p.expect('\n')
            },
            |p| p.expect('\n'),
            |p| p.expect('\r'),
        ])?;
        Ok(())
    }

    // EndOfFile <- !.
    fn parse_eof(&mut self) -> Result<(), Error> {
        self.not(|p| p.current())?;
        Ok(())
    }

    fn choice<T>(&mut self, funcs: Vec<ParseFn<T>>) -> Result<T, Error> {
        let cursor = self.cursor;
        for func in &funcs {
            match func(self) {
                Ok(o) => return Ok(o),
                Err(_) => self.cursor = cursor,
            }
        }
        Err(self.err("CHOICE".to_string()))
    }

    fn not<T>(&mut self, func: ParseFn<T>) -> Result<(), Error> {
        let cursor = self.cursor;
        let out = func(self);
        self.cursor = cursor;
        match out {
            Err(_) => Ok(()),
            Ok(_) => Err(self.err("NOT".to_string())),
        }
    }

    fn one_or_more<T>(&mut self, func: ParseFn<T>) -> Result<Vec<T>, Error> {
        let mut output = vec![func(self)?];
        output.append(&mut self.zero_or_more::<T>(func)?);
        Ok(output)
    }

    fn zero_or_more<T>(&mut self, func: ParseFn<T>) -> Result<Vec<T>, Error> {
        let mut output = vec![];
        loop {
            match func(self) {
                Ok(ch) => output.push(ch),
                Err(e) => match e {
                    Error::BacktrackError(..) => break,
                    _ => return Err(e),
                },
            }
        }
        Ok(output)
    }

    fn expect_range(&mut self, a: char, b: char) -> Result<char, Error> {
        let current = self.current()?;
        if current >= a && current <= b {
            self.next();
            return Ok(current);
        }
        Err(self.err(format!(
            "Expected char between `{}' and `{}' but got `{}' instead",
            a, b, current
        )))
    }

    fn expect_str(&mut self, expected: &str) -> Result<String, Error> {
        for c in expected.chars() {
            self.expect(c)?;
        }
        Ok(expected.to_string())
    }

    fn expect(&mut self, expected: char) -> Result<char, Error> {
        let current = self.current()?;
        if current == expected {
            self.next();
            return Ok(current);
        }
        Err(self.err(format!(
            "Expected `{}' but got `{}' instead",
            expected, current
        )))
    }

    fn any(&mut self) -> Result<char, Error> {
        let current = self.current()?;
        self.next();
        Ok(current)
    }

    fn current(&mut self) -> Result<char, Error> {
        if !self.eof() {
            return Ok(self.source[self.cursor]);
        }
        Err(self.err("EOF".to_string()))
    }

    fn eof(&self) -> bool {
        self.cursor == self.source.len()
    }

    fn next(&mut self) {
        self.cursor += 1;

        if self.cursor > self.ffp {
            self.ffp = self.cursor;
        }
    }

    fn err(&mut self, msg: String) -> Error {
        Error::BacktrackError(self.ffp, msg)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn structure_empty() {
        let mut p = Parser::new(
            "A <- 'a' /
             B <- 'b'
            ",
        );
        let out = p.parse_grammar();

        assert!(out.is_ok());
        assert_eq!(
            AST::Grammar(vec![
                AST::Definition(
                    "A".to_string(),
                    Box::new(AST::Choice(vec![
                        AST::Sequence(vec![AST::Str("a".to_string())]),
                        AST::Sequence(vec![AST::Empty])
                    ]))
                ),
                AST::Definition(
                    "B".to_string(),
                    Box::new(AST::Sequence(vec![AST::Str("b".to_string())])),
                ),
            ]),
            out.unwrap()
        );
    }

    #[test]
    fn follows_1() {
        let mut c = Compiler::new();
        let out = c.compile_str(
            "
            A <- 'a'
            B <- 'b' 'k' 'l'
            C <- ('m' / 'n') 'o'
            TerminalAfterIdentifier    <- A ';'
            ChoiceAfterIdentifier      <- A ('a' 'x' / 'b' 'y' / 'c' 'z')
            IdentifierAfterIdentifier  <- A B ('k' / 'l')
            IdChoiceAfterIdentifier    <- A C ('k' / 'l')
            EOFAfterId                 <- A !.
            ForwardIdentifier          <- A After
            After                      <- '1' / '2'
            ",
        );

        // This should be how the First sets look for the above
        // grammar:
        //
        //     First(A) = {'JustATerminal'}
        //     First(B) = {'b'}
        //     First(C) = {'m' 'n'}
        //     First(TerminalAfterIdentifier)              = First(A)
        //     First(ChoiceAfterIdentifier)                = First(A)
        //        First(ChoiceAfterIdentifier + A)         = {'a' 'b' 'c'}
        //     First(IdentifierAfterIdentifier)            = First(A)
        //        First(IdentifierAfterIdentifier + A + B) = {'k' 'l'}
        //     First(IdChoiceAfterIdentifier)              = First(A)
        //        First(IdChoiceAfterIdentifier + A)       = First(C)
        //        First(IdChoiceAfterIdentifier + A + C)   = {'k' 'l'}
        //     First(ForwardIdentifier)                    = First(A)
        //        First(ForwardIdentifier + A)             = First(After)
        //     First(After) = {'1' '2'}
        //
        // These asserts are looking at the follow set for the call
        // site of the production A within productions `*Identifier`

        assert!(out.is_ok());
        let fns: HashMap<String, usize> = c
            .funcs
            .iter()
            .map(|(k, v)| (c.strings[*k].clone(), v.addr + 2)) // 2 for call+capture
            .collect();
        let p = c.program();

        assert_eq!(
            vec![";".to_string()],
            p.expected(fns["TerminalAfterIdentifier"])
        );
        assert_eq!(
            vec!["a".to_string(), "b".to_string(), "c".to_string()],
            p.expected(fns["ChoiceAfterIdentifier"])
        );
        assert_eq!(
            vec!["b".to_string()],
            p.expected(fns["IdentifierAfterIdentifier"])
        );
        assert_eq!(
            vec!["m".to_string(), "n".to_string()],
            p.expected(fns["IdChoiceAfterIdentifier"])
        );
        assert_eq!(
            vec!["1".to_string(), "2".to_string()],
            p.expected(fns["ForwardIdentifier"])
        );
    }

    #[test]
    fn follows_2() {
        let mut c = Compiler::new();
        let out = c.compile_str(
            "
            A <- B / C
            B <- 'b' / C
            C <- 'c'
            IDAfterIDWithChoiceWithIDs <- A A
            ",
        );

        assert!(out.is_ok());
        let fns: HashMap<String, usize> = c
            .funcs
            .iter()
            .map(|(k, v)| (c.strings[*k].clone(), v.addr + 2)) // 2 for call+capture
            .collect();
        let p = c.program();

        assert_eq!(
            vec!["b".to_string(), "c".to_string()],
            p.expected(fns["IDAfterIDWithChoiceWithIDs"])
        );
    }

    #[test]
    fn choice_pick_none() -> Result<(), Error> {
        let mut parser = Parser::new("e");
        let out = parser.choice(vec![
            |p| p.expect('a'),
            |p| p.expect('b'),
            |p| p.expect('c'),
            |p| p.expect('d'),
        ]);

        assert!(out.is_err());
        assert_eq!(0, parser.cursor);

        Ok(())
    }

    #[test]
    fn choice_pick_last() -> Result<(), Error> {
        let mut parser = Parser::new("d");
        let out = parser.choice(vec![
            |p| p.expect('a'),
            |p| p.expect('b'),
            |p| p.expect('c'),
            |p| p.expect('d'),
        ]);

        assert!(out.is_ok());
        assert_eq!(1, parser.cursor);

        Ok(())
    }

    #[test]
    fn choice_pick_first() -> Result<(), Error> {
        let mut parser = Parser::new("a");
        let out = parser.choice(vec![|p| p.expect('a')]);

        assert!(out.is_ok());
        assert_eq!(1, parser.cursor);

        Ok(())
    }

    #[test]
    fn not_success_on_err() -> Result<(), Error> {
        let mut parser = Parser::new("a");
        let out = parser.not(|p| p.expect('b'));

        assert!(out.is_ok());
        assert_eq!(0, parser.cursor);

        Ok(())
    }

    #[test]
    fn not_err_on_match() -> Result<(), Error> {
        let mut parser = Parser::new("a");
        let out = parser.not(|p| p.expect('a'));

        assert!(out.is_err());
        assert_eq!(0, parser.cursor);

        Ok(())
    }

    #[test]
    fn zero_or_more() -> Result<(), Error> {
        let mut parser = Parser::new("ab2");

        let prefix = parser.zero_or_more::<char>(|p| p.expect_range('a', 'z'))?;

        assert_eq!(vec!['a', 'b'], prefix);
        assert_eq!(2, parser.cursor);

        Ok(())
    }
}
