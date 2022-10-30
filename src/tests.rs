#[cfg(test)]
mod tests {
    use crate::{compiler, format, parser, vm};
    use log::debug;

    fn compile(cc: compiler::Config, grammar: &str) -> vm::Program {
        let mut p = parser::Parser::new(grammar);
        let ast = p.parse().unwrap();
        let mut c = compiler::Compiler::new(cc);
        let program = c.compile(ast).unwrap();
        debug!("p: {}", program);
        program
    }

    fn run(program: &vm::Program, input: &str) -> Option<vm::Value> {
        let mut machine = vm::VM::new(program);
        machine.run_str(input).expect("Unexpected")
    }

    fn cc_run(cc: compiler::Config, grammar: &str, input: &str) -> Option<vm::Value> {
        let prog = compile(cc, grammar);
        let mut machine = vm::VM::new(&prog);
        machine.run_str(input).expect("Unexpected")
    }

    fn assert_success(expected: &str, value: Option<vm::Value>) {
        assert!(value.is_some());
        assert_eq!(expected.to_string(), format::value_fmt1(&value.unwrap()));
    }

    #[test]
    fn test_char() {
        let cc = compiler::Config::default();
        assert_success("A[a]", cc_run(cc, "A <- 'a'", "a"));
    }

    #[test]
    fn test_not_0() {
        let cc = compiler::Config::o0();
        assert_success("A[c]", cc_run(cc, "A <- (!('a' / 'b') .)", "c"));
    }

    #[test]
    fn test_not_opt() {
        let cc = compiler::Config::o1();
        assert_success("A[c]", cc_run(cc, "A <- (!('a' / 'b') .)", "c"));
    }

    #[test]
    fn test_and_0() {
        let cc = compiler::Config::o0();
        assert_success("A[a]", cc_run(cc, "A <- (&('a' / 'b') .)", "a"));
    }

    #[test]
    fn test_and_opt() {
        let cc = compiler::Config::o1();
        assert_success("A[a]", cc_run(cc, "A <- &'a' .", "a"));
    }

    #[test]
    fn test_choice_within_repeat() {
        let cc = compiler::Config::o0();
        assert_success(
            "A[abada]",
            cc_run(cc, "A <- ('abacate' / 'abada')+", "abada"),
        );
    }

    #[test]
    fn test_star_0() {
        let cc = compiler::Config::o0();
        assert_success("A[abab]", cc_run(cc, "A <- .*", "abab"));
    }

    #[test]
    fn test_star_opt() {
        let cc = compiler::Config::o1();
        assert_success("A[abab]", cc_run(cc, "A <- .*", "abab"));
    }

    #[test]
    fn test_var0() {
        let cc = compiler::Config::default();
        assert_success("A[11]", cc_run(cc, "A <- '1' '1'", "11"));
    }

    #[test]
    fn test_var_ending_with_zero_or_more() {
        let cc = compiler::Config::default();
        let program = compile(cc, "A <- '1'+");
        assert_success("A[111]", run(&program, "111"));
        assert_success("A[11]", run(&program, "11"));
        assert_success("A[1]", run(&program, "1"));
    }

    #[test]
    fn test_var_ending_with_option() {
        let cc = compiler::Config::default();
        let program = compile(cc, "A <- '1' '1'?");
        assert_success("A[11]", run(&program, "11"));
        assert_success("A[1]", run(&program, "1"));
    }

    #[test]
    fn test_lr0() {
        let cc = compiler::Config::default();
        let program = compile(cc, "E <- E '+n' / 'n'");
        assert_success("E[n]", run(&program, "n"));
        assert_success("E[E[n]+n]", run(&program, "n+n"));
        assert_success("E[E[E[n]+n]+n]", run(&program, "n+n+n"));
    }

    #[test]
    fn test_lr1() {
        let cc = compiler::Config::default();
        let program = compile(cc, "E <- E '+' E / 'n'+");
        assert_success("E[n]", run(&program, "n"));
        assert_success("E[E[n]+E[n]]", run(&program, "n+n"));
        assert_success("E[E[n]+E[E[n]+E[n]]]", run(&program, "n+n+n"));
        assert_success("E[E[n]+E[E[n]+E[E[n]+E[n]]]]", run(&program, "n+n+n+n"));
    }

