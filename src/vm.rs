// vm.rs --- parsing machine
//
// This machine is capable of matching patterns in strings.  The
// patterns themselves are expressed in high level text based
// languages and get compiled to programs that can be executed by this
// machine.  This module has nothing to do with how patterns get
// compiled to programs, but how programs get executted as patterns.
//

use log::debug;
use std::collections::HashMap;

#[derive(Debug, PartialEq)]
pub enum Value {
    Chr(char),
    Str(String),
    // I64(i64),
    // U64(u64),
    // F64(f64),
    Node { name: String, children: Vec<Value> },
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
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Error {
    Fail,
    LeftRec,
    Overflow,
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

impl std::fmt::Display for Program {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        writeln!(f, "Labels: {:#?}", self.labels)?;
        writeln!(f, "Strings: {:#?}", self.strings)?;

        for (i, instruction) in self.code.iter().enumerate() {
            write!(f, "{:#03} ", i)?;

            match instruction {
                Instruction::Halt => writeln!(f, "halt"),
                Instruction::Any => writeln!(f, "any"),
                Instruction::Fail => writeln!(f, "fail"),
                Instruction::FailTwice => writeln!(f, "failtwice"),
                Instruction::Return => writeln!(f, "return"),
                Instruction::Char(c) => writeln!(f, "char {:?}", c),
                Instruction::Str(i) => writeln!(f, "str {:?} {:?}", self.strings[*i], i),
                Instruction::Span(a, b) => writeln!(f, "span {:?} {:?}", a, b),
                Instruction::Choice(o) => writeln!(f, "choice {:?}", o),
                Instruction::ChoiceP(o) => writeln!(f, "choicep {:?}", o),
                Instruction::Commit(o) => writeln!(f, "commit {:?}", o),
                Instruction::CommitB(o) => writeln!(f, "commitb {:?}", o),
                Instruction::PartialCommit(u) => writeln!(f, "partialcommit {:?}", u),
                Instruction::BackCommit(u) => writeln!(f, "backcommit {:?}", u),
                Instruction::Jump(addr) => writeln!(f, "jump {:?}", addr),
                Instruction::Throw(label) => writeln!(f, "throw {:?}", label),
                Instruction::Call(addr, precedence) => {
                    let fn_addr = i + (*addr);
                    let fn_name = self.identifier(fn_addr);
                    writeln!(f, "call {:?} {:?} {:?}", fn_name, fn_addr, *precedence)
                }
                Instruction::CallB(addr, precedence) => {
                    let fn_addr = i - (*addr);
                    let fn_name = self.identifier(fn_addr);
                    writeln!(f, "callb {:?} {:?} {:?}", fn_name, fn_addr, *precedence)
                }
            }?;
        }
        write!(f, "")
    }
}

#[derive(Debug, PartialEq)]
enum StackFrameType {
    Backtrack,
    Call,
}

#[derive(Debug)]
struct StackFrame {
    ftype: StackFrameType,
    program_counter: usize,       // pc
    cursor: Result<usize, Error>, // s
    result: Result<usize, Error>, // X
    address: usize,               // pc+l
    precedence: usize,            // k
    last_capture_committed: usize,
    captures: Vec<Value>,
    predicate: bool,
}

impl StackFrame {
    fn new_backtrack(cursor: usize, pc: usize, capture: usize, predicate: bool) -> Self {
        StackFrame {
            ftype: StackFrameType::Backtrack,
            program_counter: pc,
            last_capture_committed: capture,
            cursor: Ok(cursor),
            // fields not used for backtrack frames
            address: 0,
            precedence: 0,
            result: Err(Error::Fail),
            captures: vec![],
            predicate,
        }
    }

    fn new_call(pc: usize, address: usize, precedence: usize) -> Self {
        StackFrame {
            ftype: StackFrameType::Call,
            program_counter: pc,
            cursor: Err(Error::Fail),
            result: Err(Error::Fail),
            address,
            precedence,
            last_capture_committed: 0,
            captures: vec![],
            predicate: false,
        }
    }

