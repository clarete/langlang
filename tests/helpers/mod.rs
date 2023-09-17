use langlang_lib::{compiler, format, import, vm};
use std::path::Path;

pub fn compile(cc: &compiler::Config, grammar: &str, start: &str) -> vm::Program {
    let mut loader = import::InMemoryImportLoader::default();
    loader.add_grammar("main", grammar);
    let importer = import::ImportResolver::new(loader);
    let ast = importer.resolve(Path::new("main")).unwrap();
    println!("PEG:\n{}", ast.to_string());
    let mut c = compiler::Compiler::new(cc.clone());
    let program = c.compile(&ast, start).unwrap();
    println!("PROGRAM:\n{}", program);
    program
}

pub fn compile_file(cc: &compiler::Config, grammar_file: &str, start_rule: &str) -> vm::Program {
    let importer = import::ImportResolver::new(import::RelativeImportLoader::default());
    let ast = importer.resolve(Path::new(grammar_file)).unwrap();
    let mut c = compiler::Compiler::new(cc.clone());
    c.compile(&ast, start_rule).unwrap()
}

pub fn run_str(program: &vm::Program, input: &str) -> Result<Option<vm::Value>, vm::Error> {
    let mut machine = vm::VM::new(program);
    machine.run_str(input)
}

pub fn cc_run(
    cc: &compiler::Config,
    grammar: &str,
    start: &str,
    input: &str,
) -> Result<Option<vm::Value>, vm::Error> {
    let prog = compile(cc, grammar, start);
    let mut machine = vm::VM::new(&prog);
    machine.run_str(input)
}

pub fn assert_match(expected: &str, r: Result<Option<vm::Value>, vm::Error>) {
    assert!(r.is_ok());
    let o = r.unwrap();
    assert!(o.is_some());
    let v = o.unwrap();
    assert_eq!(expected.to_string(), format::value_fmt1(&v));
}

pub fn assert_err(expected: vm::Error, r: Result<Option<vm::Value>, vm::Error>) {
    assert!(r.is_err());
    let e = r.unwrap_err();
    assert_eq!(expected, e);
}
