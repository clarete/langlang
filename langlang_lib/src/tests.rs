#[cfg(test)]
mod tests {
    use crate::{compiler, format, parser, vm};
    use log::debug;

    #[test]
    fn test_char() {
        let cc = compiler::Config::default();
        assert_match("A[a]", cc_run(&cc, "A <- 'a'", "a"));
    }

    #[test]
    fn test_str() {
        let cc = compiler::Config::default();
        let p = compile(&cc, "A <- '0x' [0-9a-fA-F]+ / '0'");

        // Note: run_str uses VM::run_str, which maps each character
        // of the input string into a `Value::Char`, and the fact that
        // `Instruction::String` can read both an entire string or a
        // set of chars allows this example to work, as the `0x` piece
        // is compiled into an `Instruction::String` call.
        assert_match("A[0xff]", run_str(&p, "0xff"));
        assert_match("A[0]", run_str(&p, "0"));

        // This won't work because "0x" is tested against "0xff" which
        // fails right away:
        //let value = run(&p, vec![vm::Value::String("0xff".to_string())]);
        //assert_match("A[0xff]", value.unwrap());

        // This works as both `f` chars get consummed by [a-f]+ one at
        // a time.
        let value = run(
            &p,
            vec![
                vm::Value::String("0x".to_string()),
                vm::Value::Char('f'),
                vm::Value::Char('f'),
            ],
        );
        assert_match("A[0xff]", value.unwrap());

        // Easiest case
        let value = run(&p, vec![vm::Value::String("0".to_string())]);
        assert_match("A[0]", value.unwrap());
    }

    #[test]
    fn test_not_0() {
        let cc = compiler::Config::o0();
        assert_match("A[c]", cc_run(&cc, "A <- (!('a' / 'b') .)", "c"));
    }

    #[test]
    fn test_not_opt() {
        let cc = compiler::Config::o1();
        assert_match("A[c]", cc_run(&cc, "A <- (!('a' / 'b') .)", "c"));
    }

    #[test]
    fn test_not_at_the_end() {
        let cc = compiler::Config::default();
        let p = compile(
            &cc,
            "
            Primary    <- Identifier !LEFTARROW
            Identifier <- [a-zA-Z_][a-zA-Z0-9_]*
            LEFTARROW  <- '<-'
            ",
        );
        assert_match("Primary[Identifier[A]]", run_str(&p, "A"));
    }

    #[test]
    fn test_and_0() {
        let cc = compiler::Config::o0();
        assert_match("A[a]", cc_run(&cc, "A <- (&('a' / 'b') .)", "a"));
    }

    #[test]
    fn test_and_opt() {
        let cc = compiler::Config::o1();
        assert_match("A[a]", cc_run(&cc, "A <- &'a' .", "a"));
    }

    #[test]
    fn test_choice_within_repeat() {
        let cc = compiler::Config::o0();
        assert_match(
            "A[abada]",
            cc_run(&cc, "A <- ('abacate' / 'abada')+", "abada"),
        );
    }

    #[test]
    fn test_star_0() {
        let cc = compiler::Config::o0();
        assert_match("A[abab]", cc_run(&cc, "A <- .*", "abab"));
    }

    #[test]
    fn test_star_opt() {
        let cc = compiler::Config::o1();
        assert_match("A[abab]", cc_run(&cc, "A <- .*", "abab"));
    }

    #[test]
    fn test_var0() {
        let cc = compiler::Config::default();
        assert_match("A[11]", cc_run(&cc, "A <- '1' '1'", "11"));
    }

    #[test]
    fn test_var_ending_with_zero_or_more() {
        let cc = compiler::Config::default();
        let program = compile(&cc, "A <- '1'*");
        assert_match("A[111]", run_str(&program, "111"));
        assert_match("A[11]", run_str(&program, "11"));
        assert_match("A[1]", run_str(&program, "1"));
        assert!(run_str(&program, "").is_none())
    }

    #[test]
    fn test_var_ending_with_one_or_more() {
        let cc = compiler::Config::default();
        let program = compile(&cc, "A <- '1'+");
        assert_match("A[111]", run_str(&program, "111"));
        assert_match("A[11]", run_str(&program, "11"));
        assert_match("A[1]", run_str(&program, "1"));
    }

    #[test]
    fn test_var_ending_with_option() {
        let cc = compiler::Config::default();
        let program = compile(&cc, "A <- '1' '1'?");
        assert_match("A[11]", run_str(&program, "11"));
        assert_match("A[1]", run_str(&program, "1"));
    }

