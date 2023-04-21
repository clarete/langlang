// vm.rs --- parsing machine
//
// This machine is capable of matching patterns in strings.  The
// patterns themselves are expressed in high level text based
// languages and get compiled to programs that can be executed by this
// machine.  This module has nothing to do with how patterns get
// compiled to programs, but how programs get executted as patterns.
//
#[cfg(debug_assertions)]
use crate::format;
use std::collections::HashMap;

#[derive(Clone, Debug, PartialEq, PartialOrd)]
pub enum Value {
    Char(char),
    Bool(bool),
    I32(i32),
    U32(u32),
    I64(i64),
    U64(u64),
    F32(f32),
    F64(f64),
    String(String),
    List(Vec<Value>),
    Node {
        name: String,
        items: Vec<Value>,
    },
    Error {
        label: String,
        message: Option<String>,
    },
}

#[derive(Clone, Debug)]
pub enum ContainerType {
    List,
    Node,
}

#[derive(Clone, Debug, PartialEq)]
pub enum CaptureType {
    /// Don't capture anything
    Disabled,
    /// Capture just the values, don't wrap them in a named node
    Unwrapped,
    /// The default type of capture, wrap captured values in a node
    /// named after the production in which the capture happened
    Wrapped,
}

#[derive(Clone, Debug)]
pub enum Instruction {
    Halt,

    // lexical
    Any,
    Char(char),
    Span(char, char),
    String(usize),

    // control flow
    Choice(usize),
    ChoiceP(usize),
    Commit(usize),
    CommitB(usize),
    Fail,
    FailTwice,
    PartialCommit(usize),
    BackCommit(usize),
    // TestChar,
    // TestAny,
    Jump(usize),
    Call(usize, usize, CaptureType),
    CallB(usize, usize, CaptureType),
    Return(CaptureType),
    Throw(usize),

    // container (list, map, node, etc)
    Open,
    Close(ContainerType),

    // value capture
    CapPush,
    CapPop,
    CapCommit,

    // semantic actions
    SemPushVal(Value),
    SemPushVar(usize),
    SemPushList(usize),
    SemPop,
    SemNegative,
    SemPositive,
    SemAdd,
    SemSub,
    SemMul,
    SemDiv,
    SemCallPrim(usize, usize),
}

impl std::fmt::Display for Instruction {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        match self {
            Instruction::Halt => write!(f, "halt"),
            Instruction::Any => write!(f, "any"),
            Instruction::Fail => write!(f, "fail"),
            Instruction::FailTwice => write!(f, "failtwice"),
            Instruction::Return(cap) => write!(f, "return {:?}", cap),
            Instruction::Char(c) => write!(f, "char {:?}", c),
            Instruction::String(i) => write!(f, "string {:?}", i),
            Instruction::Span(a, b) => write!(f, "span {:?} {:?}", a, b),
            Instruction::Choice(o) => write!(f, "choice {:?}", o),
            Instruction::ChoiceP(o) => write!(f, "choicep {:?}", o),
            Instruction::Commit(o) => write!(f, "commit {:?}", o),
            Instruction::CommitB(o) => write!(f, "commitb {:?}", o),
            Instruction::PartialCommit(u) => write!(f, "partialcommit {:?}", u),
            Instruction::BackCommit(u) => write!(f, "backcommit {:?}", u),
            Instruction::Jump(addr) => write!(f, "jump {:?}", addr),
            Instruction::Throw(label) => write!(f, "throw {:?}", label),
            Instruction::Call(addr, k, cap) => write!(f, "call {:?} {:?} {:?}", addr, k, cap),
            Instruction::CallB(addr, k, cap) => write!(f, "callb {:?} {:?} {:?}", addr, k, cap),
            Instruction::Open => write!(f, "open"),
            Instruction::Close(t) => write!(f, "close({:?})", t),
            Instruction::CapPush => write!(f, "cappush"),
            Instruction::CapPop => write!(f, "cappop"),
            Instruction::CapCommit => write!(f, "capcommit"),
            Instruction::SemPushVal(v) => write!(f, "sempushval {:?}", v),
            Instruction::SemPushVar(i) => write!(f, "sempushvar {:?}", i),
            Instruction::SemPushList(l) => write!(f, "sempushlist {}", l),
            Instruction::SemPop => write!(f, "sempop"),
            Instruction::SemNegative => write!(f, "semnegative"),
            Instruction::SemPositive => write!(f, "sempositive"),
            Instruction::SemAdd => write!(f, "semadd"),
            Instruction::SemSub => write!(f, "semsub"),
            Instruction::SemMul => write!(f, "semmul"),
            Instruction::SemDiv => write!(f, "semdiv"),
            Instruction::SemCallPrim(n, a) => write!(f, "semcallprim {}/{}", n, a),
        }
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Error {
    // Backtracking
    Fail,
    // Initial state of left recursive call
    LeftRec,
    // Something was incorrectly indexed
    Index(String),
    // Error matching the input (ffp, expected)
    Matching(usize, String),
    // End of file
    EOF,
    // Generic error for semantic actions
    SemActionErr(String),
    // wrong number of parameters for a sem action
    SemActionArity(String),
    // Signals type error in semantic action
    SemActionTypeMismatch(String),
    // A given sem action does not exist
    SemActionNotFound(String),
}

#[derive(Clone, Debug)]
pub struct Program {
    // Map with keys as the position of the first instruction of each
    // production in the source code, and values as the index in the
    // strings table where the name of the production can be found.
    identifiers: HashMap<usize, usize>,
    // Map with IDs of labels as keys and the ID of the messages
    // associated with the labels as values
    labels: HashMap<usize, usize>,
    // Map from label IDs to tuples with two things: address of the
    // recovery expression and its precedence level
    recovery: HashMap<usize, (usize, usize)>,
    // Table with strings that refer to either error labels or
    // production identifiers.  IDs are assigned in the order they are
    // requested.
    strings: Vec<String>,
    // Array of instructions that get executed by the virtual machine
    code: Vec<Instruction>,
}

impl Program {
    pub fn new(
        identifiers: HashMap<usize, usize>,
        labels: HashMap<usize, usize>,
        recovery: HashMap<usize, (usize, usize)>,
        strings: Vec<String>,
        code: Vec<Instruction>,
    ) -> Self {
        Program {
            identifiers,
            labels,
            recovery,
            strings,
            code,
        }
    }

    pub fn label(&self, id: usize) -> String {
        self.strings[id].clone()
    }

    pub fn label_message(&self, id: usize) -> Option<String> {
        if let Some(msg_id) = self.labels.get(&id) {
            return Some(self.strings[*msg_id].clone());
        }
        None
    }

    pub fn identifier(&self, address: usize) -> String {
        match self.identifiers.get(&address) {
            None => "?".to_string(),
            Some(id) => self.strings[*id].clone(),
        }
    }

