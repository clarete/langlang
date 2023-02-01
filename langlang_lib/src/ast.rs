// TODO: attach Location to AST nodes
//
// #[derive(Debug)]
// pub struct Location {
//     // how many characters have been seen since the begining of
//     // parsing
//     cursor: usize,
//     // how many end-of-line sequences seen since the begining of
//     // parsing
//     line: usize,
//     // how many characters seen since the begining of the line
//     column: usize,
// }

#[derive(Clone, Debug, Hash, PartialEq)]
pub enum AST {
    Grammar(Vec<AST>),
    Definition(String, Box<AST>),
    LabelDefinition(String, String),
    Sequence(Vec<AST>),
    Choice(Vec<AST>),
    And(Box<AST>),
    Not(Box<AST>),
    Optional(Box<AST>),
    ZeroOrMore(Box<AST>),
    OneOrMore(Box<AST>),
    Identifier(String),
    Precedence(Box<AST>, usize),
    Node(String, Vec<AST>),
    List(Vec<AST>),
    Str(String),
    Range(char, char),
    Char(char),
    Label(String, Box<AST>),
    Any,
    Empty,
}