    // -- Unicode --------------------------------------------------------------

    #[test]
    fn test_unicode_0() {
        let cc = compiler::Config::default();
        assert_match("A[♡]", cc_run(&cc, "A <- [♡]", "♡"));
        assert_match("A[♡]", cc_run(&cc, "A <- '♡'", "♡"));
    }

    // -- Left Recursion -------------------------------------------------------

    #[test]
    fn test_lr0() {
        let cc = compiler::Config::default();
        let program = compile(&cc, "E <- E '+n' / 'n'");
        assert_match("E[n]", run_str(&program, "n"));
        assert_match("E[E[n]+n]", run_str(&program, "n+n"));
        assert_match("E[E[E[n]+n]+n]", run_str(&program, "n+n+n"));
    }

    #[test]
    fn test_lr1() {
        let cc = compiler::Config::default();
        let program = compile(&cc, "E <- E '+' E / 'n'+");
        assert_match("E[n]", run_str(&program, "n"));
        assert_match("E[E[n]+E[n]]", run_str(&program, "n+n"));
        assert_match("E[E[n]+E[E[n]+E[n]]]", run_str(&program, "n+n+n"));
        assert_match("E[E[n]+E[E[n]+E[E[n]+E[n]]]]", run_str(&program, "n+n+n+n"));
    }

    #[test]
    fn test_lr2() {
        let cc = compiler::Config::default();
        let program = compile(
            &cc,
            "
             E <- M '+' E / M
             M <- M '-n' / 'n'
            ",
        );
        assert_match("E[M[n]]", run_str(&program, "n"));
        assert_match("E[M[M[n]-n]]", run_str(&program, "n-n"));
        assert_match("E[M[M[M[n]-n]-n]]", run_str(&program, "n-n-n"));
        assert_match("E[M[n]+E[M[n]+E[M[n]]]]", run_str(&program, "n+n+n"));
    }