    fn new_lrcall(
        cursor: usize,
        pc: usize,
        address: usize,
        precedence: usize,
        capture: usize,
    ) -> Self {
        StackFrame {
            ftype: StackFrameType::Call,
            program_counter: pc,
            cursor: Ok(cursor),
            result: Err(Error::LeftRec),
            address,
            precedence,
            last_capture_committed: capture,
            captures: vec![],
            predicate: false,
        }
    }
}

// #[derive(Debug)]
// enum Status {
//     Halt,
//     Continue,
// }

// pc+l: production address
//    s: subject, cursor index
type LeftRecTableKey = (usize, usize);

//         s': subject in left recursive call
// precedence: precedence level in recursive call
#[derive(Debug)]
struct LeftRecTableEntry {
    cursor: Result<usize, Error>,
    precedence: usize,
    bound: usize,
}

#[derive(Debug)]
pub struct VM {
    // Cursor within input, error means matching failed
    cursor: Result<usize, Error>,
    // Farther Failure Position
    ffp: usize,
    // Input source
    source: Vec<char>,
    // Vector of instructions and tables with literal values
    program: Program,
    // Cursor within the program
    program_counter: usize,
    // Stack of both backtrack and call frames
    stack: Vec<StackFrame>,
    // last call frame
    call_frames: Vec<usize>,
    // Memoized position of left recursive results
    lrmemo: HashMap<LeftRecTableKey, LeftRecTableEntry>,
    // Where values returned from successful match operations are stored
    captures: Vec<Value>,
    // last capture commited
    last_capture_committed: usize,
    // boolean flag that remembers if the VM is within a predicate
    within_predicate: bool,
}

impl VM {
    pub fn new(program: Program) -> Self {
        VM {
            ffp: 0,
            cursor: Ok(0),
            source: vec![],
            program,
            program_counter: 0,
            stack: vec![],
            call_frames: vec![],
            lrmemo: HashMap::new(),
            captures: vec![],
            last_capture_committed: 0,
            within_predicate: false,
        }
    }

    fn advance_cursor(&mut self) -> Result<(), Error> {
        let cursor = self.cursor.clone()? + 1;
        if cursor > self.ffp {
            self.ffp = cursor;
        }
        self.cursor = Ok(cursor);
        Ok(())
    }

    // stack management

    fn stkpeek_mut(&mut self) -> Result<&mut StackFrame, Error> {
        if !self.call_frames.is_empty() {
            let idx = self.call_frames[self.call_frames.len() - 1];
            Ok(&mut self.stack[idx])
        } else {
            Err(Error::Overflow)
        }
    }

    fn stkpeek(&self) -> Result<&StackFrame, Error> {
        if !self.call_frames.is_empty() {
            let idx = self.call_frames[self.call_frames.len() - 1];
            Ok(&self.stack[idx])
        } else {
            Err(Error::Overflow)
        }
    }

    fn stkpush(&mut self, frame: StackFrame) {
        if frame.ftype == StackFrameType::Call {
            self.call_frames.push(self.stack.len());
        }
        self.stack.push(frame);
    }

    fn stkpop(&mut self) -> Result<StackFrame, Error> {
        let frame = self.stack.pop().ok_or(Error::Overflow)?;
        if frame.ftype == StackFrameType::Call {
            self.call_frames.pop().ok_or(Error::Overflow)?;
        }
        if frame.predicate {
            self.within_predicate = false;
        }
        Ok(frame)
    }

    // functions for capturing matched values

    fn capture(&mut self, v: Value) -> Result<(), Error> {
        if self.within_predicate {
            return Ok(())
        }
        if self.call_frames.is_empty() {
            self.captures.push(v);
        } else {
            self.stkpeek_mut()?.captures.push(v);
        }
        Ok(())
    }

    fn num_captures(&self) -> Result<usize, Error> {
        if self.call_frames.is_empty() {
            Ok(self.captures.len())
        } else {
            Ok(self.stkpeek()?.captures.len())
        }
    }

    fn commit_captures(&mut self) -> Result<(), Error> {
        if self.call_frames.is_empty() {
            self.last_capture_committed = self.captures.len();
        } else {
            let mut f = self.stkpeek_mut()?;
            f.last_capture_committed = f.captures.len();
        }
        Ok(())
    }

    // evaluation

