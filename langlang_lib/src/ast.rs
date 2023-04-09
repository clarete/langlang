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
    String(String),
    Range(char, char),
    Char(char),
    Label(String, Box<AST>),
    Any,
    Empty,
}

impl std::fmt::Display for AST {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> Result<(), std::fmt::Error> {
        match self {
            AST::Grammar(exprs) => {
                for expr in exprs {
                    writeln!(f, "{}", expr)?;
                }
                Ok(())
            }
            AST::Definition(name, expr) => write!(f, "{} <- {}", name, expr),
            AST::LabelDefinition(name, msg) => write!(f, "{} = \"{}\"", name, msg),
            AST::Sequence(exprs) => {
                for expr in exprs {
                    write!(f, "{}", expr)?;
                }
                Ok(())
            }
            AST::Choice(choices) => {
                for (i, choice) in choices.iter().enumerate() {
                    write!(f, "{}", choice)?;
                    if i < choices.len() - 1 {
                        write!(f, " / ")?;
                    }
                }
                Ok(())
            }
            AST::And(expr) => write!(f, "&{}", expr),
            AST::Not(expr) => write!(f, "!{}", expr),
            AST::Optional(expr) => write!(f, "{}?", expr),
            AST::ZeroOrMore(expr) => write!(f, "{}*", expr),
            AST::OneOrMore(expr) => write!(f, "{}+", expr),
            AST::Identifier(id) => write!(f, "{}", id),
            AST::Precedence(expr, level) => write!(f, "{}{}", expr, level),
            AST::Node(name, items) => {
                write!(f, "{}: {{", name)?;
                for (i, item) in items.iter().enumerate() {
                    write!(f, "{}", item)?;
                    if i < items.len() - 1 {
                        write!(f, ", ")?;
                    }
                }
                write!(f, "}}")
            }
            AST::List(items) => {
                write!(f, "{{")?;
                for (i, item) in items.iter().enumerate() {
                    write!(f, "{}", item)?;
                    if i < items.len() - 1 {
                        write!(f, ", ")?;
                    }
                }
                write!(f, "}}")
            }
            AST::String(s) => write!(f, "{}", s),
            AST::Range(a, b) => write!(f, "[{}-{}]", a, b),
            AST::Char(c) => write!(f, "{}", c),
            AST::Label(n, expr) => write!(f, "{}^{}", expr, n),
            AST::Any => write!(f, "."),
            AST::Empty => Ok(()),
        }
    }
}