    #[test]
    fn test_lr2() {
        let cc = compiler::Config::default();
        let program = compile(
            cc,
            "
             E <- M '+' E / M
             M <- M '-n' / 'n'
            ",
        );
        assert_success("E[M[n]]", run(&program, "n"));
        assert_success("E[M[M[n]-n]]", run(&program, "n-n"));
        assert_success("E[M[M[M[n]-n]-n]]", run(&program, "n-n-n"));
        assert_success("E[M[n]+E[M[n]+E[M[n]]]]", run(&program, "n+n+n"));
    }

    #[test]
    fn test_lr3() {
        let cc = compiler::Config::default();
        let program = compile(
            cc,
            "
             E <- E '+' E
                / E '-' E
                / E '*' E
                / E '/' E
                / 'n'
            ",
        );
        // Right associative, as E is both left and right recursive,
        // without precedence
        assert_success("E[n]", run(&program, "n"));
        assert_success("E[E[n]+E[n]]", run(&program, "n+n"));
        assert_success("E[E[n]+E[E[n]+E[n]]]", run(&program, "n+n+n"));
        assert_success("E[E[n]-E[n]]", run(&program, "n-n"));
        assert_success("E[E[n]-E[E[n]-E[n]]]", run(&program, "n-n-n"));
        assert_success("E[E[n]*E[n]]", run(&program, "n*n"));
        assert_success("E[E[n]*E[E[n]*E[n]]]", run(&program, "n*n*n"));
        assert_success("E[E[n]/E[n]]", run(&program, "n/n"));
        assert_success("E[E[n]/E[E[n]/E[n]]]", run(&program, "n/n/n"));
        assert_success("E[E[n]-E[E[n]+E[n]]]", run(&program, "n-n+n"));
        assert_success("E[E[n]+E[E[n]-E[n]]]", run(&program, "n+n-n"));
        assert_success("E[E[n]+E[E[n]*E[n]]]", run(&program, "n+n*n"));
        assert_success("E[E[n]*E[E[n]+E[n]]]", run(&program, "n*n+n"));
        assert_success("E[E[n]/E[E[n]+E[n]]]", run(&program, "n/n+n"));
    }

    #[test]
    fn test_lr4() {
        let cc = compiler::Config::default();
        let program = compile(
            cc,
            "
             E <- E¹ '+' E²
                / E¹ '-' E²
                / E² '*' E³
                / E² '/' E³
                / '-' E⁴
                / '(' E¹ ')'
                / [0-9]+
            ",
        );

        // left associative with different precedences
        assert_success("E[21]", run(&program, "21"));
        assert_success("E[E[3]+E[5]]", run(&program, "3+5"));
        assert_success("E[E[3]-E[5]]", run(&program, "3-5"));
        // same precedence between addition (+) and subtraction (-)
        assert_success("E[E[E[3]-E[5]]+E[2]]", run(&program, "3-5+2"));
        assert_success("E[E[E[3]+E[5]]-E[2]]", run(&program, "3+5-2"));
        // higher precedence for multiplication (*) over addition (+) and subtraction (-)
        assert_success("E[E[3]+E[E[5]*E[2]]]", run(&program, "3+5*2"));
        assert_success("E[E[E[5]*E[2]]-E[3]]", run(&program, "5*2-3"));
        assert_success("E[E[E[E[1]*E[5]]*E[2]]+E[3]]", run(&program, "1*5*2+3"));
        // unary operator
        assert_success("E[-E[1]]", run(&program, "-1"));
        // highest precedence parenthesis
        assert_success("E[E[(E[E[3]+E[5]])]*E[2]]", run(&program, "(3+5)*2"));
    }

    #[test]
    fn test_lr5() {
        let cc = compiler::Config::default();
        assert_success(
            "L[xP[L[P[P[(n)](n)]]].xP[L[P[(n)]]].x]",
            cc_run(
                cc,
                "
                L <- P '.x' / 'x'
                P <- P '(n)' / L
                ",
                "x(n)(n).x(n).x",
            ),
        );
    }
}