    pub fn run(&mut self, input: &str) -> Result<Option<Value>, Error> {
        self.source = input.chars().collect();

        loop {
            if self.program_counter >= self.program.code.len() {
                return Err(Error::Overflow);
            }

            let (instruction, cursor) = match self.cursor {
                Ok(c) => (self.program.code[self.program_counter].clone(), c),
                Err(_) => (Instruction::Fail, 0),
            };

            debug!(
                "[{:?},{:?}] I: {:?}",
                self.program_counter, cursor, instruction
            );

            match instruction {
                Instruction::Halt => break,
                Instruction::Any => {
                    if cursor >= self.source.len() {
                        self.cursor = Err(Error::EOF);
                    } else {
                        self.capture(Value::Chr(self.source[cursor]))?;
                        self.advance_cursor()?;
                        self.program_counter += 1;
                    }
                }
                Instruction::Char(expected) => {
                    if cursor >= self.source.len() {
                        self.cursor = Err(Error::EOF);
                        continue;
                    }
                    let current = self.source[cursor];
                    if current != expected {
                        self.cursor = Err(Error::Matching(self.ffp, expected.to_string()));
                        continue;
                    }
                    self.capture(Value::Chr(self.source[cursor]))?;
                    self.advance_cursor()?;
                    self.program_counter += 1;
                }
                Instruction::Span(start, end) => {
                    if cursor >= self.source.len() {
                        self.cursor = Err(Error::EOF);
                        continue;
                    }
                    let current = self.source[cursor];
                    if current >= start && current <= end {
                        self.capture(Value::Chr(self.source[cursor]))?;
                        self.advance_cursor()?;
                        self.program_counter += 1;
                        continue;
                    }
                    self.cursor = Err(Error::Matching(self.ffp, format!("[{}-{}]", start, end)));
                }
                Instruction::Str(id) => {
                    let s = self.program.string_at(id);
                    let mut matches = 0;
                    for (i, expected) in s.chars().enumerate() {
                        let local_cursor = cursor + i;
                        if local_cursor >= self.source.len() {
                            break;
                        }
                        let current = self.source[local_cursor];
                        if current == expected {
                            self.advance_cursor()?;
                            matches += 1;
                        }
                    }
                    if matches == s.len() {
                        self.capture(Value::Str(s))?;
                    } else {
                        self.cursor = Err(Error::Matching(self.ffp, s));
                    }
                    self.program_counter += 1;
                }
                Instruction::Choice(offset) => {
                    self.stkpush(StackFrame::new_backtrack(
                        cursor,
                        self.program_counter + offset,
                        self.num_captures()?,
                        false,
                    ));
                    self.program_counter += 1;
                }
                Instruction::ChoiceP(offset) => {
                    self.stkpush(StackFrame::new_backtrack(
                        cursor,
                        self.program_counter + offset,
                        self.num_captures()?,
                        true,
                    ));
                    self.program_counter += 1;
                    self.within_predicate = true;
                }
                Instruction::Commit(offset) => {
                    self.stkpop()?;
                    self.program_counter += offset;
                    self.commit_captures()?;
                }
                Instruction::CommitB(offset) => {
                    self.stkpop()?;
                    self.program_counter -= offset;
                    self.commit_captures()?;
                }
                Instruction::PartialCommit(offset) => {
                    let idx = self.stack.len() - 1;
                    let ncaptures = self.num_captures()?;
                    let mut f = &mut self.stack[idx];
                    f.cursor = self.cursor.clone();
                    f.last_capture_committed = ncaptures;
                    // always subtracts: this opcode is currently only
                    // used when compiling the star operator (*),
                    // which always needs to send the program counter
                    // backwards.
                    self.program_counter -= offset;
                    self.commit_captures()?;
                }
                Instruction::BackCommit(offset) => {
                    let f = self.stkpop()?;
                    self.cursor = f.cursor;
                    self.program_counter += offset;
                }
                Instruction::Fail => {
                    self.fail()?;
                }
                Instruction::FailTwice => {
                    self.stkpop()?;
                    self.fail()?;
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
                        self.fail()?;
                    } else {
                        let message = self.program.label(label);
                        match self.program.recovery.get(&label) {
                            None => return Err(Error::Matching(self.ffp, message)),
                            Some(addr) => self.program_counter = *addr,
                        }
                    }
                }
            }
        }

        if self.captures.len() > 0 {
            Ok(Some(self.captures.remove(0)))
        } else {
            Ok(None)
        }
    }