    pub fn string_at(&self, id: usize) -> String {
        self.strings[id].clone()
    }
}

fn instruction_to_string(p: &Program, instruction: &Instruction, pc: usize) -> String {
    match instruction {
        Instruction::String(i) => format!("str {:?}", p.strings[*i]),
        Instruction::Call(addr, k, cap) => {
            format!("call {:?} {}, {:?}", p.identifier(pc + addr), k, cap)
        }
        Instruction::CallB(addr, k, cap) => {
            format!("callb {:?} {} {:?}", p.identifier(pc - addr), k, cap)
        }
        Instruction::Throw(label) => format!("throw {:?}", p.strings[*label]),
        Instruction::SemCallPrim(id, arity) => format!("prim {:?}/{}", p.strings[*id], arity),
        instruction => format!("{}", instruction),
    }
}

impl std::fmt::Display for Program {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        writeln!(f, "Labels: {}", self.labels.len())?;
        for (i, label) in self.labels.iter().enumerate() {
            write!(f, "  {:#04} ", i)?;
            writeln!(f, "{:?}", label)?;
        }
        writeln!(f, "Strings: {}", self.strings.len())?;
        for (i, string) in self.strings.iter().enumerate() {
            write!(f, "  {:#04} ", i)?;
            writeln!(f, "{:?}", string)?;
        }
        writeln!(f, "Addresses")?;
        for (address, id) in self.identifiers.iter() {
            write!(f, "  {:#04} ", address)?;
            writeln!(f, "{:?}", self.string_at(*id))?;
        }
        writeln!(f, "Code: {}", self.code.len())?;
        for (i, instruction) in self.code.iter().enumerate() {
            write!(f, "  {:#04} ", i)?;
            writeln!(f, "{}", instruction_to_string(self, instruction, i))?;
        }
        write!(f, "")
    }
}

#[derive(Debug, PartialEq)]
enum StackFrameType {
    Backtrack,
    Call,
    List,
}

#[derive(Debug)]
struct StackFrame {
    ftype: StackFrameType,
    program_counter: usize,       // pc
    cursor: usize,                // s
    result: Result<usize, Error>, // X
    address: usize,               // pc+l
    precedence: usize,            // k
    predicate: bool,
    recovery_label: Option<usize>,
    list: Option<Vec<Value>>,
}

impl StackFrame {
    fn new_backtrack(cursor: usize, pc: usize, predicate: bool) -> Self {
        StackFrame {
            ftype: StackFrameType::Backtrack,
            program_counter: pc,
            cursor,
            predicate,
            // fields not used for backtrack frames
            recovery_label: None,
            address: 0,
            precedence: 0,
            result: Ok(0),
            list: None,
        }
    }

    fn new_call(
        pc: usize,
        address: usize,
        precedence: usize,
        recovery_label: Option<usize>,
    ) -> Self {
        StackFrame {
            ftype: StackFrameType::Call,
            program_counter: pc,
            cursor: 0,
            result: Err(Error::Fail),
            predicate: false,
            list: None,
            address,
            precedence,
            recovery_label,
        }
    }

    fn new_lrcall(
        cursor: usize,
        pc: usize,
        address: usize,
        precedence: usize,
        recovery_label: Option<usize>,
    ) -> Self {
        StackFrame {
            ftype: StackFrameType::Call,
            program_counter: pc,
            result: Err(Error::LeftRec),
            predicate: false,
            list: None,
            cursor,
            address,
            precedence,
            recovery_label,
        }
    }

    fn new_list(cursor: usize, pc: usize, list: Vec<Value>) -> Self {
        StackFrame {
            ftype: StackFrameType::List,
            program_counter: pc,
            cursor,
            list: Some(list),
            // fields not used for list frames
            recovery_label: None,
            predicate: false,
            address: 0,
            precedence: 0,
            result: Ok(0),
        }
    }
}

#[derive(Debug, Default)]
struct CapStackFrame {
    index: usize,
    values: Vec<Value>,
}

// #[derive(Debug)]
// enum Status {
//     Halt,
//     Continue,
// }

// pc+l: production address
//    s: subject, cursor index
type LeftRecTableKey = (usize, usize);

#[derive(Debug)]
struct LeftRecTableEntry {
    // cursor (s'): subject in left recursive call
    cursor: Result<usize, Error>,
    // precedence: precedence level in recursive call
    precedence: usize,
    // counter of how deep a recursive call is
    bound: usize,
}

impl LeftRecTableEntry {
    /// Create a new LeftRecTableEntry with a custom `precedence`.
    /// The other fields of the struct receive default values.  The
    /// `cursor` is set with a LeftRec error, and the `bound` is set
    /// to zero.
    fn new(precedence: usize) -> Self {
        Self {
            cursor: Err(Error::LeftRec),
            bound: 0,
            precedence,
        }
    }
}

pub type PrimFunc = fn(vm: &mut VM, arity: usize) -> Result<Option<Value>, Error>;

#[inline]
fn _pop_u32(vm: &mut VM) -> Result<u32, Error> {
    match vm.capstkpopval()? {
        Value::U32(n) => Ok(n),
        _ => Err(Error::SemActionTypeMismatch("Not a u32".to_string())),
    }
}

#[inline]
fn _pop_text(vm: &mut VM) -> Result<String, Error> {
    match vm.capstkpopval()? {
        Value::String(s) => Ok(s),
        Value::Char(c) => Ok(c.to_string()),
        _ => Err(Error::SemActionTypeMismatch("Not a text".to_string())),
    }
}

#[inline]
fn _recursive_join_str(values: &Vec<Value>) -> Result<String, Error> {
    let mut output = String::new();
    for value in values {
        match value {
            Value::Char(c) => output.push(*c),
            Value::String(s) => output.push_str(s),
            Value::List(i) => output.push_str(&_recursive_join_str(i)?),
            Value::Node { name: _, items } => output.push_str(&_recursive_join_str(items)?),
            v => {
                return Err(Error::SemActionTypeMismatch(format!(
                    "can only join Char or String, received a {:?}",
                    v
                )))
            }
        }
    }
    Ok(output)
}

#[inline]
fn _check_arity(name: &str, expected: usize, actual: usize) -> Result<(), Error> {
    if expected != actual {
        return Err(Error::SemActionArity(format!(
            "{}() expects {} arguments, {} given",
            name, expected, actual,
        )));
    }
    Ok(())
}

fn prim_skip(vm: &mut VM, arity: usize) -> Result<Option<Value>, Error> {
    _check_arity("skip", 0, arity)?;
    let top = vm.capstktop_mut()?;
    top.values.clear();
    Ok(None)
}

fn prim_text(vm: &mut VM, arity: usize) -> Result<Option<Value>, Error> {
    _check_arity("text", 0, arity)?;
    let v = &vm.capstktop()?.values;
    Ok(Some(Value::String(_recursive_join_str(v)?)))
}

fn prim_join(vm: &mut VM, arity: usize) -> Result<Option<Value>, Error> {
    _check_arity("join", 1, arity)?;
    println!("join");
    let items = match vm.capstkpopval()? {
        Value::List(items) => items,
        v => {
            return Err(Error::SemActionTypeMismatch(format!(
                "join()'s param should be a list, got {:?} instead",
                v
            )));
        }
    };
    Ok(Some(Value::String(_recursive_join_str(&items)?)))
}

