use crate::ast::*;

pub trait Visitor<'ast>: Sized {
    fn visit_grammar(&mut self, n: &'ast Grammar) {
        walk_grammar(self, n);
    }

    fn visit_import(&mut self, n: &'ast Import) {
        walk_import(self, n)
    }

    fn visit_definition(&mut self, n: &'ast Definition) {
        walk_definition(self, n);
    }

    fn visit_expression(&mut self, n: &'ast Expression) {
        walk_expression(self, n);
    }

    fn visit_sequence(&mut self, n: &'ast Sequence) {
        walk_sequence(self, n);
    }

    fn visit_choice(&mut self, n: &'ast Choice) {
        walk_choice(self, n);
    }

    fn visit_lex(&mut self, n: &'ast Lex) {
        walk_lex(self, n);
    }

    fn visit_and(&mut self, n: &'ast And) {
        walk_and(self, n);
    }

    fn visit_not(&mut self, n: &'ast Not) {
        walk_not(self, n);
    }

    fn visit_optional(&mut self, n: &'ast Optional) {
        walk_optional(self, n);
    }

    fn visit_zero_or_more(&mut self, n: &'ast ZeroOrMore) {
        walk_zero_or_more(self, n);
    }

    fn visit_one_or_more(&mut self, n: &'ast OneOrMore) {
        walk_one_or_more(self, n);
    }

    fn visit_list(&mut self, n: &'ast List) {
        walk_list(self, n);
    }

    fn visit_node(&mut self, n: &'ast Node) {
        walk_node(self, n);
    }

    fn visit_identifier(&mut self, n: &'ast Identifier) {
        walk_identifier(self, n);
    }

    fn visit_precedence(&mut self, n: &'ast Precedence) {
        walk_precedence(self, n);
    }

    fn visit_label(&mut self, n: &'ast Label) {
        walk_label(self, n);
    }

    fn visit_literal(&mut self, n: &'ast Literal) {
        walk_literal(self, n);
    }

    fn visit_string(&mut self, _: &'ast String) {}

    fn visit_class(&mut self, _: &'ast Class) {}

    fn visit_range(&mut self, _: &'ast Range) {}

    fn visit_char(&mut self, _: &'ast Char) {}

    fn visit_any(&mut self, _: &'ast Any) {}

    fn visit_empty(&mut self, n: &'ast Empty) {
        walk_empty(self, n);
    }
}

pub fn walk_grammar<'a, V: Visitor<'a>>(visitor: &mut V, g: &'a Grammar) {
    for i in &g.imports {
        visitor.visit_import(i);
    }

    for name in &g.definition_names {
        let d = &g.definitions[name];
        visitor.visit_definition(d);
    }
}

pub fn walk_import<'a, V: Visitor<'a>>(_: &mut V, _: &'a Import) {}

pub fn walk_definition<'a, V: Visitor<'a>>(visitor: &mut V, d: &'a Definition) {
    visitor.visit_expression(&d.expr)
}

pub fn walk_expression<'a, V: Visitor<'a>>(visitor: &mut V, e: &'a Expression) {
    match e {
        Expression::Sequence(n) => visitor.visit_sequence(n),
        Expression::Choice(n) => visitor.visit_choice(n),
        Expression::Lex(n) => visitor.visit_lex(n),
        Expression::And(n) => visitor.visit_and(n),
        Expression::Not(n) => visitor.visit_not(n),
        Expression::Optional(n) => visitor.visit_optional(n),
        Expression::ZeroOrMore(n) => visitor.visit_zero_or_more(n),
        Expression::OneOrMore(n) => visitor.visit_one_or_more(n),
        Expression::Precedence(n) => visitor.visit_precedence(n),
        Expression::Label(n) => visitor.visit_label(n),
        Expression::List(n) => visitor.visit_list(n),
        Expression::Node(n) => visitor.visit_node(n),
        Expression::Identifier(n) => visitor.visit_identifier(n),
        Expression::Literal(n) => visitor.visit_literal(n),
        Expression::Empty(n) => visitor.visit_empty(n),
    }
}

pub fn walk_sequence<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a Sequence) {
    for i in &n.items {
        visitor.visit_expression(i)
    }
}

pub fn walk_choice<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a Choice) {
    for i in &n.items {
        visitor.visit_expression(i)
    }
}

pub fn walk_lex<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a Lex) {
    visitor.visit_expression(&n.expr)
}

pub fn walk_and<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a And) {
    visitor.visit_expression(&n.expr)
}

pub fn walk_not<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a Not) {
    visitor.visit_expression(&n.expr)
}

pub fn walk_optional<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a Optional) {
    visitor.visit_expression(&n.expr)
}

pub fn walk_zero_or_more<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a ZeroOrMore) {
    visitor.visit_expression(&n.expr)
}

pub fn walk_one_or_more<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a OneOrMore) {
    visitor.visit_expression(&n.expr)
}

pub fn walk_list<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a List) {
    for i in &n.items {
        visitor.visit_expression(i)
    }
}

pub fn walk_node<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a Node) {
    visitor.visit_expression(&n.expr)
}

pub fn walk_identifier<'a, V: Visitor<'a>>(_: &mut V, _: &'a Identifier) {}

pub fn walk_precedence<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a Precedence) {
    visitor.visit_expression(&n.expr)
}

pub fn walk_label<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a Label) {
    visitor.visit_expression(&n.expr)
}

pub fn walk_literal<'a, V: Visitor<'a>>(visitor: &mut V, n: &'a Literal) {
    match n {
        Literal::String(v) => visitor.visit_string(v),
        Literal::Class(v) => visitor.visit_class(v),
        Literal::Range(v) => visitor.visit_range(v),
        Literal::Char(v) => visitor.visit_char(v),
        Literal::Any(v) => visitor.visit_any(v),
    }
}

pub fn walk_empty<'a, V: Visitor<'a>>(_: &mut V, _: &'a Empty) {}
