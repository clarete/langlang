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
    Chr(char),
    Str(String),
    // I64(i64),
    // U64(u64),
    // F64(f64),
    List(Vec<Value>),
}

#[derive(Clone, Debug)]
pub enum Instruction {
    Halt,
    Any,
    Char(char),
    Span(char, char),
    Str(usize),
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
    Call(usize, usize),
    CallB(usize, usize),
    Return,
    Throw(usize),
    Open,
    Close,
    CapPush,
    CapPop,
    CapCommit,
}

impl std::fmt::Display for Instruction {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        match self {
            Instruction::Halt => write!(f, "halt"),
            Instruction::Any => write!(f, "any"),
            Instruction::Fail => write!(f, "fail"),
            Instruction::FailTwice => write!(f, "failtwice"),
            Instruction::Return => write!(f, "return"),
            Instruction::Char(c) => write!(f, "char {:?}", c),
            Instruction::Str(i) => write!(f, "str {:?}", i),
            Instruction::Span(a, b) => write!(f, "span {:?} {:?}", a, b),
            Instruction::Choice(o) => write!(f, "choice {:?}", o),
            Instruction::ChoiceP(o) => write!(f, "choicep {:?}", o),
            Instruction::Commit(o) => write!(f, "commit {:?}", o),
            Instruction::CommitB(o) => write!(f, "commitb {:?}", o),
            Instruction::PartialCommit(u) => write!(f, "partialcommit {:?}", u),
            Instruction::BackCommit(u) => write!(f, "backcommit {:?}", u),
            Instruction::Jump(addr) => write!(f, "jump {:?}", addr),
            Instruction::Throw(label) => write!(f, "throw {:?}", label),
            Instruction::Call(addr, k) => write!(f, "call {:?} {:?}", addr, k),
            Instruction::CallB(addr, k) => write!(f, "callb {:?} {:?}", addr, k),
            Instruction::Open => write!(f, "open"),
            Instruction::Close => write!(f, "close"),
            Instruction::CapPush => write!(f, "cappush"),
            Instruction::CapPop => write!(f, "cappop"),
            Instruction::CapCommit => write!(f, "capcommit"),
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
    Index,
    // Error matching the input (ffp, expected)
    Matching(usize, String),
    // End of file
    EOF,
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
    // Map with the IDs of labels as keys and the index of the first
    // line of the production with the recovery expression associated
    // to that label as values.
    recovery: HashMap<usize, usize>,
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
        recovery: HashMap<usize, usize>,
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
        match self.labels.get(&id) {
            None => self.strings[id].clone(),
            Some(sid) => self.strings[*sid].clone(),
        }
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
        Instruction::Str(i) => format!("str {:?}", p.strings[*i]),
        Instruction::Call(addr, k) => format!("call {:?} {}", p.identifier(pc + addr), k),
        Instruction::CallB(addr, k) => format!("callb {:?} {}", p.identifier(pc - addr), k),
        instruction => format!("{}", instruction),
    }
}