fn prim_unwrap(vm: &mut VM, arity: usize) -> Result<Option<Value>, Error> {
    _check_arity("unwrap", 1, arity)?;
    let items = match vm.capstkpopval()? {
        Value::Node { name: _, items } => items,
        v => {
            return Err(Error::SemActionTypeMismatch(format!(
                "unwrap()'s param should be a node, got {:?} instead",
                v
            )));
        }
    };
    if items.len() != 1 {
        return Err(Error::SemActionTypeMismatch(format!(
            "unwrap()'s param should be a node with a single value, got {} instead",
            items.len()
        )));
    }
    Ok(items.first().cloned())
}

fn prim_i32(vm: &mut VM, arity: usize) -> Result<Option<Value>, Error> {
    _check_arity("i32", 2, arity)?;
    let base = _pop_u32(vm)?;
    let input = _pop_text(vm)?;
    match i32::from_str_radix(&input, base) {
        Ok(value) => Ok(Some(Value::I32(value))),
        Err(err) => Err(Error::SemActionErr(err.to_string())),
    }
}

fn prim_u32(vm: &mut VM, arity: usize) -> Result<Option<Value>, Error> {
    _check_arity("u32", 2, arity)?;
    let base = _pop_u32(vm)?;
    let input = _pop_text(vm)?;
    match u32::from_str_radix(&input, base) {
        Ok(value) => Ok(Some(Value::U32(value))),
        Err(err) => Err(Error::SemActionErr(err.to_string())),
    }
}

fn prim_i64(vm: &mut VM, arity: usize) -> Result<Option<Value>, Error> {
    _check_arity("i64", 2, arity)?;
    let base = _pop_u32(vm)?;
    let input = _pop_text(vm)?;
    match i64::from_str_radix(&input, base) {
        Ok(value) => Ok(Some(Value::I64(value))),
        Err(err) => Err(Error::SemActionErr(err.to_string())),
    }
}

fn prim_u64(vm: &mut VM, arity: usize) -> Result<Option<Value>, Error> {
    _check_arity("u64", 2, arity)?;
    let base = _pop_u32(vm)?;
    let input = _pop_text(vm)?;
    match u64::from_str_radix(&input, base) {
        Ok(value) => Ok(Some(Value::U64(value))),
        Err(err) => Err(Error::SemActionErr(err.to_string())),
    }
}

fn prim_f32(vm: &mut VM, arity: usize) -> Result<Option<Value>, Error> {
    _check_arity("f32", 1, arity)?;
    let input = _pop_text(vm)?;
    match input.parse::<f32>() {
        Ok(value) => Ok(Some(Value::F32(value))),
        Err(err) => Err(Error::SemActionErr(err.to_string())),
    }
}

fn prim_f64(vm: &mut VM, arity: usize) -> Result<Option<Value>, Error> {
    _check_arity("f64", 1, arity)?;
    let input = _pop_text(vm)?;
    match input.parse::<f64>() {
        Ok(value) => Ok(Some(Value::F64(value))),
        Err(err) => Err(Error::SemActionErr(err.to_string())),
    }
}

pub struct VM<'a> {
    // Cursor position at the input
    cursor: usize,
    // Farther Failure Position
    ffp: usize,
    // Vector of instructions and tables with literal values
    program: &'a Program,
    // Cursor within the program
    program_counter: usize,
    // Stack of both backtrack and call frames
    stack: Vec<StackFrame>,
    // last call frame
    call_frames: Vec<usize>,
    // Memoized position of left recursive results
    lrmemo: HashMap<LeftRecTableKey, LeftRecTableEntry>,
    // Where values returned from successful match operations are stored
    captures: Vec<CapStackFrame>,
    // boolean flag that remembers if the VM is within a predicate
    within_predicate: bool,
    // stack for manipulating values while running semantic actions
    value_stack: Vec<Value>,
    // primitive functions
    prims: HashMap<String, PrimFunc>,
}

impl<'a> VM<'a> {
    pub fn new(program: &'a Program) -> Self {
        VM {
            program,
            ffp: 0,
            cursor: 0,
            program_counter: 0,
            stack: vec![],
            call_frames: vec![],
            lrmemo: HashMap::new(),
            captures: vec![],
            value_stack: vec![],
            within_predicate: false,
            prims: HashMap::from([
                ("skip".to_string(), prim_skip as PrimFunc),
                ("text".to_string(), prim_text as PrimFunc),
                ("join".to_string(), prim_join as PrimFunc),
                ("i32".to_string(), prim_i32 as PrimFunc),
                ("i64".to_string(), prim_i64 as PrimFunc),
                ("u32".to_string(), prim_u32 as PrimFunc),
                ("u64".to_string(), prim_u64 as PrimFunc),
                ("f32".to_string(), prim_f32 as PrimFunc),
                ("f64".to_string(), prim_f64 as PrimFunc),
                ("unwrap".to_string(), prim_unwrap as PrimFunc),
            ]),
        }
    }

    fn advance_cursor(&mut self) -> Result<(), Error> {
        let cursor = self.cursor + 1;
        if cursor > self.ffp {
            self.ffp = cursor;
        }
        self.cursor = cursor;
        Ok(())
    }

    // stack management

    fn stktop(&self) -> Result<usize, Error> {
        if self.call_frames.is_empty() {
            return Err(Error::Index("peek_call_frame".to_string()));
        }
        Ok(self.call_frames[self.call_frames.len() - 1])
    }

    fn stkpeek_mut(&mut self) -> Result<&mut StackFrame, Error> {
        let idx = self.stktop()?;
        Ok(&mut self.stack[idx])
    }

    fn stkpeek(&self) -> Result<&StackFrame, Error> {
        let idx = self.stktop()?;
        Ok(&self.stack[idx])
    }

    fn stkpush(&mut self, frame: StackFrame) {
        if frame.ftype == StackFrameType::Call {
            self.call_frames.push(self.stack.len());
        }
        self.stack.push(frame);
    }

    fn stkpop(&mut self) -> Result<StackFrame, Error> {
        let frame = self
            .stack
            .pop()
            .ok_or_else(|| Error::Index("pop_stack_frame".to_string()))?;
        if frame.ftype == StackFrameType::Call {
            self.call_frames
                .pop()
                .ok_or_else(|| Error::Index("pop_call_frame".to_string()))?;
        }
        if frame.predicate {
            self.within_predicate = false;
        }
        Ok(frame)
    }

    // functions for capturing matched values

    fn capstktop(&mut self) -> Result<&CapStackFrame, Error> {
        if self.captures.is_empty() {
            panic!("capstktop() tried to peek at empty capture vector");
        }
        let idx = self.captures.len() - 1;
        Ok(&self.captures[idx])
    }

    fn capstktop_mut(&mut self) -> Result<&mut CapStackFrame, Error> {
        if self.captures.is_empty() {
            panic!("capstktop_mut() tried to peek at empty capture vector");
        }
        let idx = self.captures.len() - 1;
        Ok(&mut self.captures[idx])
    }

    fn capstkpush(&mut self) {
        self.captures.push(CapStackFrame::default());
    }

