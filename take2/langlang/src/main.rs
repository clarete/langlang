use std::fs;
use std::io::{self, Write};

mod parser;
mod vm;

#[derive(Debug)]
pub enum ShellError {
    ParsingError(parser::Error),
    RuntimeError,
    IOError,
}

impl std::fmt::Display for ShellError {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        match self {
            ShellError::ParsingError(e) => write!(f, "{:#?}", e),
            ShellError::RuntimeError => write!(f, "Runtime Error"),
            ShellError::IOError => write!(f, "Input/Output Error"),
        }
    }
}

impl std::error::Error for ShellError {}

impl From<io::Error> for ShellError {
    fn from(_: io::Error) -> Self {
        ShellError::IOError
    }
}

impl From<parser::Error> for ShellError {
    fn from(e: parser::Error) -> Self {
        ShellError::ParsingError(e)
    }
}

impl From<vm::Error> for ShellError {
    fn from(_: vm::Error) -> Self {
        ShellError::RuntimeError
    }
}

fn shell() -> Result<(), ShellError> {
    println!("welcome to langlang. use Ctrl-D to get outta here.");

    let data = fs::read_to_string("/home/lincoln/src/github.com/clarete/langlang/lib/abnf.peg")?;
    println!("Text:\n{}", data);
    let mut c = parser::Compiler::new();
    c.compile_str(data.as_str())?;
    let p = c.program();

    loop {
        // display prompt
        print!("langlang% ");
        io::stdout().flush().expect("can't flush stdout");

        // read the next line typed in
        let mut line = String::new();
        io::stdin().read_line(&mut line)?;

        // handle Ctrl-D
        if line.as_str() == "" {
            println!();
            break;
        }

        // skip empty lines
        if line.as_str() == "\n" {
            continue;
        }

        // removed the unwanted last \n
        line.pop();

        // run the line
        let mut m = vm::VM::new(p.clone());
        match m.run(&line) {
            Ok(Some(v)) => println!("{:#?}", v),
            Ok(None) => println!("not much"),
            Err(e) => println!("{:?}", e),
        }
    }

    Ok(())
}

fn run_grammar_on_input(grammar_file: &str, input_file: &str) -> Result<(), ShellError> {
    let grammar_data = fs::read_to_string(grammar_file)?;
    let mut c = parser::Compiler::new();
    c.compile_str(grammar_data.as_str())?;
    let p = c.program();
    println!("{}", p.to_string());
    let input_data = fs::read_to_string(input_file)?;
    let mut m = vm::VM::new(p);
    match m.run(&input_data) {
        Ok(Some(v)) => println!("{:#?}", v),
        Ok(None) => println!("not much"),
        Err(e) => println!("{:?}", e),
    }
    Ok(())
}

fn run_grammar_on_input_from_cmd() -> Result<(), ShellError> {
    let grammar_file = std::env::args().nth(2).expect("no grammar given");
    let input_file = std::env::args().nth(3).expect("no grammar given");
    run_grammar_on_input(grammar_file.as_str(), input_file.as_str())
}

fn cmd() -> Result<(), ShellError> {
    let command = std::env::args().nth(1).expect("no command given");

    match command.as_str() {
        "shell" => shell()?,
        "run" => run_grammar_on_input_from_cmd()?,
        nope => println!("Invalid Command {}", nope),
    }
    Ok(())
}

fn main() {
    env_logger::init();

    if let Err(ShellError::ParsingError(e)) = cmd() {
        println!("{}", e.to_string());
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn throw_1_not() {
        let mut c = parser::Compiler::new();
        // The label `l` is being used within a not predicate, so it
        // should behave like a regular `fail` level.
        c.compile_str(
            "G0 <- (!G1 .)+^l2
             G1 <- 'abcd' / G2
             G2 <- 'a' 'b'^l / 'c'
            ",
        )
        .unwrap();

        let mut v = vm::VM::new(c.program());
        let result = v.run("ak");

        assert!(result.is_ok());
        assert_eq!(
            Ok(Some(&vm::Value::Node {
                name: "G0".to_string(),
                children: vec![vm::Value::Chr('a'), vm::Value::Chr('k')],
            })),
            result
        );
    }

    #[test]
    fn throw_2_not() {
        let mut c = parser::Compiler::new();
        // first label `l` is thrown from within a not predicate, but
        // then it will try to match the input that doesn't have what
        // it wants and must throw an error out side the not predicate
        // proving that `fail` correctly flips the predicate state
        // back to false.
        c.compile_str(
            "label l = 'foo'
             G0 <- !G1 . G1
             G1 <- 'a' 'b'^l / 'c'
            ",
        )
        .unwrap();

        let mut v = vm::VM::new(c.program());
        let result = v.run("aac");
        assert!(result.is_err());
        assert_eq!(Err(vm::Error::Matching(2, "foo".to_string())), result);
    }
}
