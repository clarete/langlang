use std::fs;
use std::io::{self, Write};

use langlang::{compiler, parser, vm};

#[derive(Debug)]
pub enum ShellError {
    CompilerError(compiler::Error),
    ParserError(parser::Error),
    RuntimeError(vm::Error),
    IOError(io::Error),
}

impl std::fmt::Display for ShellError {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        match self {
            ShellError::ParserError(e) => write!(f, "Parsing Error: {:#?}", e),
            ShellError::CompilerError(e) => write!(f, "Compiler Error: {:#?}", e),
            ShellError::RuntimeError(e) => write!(f, "Runtime Error: {:#?}", e),
            ShellError::IOError(e) => write!(f, "Input/Output Error: {:#?}", e),
        }
    }
}

impl std::error::Error for ShellError {}

impl From<io::Error> for ShellError {
    fn from(e: io::Error) -> Self {
        ShellError::IOError(e)
    }
}

impl From<compiler::Error> for ShellError {
    fn from(e: compiler::Error) -> Self {
        ShellError::CompilerError(e)
    }
}

impl From<parser::Error> for ShellError {
    fn from(e: parser::Error) -> Self {
        ShellError::ParserError(e)
    }
}

impl From<vm::Error> for ShellError {
    fn from(e: vm::Error) -> Self {
        ShellError::RuntimeError(e)
    }
}

fn shell() -> Result<(), ShellError> {
    let file_name = std::env::args().nth(1).expect("no grammar given");
    let data = fs::read_to_string(&file_name)?;

    println!("welcome to langlang. use Ctrl-D to get outta here.");
    println!("loaded: {}", file_name);

    let mut p = parser::Parser::new(data.as_str());
    let ast = p.parse()?;

    let mut compiler = compiler::Compiler::default();
    let program = compiler.compile(ast)?;
    println!("{}", program);
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
        let mut m = vm::VM::new(program.clone());
        match m.run_str(&line) {
            Ok(Some(v)) => println!("{:#?}", v),
            Ok(None) => println!("not much"),
            Err(e) => return Err(ShellError::RuntimeError(e)),
        }
    }

    Ok(())
}

fn main() {
    env_logger::init();

    if let Err(e) = shell() {
        println!("{}", e);
    }
}