    fn capstkpop(&mut self, n: String) -> Result<CapStackFrame, Error> {
        self.captures
            .pop()
            .ok_or_else(|| Error::Index(format!("pop_capture_{}", n)))
    }

    fn capstkpopval(&mut self) -> Result<Value, Error> {
        let v = self
            .value_stack
            .pop()
            .ok_or_else(|| Error::Index("capstkpopval".to_string()))?;
        Ok(v)
    }

    /// pushes a new value onto the frame on top of the capture stack
    fn capture(&mut self, v: Value) -> Result<(), Error> {
        if self.within_predicate {
            return Ok(());
        }
        self.capstktop_mut()?.values.push(v);
        Ok(())
    }

    fn capture_flatten(&mut self, address: usize, items: Vec<Value>) -> Result<(), Error> {
        let name = self.program.identifier(address);
        match &items[..] {
            [] => {}
            [Value::Node { name: n, .. }] if *n == name && items.len() == 1 => {
                self.capture(items[0].clone())?;
            }
            _ => {
                self.capture(Value::Node { name, items })?;
            }
        }
        Ok(())
    }

    fn capture_node(
        &mut self,
        address: usize,
        capframe: CapStackFrame,
        capture_type: CaptureType,
    ) -> Result<(), Error> {
        match capture_type {
            CaptureType::Disabled => {}
            CaptureType::Wrapped => {
                // base case for regular rules returning what's inside the
                // capture frame that was just popped
                let items = capframe.values;
                if !items.is_empty() {
                    let name = self.program.identifier(address);
                    self.capture(Value::Node { name, items })?;
                }
            }
            CaptureType::Unwrapped => {
                for v in capframe.values {
                    self.capture(v)?;
                }
            }
        }
        Ok(())
    }

    fn capture_value(&mut self, v: Value) -> Result<(), Error> {
        self.value_stack.push(v);
        Ok(())
    }

    /// mark all values captured on the top of the stack as commited
    fn commit_captures(&mut self) -> Result<(), Error> {
        let top = self.capstktop_mut()?;
        let (idx, len) = (top.index, top.values.len());
        top.index = len;
        if idx != len {
            self.dbg_captures()?;
        }
        Ok(())
    }

    // evaluation

    pub fn run_str(&mut self, input: &str) -> Result<Option<Value>, Error> {
        let source = input.chars().map(Value::Char).collect::<Vec<Value>>();
        self.run(source)
    }

