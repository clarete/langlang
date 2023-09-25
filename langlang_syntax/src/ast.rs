use std::collections::HashMap;
use std::fmt::Debug;
use std::string::{String as StdString, ToString};

use langlang_value::source_map::Span;

/// Grammar is the top-level AST node for the input grammar language.
#[derive(Debug)]
pub struct Grammar {
    pub span: Span,
    pub imports: Vec<Import>,
    pub definition_names: Vec<StdString>,
    pub definitions: HashMap<StdString, Definition>,
}

impl Grammar {
    pub fn new(
        span: Span,
        imports: Vec<Import>,
        definition_names: Vec<StdString>,
        definitions: HashMap<StdString, Definition>,
    ) -> Self {
        Self {
            span,
            imports,
            definition_names,
            definitions,
        }
    }

    pub fn add_definition(&mut self, d: &Definition) {
        if self.definitions.get(&d.name).is_none() {
            self.definition_names.push(d.name.clone());
            self.definitions.insert(d.name.clone(), d.clone());
        }
    }
}

impl ToString for Grammar {
    fn to_string(&self) -> StdString {
        let mut output = StdString::new();
        for i in &self.imports {
            output.push_str(&i.to_string());
            output.push('\n');
        }
        for name in &self.definition_names {
            let d = &self.definitions[name];
            output.push_str(&d.to_string());
            output.push('\n');
        }
        output
    }
}

/// Import represents an import node and contains both names to be
/// imported and the path to import the names from.
#[derive(Clone, Debug)]
pub struct Import {
    pub span: Span,
    pub path: StdString,
    pub names: Vec<StdString>,
}

impl ToString for Import {
    fn to_string(&self) -> StdString {
        format!(
            "@import {} from \"{}\"",
            fmtlistsep(", ", &self.names),
            self.path
        )
    }
}

impl Import {
    pub fn new(span: Span, path: StdString, names: Vec<StdString>) -> Self {
        Self { span, path, names }
    }
}

/// Definition represents a single production definition.  It stores
/// both the name and the expression associated with the production.
#[derive(Clone, Debug)]
pub struct Definition {
    pub span: Span,
    pub name: StdString,
    pub expr: Expression,
}

impl Definition {
    pub fn new(span: Span, name: StdString, expr: Expression) -> Self {
        Self { span, name, expr }
    }
}

impl ToString for Definition {
    fn to_string(&self) -> StdString {
        format!("{} <- {}", self.name, self.expr.to_string())
    }
}

pub trait IsSyntactic {
    fn is_syntactic(&self) -> bool {
        false
    }
}

fn is_syntactic_list<T: IsSyntactic>(items: &[T]) -> bool {
    items
        .iter()
        .map(|i| i.is_syntactic())
        .reduce(|acc, i| acc && i)
        .unwrap_or(false)
}

#[derive(Clone, Debug, PartialEq)]
pub enum Expression {
    Sequence(Sequence),
    Choice(Choice),
    Lex(Lex),
    And(And),
    Not(Not),
    Optional(Optional),
    ZeroOrMore(ZeroOrMore),
    OneOrMore(OneOrMore),
    Precedence(Precedence),
    Label(Label),
    List(List),
    Node(Node),
    Identifier(Identifier),
    Literal(Literal),
    Empty(Empty),
}

impl IsSyntactic for Expression {
    fn is_syntactic(&self) -> bool {
        match self {
            Expression::Choice(v) => is_syntactic_list(&v.items),
            Expression::Sequence(v) => v.is_syntactic(),
            Expression::Lex(_) => true,
            Expression::And(v) => v.expr.is_syntactic(),
            Expression::Not(v) => v.expr.is_syntactic(),
            Expression::Optional(v) => v.expr.is_syntactic(),
            Expression::ZeroOrMore(v) => v.expr.is_syntactic(),
            Expression::OneOrMore(v) => v.expr.is_syntactic(),
            Expression::Precedence(v) => v.expr.is_syntactic(),
            Expression::Label(v) => v.expr.is_syntactic(),
            Expression::List(v) => is_syntactic_list(&v.items),
            Expression::Node(v) => v.expr.is_syntactic(),
            Expression::Identifier(_) => false,
            Expression::Literal(_) => true,
            Expression::Empty(_) => true,
        }
    }
}

