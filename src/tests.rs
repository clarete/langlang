#[cfg(test)]
mod tests {
    use crate::{compiler, parser, vm};
    use log::debug;

    fn compile_and_run(cc: compiler::Config, grammar: &str, input: &str) -> Option<vm::Value> {
        let mut p = parser::Parser::new(grammar);
        let ast = p.parse_grammar().unwrap();
        let mut c = compiler::Compiler::new(cc);
        let program = c.compile(ast).unwrap();
        debug!("p: {}", program);
        let mut machine = vm::VM::new(program);
        machine.run(input).unwrap()
    }

    #[test]
    fn test_char() {
        let cc = compiler::Config::default().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- 'a'", "a");
        assert_eq!(
            Some(vm::Value::Node {
                name: "A".to_string(),
                children: vec![vm::Value::Str("a".to_string())],
            }),
            value
        );
    }

    #[test]
    fn test_not_0() {
        let cc = compiler::Config::o0().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- (!('a' / 'b') .)", "c");
        assert_eq!(
            Some(vm::Value::Node {
                name: "A".to_string(),
                children: vec![vm::Value::Chr('c')],
            }),
            value
        );
    }

    #[test]
    fn test_not_opt() {
        let cc = compiler::Config::o1().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- (!('a' / 'b') .)", "c");
        assert_eq!(
            Some(vm::Value::Node {
                name: "A".to_string(),
                children: vec![vm::Value::Chr('c')],
            }),
            value
        );
    }

    #[test]
    fn test_and_0() {
        let cc = compiler::Config::o0().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- (&('a' / 'b') .)", "a");
        assert_eq!(
            Some(vm::Value::Node {
                name: "A".to_string(),
                children: vec![vm::Value::Chr('a')],
            }),
            value
        );
    }

    #[test]
    fn test_and_opt() {
        let cc = compiler::Config::o1().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- &'a' .", "a");
        assert_eq!(
            Some(vm::Value::Node {
                name: "A".to_string(),
                children: vec![vm::Value::Chr('a')],
            }),
            value
        );
    }

    #[test]
    fn test_star_opt() {
        let cc = compiler::Config::o1().with_disabled_precedence();
        let value = compile_and_run(cc, "A <- .*", "abab");
        assert_eq!(
            Some(vm::Value::Node {
                name: "A".to_string(),
                children: vec![
                    vm::Value::Chr('a'),
                    vm::Value::Chr('b'),
                    vm::Value::Chr('a'),
                    vm::Value::Chr('b'),
                ],
            }),
            value
        );
    }
}
