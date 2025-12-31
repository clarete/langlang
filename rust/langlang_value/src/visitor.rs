use crate::value::*;

pub trait Visitor<'a>: Sized {
    fn visit_value(&mut self, n: &'a Value) {
        walk_value(self, n);
    }

    fn visit_list(&mut self, n: &'a List) {
        walk_list(self, n);
    }

    fn visit_node(&mut self, n: &'a Node) {
        walk_node(self, n);
    }

    fn visit_char(&mut self, _: &'a Char) {}

    fn visit_string(&mut self, _: &'a String) {}

    fn visit_error(&mut self, _: &'a Error) {}
}

pub fn walk_value<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a Value) {
    match n {
        Value::Char(v) => visitor.visit_char(v),
        Value::String(v) => visitor.visit_string(v),
        Value::List(v) => visitor.visit_list(v),
        Value::Node(v) => visitor.visit_node(v),
        Value::Error(v) => visitor.visit_error(v),
    }
}

pub fn walk_list<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a List) {
    for v in &n.values {
        visitor.visit_value(v)
    }
}

pub fn walk_node<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a Node) {
    for v in &n.items {
        visitor.visit_value(v)
    }
}