    pub fn run(&mut self, input: Vec<Value>) -> Result<Option<Value>, Error> {
        let mut source = input;
        self.capstkpush();
        loop {
            self.dbg_instruction();
            match self.program.code[self.program_counter] {
                Instruction::Halt => break,

                // Terminal Matchers
                Instruction::Any => {
                    self.program_counter += 1;
                    if self.cursor >= source.len() {
                        self.fail(Error::EOF)?;
                        continue;
                    }
                    self.capture(source[self.cursor].clone())?;
                    self.advance_cursor()?;
                }
                Instruction::Char(expected) => {
                    self.program_counter += 1;
                    if self.cursor >= source.len() {
                        self.fail(Error::EOF)?;
                        continue;
                    }
                    let current = &source[self.cursor];
                    if current != &Value::Char(expected) {
                        self.fail(Error::Matching(self.ffp, expected.to_string()))?;
                        continue;
                    }
                    self.capture(current.clone())?;
                    self.advance_cursor()?;
                }
                Instruction::Span(start, end) => {
                    self.program_counter += 1;
                    if self.cursor >= source.len() {
                        self.fail(Error::EOF)?;
                        continue;
                    }
                    let current = &source[self.cursor];
                    if current >= &Value::Char(start) && current <= &Value::Char(end) {
                        self.capture(current.clone())?;
                        self.advance_cursor()?;
                        continue;
                    }
                    self.fail(Error::Matching(self.ffp, format!("[{}-{}]", start, end)))?;
                }
                Instruction::String(id) => {
                    self.program_counter += 1;
                    if self.cursor >= source.len() {
                        self.fail(Error::EOF)?;
                        continue;
                    }
                    let expected = self.program.string_at(id);
                    match &source[self.cursor] {
                        Value::String(s) if s == &expected => {
                            self.capture(Value::String(expected))?;
                            self.advance_cursor()?;
                            continue;
                        }
                        _ => {
                            let mut expected_chars = expected.chars();
                            match loop {
                                let current_char = match expected_chars.next() {
                                    None => break Ok(()),
                                    Some(c) => c,
                                };
                                if self.cursor >= source.len() {
                                    break Err(Error::EOF);
                                }
                                if source[self.cursor] != Value::Char(current_char) {
                                    break Err(Error::Matching(self.ffp, expected.clone()));
                                }
                                self.advance_cursor()?;
                            } {
                                Ok(()) => self.capture(Value::String(expected))?,
                                Err(e) => self.fail(e)?,
                            }
                        }
                    }
                }

                // Control flow
                Instruction::Choice(offset) => {
                    self.commit_captures()?;
                    self.stkpush(StackFrame::new_backtrack(
                        self.cursor,
                        self.program_counter + offset,
                        false,
                    ));
                    self.program_counter += 1;
                }
                Instruction::ChoiceP(offset) => {
                    self.commit_captures()?;
                    self.stkpush(StackFrame::new_backtrack(
                        self.cursor,
                        self.program_counter + offset,
                        true,
                    ));
                    self.program_counter += 1;
                    self.within_predicate = true;
                }
                Instruction::Commit(offset) => {
                    self.stkpop()?;
                    self.program_counter += offset;
                }
                Instruction::CommitB(offset) => {
                    self.stkpop()?;
                    self.program_counter -= offset;
                }
                Instruction::PartialCommit(offset) => {
                    let idx = self.stack.len() - 1;
                    let mut f = &mut self.stack[idx];
                    f.cursor = self.cursor;
                    // always subtracts: this opcode is currently only
                    // used when compiling the star operator (*),
                    // which always needs to send the program counter
                    // backwards.
                    self.program_counter -= offset;
                }
                Instruction::BackCommit(offset) => {
                    let f = self.stkpop()?;
                    self.cursor = f.cursor;
                    self.program_counter += offset;
                }
                Instruction::Fail => {
                    self.fail(Error::Fail)?;
                }
                Instruction::FailTwice => {
                    self.stkpop()?;
                    self.fail(Error::Fail)?;
                }
                Instruction::Jump(index) => {
                    self.program_counter = index;
                }
                Instruction::Call(offset, precedence, ref capture) => {
                    self.inst_call(self.program_counter + offset, precedence, None)?;
                }
                Instruction::CallB(offset, precedence, ref capture) => {
                    self.inst_call(self.program_counter - offset, precedence, None)?;
                }
                Instruction::Return(ref capture_type) => {
                    self.inst_return(capture_type.clone())?;
                }

                // Error Reporting/Recovery
                Instruction::Throw(label) => {
                    if self.within_predicate {
                        self.program_counter += 1;
                        self.fail(Error::Fail)?;
                    } else {
                        let message = self.program.label(label);
                        match self.program.recovery.get(&label) {
                            None => return Err(Error::Matching(self.ffp, message)),
                            Some((addr, precedence)) => {
                                self.inst_call(*addr, *precedence, Some(label))?
                            }
                        }
                    }
                }

                // Data Structure Matching
                Instruction::Open => {
                    self.program_counter += 1;
                    match &source[self.cursor] {
                        Value::List(ref items) => {
                            self.capstkpush();
                            self.stkpush(StackFrame::new_list(
                                self.cursor,
                                self.program_counter,
                                source.to_vec(),
                            ));
                            source = items.to_vec();
                            self.cursor = 0;
                        }
                        Value::Node { name, items } => {
                            self.capstkpush();
                            self.stkpush(StackFrame::new_list(
                                self.cursor,
                                self.program_counter,
                                source.to_vec(),
                            ));
                            let mut tmp = vec![Value::String(name.clone())];
                            tmp.extend(items.to_vec());
                            source = tmp;
                            self.cursor = 0;
                        }
                        _ => self.fail(Error::Matching(self.ffp, "Not a list".to_string()))?,
                    }
                }
                Instruction::Close(ref container_type) => {
                    self.program_counter += 1;
                    let capsframe = self.capstkpop("close".to_string())?;
                    self.capture(match container_type {
                        ContainerType::List => Value::List(capsframe.values),
                        ContainerType::Node => Value::Node {
                            name: match &capsframe.values[0] {
                                Value::String(s) => s.clone(),
                                _ => panic!("node name must be a string"),
                            },
                            items: capsframe.values[1..].to_vec(),
                        },
                    })?;
                    let frame = self.stkpop()?;
                    self.cursor = frame.cursor + 1;
                    source = frame
                        .list
                        .ok_or_else(|| Error::Index("pop_frame_list".to_string()))?;
                }

                // Capture Stack
                Instruction::CapPush => {
                    self.program_counter += 1;
                    if !self.within_predicate {
                        self.capstkpush();
                    }
                }
                Instruction::CapPop => {
                    self.program_counter += 1;
                    if !self.within_predicate {
                        let values = self.capstkpop("cap_pop".to_string())?.values;
                        for v in values {
                            self.capture(v)?;
                        }
                    }
                }
                Instruction::CapCommit => {
                    self.program_counter += 1;
                    if !self.within_predicate {
                        self.commit_captures()?;
                    }
                }

                // Semantic Actions
                Instruction::SemPushVal(ref v) => {
                    self.program_counter += 1;
                    self.capture_value(v.clone())?;
                }
                Instruction::SemPushVar(i) => {
                    self.program_counter += 1;
                    let top = self.capstktop()?;
                    let v = top.values[i].clone();
                    self.capture_value(v)?;
                }
                Instruction::SemPushList(len) => {
                    self.program_counter += 1;
                    let mut v = Vec::with_capacity(len);
                    for _ in 0..len {
                        let value = self.capstkpopval()?;
                        v.push(value);
                    }
                    let rev = v.into_iter().rev().collect();
                    self.capture_value(Value::List(rev))?;
                }
                Instruction::SemPop => {
                    self.program_counter += 1;
                    self.capstktop_mut()?.values.clear();
                    if let Some(v) = self.value_stack.pop() {
                        self.capture(v)?;
                    }
                }
                Instruction::SemNegative => {
                    self.program_counter += 1;
                    match self.capstkpopval()? {
                        Value::I64(v) => self.capture_value(Value::I64(-v))?,
                        x => {
                            return Err(Error::SemActionTypeMismatch(format!(
                                "Value `{:?}` isn't an integer",
                                x
                            )));
                        }
                    };
                }
                Instruction::SemPositive => {
                    self.program_counter += 1;
                    let v = self.capstkpopval()?;
                    self.capture_value(pos_val(v)?)?;
                }
                Instruction::SemAdd => {
                    self.program_counter += 1;
                    let b = self.capstkpopval()?;
                    let a = self.capstkpopval()?;
                    self.capture_value(add_val(a, b)?)?;
                }
                Instruction::SemSub => {
                    self.program_counter += 1;
                    let b = self.capstkpopval()?;
                    let a = self.capstkpopval()?;
                    self.capture_value(sub_val(a, b)?)?;
                }
                Instruction::SemMul => {
                    self.program_counter += 1;
                    // let b = self.capstkpopval()?;
                    // let a = self.capstkpopval()?;
                    // self.capture_value(mul_val(a, b)?)?;
                }
                Instruction::SemDiv => {
                    self.program_counter += 1;
                    // let b = self.capstkpopval()?;
                    // let a = self.capstkpopval()?;
                    // self.capture_value(div_val(a, b)?)?;
                }
                Instruction::SemCallPrim(id, arity) => {
                    self.program_counter += 1;
                    let prim_name = self.program.string_at(id);
                    let result = match self.prims.get(&prim_name) {
                        None => return Err(Error::SemActionNotFound(prim_name)),
                        Some(f) => f(self, arity),
                    };
                    if let Some(value) = result? {
                        self.capture_value(value)?;
                    }
                }
            }
        }

        if !self.captures.is_empty() {
            self.dbg_captures()?;
            Ok(self.capstkpop("final".to_string())?.values.pop())
        } else {
            Ok(None)
        }
    }

    fn inst_call(
        &mut self,
        address: usize,
        precedence: usize,
        recovery_label: Option<usize>,
    ) -> Result<(), Error> {
        // There is no precedence level set, which means this is *not*
        // a left recursive call.  So all we need to do is to push a
        // new frame for both the capture and the backtrack/call stack
        // and set the program counter appropriately
        if precedence == 0 {
            self.capstkpush();
            self.stkpush(StackFrame::new_call(
                self.program_counter + 1,
                address,
                precedence,
                recovery_label,
            ));
            self.program_counter = address;
            return Ok(());
        }

        // from this point on, we're handling left recursive calls.
        let cursor = self.cursor;
        let key = (address, cursor);
        match self.lrmemo.get(&key) {
            // When there isn't a memoised leftrec entry, it means
            // that it's a left recursive call with bound zero (0), so
            // we push a new frame in both the capture and the regular
            // backtrack/call stack, point the program counter to
            // where the function being called is and move on.
            None => {
                self.dbg("- lvar.{{1, 2}}");
                self.capstkpush();
                self.stkpush(StackFrame::new_lrcall(
                    cursor,
                    self.program_counter + 1,
                    address,
                    precedence,
                    recovery_label,
                ));
                self.program_counter = address;
                self.lrmemo.insert(key, LeftRecTableEntry::new(precedence));
            }
            // if there is already a leftrec entry in the memoization
            // table, it means that we're hitting a left recursive
            // call.  If the previous call's bound was zero (0) or if
            // the precedence of the current call is lower than the
            // one in the table, then it fails the call.  Otherwise,
            // we wrap the current set of captured values into a new
            // node and push it into the capture stack.
            Some(entry) => {
                if matches!(entry.cursor, Err(Error::LeftRec)) || precedence < entry.precedence {
                    self.dbg("- lvar.{{3,5}}");
                    self.fail(Error::Fail)?;
                } else {
                    self.dbg("- lvar.4");
                    self.program_counter += 1;
                    self.cursor = entry.cursor.clone()?;
                    let frame = self.capstktop_mut()?;
                    let values = frame.values.drain(..frame.index).collect();
                    frame.values.clear();
                    self.capture_flatten(address, values)?;
                    self.commit_captures()?;
                }
            }
        }
        self.dbg_captures()?;
        Ok(())
    }