    fn inst_call(&mut self, address: usize, precedence: usize) -> Result<(), Error> {
        debug!("       . call({:?})", self.program.identifier(address));
        let cursor = self.cursor.clone()?;
        if precedence == 0 {
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
                debug!("       . lvar.{{1, 2}}");
                self.stkpush(StackFrame::new_lrcall(
                    cursor,
                    self.program_counter + 1,
                    address,
                    precedence,
                    self.num_captures()?,
                ));
                self.program_counter = address;
                self.lrmemo.insert(
                    key,
                    LeftRecTableEntry {
                        cursor: Err(Error::LeftRec),
                        precedence,
                        bound: 0,
                    },
                );
            }
            Some(entry) => {
                if matches!(entry.cursor, Err(_)) || precedence < entry.precedence {
                    debug!("       . lvar.{{3,5}}");
                    self.fail()?;
                } else {
                    debug!("       . lvar.4");
                    self.program_counter += 1;
                    self.cursor = Ok(entry.cursor.clone()?);
                }
            }
        }
        Ok(())
    }

    fn inst_return(&mut self) -> Result<(), Error> {
        let cursor = self.cursor.clone()?;
        let mut frame = self.stkpeek_mut()?;
        let address = frame.address;

        if frame.precedence == 0 {
            let frame = self.stkpop()?;
            self.program_counter = frame.program_counter;
            self.capture(Value::Node {
                name: self.program.identifier(address),
                children: frame.captures,
            })?;
            return Ok(());
        }

        if matches!(frame.result, Err(Error::LeftRec)) || cursor > frame.result.clone()? {
            frame.result = Ok(cursor);
            let frame_cursor = frame.cursor.clone();
            let frame_precedence = frame.precedence;
            let key = (address, frame_cursor.clone()?);
            let mut entry = &mut self.lrmemo.get_mut(&key).ok_or(Error::Fail)?;
            entry.cursor = Ok(cursor);
            entry.bound += 1;
            entry.precedence = frame_precedence;

            self.cursor = frame_cursor;
            self.program_counter = address;
        } else {
            debug!("       . inc.3");
            let frame = self.stkpop()?;
            let pc = frame.program_counter;
            let key = (frame.address, frame.cursor?);
            self.cursor = frame.result;
            self.program_counter = pc;
            debug!("       . captures so far: {:#?}", frame.captures);
            self.lrmemo.remove(&key);
        }
        Ok(())
    }

    fn fail(&mut self) -> Result<(), Error> {
        debug!("       . fail");
        let error = match self.cursor.clone() {
            Err(e) => e,
            Ok(_) => Error::Fail,
        };
        let frame = loop {
            match self.stack.pop() {
                None => {
                    debug!("       . none");
                    self.cursor = Err(error.clone());
                    return Err(error);
                }
                Some(f) => {
                    debug!("       . pop {:#?}", f);
                    if let Ok(cursor) = f.cursor {
                        self.cursor = Ok(cursor);
                    }
                    if matches!(f.result, Err(Error::LeftRec)) {
                        let key = (f.address, f.cursor.clone()?);
                        debug!("       . lrfail: {:?}", key);
                        self.lrmemo.remove(&key);
                    }
                    if f.ftype == StackFrameType::Backtrack {
                        if f.predicate {
                            self.within_predicate = false;
                        }

                        let len = self.num_captures()?;
                        if self.call_frames.is_empty() {
                            let first = std::cmp::min(self.last_capture_committed, len);
                            self.captures.drain(first..len);
                        } else {
                            let first = std::cmp::min(f.last_capture_committed, len);
                            let top = self.stkpeek_mut()?;
                            top.captures.drain(first..len);
                        }

                        break f;
                    } else {
                        self.call_frames.pop();
                    }
                    if let Ok(result) = f.result {
                        if result > 0 {
                            debug!("       . inc.2");
                            self.cursor = Ok(result);
                            break f;
                        }
                    }
                }
            }
        };

        self.program_counter = frame.program_counter;

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

        let mut vm = VM::new(program);
        let result = vm.run("a");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(1, vm.cursor.unwrap());
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

        let mut vm = VM::new(program);
        let result = vm.run("b");

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

        let mut vm = VM::new(program);
        let result = vm.run("a");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(1, vm.cursor.unwrap());
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

        let mut vm = VM::new(program);
        let result = vm.run("9");

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

        let mut vm = VM::new(program);
        let result = vm.run("abcd");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(3, vm.cursor.unwrap());
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

        let mut vm = VM::new(program);
        let result = vm.run("");

        assert!(result.is_err());
        assert_eq!(Error::EOF, result.unwrap_err());
        assert!(vm.cursor.is_err());
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

        let mut vm = VM::new(program);
        let result = vm.run("foo");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(0, vm.cursor.unwrap());
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

        let mut vm = VM::new(program);
        let result = vm.run("foo");

        assert!(result.is_err());
        assert_eq!(Error::Fail, result.unwrap_err());
        assert!(vm.cursor.is_err());
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

        let mut vm = VM::new(program);
        let result = vm.run("c");

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

        let mut vm = VM::new(program);
        let result = vm.run("a");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(1, vm.cursor.unwrap());
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

        let mut vm = VM::new(program);
        let result = vm.run("b");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(1, vm.cursor.unwrap());
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

        let mut vm = VM::new(program);
        let result = vm.run("aab");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(2, vm.cursor.unwrap());
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

        let mut vm = VM::new(program);
        let result = vm.run("b");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(0, vm.cursor.unwrap());
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

        let mut vm = VM::new(program);
        let result = vm.run("1+1");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(3, vm.cursor.unwrap());
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

        let mut vm = VM::new(program);
        let result = vm.run("1+2");

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

        let mut vm = VM::new(program);
        let result = vm.run("321");

        assert!(result.is_err());
        assert!(vm.cursor.is_err());
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

        let mut vm = VM::new(program);
        let result = vm.run("n+n+n");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(5, vm.cursor.unwrap());
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

        let mut vm = VM::new(program);
        let result = vm.run("0+1");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(3, vm.cursor.unwrap());
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

        let mut vm = VM::new(program);
        let result = vm.run("0+1*1");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(5, vm.cursor.unwrap());
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
        let mut vm = VM::new(program);
        let result = vm.run("axyz");

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

        let mut vm = VM::new(program);
        let result = vm.run("abacate");

        assert!(vm.cursor.is_ok());
        assert_eq!(7, vm.cursor.unwrap());

        assert!(result.is_ok());
        let r = result.unwrap();
        assert!(r.is_some());
        assert_eq!(
            Value::Node {
                name: "G".to_string(),
                children: vec![Value::Str("abacate".to_string())],
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
                Instruction::Call(2, 0),
                Instruction::Halt,
                Instruction::Str(1),
                Instruction::Return,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run("abacaxi");

        assert!(result.is_err());
        assert_eq!(
            Error::Matching(5, "abacate".to_string()),
            result.unwrap_err(),
        );
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

        let mut vm = VM::new(program);
        let result = vm.run("abada");

        assert!(vm.cursor.is_ok());
        assert_eq!(5, vm.cursor.unwrap());

        assert!(result.is_ok());
        let r = result.unwrap();
        assert!(r.is_some());
        assert_eq!(
            Value::Node {
                name: "G".to_string(),
                children: vec![
                    Value::Chr('a'),
                    Value::Chr('b'),
                    Value::Chr('a'),
                    Value::Chr('d'),
                    Value::Chr('a'),
                ]
            },
            r.unwrap()
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

        let mut vm = VM::new(program);
        let result = vm.run("1");

        assert!(vm.cursor.is_ok());
        assert_eq!(1, vm.cursor.unwrap());

        assert!(result.is_ok());
        let r = result.unwrap();
        assert!(r.is_some());
        assert_eq!(
            Value::Node {
                name: "G".to_string(),
                children: vec![Value::Node {
                    name: "D".to_string(),
                    children: vec![Value::Chr('1')],
                }],
            },
            r.unwrap(),
        );
    }

    #[test]
    fn capture_choice_within_repeat() {
        // G <- ('abacate' / 'abada')+
        let identifiers = [(2, 0)].iter().cloned().collect();
        let program = Program {
            identifiers,
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["G".to_string()],
            code: vec![
                /* 00 */ Instruction::Call(2, 0),
                /* 01 */ Instruction::Halt,
                /* 02 */ Instruction::Choice(9),
                /* 03 */ Instruction::Char('a'),
                /* 04 */ Instruction::Char('b'),
                /* 05 */ Instruction::Char('a'),
                /* 06 */ Instruction::Char('c'),
                /* 07 */ Instruction::Char('a'),
                /* 08 */ Instruction::Char('t'),
                /* 09 */ Instruction::Char('e'),
                /* 10 */ Instruction::Commit(6),
                /* 11 */ Instruction::Char('a'),
                /* 12 */ Instruction::Char('b'),
                /* 13 */ Instruction::Char('a'),
                /* 14 */ Instruction::Char('d'),
                /* 15 */ Instruction::Char('a'),
                /* 16 */ Instruction::Choice(16),
                /* 17 */ Instruction::Choice(9),
                /* 18 */ Instruction::Char('a'),
                /* 19 */ Instruction::Char('b'),
                /* 20 */ Instruction::Char('a'),
                /* 21 */ Instruction::Char('c'),
                /* 22 */ Instruction::Char('a'),
                /* 23 */ Instruction::Char('t'),
                /* 24 */ Instruction::Char('e'),
                /* 25 */ Instruction::Commit(6),
                /* 26 */ Instruction::Char('a'),
                /* 27 */ Instruction::Char('b'),
                /* 28 */ Instruction::Char('a'),
                /* 29 */ Instruction::Char('d'),
                /* 30 */ Instruction::Char('a'),
                /* 31 */ Instruction::CommitB(15),
                /* 32 */ Instruction::Return,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run("abada");

        assert!(vm.cursor.is_ok());
        assert_eq!(5, vm.cursor.unwrap());

        assert!(result.is_ok());
        let r = result.unwrap();
        assert!(r.is_some());
        assert_eq!(
            Value::Node {
                name: "G".to_string(),
                children: vec![
                    Value::Chr('a'),
                    Value::Chr('b'),
                    Value::Chr('a'),
                    Value::Chr('d'),
                    Value::Chr('a'),
                ]
            },
            r.unwrap()
        );
    }

    #[test]
    fn capture_leftrec() {
        // E <- E:1 '+' E:2
        //    / E:2 '*' E:3
        //    / D
        // D <- [0-9]+
        let identifiers = [(2, 0), (14, 1)].iter().cloned().collect();
        let program = Program {
            identifiers,
            labels: HashMap::new(),
            recovery: HashMap::new(),
            strings: vec!["E".to_string(), "D".to_string()],
            code: vec![
                /* 00 */ Instruction::Call(2, 1),
                /* 01 */ Instruction::Halt,
                // / E:1 '+' E:2
                /* 02 */ Instruction::Choice(5),
                /* 03 */ Instruction::CallB(1, 1),
                /* 04 */ Instruction::Char('+'),
                /* 05 */ Instruction::CallB(3, 2),
                /* 06 */ Instruction::Commit(7),
                // / E:2 '*' E:2
                /* 07 */ Instruction::Choice(5),
                /* 08 */ Instruction::CallB(6, 2),
                /* 09 */ Instruction::Char('*'),
                /* 10 */ Instruction::CallB(8, 3),
                /* 11 */ Instruction::Commit(2),
                // / D
                /* 12 */ Instruction::Call(2, 0),
                /* 13 */ Instruction::Return,
                // D
                /* 14 */ Instruction::Span('0', '9'),
                /* 15 */ Instruction::Choice(3),
                /* 16 */ Instruction::Span('0', '9'),
                /* 17 */ Instruction::CommitB(2),
                /* 18 */ Instruction::Return,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run("12+34*56");

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(8, vm.cursor.unwrap());

        assert_eq!(
            vec![Value::Node {
                name: "E".to_string(),
                children: vec![
                    Value::Node {
                        name: "D".to_string(),
                        children: vec![Value::Chr('1'), Value::Chr('2')],
                    },
                    Value::Chr('+'),
                    Value::Node {
                        name: "E".to_string(),
                        children: vec![
                            Value::Node {
                                name: "D".to_string(),
                                children: vec![Value::Chr('3'), Value::Chr('4')],
                            },
                            Value::Chr('*'),
                            Value::Node {
                                name: "E".to_string(),
                                children: vec![Value::Node {
                                    name: "D".to_string(),
                                    children: vec![Value::Chr('5'), Value::Chr('6')],
                                }],
                            }
                        ],
                    }
                ],
            }],
            vm.captures
        );
    }
}
