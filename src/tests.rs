#[cfg(test)]
mod tests {
    use crate::{compiler, format, parser, vm};
    use log::debug;

    fn compile(cc: compiler::Config, grammar: &str) -> vm::Program {
        let mut p = parser::Parser::new(grammar);
        let ast = p.parse_grammar().unwrap();
        let mut c = compiler::Compiler::new(cc);
        let program = c.compile(ast).unwrap();
        debug!("p: {}", program);
        program
    }

    fn run(program: vm::Program, input: &str) -> Option<vm::Value> {
        let mut machine = vm::VM::new(program);
        match machine.run(input) {
            Ok(opt) => opt,
            Err(err) => panic!("Unexpected Error: {:?}", err),
        }
    }

    fn compile_and_run(cc: compiler::Config, grammar: &str, input: &str) -> Option<vm::Value> {
        let program = compile(cc, grammar);
        run(program, input)
    }

    fn assert_success(expected: &str, value: Option<vm::Value>) {
        assert!(value.is_some());
        assert_eq!(expected.to_string(), format::value_fmt1(&value.unwrap()));
    }

    #[test]
    fn test_char() {
        let cc = compiler::Config::default().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- 'a'", "a");
        assert_success("A[a]", value);
    }

    #[test]
    fn test_not_0() {
        let cc = compiler::Config::o0().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- (!('a' / 'b') .)", "c");
        assert_success("A[c]", value);
    }

    #[test]
    fn test_not_opt() {
        let cc = compiler::Config::o1().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- (!('a' / 'b') .)", "c");
        assert_success("A[c]", value);
    }

    #[test]
    fn test_and_0() {
        let cc = compiler::Config::o0().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- (&('a' / 'b') .)", "a");
        assert_success("A[a]", value);
    }

    #[test]
    fn test_and_opt() {
        let cc = compiler::Config::o1().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- &'a' .", "a");
        assert_success("A[a]", value);
    }

    #[test]
    fn test_choice_within_repeat() {
        let cc = compiler::Config::o0().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- ('abacate' / 'abada')+", "abada");
        assert_success("A[abada]", value);
    }

    #[test]
    fn test_star_opt() {
        let cc = compiler::Config::o1().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- .*", "abab");
        assert_success("A[abab]", value);
    }
}