impl ToString for Expression {
    fn to_string(&self) -> StdString {
        match self {
            Expression::Choice(v) => format!("({})", fmtlistsep(" / ", &v.items)),
            Expression::Sequence(v) => fmtlistsep(" ", &v.items),
            Expression::Lex(v) => fmtprefix("#", &v.expr),
            Expression::And(v) => fmtprefix("&", &v.expr),
            Expression::Not(v) => fmtprefix("!", &v.expr),
            Expression::Optional(v) => fmtsuffix("?", &v.expr),
            Expression::ZeroOrMore(v) => fmtsuffix("*", &v.expr),
            Expression::OneOrMore(v) => fmtsuffix("+", &v.expr),
            Expression::Precedence(v) => format!("{}{}", v.expr.to_string(), v.precedence),
            Expression::Label(v) => format!("{}^{}", v.expr.to_string(), v.label),
            Expression::List(v) => format!("[{}]", fmtlistsep(", ", &v.items)),
            Expression::Node(v) => format!("{} {{{}}}", v.name, v.expr.to_string()),
            Expression::Identifier(v) => v.name.to_string(),
            Expression::Literal(v) => v.to_string(),
            Expression::Empty(_) => "".to_string(),
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Sequence {
    pub span: Span,
    pub items: Vec<Expression>,
}

impl Sequence {
    pub fn new_expr(span: Span, items: Vec<Expression>) -> Expression {
        Expression::Sequence(Self { span, items })
    }
}

impl IsSyntactic for Sequence {
    fn is_syntactic(&self) -> bool {
        is_syntactic_list(&self.items)
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Choice {
    pub span: Span,
    pub items: Vec<Expression>,
}

impl Choice {
    pub fn new_expr(span: Span, items: Vec<Expression>) -> Expression {
        Expression::Choice(Choice::new(span, items))
    }

    pub fn new(span: Span, items: Vec<Expression>) -> Self {
        Self { span, items }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Lex {
    pub span: Span,
    pub expr: Box<Expression>,
}

impl Lex {
    pub fn new_expr(span: Span, expr: Box<Expression>) -> Expression {
        Expression::Lex(Lex::new(span, expr))
    }

    pub fn new(span: Span, expr: Box<Expression>) -> Self {
        Self { span, expr }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct And {
    pub span: Span,
    pub expr: Box<Expression>,
}

impl And {
    pub fn new_expr(span: Span, expr: Box<Expression>) -> Expression {
        Expression::And(Self::new(span, expr))
    }

    pub fn new(span: Span, expr: Box<Expression>) -> Self {
        Self { span, expr }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Not {
    pub span: Span,
    pub expr: Box<Expression>,
}

impl Not {
    pub fn new_expr(span: Span, expr: Box<Expression>) -> Expression {
        Expression::Not(Self { span, expr })
    }

    pub fn new(span: Span, expr: Box<Expression>) -> Self {
        Self { span, expr }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Optional {
    pub span: Span,
    pub expr: Box<Expression>,
}

impl Optional {
    pub fn new_expr(span: Span, expr: Box<Expression>) -> Expression {
        Expression::Optional(Self { span, expr })
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct ZeroOrMore {
    pub span: Span,
    pub expr: Box<Expression>,
}

impl ZeroOrMore {
    pub fn new_expr(span: Span, expr: Box<Expression>) -> Expression {
        Expression::ZeroOrMore(Self { span, expr })
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct OneOrMore {
    pub span: Span,
    pub expr: Box<Expression>,
}

impl OneOrMore {
    pub fn new_expr(span: Span, expr: Box<Expression>) -> Expression {
        Expression::OneOrMore(Self { span, expr })
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Precedence {
    pub span: Span,
    pub expr: Box<Expression>,
    pub precedence: usize,
}

impl Precedence {
    pub fn new_expr(span: Span, expr: Box<Expression>, precedence: usize) -> Expression {
        Expression::Precedence(Self {
            span,
            expr,
            precedence,
        })
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Label {
    pub span: Span,
    pub label: StdString,
    pub expr: Box<Expression>,
}

impl Label {
    pub fn new_expr(span: Span, label: StdString, expr: Box<Expression>) -> Expression {
        Expression::Label(Self { span, label, expr })
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct List {
    pub span: Span,
    pub items: Vec<Expression>,
}

impl List {
    pub fn new_expr(span: Span, items: Vec<Expression>) -> Expression {
        Expression::List(Self { span, items })
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Node {
    pub span: Span,
    pub name: StdString,
    pub expr: Box<Expression>,
}

impl Node {
    pub fn new_expr(span: Span, name: StdString, expr: Box<Expression>) -> Expression {
        Expression::Node(Self { span, name, expr })
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Identifier {
    pub span: Span,
    pub name: StdString,
}

impl Identifier {
    pub fn new_expr(span: Span, name: StdString) -> Expression {
        Expression::Identifier(Self::new(span, name))
    }

    pub fn new(span: Span, name: StdString) -> Self {
        Self { span, name }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub enum Literal {
    String(String),
    Class(Class),
    Range(Range),
    Char(Char),
    Any(Any),
}

impl ToString for Literal {
    fn to_string(&self) -> StdString {
        match self {
            Literal::String(v) => format!("\"{}\"", v.value),
            Literal::Class(v) => v.to_string(),
            Literal::Range(v) => format!("{}-{}", v.start, v.end),
            Literal::Char(v) => v.to_string(),
            Literal::Any(_) => ".".to_string(),
        }
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct String {
    pub span: Span,
    pub value: StdString,
}

impl String {
    pub fn new_expr(span: Span, value: StdString) -> Expression {
        Expression::Literal(Literal::String(Self { span, value }))
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Class {
    pub span: Span,
    pub literals: Vec<Literal>,
}

impl Class {
    pub fn new_expr(span: Span, literals: Vec<Literal>) -> Expression {
        Expression::Literal(Literal::Class(Self { span, literals }))
    }
}

impl ToString for Class {
    fn to_string(&self) -> StdString {
        let mut output = StdString::new();
        output.push('[');
        for l in &self.literals {
            output.push_str(&l.to_string());
        }
        output.push(']');
        output
    }
}

#[derive(Clone, Debug, PartialEq)]
pub struct Range {
    pub span: Span,
    pub start: char,
    pub end: char,
}

impl Range {
    pub fn new(span: Span, start: char, end: char) -> Self {
        Self { span, start, end }
    }
}

/// Char stores the position and value of a single character matcher
#[derive(Clone, Debug, PartialEq)]
pub struct Char {
    pub span: Span,
    pub value: char,
}

impl Char {
    pub fn new(span: Span, value: char) -> Self {
        Self { span, value }
    }
}

impl ToString for Char {
    fn to_string(&self) -> StdString {
        match self.value {
            '\n' => "\\n".to_string(),
            _ => format!("{}", self.value),
        }
    }
}

/// Any is the operator that matches anything but EOF
#[derive(Clone, Debug, PartialEq)]
pub struct Any {
    pub span: Span,
}

impl Any {
    pub fn new_expr(span: Span) -> Expression {
        Expression::Literal(Literal::Any(Self { span }))
    }
}

/// Empty represents the empty alternative of an ordered choice
/// operator.  Both start and end of such span are the same as no
/// input is consumed.
#[derive(Clone, Debug, PartialEq)]
pub struct Empty {
    pub span: Span,
}

impl Empty {
    pub fn new_expr(span: Span) -> Expression {
        Expression::Empty(Self { span })
    }
}

// formatting functions

fn fmtlistsep<T: ToString>(sep: &str, items: &Vec<T>) -> StdString {
    let mut output = StdString::new();
    let len = items.len();

    for (index, item) in items.iter().enumerate() {
        output.push_str(&item.to_string());
        if index < len - 1 {
            output.push_str(sep);
        }
    }

    output
}

fn fmtprefix(prefix: &str, node: &Expression) -> StdString {
    if tree_height(node) > 1 {
        return format!("{}({})", prefix, node.to_string());
    }
    if let Expression::Sequence(seq) = node {
        if seq.items.len() > 1 {
            return format!("{}({})", prefix, node.to_string());
        }
    }
    format!("{}{}", prefix, node.to_string())
}

fn fmtsuffix(suffix: &str, node: &Expression) -> StdString {
    if tree_height(node) > 1 {
        return format!("({}){}", node.to_string(), suffix);
    }
    if let Expression::Sequence(seq) = node {
        if seq.items.len() > 1 {
            return format!("({}){}", node.to_string(), suffix);
        }
    }
    format!("{}{}", node.to_string(), suffix)
}

fn tree_height(n: &Expression) -> usize {
    match n {
        Expression::Sequence(v) => items_height(&v.items),
        Expression::Choice(v) => items_height(&v.items) + 1,
        Expression::Lex(v) => tree_height(&v.expr) + 1,
        Expression::And(v) => tree_height(&v.expr) + 1,
        Expression::Not(v) => tree_height(&v.expr) + 1,
        Expression::Optional(v) => tree_height(&v.expr) + 1,
        Expression::ZeroOrMore(v) => tree_height(&v.expr) + 1,
        Expression::OneOrMore(v) => tree_height(&v.expr) + 1,
        Expression::Precedence(v) => tree_height(&v.expr) + 1,
        Expression::Label(v) => tree_height(&v.expr) + 1,
        Expression::List(v) => items_height(&v.items) + 1,
        Expression::Node(v) => tree_height(&v.expr) + 1,
        Expression::Identifier(_) => 1,
        Expression::Literal(_) => 1,
        Expression::Empty(_) => 1,
    }
}

fn items_height(items: &[Expression]) -> usize {
    items
        .iter()
        .map(tree_height)
        .fold(usize::MIN, |a, b| a.max(b))
}
