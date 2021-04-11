// vm.rs --- parsing machine
//
// This machine is capable of matching patterns in strings.  The
// patterns themselves are expressed in high level text based
// languages and get compiled to programs that can be executed by this
// machine.  This module has nothing to do with how patterns get
// compiled to programs, but how programs get executted as patterns.
//

#[derive(Debug, PartialEq)]
enum Value {
    // I64(i64),
    // U64(u64),
    // F64(f64),
    Chr(char),
    // Str(String),
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
    Call(usize),
    Return,
    // Throw(usize),
}

#[derive(Clone, Debug, PartialEq)]
enum Error {
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
    cursor: Option<usize>,
    program_counter: usize,
}

// #[derive(Debug)]
// enum Status {
//     Halt,
//     Continue,
// }

#[derive(Debug)]
struct VM {
    fpp: usize,
    cursor: usize,
    source: Vec<char>,
    program: Program,
    program_counter: usize,
    stack: Vec<StackFrame>,
    failure: Option<Error>,
    captured_stack: Vec<Vec<Value>>,
    captured_pointer: usize,
}

impl VM {
    fn new(program: Program) -> Self {
        return VM {
            fpp: 0,
            cursor: 0,
            source: vec![],
            program_counter: 0,
            program: program,
            stack: vec![],
            failure: None,
            captured_stack: vec![vec![]],
            captured_pointer: 0,
        };
    }

    fn update_cursor(&mut self) {
        self.cursor += 1;
        if self.cursor > self.fpp {
            self.fpp = self.cursor;
        }
    }

    fn push_call(&mut self, program_counter: usize) {
        self.stack.push(StackFrame {
            cursor: None,
            program_counter: program_counter,
        })
    }

    fn push_backtracking(&mut self, cursor: usize, program_counter: usize) {
        self.stack.push(StackFrame {
            cursor: Some(cursor),
            program_counter: program_counter,
        })
    }

    fn capture(&mut self, v: Value) {
        self.captured_stack[self.captured_pointer].push(v);
    }

    fn fail(&mut self) -> bool {
        loop {
            match self.stack.pop() {
                None => break,
                Some(frame) => match frame.cursor {
                    None => continue,
                    Some(cursor) => {
                        self.cursor = cursor;
                        self.program_counter = frame.program_counter;
                        return true;
                    }
                },
            }
        }
        false
    }

    fn run(&mut self, input: &String) -> Result<(), Error> {
        self.source = input.chars().collect();

        loop {
            if self.program_counter >= self.program.code.len() {
                return Err(Error::Overflow);
            }

            let instruction = match self.failure {
                Some(_) => Instruction::Fail,
                None => self.program.code[self.program_counter].clone(),
            };

            println!("I: {:?}", instruction);

            match instruction {
                Instruction::Halt => break,
                Instruction::Any => {
                    if self.cursor >= self.source.len() {
                        self.failure = Some(Error::EOF);
                    } else {
                        self.capture(Value::Chr(self.source[self.cursor]));
                        self.update_cursor();
                        self.program_counter += 1;
                    }
                }
                Instruction::Char(expected) => {
                    let current = self.source[self.cursor];
                    if current != expected {
                        self.failure = Some(Error::Matching(format!(
                            "Expected {}, but got {} instead",
                            expected, current,
                        )));
                    } else {
                        self.capture(Value::Chr(current));
                        self.update_cursor();
                        self.program_counter += 1;
                    }
                }
                Instruction::Choice(offset) => {
                    self.push_backtracking(self.cursor, self.program_counter + offset);
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
                    if !self.fail() {
                        let failure = self.failure.clone();
                        let err = failure.unwrap_or(Error::Matching("ERROR".to_string()));
                        return Err(err);
                    }
                    self.failure = None;
                }
                Instruction::Jump(index) => {
                    self.program_counter = index;
                }
                Instruction::Call(offset) => {
                    self.push_call(self.program_counter + 1);
                    self.program_counter += offset;
                }
                Instruction::Return => match self.stack.pop() {
                    None => return Err(Error::Overflow),
                    Some(frame) => self.program_counter = frame.program_counter,
                },
            }
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
        let input = "a".to_string();
        // G <- 'a'
        let program = Program {
            code: vec![Instruction::Char('a'), Instruction::Halt],
        };

        let mut vm = VM::new(program);
        let result = vm.run(&input);

        assert!(result.is_ok());
        assert_eq!(vec![vec![Value::Chr('a')]], vm.captured_stack);
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

        assert_eq!(3, vm.cursor);
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
        assert_eq!(Error::EOF, result.unwrap_err());
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
        assert_eq!(0, vm.cursor);
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
        assert_eq!(Error::Matching("ERROR".to_string()), result.unwrap_err());
        //assert_eq!(0, vm.cursor);
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
        assert_eq!(0, vm.cursor);
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
        assert_eq!(1, vm.cursor);
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
        assert_eq!(1, vm.cursor);
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
        assert_eq!(2, vm.cursor);
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
        assert_eq!(0, vm.cursor);
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
                Instruction::Call(2),
                Instruction::Jump(11),
                // G
                Instruction::Call(4),
                Instruction::Char('+'),
                Instruction::Call(2),
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
        // assert_eq!(vec![vec![]] as Vec<Vec<Value>>, vm.captured_stack);
        assert_eq!(3, vm.cursor);
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
                Instruction::Call(2),
                Instruction::Jump(11),
                // G
                Instruction::Call(4),
                Instruction::Char('+'),
                Instruction::Call(2),
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
        assert_eq!(2, vm.cursor);
    }
}
