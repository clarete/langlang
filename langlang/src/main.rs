use clap::{Parser, Subcommand};
use log::warn;
use std::io::Write;
use std::path::PathBuf;
use std::{fs, io};

use langlang_lib::{compiler, format, parser, vm};

/// Enumeration of all sub commands supported by this binary
#[derive(Subcommand)]
enum Command {
    /// Run a grammar file against an input file.  If the input file
    /// is not provided, the user will be dropped into an interactive
    /// shell.
    Run {
        /// Path to the grammar file to be executed
        #[arg(short, long)]
        grammar_file: std::path::PathBuf,

        /// Path to the content to be matched against the grammar;
        /// Omitting it will drop you in an interactive shell
        #[arg(short, long)]
        input_file: Option<std::path::PathBuf>,

        /// Configure the output before printing it out in the screen
        #[arg(short, long)]
        output_format: Option<String>,
    },
}

/// langlang provides a set of subcommands with different functionality.
#[derive(Parser)]
#[command(author, version, about, long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Command,
}

#[derive(Debug)]
pub enum Error {
    CompilerError(compiler::Error),
    ParserError(parser::Error),
    RuntimeError(vm::Error),
    IOError(io::Error),
}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        match self {
            Error::ParserError(e) => write!(f, "Parsing Error: {:#?}", e),
            Error::CompilerError(e) => write!(f, "Compiler Error: {:#?}", e),
            Error::RuntimeError(e) => write!(f, "Runtime Error: {:#?}", e),
            Error::IOError(e) => write!(f, "Input/Output Error: {:#?}", e),
        }
    }
}

impl std::error::Error for Error {}

impl From<io::Error> for Error {
    fn from(e: io::Error) -> Self {
        Error::IOError(e)
    }
}

impl From<compiler::Error> for Error {
    fn from(e: compiler::Error) -> Self {
        Error::CompilerError(e)
    }
}

impl From<parser::Error> for Error {
    fn from(e: parser::Error) -> Self {
        Error::ParserError(e)
    }
}

impl From<vm::Error> for Error {
    fn from(e: vm::Error) -> Self {
        Error::RuntimeError(e)
    }
}

type FormattingFunc = fn(v: &vm::Value) -> String;

fn formatter(name: &str) -> FormattingFunc {
    match name {
        "fmt0" => format::value_fmt0,
        "fmt1" => format::value_fmt1,
        "fmt2" => format::value_fmt2,
        "html" => format::value_html,
        _ => {
            warn!("oh no! an invalud formatter: {}", name);
            format::value_fmt0
        }
    }
}

fn command_run(
    grammar_file: &PathBuf,
    input_file: &Option<PathBuf>,
    output_format: &Option<String>,
) -> Result<(), Error> {
    let grammar = fs::read_to_string(grammar_file)?;
    let ast = parser::Parser::new(&grammar).parse()?;
    let program = compiler::Compiler::default().compile(ast)?;
    let fmt = formatter(output_format.as_ref().unwrap_or(&"fmt".to_string()));

    match input_file {
        Some(input_file) => {
            let input_data = fs::read_to_string(input_file)?;
            let mut m = vm::VM::new(&program);
            match m.run_str(&input_data)? {
                None => println!("not much"),
                Some(v) => println!("{}", fmt(&v)),
            }
        }
        None => {
            // Shell
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
                let mut m = vm::VM::new(&program);
                match m.run_str(&line)? {
                    None => println!("not much"),
                    Some(v) => println!("{}", fmt(&v)),
                }
            }
        }
    }
    Ok(())
}

fn run() -> Result<(), Error> {
    let cli = Cli::parse();
    match &cli.command {
        Command::Run {
            grammar_file,
            input_file,
            output_format,
        } => {
            command_run(grammar_file, input_file, output_format)?;
        }
    }
    Ok(())
}

fn main() {
    env_logger::init();

    if let Err(e) = run() {
        println!("{}", e);
    }
}
