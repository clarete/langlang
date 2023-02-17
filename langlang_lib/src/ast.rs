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

#[derive(Clone, Debug, PartialEq)]
pub enum SemExprUnaryOp {
    Negative,
    Positive,
}

impl std::fmt::Display for SemExprUnaryOp {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> Result<(), std::fmt::Error> {
        match self {
            SemExprUnaryOp::Positive => write!(f, "+"),
            SemExprUnaryOp::Negative => write!(f, "-"),
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub enum SemExprBinaryOp {
    Addition,
    Subtraction,
    Division,
    Multiplication,
}

impl std::fmt::Display for SemExprBinaryOp {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> Result<(), std::fmt::Error> {
        match self {
            SemExprBinaryOp::Addition => write!(f, "+"),
            SemExprBinaryOp::Subtraction => write!(f, "-"),
            SemExprBinaryOp::Division => write!(f, "/"),
            SemExprBinaryOp::Multiplication => write!(f, "*"),
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub enum SemExpr {
    Identifier(String),
    Value(SemValue),
    BinaryOp(Box<SemExpr>, SemExprBinaryOp, Box<SemExpr>),
    UnaryOp(SemExprUnaryOp, Box<SemExpr>),
    Call(String, Vec<SemExpr>),
}

impl std::fmt::Display for SemExpr {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> Result<(), std::fmt::Error> {
        match self {
            SemExpr::BinaryOp(left, op, right) => write!(f, "{} {} {}", left, op, right),
            SemExpr::UnaryOp(operator, operand) => write!(f, "{}{}", operator, operand),
            SemExpr::Call(name, params) => {
                write!(f, "{}(", name)?;
                for (i, param) in params.iter().enumerate() {
                    write!(f, "{}", param)?;
                    if i < params.len() - 1 {
                        write!(f, ", ")?;
                    }
                }
                write!(f, ")")
            }
            SemExpr::Value(v) => write!(f, "{}", v),
            SemExpr::Identifier(i) => write!(f, "{}", i),
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub enum SemValue {
    List(Vec<SemExpr>),
    Literal(String),
    Number(i64),
    Variable(usize),
}

impl std::fmt::Display for SemValue {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> Result<(), std::fmt::Error> {
        match self {
            SemValue::Number(n) => write!(f, "{}", n),
            SemValue::Literal(l) => write!(f, "'{}'", l),
            SemValue::Variable(v) => write!(f, "%{}", v),
            SemValue::List(items) => {
                write!(f, "{{")?;
                for (i, item) in items.iter().enumerate() {
                    write!(f, "{}", item)?;
                    if i < items.len() - 1 {
                        write!(f, ", ")?;
                    }
                }
                write!(f, "}}")
            }
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub enum AST {
    Grammar(Vec<AST>),
    Definition(String, Box<AST>),
    LabelDefinition(String, String),
    SemanticAction(String, Box<SemExpr>),
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
            AST::SemanticAction(name, expr) => write!(f, "{} -> {}", name, expr),
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
            AST::Str(s) => write!(f, "{}", s),
            AST::Range(a, b) => write!(f, "[{}-{}]", a, b),
            AST::Char(c) => write!(f, "{}", c),
            AST::Label(n, expr) => write!(f, "{}^{}", expr, n),
            AST::Any => write!(f, "."),
            AST::Empty => Ok(()),
        }
    }
}
