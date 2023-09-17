use std::collections::BTreeMap;

use langlang_syntax::ast;
use langlang_syntax::ast::IsSyntactic;
use langlang_syntax::source_map::Span;

use crate::consts::WHITE_SPACE_RULE_NAME;

#[derive(Default)]
pub(crate) struct WhiteSpaceHandlerInjector {
    // depth of use of the lex ('#') operator
    lex_level: usize,
}

impl WhiteSpaceHandlerInjector {
    pub(crate) fn run(&mut self, grammar: &ast::Grammar) -> ast::Grammar {
        let mut definitions = BTreeMap::new();
        let mut definition_names = Vec::new();

        for name in &grammar.definition_names {
            let d = &grammar.definitions[name];
            definition_names.push(name.clone());

            if name == WHITE_SPACE_RULE_NAME {
                definitions.insert(name.clone(), d.clone());
                continue;
            }

            definitions.insert(
                name.to_owned(),
                ast::Definition::new(
                    d.span.clone(),
                    d.name.clone(),
                    self.expand_expr(&d.expr, true),
                ),
            );
        }

        ast::Grammar::new(
            grammar.span.clone(),
            grammar.imports.to_vec(),
            definition_names,
            definitions,
        )
    }

    fn expand_expr(&mut self, expr: &ast::Expression, consume_first: bool) -> ast::Expression {
        match expr {
            ast::Expression::Lex(node) => {
                self.lex_level += 1;
                let expr = self.expand_expr(&node.expr, true);
                self.lex_level -= 1;
                ast::Lex::new_expr(node.span.clone(), Box::new(expr))
            }
            ast::Expression::Sequence(node) => {
                let should_consume_spaces = self.lex_level == 0 && !node.is_syntactic();
                let mut items: Vec<ast::Expression> = vec![];
                for (i, item) in node.items.iter().enumerate() {
                    if should_consume_spaces && !(i == 0 && !consume_first) {
                        match item {
                            ast::Expression::Lex(_) => {}
                            _ => items.push(mkwscall(&node.span)),
                        }
                    }
                    items.push(self.expand_expr(item, true));
                }
                ast::Sequence::new_expr(node.span.clone(), items)
            }
            ast::Expression::Choice(node) => {
                if expr.is_syntactic() {
                    return ast::Choice::new_expr(
                        node.span.clone(),
                        node.items
                            .iter()
                            .map(|i| self.expand_expr(i, true))
                            .collect(),
                    );
                }
                ast::Sequence::new_expr(
                    node.span.clone(),
                    vec![
                        mkwscall(&node.span),
                        ast::Choice::new_expr(
                            node.span.clone(),
                            node.items
                                .iter()
                                .map(|i| self.expand_expr(i, false))
                                .collect(),
                        ),
                    ],
                )
            }
            ast::Expression::And(node) => ast::And::new_expr(
                node.span.clone(),
                Box::new(self.expand_expr(&node.expr, true)),
            ),
            ast::Expression::Not(node) => ast::Not::new_expr(
                node.span.clone(),
                Box::new(self.expand_expr(&node.expr, true)),
            ),
            ast::Expression::Optional(node) => ast::Optional::new_expr(
                node.span.clone(),
                Box::new(self.expand_expr(&node.expr, true)),
            ),
            ast::Expression::ZeroOrMore(node) => ast::ZeroOrMore::new_expr(
                node.span.clone(),
                Box::new(self.expand_expr(&node.expr, true)),
            ),
            ast::Expression::OneOrMore(node) => ast::OneOrMore::new_expr(
                node.span.clone(),
                Box::new(self.expand_expr(&node.expr, true)),
            ),
            ast::Expression::Precedence(node) => ast::Precedence::new_expr(
                node.span.clone(),
                Box::new(self.expand_expr(&node.expr, true)),
                node.precedence,
            ),
            ast::Expression::Label(node) => ast::Label::new_expr(
                node.span.clone(),
                node.label.clone(),
                Box::new(self.expand_expr(&node.expr, true)),
            ),
            _ => expr.clone(),
        }
    }
}

fn mkwscall(span: &Span) -> ast::Expression {
    ast::Identifier::new_expr(span.clone(), WHITE_SPACE_RULE_NAME.to_string())
}
