use std::fs;
use langlang::{parser, vm};

fn run_grammar_on_input_from_cmd() -> Result<(), std::io::Error> {
    let grammar_file = std::env::args().nth(1).expect("no grammar given");
    let input_file = std::env::args().nth(2).expect("no input given");
    let grammar_data = fs::read_to_string(grammar_file)?;
    let mut c = parser::Compiler::new();
    if let Err(e) = c.compile_str(grammar_data.as_str()) {
        return Err(std::io::Error::new(std::io::ErrorKind::NotFound, e.to_string()));
    }

    let p = c.program();
    println!("Compiled:\n{}", p);

    let input_data = fs::read_to_string(input_file)?;
    let mut m = vm::VM::new(p);
    match m.run(&input_data) {
        Ok(Some(v)) => println!("{:#?}", v),
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
