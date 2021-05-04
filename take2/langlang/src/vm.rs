// vm.rs --- parsing machine
//
// This machine is capable of matching patterns in strings.  The
// patterns themselves are expressed in high level text based
// languages and get compiled to programs that can be executed by this
// machine.  This module has nothing to do with how patterns get
// compiled to programs, but how programs get executted as patterns.
//

use std::collections::HashMap;

#[derive(Debug, PartialEq)]
enum Value {
    Chr(char),
    // I64(i64),
    // U64(u64),
    // F64(f64),
    // Str(String),
    // Node(Vec<Value>),
}

#[derive(Clone, Debug)]
enum Instruction {
    Halt,
    Any,
    Char(char),
    // Span(char, char),
    Choice(usize),
    Commit(usize),
    CommitB(usize),
    Fail,
    // FailTwice,
    // PartialCommit,
    // BackCommit,
    // TestChar,
    // TestAny,
    Jump(usize),
    Call(usize, usize),
    CallB(usize, usize),
    Return,
    // Throw(usize),
}

#[derive(Clone, Debug, PartialEq)]
enum Error {
    Fail,
    LeftRec,
    Overflow,
    Matching(String),
    EOF,
}

#[derive(Debug)]
struct Program {
    code: Vec<Instruction>,
    // source_mapping: ...
}

#[derive(Debug)]
struct StackFrame {
    program_counter: usize,        // pc
    cursor: Result<usize, Error>,  // s
    result: Result<usize, Error>,  // X
    callable_index: Option<usize>, // pc+l
    precedence: Option<usize>,     // k
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
struct VM {
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
    // Stack of captured values
    captured_stack: Vec<Vec<Value>>,
    lrmemo: HashMap<LeftRecTableKey, LeftRecTableEntry>,
}

impl VM {
    fn new(program: Program) -> Self {
        return VM {
            ffp: 0,
            cursor: Ok(0),
            source: vec![],
            program: program,
            program_counter: 0,
            stack: vec![],
            captured_stack: vec![vec![]],
            lrmemo: HashMap::new(),
        };
    }

    fn advance_cursor(&mut self) -> Result<(), Error> {
        let cursor = self.cursor.clone()? + 1;
        if cursor > self.ffp {
            self.ffp = cursor;
        }
        self.cursor = Ok(cursor);
        Ok(())
    }

    fn stkpeek(&mut self) -> Result<&mut StackFrame, Error> {
        let len = self.stack.len();
        if len < 1 {
            return Err(Error::Overflow)?;
        }
        Ok(&mut self.stack[len - 1])
    }

    fn stkpop(&mut self) -> Result<StackFrame, Error> {
        self.stack.pop().ok_or(Error::Fail)
    }

    fn capture(&mut self, v: Value) {
        let i = self.captured_stack.len() - 1;
        self.captured_stack[i].push(v);
    }

    fn pop_captured(&mut self) {
        let i = self.captured_stack.len() - 1;
        self.captured_stack[i].pop();
    }

