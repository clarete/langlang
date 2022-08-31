use log::warn;
use std::fs;

use langlang::{compiler, format, parser, vm};

type FormattingFunc = fn(v: &vm::Value) -> String;

fn formatter(name: &str) -> FormattingFunc {
    match name {
        "fmt1" => format::value_fmt1,
        "fmt2" => format::value_fmt2,
        "" => format::value_fmt0,
        _ => {
            warn!("oh no! an invalud formatter: {}", name);
            format::value_fmt0
        }
    }
}

fn run_grammar_on_input_from_cmd() -> Result<(), std::io::Error> {
    let grammar_file = std::env::args().nth(1).expect("no grammar given");
    let input_file = std::env::args().nth(2).expect("no input given");
    let fmt = formatter(std::env::args().nth(3).unwrap_or("fmt0".to_string()).as_str());
    let grammar_data = fs::read_to_string(grammar_file)?;

    let mut p = parser::Parser::new(grammar_data.as_str());
    let ast = match p.parse_grammar() {
        Ok(a) => a,
        Err(e) => return Err(std::io::Error::new(
            std::io::ErrorKind::NotFound,
            e.to_string(),
        )),
    };

    let mut c = compiler::Compiler::default();
    let program = match c.compile(ast) {
        Ok(p) => p,
        Err(e) => return Err(std::io::Error::new(
            std::io::ErrorKind::NotFound,
            e.to_string(),
        )),
    };

    println!("Compiled:\n{}", program);

    let input_data = fs::read_to_string(input_file)?;
    let mut m = vm::VM::new(program);
    match m.run(&input_data) {
        Ok(Some(v)) => println!("{}", fmt(v)),
        Ok(None) => println!("not much"),
        Err(e) => println!("{:?}", e),
    }
    Ok(())
}

fn main() {
    env_logger::init();

    if let Err(e) = run_grammar_on_input_from_cmd() {
        println!("{}", e);
    }
}