    #[test]
    fn test_lr3() {
        let cc = compiler::Config::default();
        let program = compile(
            &cc,
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
        assert_match("E[n]", run_str(&program, "n"));
        assert_match("E[E[n]+E[n]]", run_str(&program, "n+n"));
        assert_match("E[E[n]+E[E[n]+E[n]]]", run_str(&program, "n+n+n"));
        assert_match("E[E[n]-E[n]]", run_str(&program, "n-n"));
        assert_match("E[E[n]-E[E[n]-E[n]]]", run_str(&program, "n-n-n"));
        assert_match("E[E[n]*E[n]]", run_str(&program, "n*n"));
        assert_match("E[E[n]*E[E[n]*E[n]]]", run_str(&program, "n*n*n"));
        assert_match("E[E[n]/E[n]]", run_str(&program, "n/n"));
        assert_match("E[E[n]/E[E[n]/E[n]]]", run_str(&program, "n/n/n"));
        assert_match("E[E[n]-E[E[n]+E[n]]]", run_str(&program, "n-n+n"));
        assert_match("E[E[n]+E[E[n]-E[n]]]", run_str(&program, "n+n-n"));
        assert_match("E[E[n]+E[E[n]*E[n]]]", run_str(&program, "n+n*n"));
        assert_match("E[E[n]*E[E[n]+E[n]]]", run_str(&program, "n*n+n"));
        assert_match("E[E[n]/E[E[n]+E[n]]]", run_str(&program, "n/n+n"));
    }

    #[test]
    fn test_lr4() {
        let cc = compiler::Config::default();
        let program = compile(
            &cc,
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
        assert_match("E[21]", run_str(&program, "21"));
        assert_match("E[E[3]+E[5]]", run_str(&program, "3+5"));
        assert_match("E[E[3]-E[5]]", run_str(&program, "3-5"));
        // same precedence between addition (+) and subtraction (-)
        assert_match("E[E[E[3]-E[5]]+E[2]]", run_str(&program, "3-5+2"));
        assert_match("E[E[E[3]+E[5]]-E[2]]", run_str(&program, "3+5-2"));
        // higher precedence for multiplication (*) over addition (+) and subtraction (-)
        assert_match("E[E[3]+E[E[5]*E[2]]]", run_str(&program, "3+5*2"));
        assert_match("E[E[E[5]*E[2]]-E[3]]", run_str(&program, "5*2-3"));
        assert_match("E[E[E[E[1]*E[5]]*E[2]]+E[3]]", run_str(&program, "1*5*2+3"));
        // unary operator
        assert_match("E[-E[1]]", run_str(&program, "-1"));
        // highest precedence parenthesis
        assert_match("E[E[(E[E[3]+E[5]])]*E[2]]", run_str(&program, "(3+5)*2"));
    }

    #[test]
    fn test_lr5() {
        let cc = compiler::Config::default();
        assert_match(
            "L[xP[L[P[P[(n)](n)]]].xP[L[P[(n)]]].x]",
            cc_run(
                &cc,
                "
                L <- P '.x' / 'x'
                P <- P '(n)' / L
                ",
                "x(n)(n).x(n).x",
            ),
        );
    }

    // -- Lists ----------------------------------------------------------------

    #[test]
    fn test_list_with_no_list() {
        let cc = compiler::Config::default();
        let program = compile(&cc, "A <- { 'aba' }");
        let result = run(
            &program,
            vec![
                vm::Value::Char('a'),
                vm::Value::Char('b'),
                vm::Value::Char('a'),
            ],
        );
        assert!(result.is_err());
        assert_eq!(
            result.unwrap_err(),
            vm::Error::Matching(0, "Not a list".to_string())
        );
    }

    #[test]
    fn test_list_0() {
        let cc = compiler::Config::default();
        let p = compile(&cc, "A <- { 'aba' }");

        let input_with_chr = vec![vm::Value::List(vec![
            vm::Value::Char('a'),
            vm::Value::Char('b'),
            vm::Value::Char('a'),
        ])];
        assert_match("A[[aba]]", run(&p, input_with_chr).unwrap());

        let input_with_str = vec![vm::Value::List(vec![vm::Value::String("aba".to_string())])];
        assert_match("A[[aba]]", run(&p, input_with_str).unwrap())
    }

    #[test]
    fn test_list_nested_0() {
        let cc = compiler::Config::default();
        let p = compile(&cc, "A <- { { 'aba' } 'cate' }");

        let input_with_chr = vec![vm::Value::List(vec![
            vm::Value::List(vec![
                vm::Value::Char('a'),
                vm::Value::Char('b'),
                vm::Value::Char('a'),
            ]),
            vm::Value::Char('c'),
            vm::Value::Char('a'),
            vm::Value::Char('t'),
            vm::Value::Char('e'),
        ])];
        assert_match("A[[[aba]cate]]", run(&p, input_with_chr).unwrap());

        let input_with_str = vec![vm::Value::List(vec![
            vm::Value::List(vec![vm::Value::String("aba".to_string())]),
            vm::Value::String("cate".to_string()),
        ])];
        assert_match("A[[[aba]cate]]", run(&p, input_with_str).unwrap());
    }

    #[test]
    fn test_node_0() {
        let cc = compiler::Config::default();
        let p = compile(&cc, "A <- { A: 'aba' }");

        let input_with_chr = vec![vm::Value::Node {
            name: "A".to_string(),
            items: vec![
                vm::Value::Char('a'),
                vm::Value::Char('b'),
                vm::Value::Char('a'),
            ],
        }];
        assert_match("A[A[aba]]", run(&p, input_with_chr).unwrap());
    }

    // -- Error Recovery -------------------------------------------------------

    #[test]
    fn test_manual_recovery() {
        let cc = compiler::Config::default();
        let program = compile(
            &cc,
            "
            P          <- Stm+
            Stm        <- IfStm / WhileStm / AssignStm
            IfStm      <- IF LPAR^iflpar Expr^ifexpr RPAR^ifrpar Body^ifbody
            WhileStm   <- WHILE LPAR^wlpar Expr^wexpr RPAR^wrpar Body^wbody
            AssignStm  <- Identifier EQ^assigneq Expr^assignexpr SEMI^assignsemi
            Body       <- LBRK Stm* RBRK
                        / Stm

            IF         <- 'if'    _
            WHILE      <- 'while' _
            LPAR       <- '('     _
            RPAR       <- ')'     _
            LBRK       <- '{'     _
            RBRK       <- '}'     _
            EQ         <- '='     _
            SEMI       <- ';'     _

            Expr       <- Bool / Identifier / Number
            Bool       <- ('true' / 'false')     _
            Identifier <- [a-zA-Z_][a-zA-Z0-9_]* _
            Number     <- [1-9][0-9]*

            _   <- (' ' / '\n')*
            EOF <- !.

            // recovery expressions for the labels declared above

            iflpar     <- (!(Bool / Identifier / Number) .)*  // first(Expr)
            ifrpar     <- (!LBRK .)* // first(Body)
            assigneq   <- _
            assignexpr <- _
            assignsemi <- _
            ",
        );

        // makes sure the above grammar works
        assert_match(
            "P[Stm[IfStm[IF[if_[ ]]LPAR[(]Expr[Bool[false]]RPAR[)_[ ]]Body[LBRK[{]RBRK[}]]]]]",
            run_str(&program, "if (false) {}"),
        );
        assert_match(
            "P[Stm[WhileStm[WHILE[while_[ ]]LPAR[(]Expr[Bool[false]]RPAR[)_[ ]]Body[LBRK[{]RBRK[}]]]]]",
            run_str(&program, "while (false) {}"),
        );
        assert_match(
            "P[Stm[AssignStm[Identifier[var_[ ]]EQ[=_[ ]]Expr[Number[1]]SEMI[;]]]]",
            run_str(&program, "var = 1;"),
        );
        assert_match(
            "P[Stm[IfStm[IF[if_[ ]]LPAR[(]Expr[Bool[false]]RPAR[)_[ ]]Body[LBRK[{_[ ]]Stm[AssignStm[Identifier[var_[ ]]EQ[=_[ ]]Expr[Number[1]]SEMI[;_[ ]]]]RBRK[}]]]]]",
            run_str(&program, "if (false) { var = 1; }"),
        );

        // missing semicolon (`;`) at the end of the assignment statement
        assert_match(
            "P[Stm[AssignStm[Identifier[var_[ ]]EQ[=_[ ]]Expr[Number[1]]Error[assignsemi]]]]",
            run_str(&program, "var = 1"),
        );

        // Missing left parenthesis ('(') right after the if token
        assert_match(
            "P[Stm[IfStm[IF[if_[ ]]Error[iflpar]Expr[Bool[false]]RPAR[)_[ ]]Body[LBRK[{]RBRK[}]]]]]",
            run_str(&program, "if false) {}"),
        );

        // missing both left parenthesis and semicolon
        assert_match(
            "P[Stm[IfStm[IF[if_[ ]]Error[iflpar]Expr[Bool[false]]RPAR[)_[ ]]Body[LBRK[{_[ ]]Stm[AssignStm[Identifier[var_[ ]]EQ[=_[ ]]Expr[Number[1]]Error[assignsemi]]]RBRK[}]]]]]",
            run_str(&program, "if false) { var = 1 }"),
        );
    }

    // -- Expand Grammar -------------------------------------------------------

    #[test]
    fn test_expand_tree_0() {
        let cc = compiler::Config::default();

        // Program that parses the initial input
        let input_grammar = "A <- 'F'";
        let program = compile(&cc, input_grammar);
        let output = run_str(&program, "F");

        // Program that parses the output obtained upon successful
        // parsing with the initial program
        let original_ast = parser::Parser::new(input_grammar).parse().unwrap();
        let rewrite = parser::expand(original_ast).unwrap();
        let mut c = compiler::Compiler::new(cc);
        let list_program = c.compile(rewrite).unwrap();
        let value = run(&list_program, vec![output.unwrap()]).unwrap();
        assert_match("A[A[F]]", value);
    }

    // -- Test Helpers ---------------------------------------------------------

    fn compile(cc: &compiler::Config, grammar: &str) -> vm::Program {
        let mut p = parser::Parser::new(grammar);
        let ast = p.parse().unwrap();
        let mut c = compiler::Compiler::new(cc.clone());
        let program = c.compile(ast).unwrap();
        debug!("p: {}", program);
        program
    }

    fn run_str(program: &vm::Program, input: &str) -> Option<vm::Value> {
        let mut machine = vm::VM::new(program);
        machine.run_str(input).expect("Unexpected")
    }

    fn run(program: &vm::Program, input: Vec<vm::Value>) -> Result<Option<vm::Value>, vm::Error> {
        let mut machine = vm::VM::new(program);
        machine.run(input)
    }

    fn cc_run(cc: &compiler::Config, grammar: &str, input: &str) -> Option<vm::Value> {
        let prog = compile(cc, grammar);
        let mut machine = vm::VM::new(&prog);
        machine.run_str(input).expect("Unexpected")
    }

    fn assert_match(expected: &str, value: Option<vm::Value>) {
        assert!(value.is_some());
        assert_eq!(expected.to_string(), format::value_fmt1(&value.unwrap()));
    }
}