    fn run(&mut self, input: &String) -> Result<(), Error> {
        self.source = input.chars().collect();

        loop {
            if self.program_counter >= self.program.code.len() {
                return Err(Error::Overflow);
            }

            let (instruction, cursor) = match self.cursor {
                Ok(c) => (self.program.code[self.program_counter].clone(), c),
                Err(_) => (Instruction::Fail, 0),
            };

            println!("[{:?}] I: {:?}", cursor, instruction);

            match instruction {
                Instruction::Halt => break,
                Instruction::Any => {
                    if cursor >= self.source.len() {
                        self.cursor = Err(Error::EOF);
                    } else {
                        self.capture(Value::Chr(self.source[cursor]));
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
                        self.cursor = Err(Error::Matching(format!(
                            "Expected {}, but got {} instead",
                            expected, current,
                        )));
                        continue;
                    }
                    self.capture(Value::Chr(current));
                    self.advance_cursor()?;
                    self.program_counter += 1;
                }
                Instruction::Choice(offset) => {
                    self.stack.push(StackFrame {
                        program_counter: self.program_counter + offset,
                        cursor: Ok(cursor),
                        callable_index: None,
                        result: Err(Error::Fail),
                        precedence: None,
                    });
                    self.program_counter += 1;
                }
                Instruction::Commit(offset) => {
                    self.stack.pop();
                    self.program_counter += offset;
                }
                Instruction::CommitB(offset) => {
                    self.stack.pop();
                    self.program_counter -= offset;
                }
                Instruction::Fail => {
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
            }
        }
        Ok(())
    }

    fn inst_call(&mut self, callable_index: usize, precedence: usize) -> Result<(), Error> {
        let cursor = self.cursor.clone()?;
        if precedence == 0 {
            self.stack.push(StackFrame {
                program_counter: self.program_counter + 1,
                cursor: Err(Error::Fail),
                result: Err(Error::Fail),
                callable_index: Some(callable_index),
                precedence: Some(precedence),
            });
            self.program_counter = callable_index;
            return Ok(());
        }
        let key = (callable_index, cursor);
        match self.lrmemo.get(&key) {
            None => {
                // println!("lvar.{{1, 2}}");
                self.captured_stack.push(vec![]);
                self.stack.push(StackFrame {
                    program_counter: self.program_counter + 1,
                    cursor: Ok(cursor),
                    result: Err(Error::LeftRec),
                    callable_index: Some(callable_index),
                    precedence: Some(precedence),
                });
                self.program_counter = callable_index;
                self.lrmemo.insert(
                    key,
                    LeftRecTableEntry {
                        cursor: Err(Error::Matching("LEFTREC".to_string())),
                        precedence: precedence,
                        bound: 0,
                    },
                );
                //self.fail()?;
            }
            Some(entry) => {
                match entry.cursor {
                    Err(_) => {
                        // println!("lvar.{{3, 5}}.1");
                        self.fail()?;
                    },
                    Ok(cursor) => {
                        if precedence < entry.precedence {
                            // println!("lvar.{{3, 5}}.2");
                            self.fail()?;
                        } else {
                            // println!("lvar.4");
                            self.cursor = Ok(cursor);
                            self.program_counter += 1;
                        }
                    }
                }
            }
        }
        Ok(())
    }

    fn inst_return(&mut self) -> Result<(), Error> {
        let cursor = self.cursor.clone()?;
        let mut frame = self.stkpeek()?;

        if let Some(precedence) = frame.precedence {
            if precedence == 0 {
                self.program_counter = frame.program_counter;
                self.stkpop()?;
                return Ok(());
            }
        }

        if frame.result.is_err() || cursor > frame.result.clone()?  {
            if let Some(callable_index) = frame.callable_index {
                // println!("{{lvar, inc}}.1");

                frame.result = Ok(cursor);

                let frame_cursor = frame.cursor.clone();
                let key = (callable_index, frame_cursor.clone().unwrap());
                let mut entry = &mut self.lrmemo.get_mut(&key).ok_or(Error::Fail)?;

                entry.cursor = Ok(cursor);
                entry.bound += 1;
                entry.precedence = entry.precedence;

                self.cursor = frame_cursor;
                self.program_counter = callable_index;
            }
        } else {
            // println!("inc.3");
            let pc = frame.program_counter;
            self.cursor = frame.result.clone();
            self.program_counter = pc;
            self.stkpop()?;
        }
        Ok(())
    }

    fn fail(&mut self) -> Result<(), Error> {
        // println!("::::::FAIL");
        let error = match self.cursor.clone() {
            Err(e) => e,
            Ok(_) => Error::Fail,
        };
        let frame = loop {
            match self.stack.pop() {
                None => {
                    self.cursor = Err(error.clone());
                    return Err(error);
                },
                Some(f) => {
                    if let Ok(cursor) = f.cursor {
                        self.cursor = Ok(cursor);
                        break f;
                    }
                },
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
        let input = "a".to_string();
        // G <- 'a'
        let program = Program {
            code: vec![Instruction::Char('a'), Instruction::Halt],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

        assert!(result.is_ok());
        assert_eq!(vec![vec![Value::Chr('a')]], vm.captured_stack);
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
        let input = "b".to_string();
        // G <- 'a'
        let program = Program {
            code: vec![Instruction::Char('a'), Instruction::Halt],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

        assert!(result.is_err());
        assert_eq!(
            Error::Matching("Expected a, but got b instead".to_string()),
            result.unwrap_err(),
        );
        assert_eq!(0, vm.ffp);
    }

    // (any.1)
    //   i ≤ |s|
    // -----------------
    // match . s i = i+1
    #[test]
    fn any_1() {
        let input = "abcd".to_string();
        let program = Program {
            code: vec![
                Instruction::Any,
                Instruction::Any,
                Instruction::Any,
                Instruction::Halt,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

        assert!(result.is_ok());
        assert_eq!(
            vec![vec![Value::Chr('a'), Value::Chr('b'), Value::Chr('c')]],
            vm.captured_stack,
        );

        assert!(vm.cursor.is_ok());
        assert_eq!(3, vm.cursor.unwrap());
    }

    // (any.2)
    // i > |s|
    // -----------------
    // match . s i = nil
    #[test]
    fn any_2_eof() {
        let input = "".to_string();
        let program = Program {
            code: vec![Instruction::Any],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

        assert!(result.is_err());
        assert_eq!(Error::EOF, result.clone().unwrap_err());
        assert!(vm.cursor.is_err());
        assert_eq!(vm.cursor.unwrap_err(), result.unwrap_err())
    }

    // (not.1)
    // match p s i = nil
    // -----------------
    // match !p s i = i
    #[test]
    fn not_1() {
        let input = "foo".to_string();
        // G <- !'a'
        let program = Program {
            code: vec![
                Instruction::Choice(4),
                Instruction::Char('a'),
                Instruction::Commit(1),
                Instruction::Fail,
                Instruction::Halt,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

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
        let input = "foo".to_string();
        // G <- !'f'
        let program = Program {
            code: vec![
                Instruction::Choice(4),
                Instruction::Char('f'),
                Instruction::Commit(1),
                Instruction::Fail,
                Instruction::Halt,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

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
        let input = "c".to_string();
        // G <- 'a' / 'b'
        let program = Program {
            code: vec![
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::Commit(2),
                Instruction::Char('b'),
                Instruction::Halt,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

        assert!(result.is_err());
        // currently shows the last error
        assert_eq!(
            Error::Matching("Expected b, but got c instead".to_string()),
            result.unwrap_err()
        );
        assert_eq!(0, vm.ffp);
    }

    // (ord.2)
    // match p1 s i = i+j
    // -----------------------
    // match p1 / p2 s i = i+j (ord.2)
    #[test]
    fn ord_2() {
        let input = "a".to_string();
        // G <- 'a' / 'b'
        let program = Program {
            code: vec![
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::Commit(2),
                Instruction::Char('b'),
                Instruction::Halt,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

        assert!(result.is_ok());
        assert_eq!(vec![vec![Value::Chr('a')]], vm.captured_stack);
        assert!(vm.cursor.is_ok());
        assert_eq!(1, vm.cursor.unwrap());
    }

    // (ord.3)
    // match p1 s i = nil    match p2 s i = i+k
    // ----------------------------------------
    //         match p1 / p2 s i = i+k
    #[test]
    fn ord_3() {
        let input = "b".to_string();
        // G <- 'a' / 'b'
        let program = Program {
            code: vec![
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::Commit(2),
                Instruction::Char('b'),
                Instruction::Halt,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

        assert!(result.is_ok());
        assert_eq!(vec![vec![Value::Chr('b')]], vm.captured_stack);
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
        let input = "aab".to_string();
        // G <- 'a*'
        let program = Program {
            code: vec![
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::CommitB(2),
                Instruction::Halt,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

        assert!(result.is_ok());
        assert_eq!(
            vec![vec![Value::Chr('a'), Value::Chr('a')]],
            vm.captured_stack
        );
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
        let input = "b".to_string();
        // G <- 'a*'
        let program = Program {
            code: vec![
                Instruction::Choice(3),
                Instruction::Char('a'),
                Instruction::CommitB(2),
                Instruction::Halt,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

        assert!(result.is_ok());
        assert_eq!(vec![vec![]] as Vec<Vec<Value>>, vm.captured_stack);
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
        let input = "1+1".to_string();
        // G <- D '+' D
        // D <- '0' / '1'
        let program = Program {
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
        let result = vm.run(&input);

        assert!(result.is_ok());
        // assert_eq!(
        //     vec![vec![Value::Node(vec![
        //         Value::Node(vec![Value::Chr('1')]),
        //         Value::Chr('+'),
        //         Value::Node(vec![Value::Chr('1')]),
        //     ])]] as Vec<Vec<Value>>,
        //     vm.captured_stack
        // );
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
        let input = "1+2".to_string();
        // G <- D '+' D
        // D <- '0' / '1'
        let program = Program {
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
        let result = vm.run(&input);

        assert!(result.is_err());
        assert_eq!(
            Error::Matching("Expected 1, but got 2 instead".to_string()),
            result.unwrap_err()
        );
        assert_eq!(2, vm.ffp);
    }

    // (lvar.1)
    //
    // (A, xyz) not in L G[P(A)] xyz L[(A, xyz) -> fail] -peg-> (yz, x')
    //     G[P(A)] xyz L[(A, xyz) -> (yz,'x, k)] -inc-> (z, (xy)')
    // -----------------------------------------------------------------
    //                  G[Ak] xyz L -peg-> (z, A[(xy)'])
    #[test]
    fn var_1_lr() {
        let input = "n+n".to_string();
        // G <- G '+' 'n' / 'n'
        let program = Program {
            code: vec![
                Instruction::Call(2, 1),
                Instruction::Jump(9),
                Instruction::Choice(5),
                Instruction::CallB(1, 1),
                Instruction::Char('+'),
                Instruction::Char('n'),
                Instruction::Commit(2),
                Instruction::Char('n'),
                Instruction::Return,
                Instruction::Halt,
            ],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

        assert!(result.is_ok());
        assert!(vm.cursor.is_ok());
        assert_eq!(3, vm.cursor.unwrap());
    }
}
