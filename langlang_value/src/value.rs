use std::string::String as StdString;

use crate::source_map::Span;

#[derive(Clone, Debug, PartialEq, PartialOrd)]
pub enum Value {
    Char(Char),
    String(String),
    List(List),
    Node(Node),
    Error(Error),
}

impl Value {
    pub fn span(&self) -> Span {
        match self {
            Value::Char(v) => v.span.clone(),
            Value::String(v) => v.span.clone(),
            Value::List(v) => v.span.clone(),
            Value::Node(v) => v.span.clone(),
            Value::Error(v) => v.span.clone(),
        }
    }

    pub fn compare(&self, other: Value) -> bool {
        match (self, other) {
            (Value::Char(a), Value::Char(b)) => a.value == b.value,
            (Value::String(a), Value::String(b)) => a.value == b.value,
            _ => false,
        }
    }
}

#[derive(Clone, Debug, PartialEq, PartialOrd)]
pub struct Char {
    pub span: Span,
    pub value: char,
}

impl Char {
    pub fn new_val(span: Span, value: char) -> Value {
        Value::Char(Self::new(span, value))
    }

    pub fn new(span: Span, value: char) -> Self {
        Self { span, value }
    }
}

#[derive(Clone, Debug, PartialEq, PartialOrd)]
pub struct String {
    pub span: Span,
    pub value: StdString,
}

impl String {
    pub fn new_val(span: Span, value: StdString) -> Value {
        Value::String(Self::new(span, value))
    }

    pub fn new(span: Span, value: StdString) -> Self {
        Self { span, value }
    }
}

#[derive(Clone, Debug, PartialEq, PartialOrd)]
pub struct List {
    pub span: Span,
    pub values: Vec<Value>,
}

impl List {
    pub fn new_val(span: Span, values: Vec<Value>) -> Value {
        Value::List(Self::new(span, values))
    }

    pub fn new(span: Span, values: Vec<Value>) -> Self {
        Self { span, values }
    }
}

#[derive(Clone, Debug, PartialEq, PartialOrd)]
pub struct Node {
    pub span: Span,
    pub name: StdString,
    pub items: Vec<Value>,
}

impl Node {
    pub fn new_val(span: Span, name: StdString, items: Vec<Value>) -> Value {
        Value::Node(Self::new(span, name, items))
    }

    pub fn new(span: Span, name: StdString, items: Vec<Value>) -> Self {
        Self { span, name, items }
    }
}

#[derive(Clone, Debug, PartialEq, PartialOrd)]
pub struct Error {
    pub span: Span,
    pub label: StdString,
    pub message: Option<StdString>,
}

impl Error {
    pub fn new_val(span: Span, label: StdString, message: Option<StdString>) -> Value {
        Value::Error(Self::new(span, label, message))
    }

    pub fn new(span: Span, label: StdString, message: Option<StdString>) -> Self {
        Self {
            span,
            label,
            message,
        }
    }
}