    fn inst_return(&mut self, capture_type: CaptureType) -> Result<(), Error> {
        let cursor = self.cursor;
        let frame = self.stkpeek()?;
        let address = frame.address;

        if frame.precedence == 0 {
            let frame = self.stkpop()?;
            let capframe = self.capstkpop("capture_node_unwrapped".to_string())?;
            self.program_counter = frame.program_counter;

            // Recovery labels are captured as Error nodes
            if let Some(label_id) = frame.recovery_label {
                let label = self.program.identifier(address);
                let message = self.program.label_message(label_id);
                self.capture(Value::Error { label, message })?;
                return Ok(());
            }

            // Let the `capture_node` method deal with values captured
            // within the rule being returned from
            self.capture_node(address, capframe, capture_type)?;

            return Ok(());
        }

        // left recursive cases

        if matches!(frame.result, Err(Error::LeftRec)) || cursor > frame.result.clone()? {
            self.dbg("- {{lvar,inc}}.1");
            let mut frame = self.stkpeek_mut()?;
            frame.result = Ok(cursor);
            let frame_cursor = frame.cursor;
            let frame_precedence = frame.precedence;
            let key = (address, frame_cursor);
            let mut entry = &mut self.lrmemo.get_mut(&key).ok_or(Error::Fail)?;
            entry.cursor = Ok(cursor);
            entry.bound += 1;
            entry.precedence = frame_precedence;

            // call the same address we just returned from, to try to
            // increment the left recursive bound once more
            self.program_counter = address;
            self.cursor = frame_cursor;
            self.commit_captures()?;
            return Ok(());
        }
        self.dbg("- inc.3");
        let frame = self.stkpop()?;
        self.cursor = frame.result?;
        self.program_counter = frame.program_counter;
        let key = (frame.address, frame.cursor);
        self.lrmemo.remove(&key);
        let mut capframe = self.capstkpop("return_inc3".to_string())?;
        if capframe.index > 0 {
            let values = capframe.values.drain(..capframe.index).collect();
            capframe.values.clear();
            self.capture_flatten(address, values)?;
        }
        self.dbg_captures()?;
        Ok(())
    }

    fn fail(&mut self, error: Error) -> Result<(), Error> {
        self.dbg_instruction_fail();
        let frame = loop {
            match self.stkpop() {
                Err(_) => return Err(error),
                Ok(f) => {
                    if matches!(f.result, Err(Error::LeftRec)) {
                        self.dbg("- lvar.2");
                        let key = (f.address, f.cursor);
                        self.lrmemo.remove(&key);
                    }
                    if f.ftype == StackFrameType::Backtrack {
                        let top = self.capstktop_mut()?;
                        top.values.drain(top.index..);
                        self.dbg_captures()?;
                        break f;
                    } else {
                        self.capstkpop("fail".to_string())?;
                    }
                    if let Ok(result) = f.result {
                        if result > 0 {
                            self.dbg("- inc.2");
                            self.cursor = result;
                            break f;
                        }
                    }
                }
            }
        };
        self.program_counter = frame.program_counter;
        self.cursor = frame.cursor;
        Ok(())
    }

    fn dbg(&self, _m: &str) {
        #[cfg(debug_assertions)]
        {
            for _ in 0..self.call_frames.len() {
                eprint!("    ");
            }
            eprintln!("{}", _m);
        }
    }

    fn dbg_instruction(&self) {
        #[cfg(debug_assertions)]
        {
            eprint!("{:#04}, {:#04} ", self.program_counter, self.cursor);
            self.dbg(&instruction_to_string(
                self.program,
                &self.program.code[self.program_counter],
                self.program_counter,
            ));
        }
    }

    fn dbg_instruction_fail(&self) {
        #[cfg(debug_assertions)]
        {
            eprint!("{:#04}, {:#04} ", self.program_counter, self.cursor);
            for _ in 0..self.call_frames.len() {
                eprint!("    ");
            }
            eprintln!("fail");
        }
    }

    fn dbg_captures(&self) -> Result<(), Error> {
        #[cfg(debug_assertions)]
        {
            let top = if self.captures.is_empty() {
                return Err(Error::Index("pop_capture_dbg".to_string()));
            } else {
                &self.captures[self.captures.len() - 1]
            };
            if top.values.is_empty() {
                self.dbg(&format!("- captures[{}]: []", top.index));
                return Ok(());
            }
            self.dbg(&format!(
                "- captures[{}]: {:?}",
                top.index,
                top.values
                    .iter()
                    .map(format::value_fmt1)
                    .collect::<Vec<_>>()
                    .join(", ")
            ));
        }
        Ok(())
    }
}

fn add_val(a: Value, b: Value) -> Result<Value, Error> {
    match (a, b) {
        (Value::I32(a), Value::I32(b)) => Ok(Value::I32(a + b)),
        (Value::U32(a), Value::U32(b)) => Ok(Value::U32(a + b)),
        (Value::I64(a), Value::I64(b)) => Ok(Value::I64(a + b)),
        (Value::U64(a), Value::U64(b)) => Ok(Value::U64(a + b)),
        (a, b) => Err(Error::SemActionTypeMismatch(format!(
            "can't sum {:?} and {:?}",
            a, b,
        ))),
    }
}

fn sub_val(a: Value, b: Value) -> Result<Value, Error> {
    match (a, b) {
        (Value::I32(a), Value::I32(b)) => Ok(Value::I32(a - b)),
        (Value::U32(a), Value::U32(b)) => Ok(Value::U32(a - b)),
        (Value::I64(a), Value::I64(b)) => Ok(Value::I64(a - b)),
        (Value::U64(a), Value::U64(b)) => Ok(Value::U64(a - b)),
        (a, b) => Err(Error::SemActionTypeMismatch(format!(
            "can't subtract {:?} and {:?}",
            a, b,
        ))),
    }
}

