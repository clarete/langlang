mod helpers;

use langlang_lib::{compiler, vm};

#[test]
fn is_syntactic_sequence_with_literals() {
    // sequence with literal terminals is always syntactic
    helpers::assert_match("Syntactic0[abc]", run("Syntactic0", "abc"));

    // It doesn't expect spaces between the sequence items
    helpers::assert_err(
        vm::Error::Matching(1, "b".to_string()),
        run("Syntactic0", "a b c"),
    );
}

#[test]
fn is_not_syntactic_sequence_with_identifiers() {
    // sequence with grammar nodes that are not terminals are not syntactic
    helpers::assert_match(
        "NotSyntactic0[Syntactic0[abc]Syntactic0[abc]!]",
        run("NotSyntactic0", "abcabc!"),
    );

    // Optional spaces are introduced between the items within the top-level sequence
    helpers::assert_match(
        "NotSyntactic0[Syntactic0[abc]Syntactic0[abc]!]",
        run("NotSyntactic0", "abc abc !"),
    );
}

#[test]
fn test_lexification_on_single_item() {
    // Lexification operator on a single item within a syntactic rule
    helpers::assert_match("Ordinal[Decimal[1]st]", run("Ordinal", "1st"));

    // There should be no spaces between the decimal and ordinal string
    helpers::assert_err(
        vm::Error::Matching(1, "ord".to_string()),
        run("Ordinal", "1 st"),
    );
}

fn run(start: &str, input: &str) -> Result<Option<vm::Value>, vm::Error> {
    let cc = compiler::Config::default();
    let program = helpers::compile_file(&cc, "wshinsert.peg", start);
    helpers::run_str(&program, input)
}
