use log::warn;
use std::fs;

use langlang::{format, parser, vm};

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
    let mut c = parser::Compiler::default();
    if let Err(e) = c.compile_str(grammar_data.as_str()) {
        return Err(std::io::Error::new(
            std::io::ErrorKind::NotFound,
            e.to_string(),
        ));
    }

    let p = c.program();
    println!("Compiled:\n{}", p);

    let input_data = fs::read_to_string(input_file)?;
    let mut m = vm::VM::new(p);
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
