use std::io::Write;
use std::path::{Path, PathBuf};
use std::{fs, io};

use langlang_lib::vm::VM;
use langlang_lib::{compiler, import};
use langlang_value::format;
use langlang_value::value::Value;

use clap::{Parser, Subcommand};

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

        /// Choose what's the first production to run
        #[arg(short, long)]
        start_rule: Option<String>,

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

type FormattingFunc = fn(v: &Value);

fn outputfn(name: &str) -> FormattingFunc {
    match name {
        "nil" => |_| {},
        "compact" => |v| println!("{}", format::compact(v)),
        "html" => |v| println!("{}", format::html(v)),
        "indented" => |v| println!("{}", format::indented(v)),
        "raw" => |v| println!("{}", format::raw(v)),
        _ => |_| println!(""),
    }
}

fn command_run(
    grammar_file: &Path,
    start_rule: &Option<String>,
    input_file: &Option<PathBuf>,
    output_format: &Option<String>,
) -> Result<(), langlang_lib::Error> {
    let importer = import::ImportResolver::new(import::RelativeImportLoader::default());
    let ast = importer.resolve(grammar_file)?;
    // This is a little ugly but it's converting from &Option<String> to Option<&str>
    let program = compiler::Compiler::default().compile(
        &ast,
        match start_rule {
            Some(n) => Some(&n),
            None => None,
        },
    )?;
    let fmt = outputfn(output_format.as_ref().unwrap_or(&"raw".to_string()));

    match input_file {
        Some(input_file) => {
            let input_data = fs::read_to_string(input_file)?;
            let mut m = VM::new(&program);
            match m.run_str(&input_data)? {
                None => println!("not much"),
                Some(v) => fmt(&v),
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
                let mut m = VM::new(&program);
                match m.run_str(&line)? {
                    None => println!("not much"),
                    Some(v) => fmt(&v),
                }
            }
        }
    }
    Ok(())
}

fn run() -> Result<(), langlang_lib::Error> {
    let cli = Cli::parse();
    match &cli.command {
        Command::Run {
            grammar_file,
            start_rule,
            input_file,
            output_format,
        } => {
            command_run(grammar_file, start_rule, input_file, output_format)?;
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