fn pos_val(v: Value) -> Result<Value, Error> {
    match v {
        Value::I32(v) => Ok(Value::I32(v.abs())),
        Value::I64(v) => Ok(Value::I64(v.abs())),
        Value::U32(v) => Ok(Value::U32(v)),
        Value::U64(v) => Ok(Value::U64(v)),
        Value::F32(v) => Ok(Value::F32(v.abs())),
        Value::F64(v) => Ok(Value::F64(v.abs())),
        x => Err(Error::SemActionTypeMismatch(format!(
            "Value `{:?}` isn't a number",
            x
        ))),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    // (ch.1)
    //
    // s[i] = 'c'
    // -------------------
    // match 'c' s i = i+1
    #[test]
    fn ch_1() {
        // G <- 'a'
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Char('a'),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("a");

        assert!(result.is_ok());
        assert_eq!(1, vm.cursor);
    }

    // (ch.2)
    //
    // s[i] != 'c'
    // -------------------
    // match 'c' s i = nil
    #[test]
    fn ch_2() {
        // G <- 'a'
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Char('a'),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("b");

        assert!(result.is_err());
        assert_eq!(Error::Matching(0, "a".to_string()), result.unwrap_err());
    }

    // (span.1)
    //
    // 'a' <= s[i] <= 'z'
    // -------------------
    // match 'c' s i = i+1
    #[test]
    fn span_1() {
        // G <- [a-z]
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Span('a', 'z'),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("a");

        assert!(result.is_ok());
        assert_eq!(1, vm.cursor);
    }

    // (span.2)
    //
    // NOT 'a' <= s[i] <= 'z'
    // -------------------
    // match 'c' s i = nil
    #[test]
    fn span_2() {
        // G <- [a-z]
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Span('a', 'z'),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("9");

        assert!(result.is_err());
        assert_eq!(Error::Matching(0, "[a-z]".to_string()), result.unwrap_err());
    }

    // (any.1)
    //   i ≤ |s|
    // -----------------
    // match . s i = i+1
    #[test]
    fn any_1() {
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Any,
                Instruction::Any,
                Instruction::Any,
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("abcd");

        assert!(result.is_ok());
        assert_eq!(3, vm.cursor);
    }

    // (any.2)
    // i > |s|
    // -----------------
    // match . s i = nil
    #[test]
    fn any_2_eof() {
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Any,
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("");

        assert!(result.is_err());
        assert_eq!(Error::EOF, result.unwrap_err());
        // assert!(vm.cursor.is_err());
        //assert_eq!(vm.cursor.unwrap_err(), result.unwrap_err())
    }

    // (not.1)
    // match p s i = nil
    // -----------------
    // match !p s i = i
    #[test]
    fn not_1() {
        // G <- !'a'
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Choice(4),
                Instruction::Char('a'),
                Instruction::Commit(1),
                Instruction::Fail,
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("foo");

        assert!(result.is_ok());
        assert_eq!(0, vm.cursor);
        assert_eq!(0, vm.ffp);
    }

    // (not.2)
    // match p s i = i+j
    // ------------------
    // match !p s i = nil
    #[test]
    fn not_2() {
        // G <- !'f'
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Choice(4),
                Instruction::Char('f'),
                Instruction::Commit(1),
                Instruction::Fail,
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("foo");

        assert!(result.is_err());
        assert_eq!(Error::Fail, result.unwrap_err());
        // assert!(vm.cursor.is_err());
        assert_eq!(1, vm.ffp);
    }

    // (ord.1)
    // match p1 s i = nil    match p2 s i = nil
    // ----------------------------------------
    //         match p1 / p2 s i = nil
    #[test]
    fn ord_1() {
        // G <- 'a' / 'b'
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::Commit(2),
                Instruction::Char('b'),
                Instruction::Halt,
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("c");

        assert!(result.is_err());
        // currently shows the last error
        assert_eq!(Error::Matching(0, "b".to_string()), result.unwrap_err());
    }

    // (ord.2)
    // match p1 s i = i+j
    // -----------------------
    // match p1 / p2 s i = i+j (ord.2)
    #[test]
    fn ord_2() {
        // G <- 'a' / 'b'
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::Commit(2),
                Instruction::Char('b'),
                Instruction::Halt,
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("a");

        assert!(result.is_ok());
        assert_eq!(1, vm.cursor);
    }

    // (ord.3)
    // match p1 s i = nil    match p2 s i = i+k
    // ----------------------------------------
    //         match p1 / p2 s i = i+k
    #[test]
    fn ord_3() {
        // G <- 'a' / 'b'
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::Commit(2),
                Instruction::Char('b'),
                Instruction::Halt,
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("b");

        assert!(result.is_ok());
        assert_eq!(1, vm.cursor);
        assert_eq!(1, vm.ffp);
    }

    // (rep.1)
    // match p s i = i+j    match p∗ s i + j = i+j+k
    // ----------------------------------------------
    //            match p∗ s i = i+j+k
    #[test]
    fn rep_1() {
        // G <- 'a*'
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::CommitB(2),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("aab");

        assert!(result.is_ok());
        assert_eq!(2, vm.cursor);
        assert_eq!(2, vm.ffp);
    }

    // (rep.2)
    // match p s i = nil
    // -----------------
    // match p∗ s i = i
    #[test]
    fn rep_2() {
        // G <- 'a*'
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::CommitB(2),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("b");

        assert!(result.is_ok());
        assert_eq!(0, vm.cursor);
        assert_eq!(0, vm.ffp);
    }

    // (var.1)
    // match g g(Ak) s i = i+j
    // -----------------------
    // match g Ak s i = i+j
    #[test]
    fn var_1() {
        // G <- D '+' D
        // D <- '0' / '1'
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Jump(11),
                // G
                Instruction::Call(4, 0, CaptureType::Wrapped),
                Instruction::Char('+'),
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Return(CaptureType::Wrapped),
                // D
                Instruction::Choice(3),
                Instruction::Char('0'),
                Instruction::Commit(2),
                Instruction::Char('1'),
                Instruction::Return(CaptureType::Wrapped),
                Instruction::Halt,
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("1+1");

        assert!(result.is_ok());
        assert_eq!(3, vm.cursor);
        assert_eq!(3, vm.ffp);
    }

    // (var.2)
    // match g g(Ak) s i = nil
    // -----------------------
    //  match g Ak s i = nil
    #[test]
    fn var_2() {
        // G <- D '+' D
        // D <- '0' / '1'
        let program = Program {
            identifiers: HashMap::new(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Jump(11),
                // G
                Instruction::Call(4, 0, CaptureType::Wrapped),
                Instruction::Char('+'),
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Return(CaptureType::Wrapped),
                // D
                Instruction::Choice(3),
                Instruction::Char('0'),
                Instruction::Commit(2),
                Instruction::Char('1'),
                Instruction::Return(CaptureType::Wrapped),
                Instruction::Halt,
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("1+2");

        assert!(result.is_err());
        assert_eq!(Error::Matching(2, "1".to_string()), result.unwrap_err());
    }

    #[test]
    fn lrvar_err() {
        let identifiers = [(2, 0)].iter().cloned().collect();

        // G <- G '+' 'n' / 'n'
        let program = Program {
            identifiers,
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["E".to_string()],
            code: vec![
                Instruction::Call(2, 1, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Choice(5),
                Instruction::CallB(1, 1, CaptureType::Wrapped),
                Instruction::Char('+'),
                Instruction::Char('n'),
                Instruction::Commit(2),
                Instruction::Char('n'),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("321");

        assert!(result.is_err());
        // assert!(vm.cursor.is_err());
        assert_eq!(0, vm.ffp);
    }

    // (lvar.1)
    #[test]
    fn lrvar_1() {
        let identifiers = [(2, 0)].iter().cloned().collect();

        // G <- G '+' 'n' / 'n'
        let program = Program {
            identifiers,
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["E".to_string()],
            code: vec![
                Instruction::Call(2, 1, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::Choice(5),
                Instruction::CallB(1, 1, CaptureType::Wrapped),
                Instruction::Char('+'),
                Instruction::Char('n'),
                Instruction::Commit(2),
                Instruction::Char('n'),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("n+n+n");

        assert!(result.is_ok());
        assert_eq!(5, vm.cursor);
    }

    #[test]
    fn lrvar_2() {
        let identifiers = [(2, 0), (9, 1)].iter().cloned().collect();

        // E <- E:1 '+' E:2
        //    / D
        // D <- '0' / '1'
        let program = Program {
            identifiers,
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["E".to_string(), "D".to_string()],
            code: vec![
                Instruction::Call(2, 1, CaptureType::Wrapped),
                Instruction::Halt,
                // / E:1 '+' E:1
                Instruction::Choice(5),
                Instruction::CallB(1, 1, CaptureType::Wrapped),
                Instruction::Char('+'),
                Instruction::CallB(3, 1, CaptureType::Wrapped),
                Instruction::Commit(2),
                // / D
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Return(CaptureType::Wrapped),
                // D
                Instruction::Choice(3),
                Instruction::Char('0'),
                Instruction::Commit(2),
                Instruction::Char('1'),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("0+1");

        assert!(result.is_ok());
        assert_eq!(3, vm.cursor);
    }

    #[test]
    fn lrvar_3() {
        let identifiers = [(2, 0), (9, 1)].iter().cloned().collect();

        // E <- E:1 '+' E:2
        //    / E:2 '*' E:3
        //    / D
        // D <- '0' / '1'
        let program = Program {
            identifiers,
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["E".to_string(), "D".to_string()],
            code: vec![
                Instruction::Call(2, 1, CaptureType::Wrapped),
                Instruction::Halt,
                // / E:1 '+' E:2
                Instruction::Choice(5),
                Instruction::CallB(1, 1, CaptureType::Wrapped),
                Instruction::Char('+'),
                Instruction::CallB(3, 2, CaptureType::Wrapped),
                Instruction::Commit(7),
                // / E:2 '*' E:2
                Instruction::Choice(5),
                Instruction::CallB(6, 2, CaptureType::Wrapped),
                Instruction::Char('*'),
                Instruction::CallB(8, 3, CaptureType::Wrapped),
                Instruction::Commit(2),
                // / D
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Return(CaptureType::Wrapped),
                // D
                Instruction::Choice(3),
                Instruction::Char('0'),
                Instruction::Commit(2),
                Instruction::Char('1'),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("0+1*1");

        assert!(result.is_ok());
        assert_eq!(5, vm.cursor);
    }

    #[test]
    fn throw_1() {
        let identifiers = [(2, 0)].iter().cloned().collect();
        let labels = [(1, 1)].iter().cloned().collect();
        let strings = vec!["G".to_string(), "Not really b".to_string()];

        // G <- 'a' 'b'^l / 'c'
        let program = Program {
            identifiers,
            labels,
            strings,
            recovery: HashMap::new(),
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                // G
                Instruction::Choice(7),
                Instruction::Char('a'),
                Instruction::Choice(3),
                Instruction::Char('b'),
                Instruction::Commit(2),
                Instruction::Throw(1),
                Instruction::Commit(2),
                Instruction::Char('c'),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };
        let mut vm = VM::new(&program);
        let result = vm.run_str("axyz");

        assert!(result.is_err());
        assert_eq!(
            Error::Matching(1, "Not really b".to_string()),
            result.unwrap_err()
        );
    }

    #[test]
    fn str_1() {
        let program = Program {
            identifiers: [(2, 0)].iter().cloned().collect(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string(), "abacate".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::String(1),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("abacate");

        assert_eq!(7, vm.cursor);
        assert!(result.is_ok());
        let r = result.unwrap();
        assert!(r.is_some());
        assert_eq!(
            Value::Node {
                name: "G".to_string(),
                items: vec![Value::String("abacate".to_string())],
            },
            r.unwrap()
        );
    }

    #[test]
    fn str_2() {
        let program = Program {
            identifiers: [(2, 0)].iter().cloned().collect(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string(), "abacate".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::String(1),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("abacaxi");

        assert!(result.is_err());
        assert_eq!(
            Error::Matching(5, "abacate".to_string()),
            result.unwrap_err(),
        );
    }

    #[test]
    fn str_3() {
        let program = Program {
            identifiers: [(2, 0)].iter().cloned().collect(),
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string(), "abacate".to_string()],
            code: vec![
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                Instruction::String(1),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("a");

        assert!(result.is_err());
        assert_eq!(Error::EOF, result.unwrap_err());
    }

    #[test]
    fn capture_choice_0() {
        // G <- 'abacate' / 'abada'
        let identifiers = [(2, 0)].iter().cloned().collect();

        let program = Program {
            identifiers,
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                // Call to first production follwed by the end of the matching
                Instruction::Call(2, 0, CaptureType::Wrapped),
                Instruction::Halt,
                // Body of production G
                Instruction::Choice(9),
                Instruction::Char('a'),
                Instruction::Char('b'),
                Instruction::Char('a'),
                Instruction::Char('c'),
                Instruction::Char('a'),
                Instruction::Char('t'),
                Instruction::Char('e'),
                Instruction::Commit(6),
                Instruction::Char('a'),
                Instruction::Char('b'),
                Instruction::Char('a'),
                Instruction::Char('d'),
                Instruction::Char('a'),
                Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("abada");

        assert_eq!(5, vm.cursor);

        assert!(result.is_ok());
        let r = result.unwrap();
        assert!(r.is_some());
        assert_eq!(
            Value::Node {
                name: "G".to_string(),
                items: vec![
                    Value::Char('a'),
                    Value::Char('b'),
                    Value::Char('a'),
                    Value::Char('d'),
                    Value::Char('a'),
                ],
            },
            r.unwrap(),
        );
    }

    #[test]
    fn capture_choice_within_var() {
        // G <- D
        // D <- '0' / '1'
        let identifiers = [(2, 0), (4, 1)].iter().cloned().collect();
        let program = Program {
            identifiers,
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string(), "D".to_string()],
            code: vec![
                /* 00 */ Instruction::Call(2, 0, CaptureType::Wrapped),
                /* 01 */ Instruction::Halt,
                // G
                /* 02 */ Instruction::Call(2, 0, CaptureType::Wrapped),
                /* 03 */ Instruction::Return(CaptureType::Wrapped),
                // D
                /* 04 */ Instruction::Choice(3),
                /* 05 */ Instruction::Char('0'),
                /* 06 */ Instruction::Commit(2),
                /* 07 */ Instruction::Char('1'),
                /* 08 */ Instruction::Return(CaptureType::Wrapped),
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("1");

        assert_eq!(1, vm.cursor);

        assert!(result.is_ok());
        let r = result.unwrap();
        assert!(r.is_some());
        assert_eq!(
            Value::Node {
                name: "G".to_string(),
                items: vec![Value::Node {
                    name: "D".to_string(),
                    items: vec![Value::Char('1')],
                }],
            },
            r.unwrap(),
        );
    }
}
