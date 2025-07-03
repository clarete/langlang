mod helpers;
use helpers::{assert_match, compile_file, run_str};

use langlang_lib::compiler;

#[test]
fn test_import() {
    let cc = compiler::Config::default();
    let program = compile_file(&cc, "./import_gr_expr.peg", None);

    // This test ensures that `LabelHex` is also imported as a
    // dependency from the `Hexadecimal` production.
    assert_match(
        "Expr[Term[Multi[Primary[Value[Number[Hexadecimal[0xError[LabelHex]]]]]]+Multi[Primary[Value[Number[Decimal[3]]]]]]]",
        run_str(&program, "0xG + 3"),
    )
}