impl std::fmt::Display for Program {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        writeln!(f, "Labels: {:#?}", self.labels)?;
        writeln!(f, "Strings: {:#?}", self.strings)?;
        for (i, instruction) in self.code.iter().enumerate() {
            write!(f, "{:#03} ", i)?;
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
            address: 0,
            precedence: 0,
            result: Ok(0),
            list: None,
        }
    }

    fn new_call(pc: usize, address: usize, precedence: usize) -> Self {
        StackFrame {
            ftype: StackFrameType::Call,
            program_counter: pc,
            cursor: 0,
            result: Err(Error::Fail),
            predicate: false,
            list: None,
            address,
            precedence,
        }
    }

    fn new_lrcall(cursor: usize, pc: usize, address: usize, precedence: usize) -> Self {
        StackFrame {
            ftype: StackFrameType::Call,
            program_counter: pc,
            result: Err(Error::LeftRec),
            predicate: false,
            list: None,
            cursor,
            address,
            precedence,
        }
    }

    fn new_list(cursor: usize, pc: usize, list: Vec<Value>) -> Self {
        StackFrame {
            ftype: StackFrameType::List,
            program_counter: pc,
            cursor,
            list: Some(list),
            // fields not used for list frames
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

#[derive(Debug)]
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
            within_predicate: false,
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
            return Err(Error::Index);
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
        let frame = self.stack.pop().ok_or(Error::Index)?;
        if frame.ftype == StackFrameType::Call {
            self.call_frames.pop().ok_or(Error::Index)?;
        }
        if frame.predicate {
            self.within_predicate = false;
        }
        Ok(frame)
    }

    // functions for capturing matched values

    fn capstktop_mut(&mut self) -> Result<&mut CapStackFrame, Error> {
        if self.captures.is_empty() {
            return Err(Error::Index);
        }
        let idx = self.captures.len() - 1;
        Ok(&mut self.captures[idx])
    }

    fn capstkpush(&mut self) {
        self.captures.push(CapStackFrame::default())
    }

    fn capstkpop(&mut self) -> Result<CapStackFrame, Error> {
        self.captures.pop().ok_or(Error::Index)
    }

    /// pushes a new value onto the frame on top of the capture stack
    fn capture(&mut self, v: Value) -> Result<(), Error> {
        if self.within_predicate {
            return Ok(());
        }
        self.capstktop_mut()?.values.push(v);
        Ok(())
    }

    fn capture_flatten(&mut self, address: usize, children: Vec<Value>) -> Result<(), Error> {
        let name = self.program.identifier(address);
        match &children[..] {
            [] => {}
            [Value::List(ch)] if ch.len() == 2 && ch[0] == Value::Str(name) => {
                self.capture(children[0].clone())?;
            }
            _ => {
                let name = self.program.identifier(address);
                let items = vec![Value::Str(name), Value::List(children)];
                self.capture(Value::List(items))?;
            }
        }
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
        let source = input.chars().map(Value::Chr).collect::<Vec<Value>>();
        self.run(source)
    }

    pub fn run(&mut self, input: Vec<Value>) -> Result<Option<Value>, Error> {
        let mut source = input;
        self.capstkpush();
        loop {
            self.dbg_instruction();
            match self.program.code[self.program_counter] {
                Instruction::Halt => break,
                Instruction::CapPush => {
                    self.program_counter += 1;
                    if !self.within_predicate {
                        self.capstkpush();
                    }
                }
                Instruction::CapPop => {
                    self.program_counter += 1;
                    if !self.within_predicate {
                        for c in self.capstkpop()?.values {
                            self.capture(c)?;
                        }
                    }
                }
                Instruction::CapCommit => {
                    self.program_counter += 1;
                    self.commit_captures()?;
                }
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
                    if current != &Value::Chr(expected) {
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
                    if current >= &Value::Chr(start) && current <= &Value::Chr(end) {
                        self.capture(current.clone())?;
                        self.advance_cursor()?;
                        continue;
                    }
                    self.fail(Error::Matching(self.ffp, format!("[{}-{}]", start, end)))?;
                }
                Instruction::Str(id) => {
                    self.program_counter += 1;
                    if self.cursor >= source.len() {
                        self.fail(Error::EOF)?;
                        continue;
                    }
                    let expected = self.program.string_at(id);
                    match &source[self.cursor] {
                        Value::Str(s) if s == &expected => {
                            self.capture(Value::Str(expected))?;
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
                                if source[self.cursor] != Value::Chr(current_char) {
                                    break Err(Error::Matching(self.ffp, expected.clone()));
                                }
                                self.advance_cursor()?;
                            } {
                                Ok(()) => self.capture(Value::Str(expected))?,
                                Err(e) => self.fail(e)?,
                            }
                        }
                    }
                }
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
                Instruction::Call(offset, precedence) => {
                    self.inst_call(self.program_counter + offset, precedence)?;
                }
                Instruction::CallB(offset, precedence) => {
                    self.inst_call(self.program_counter - offset, precedence)?;
                }
                Instruction::Return => {
                    self.inst_return()?;
                }
                Instruction::Throw(label) => {
                    self.program_counter += 1;
                    if self.within_predicate {
                        self.fail(Error::Fail)?;
                    } else {
                        let message = self.program.label(label);
                        match self.program.recovery.get(&label) {
                            None => return Err(Error::Matching(self.ffp, message)),
                            Some(addr) => self.program_counter = *addr,
                        }
                    }
                }
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
                        _ => self.fail(Error::Matching(self.ffp, "Not a list".to_string()))?,
                    }
                }
                Instruction::Close => {
                    self.program_counter += 1;
                    let capsframe = self.capstkpop()?;
                    self.capture(Value::List(capsframe.values))?;
                    let frame = self.stkpop()?;
                    self.cursor = frame.cursor + 1;
                    source = frame.list.ok_or(Error::Index)?;
                }
            }
        }

        if !self.captures.is_empty() {
            self.dbg_captures()?;
            Ok(self.capstkpop()?.values.pop())
        } else {
            Ok(None)
        }
    }

    fn inst_call(&mut self, address: usize, precedence: usize) -> Result<(), Error> {
        let cursor = self.cursor;
        if precedence == 0 {
            self.capstkpush();
            self.stkpush(StackFrame::new_call(
                self.program_counter + 1,
                address,
                precedence,
            ));
            self.program_counter = address;
            return Ok(());
        }
        let key = (address, cursor);
        match self.lrmemo.get(&key) {
            None => {
                self.dbg("- lvar.{{1, 2}}");
                self.capstkpush();
                self.stkpush(StackFrame::new_lrcall(
                    cursor,
                    self.program_counter + 1,
                    address,
                    precedence,
                ));
                self.program_counter = address;
                self.lrmemo.insert(
                    key,
                    LeftRecTableEntry {
                        cursor: Err(Error::LeftRec),
                        bound: 0,
                        precedence,
                    },
                );
            }
            Some(entry) => {
                if matches!(entry.cursor, Err(Error::LeftRec)) || precedence < entry.precedence {
                    self.dbg("- lvar.{{3,5}}");
                    self.fail(Error::Fail)?;
                } else {
                    self.dbg("- lvar.4");
                    self.program_counter += 1;
                    self.cursor = entry.cursor.clone()?;
                    let frame = self.capstktop_mut()?;
                    let children: Vec<_> = frame.values.drain(..frame.index).collect();
                    frame.values.clear();
                    self.capture_flatten(address, children)?;
                    self.commit_captures()?;
                }
            }
        }
        self.dbg_captures()?;
        Ok(())
    }

    fn inst_return(&mut self) -> Result<(), Error> {
        let cursor = self.cursor;
        let frame = self.stkpeek()?;
        let address = frame.address;

        if frame.precedence == 0 {
            let frame = self.stkpop()?;
            self.program_counter = frame.program_counter;
            let children = self.capstkpop()?.values;
            if !children.is_empty() {
                let name = self.program.identifier(address);
                let items = vec![Value::Str(name), Value::List(children)];
                self.capture(Value::List(items))?;
            }
            return Ok(());
        }
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
        let mut capframe = self.capstkpop()?;
        if capframe.index > 0 {
            let children: Vec<_> = capframe.values.drain(..capframe.index).collect();
            capframe.values.clear();
            self.capture_flatten(address, children)?;
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
                        self.capstkpop()?;
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
        self.dbg(&instruction_to_string(
            self.program,
            &self.program.code[self.program_counter],
            self.program_counter,
        ));
    }

    fn dbg_instruction_fail(&self) {
        #[cfg(debug_assertions)]
        {
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
                return Err(Error::Index);
            } else {
                &self.captures[self.captures.len() - 1]
            };
            if top.values.is_empty() {
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Char('a'),
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Char('a'),
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Span('a', 'z'),
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Span('a', 'z'),
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Any,
                Instruction::Any,
                Instruction::Any,
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Any,
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Choice(4),
                Instruction::Char('a'),
                Instruction::Commit(1),
                Instruction::Fail,
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Choice(4),
                Instruction::Char('f'),
                Instruction::Commit(1),
                Instruction::Fail,
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::Commit(2),
                Instruction::Char('b'),
                Instruction::Halt,
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::Commit(2),
                Instruction::Char('b'),
                Instruction::Halt,
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::Commit(2),
                Instruction::Char('b'),
                Instruction::Halt,
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::CommitB(2),
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::CommitB(2),
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Jump(11),
                // G
                Instruction::Call(4, 0),
                Instruction::Char('+'),
                Instruction::Call(2, 0),
                Instruction::Return,
                // D
                Instruction::Choice(3),
                Instruction::Char('0'),
                Instruction::Commit(2),
                Instruction::Char('1'),
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Jump(11),
                // G
                Instruction::Call(4, 0),
                Instruction::Char('+'),
                Instruction::Call(2, 0),
                Instruction::Return,
                // D
                Instruction::Choice(3),
                Instruction::Char('0'),
                Instruction::Commit(2),
                Instruction::Char('1'),
                Instruction::Return,
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
                Instruction::Call(2, 1),
                Instruction::Halt,
                Instruction::Choice(5),
                Instruction::CallB(1, 1),
                Instruction::Char('+'),
                Instruction::Char('n'),
                Instruction::Commit(2),
                Instruction::Char('n'),
                Instruction::Return,
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
                Instruction::Call(2, 1),
                Instruction::Halt,
                Instruction::Choice(5),
                Instruction::CallB(1, 1),
                Instruction::Char('+'),
                Instruction::Char('n'),
                Instruction::Commit(2),
                Instruction::Char('n'),
                Instruction::Return,
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
                Instruction::Call(2, 1),
                Instruction::Halt,
                // / E:1 '+' E:1
                Instruction::Choice(5),
                Instruction::CallB(1, 1),
                Instruction::Char('+'),
                Instruction::CallB(3, 1),
                Instruction::Commit(2),
                // / D
                Instruction::Call(2, 0),
                Instruction::Return,
                // D
                Instruction::Choice(3),
                Instruction::Char('0'),
                Instruction::Commit(2),
                Instruction::Char('1'),
                Instruction::Return,
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
                Instruction::Call(2, 1),
                Instruction::Halt,
                // / E:1 '+' E:2
                Instruction::Choice(5),
                Instruction::CallB(1, 1),
                Instruction::Char('+'),
                Instruction::CallB(3, 2),
                Instruction::Commit(7),
                // / E:2 '*' E:2
                Instruction::Choice(5),
                Instruction::CallB(6, 2),
                Instruction::Char('*'),
                Instruction::CallB(8, 3),
                Instruction::Commit(2),
                // / D
                Instruction::Call(2, 0),
                Instruction::Return,
                // D
                Instruction::Choice(3),
                Instruction::Char('0'),
                Instruction::Commit(2),
                Instruction::Char('1'),
                Instruction::Return,
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
                Instruction::Call(2, 0),
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
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Str(1),
                Instruction::Return,
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("abacate");

        assert_eq!(7, vm.cursor);
        assert!(result.is_ok());
        let r = result.unwrap();
        assert!(r.is_some());
        assert_eq!(
            Value::List(vec![
                Value::Str("G".to_string()),
                Value::List(vec![Value::Str("abacate".to_string()),]),
            ]),
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Str(1),
                Instruction::Return,
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Str(1),
                Instruction::Return,
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
                Instruction::Call(2, 0),
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
                Instruction::Return,
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("abada");

        assert_eq!(5, vm.cursor);

        assert!(result.is_ok());
        let r = result.unwrap();
        assert!(r.is_some());
        assert_eq!(
            Value::List(vec![
                Value::Str("G".to_string()),
                Value::List(vec![
                    Value::Chr('a'),
                    Value::Chr('b'),
                    Value::Chr('a'),
                    Value::Chr('d'),
                    Value::Chr('a'),
                ])
            ]),
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
                /* 00 */ Instruction::Call(2, 0),
                /* 01 */ Instruction::Halt,
                // G
                /* 02 */ Instruction::Call(2, 0),
                /* 03 */ Instruction::Return,
                // D
                /* 04 */ Instruction::Choice(3),
                /* 05 */ Instruction::Char('0'),
                /* 06 */ Instruction::Commit(2),
                /* 07 */ Instruction::Char('1'),
                /* 08 */ Instruction::Return,
            ],
        };

        let mut vm = VM::new(&program);
        let result = vm.run_str("1");

        assert_eq!(1, vm.cursor);

        assert!(result.is_ok());
        let r = result.unwrap();
        assert!(r.is_some());
        assert_eq!(
            Value::List(vec![
                Value::Str("G".to_string()),
                Value::List(vec![Value::List(vec![
                    Value::Str("D".to_string()),
                    Value::List(vec![Value::Chr('1'),]),
                ]),]),
            ]),
            r.unwrap(),
        );
    }
}
