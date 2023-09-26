use crate::ast;
use langlang_value::source_map::{Position, Span};
use std::collections::HashMap;

#[derive(Debug)]
pub enum Error {
    BacktrackError(usize, String),
}

impl std::error::Error for Error {}

impl std::fmt::Display for Error {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        match self {
            Error::BacktrackError(i, m) => write!(f, "Syntax Error: {}: {}", i, m),
        }
    }
}

pub fn parse(input: &str) -> Result<ast::Grammar, Error> {
    let mut p = Parser::new(input);
    p.parse_grammar()
}

pub struct Parser {
    ffp: usize,
    cursor: usize,
    line: usize,
    column: usize,
    source: Vec<char>,
}

type ParseFn<T> = fn(&mut Parser) -> Result<T, Error>;

impl Parser {
    pub fn new(s: &str) -> Self {
        return Parser {
            ffp: 0,
            cursor: 0,
            line: 0,
            column: 0,
            source: s.chars().collect(),
        };
    }

    // GR: Grammar <- Spacing Import* Definition* EndOfFile
    pub fn parse_grammar(&mut self) -> Result<ast::Grammar, Error> {
        self.parse_spacing()?;
        let start = self.pos();
        let imports = self.zero_or_more(|p| p.parse_import())?;
        let mut defs = HashMap::new();
        let mut def_names = Vec::new();
        self.zero_or_more(|p| {
            let def = p.parse_definition()?;
            def_names.push(def.name.clone());
            defs.insert(def.name.clone(), def);
            Ok(())
        })?;
        self.parse_eof()?;
        let span = self.span_from(start);
        Ok(ast::Grammar::new(span, imports, def_names, defs))
    }

    // GR: Import <- "@import" Identifier ("," Identifier)* "from" Literal
    fn parse_import(&mut self) -> Result<ast::Import, Error> {
        self.parse_spacing()?;
        let start = self.pos();
        self.expect_str("@import")?;
        let mut names = vec![self.parse_identifier()?];
        names.append(&mut self.zero_or_more(|p| {
            p.parse_spacing()?;
            p.expect(',')?;
            p.parse_identifier()
        })?);
        self.parse_spacing()?;
        self.expect_str("from")?;
        self.parse_spacing()?;
        let path = self.parse_literal_string()?;
        let span = self.span_from(start);
        Ok(ast::Import::new(span, path, names))
    }

    // GR: Definition <- Identifier LEFTARROW Expression
    fn parse_definition(&mut self) -> Result<ast::Definition, Error> {
        self.parse_spacing()?;
        let start = self.pos();
        let id = self.parse_identifier()?;

        self.parse_spacing()?;
        self.expect('<')?;
        self.expect('-')?;

        let expr = self.parse_expression()?;
        let span = self.span_from(start);
        Ok(ast::Definition::new(span, id, expr))
    }

    // GR: Expression <- Sequence (SLASH Sequence)*
    fn parse_expression(&mut self) -> Result<ast::Expression, Error> {
        let start = self.pos();
        let first = self.parse_sequence()?;
        let mut choices = vec![first];
        choices.append(&mut self.zero_or_more(|p| {
            p.expect('/')?;
            p.parse_spacing()?;
            p.parse_sequence()
        })?);
        Ok(if choices.len() == 1 {
            choices.remove(0)
        } else {
            let span = self.span_from(start);
            ast::Choice::new_expr(span, choices)
        })
    }

    // GR: Sequence <- Prefix*
    fn parse_sequence(&mut self) -> Result<ast::Expression, Error> {
        let start = self.pos();
        let seq = self.zero_or_more(|p| p.parse_prefix())?;
        let span = self.span_from(start);
        Ok(if seq.is_empty() {
            let empty = vec![ast::Empty::new_expr(span.clone())];
            ast::Sequence::new_expr(span, empty)
        } else {
            ast::Sequence::new_expr(span, seq)
        })
    }

    // GR: Prefix <- ('#' / '&' / '!')? Labeled
    fn parse_prefix(&mut self) -> Result<ast::Expression, Error> {
        self.parse_spacing()?;
        let start = self.pos();
        let prefix = self.choice(vec![
            |p| p.expect_str("#"),
            |p| p.expect_str("&"),
            |p| p.expect_str("!"),
            |_| Ok("".to_string()),
        ])?;
        let labeled = self.parse_labeled()?;
        let span = self.span_from(start);
        Ok(match prefix.as_str() {
            "#" => ast::Expression::Lex(ast::Lex::new(span, Box::new(labeled))),
            "&" => ast::Expression::And(ast::And::new(span, Box::new(labeled))),
            "!" => ast::Expression::Not(ast::Not::new(span, Box::new(labeled))),
            _ => labeled,
        })
    }

    // GR: Labeled <- Suffix Label?
    fn parse_labeled(&mut self) -> Result<ast::Expression, Error> {
        self.parse_spacing()?;
        let start = self.pos();
        let suffix = self.parse_suffix()?;
        Ok(match self.parse_label() {
            Ok(label) => {
                let span = self.span_from(start);
                ast::Label::new_expr(span, label, Box::new(suffix))
            }
            _ => suffix,
        })
    }

    // GR: Label <- [^⇑] Identifier
    fn parse_label(&mut self) -> Result<String, Error> {
        self.choice(vec![|p| p.expect_str("^"), |p| p.expect_str("⇑")])?;
        self.parse_identifier()
    }

    // GR: Suffix <- Primary (QUESTION / STAR / PLUS / Superscript)?
    fn parse_suffix(&mut self) -> Result<ast::Expression, Error> {
        self.parse_spacing()?;
        let start = self.pos();
        let primary = self.parse_primary()?;

        self.parse_spacing()?;
        let suffix = self.choice(vec![
            |p| p.expect_str("?"),
            |p| p.expect_str("*"),
            |p| p.expect_str("+"),
            |p| p.expect_str("¹"),
            |p| p.expect_str("²"),
            |p| p.expect_str("³"),
            |p| p.expect_str("⁴"),
            |p| p.expect_str("⁵"),
            |p| p.expect_str("⁶"),
            |p| p.expect_str("⁷"),
            |p| p.expect_str("⁸"),
            |p| p.expect_str("⁹"),
            |_| Ok("".to_string()),
        ])?;
        let span = self.span_from(start);
        Ok(match suffix.as_ref() {
            "?" => ast::Optional::new_expr(span, Box::new(primary)),
            "*" => ast::ZeroOrMore::new_expr(span, Box::new(primary)),
            "+" => ast::OneOrMore::new_expr(span, Box::new(primary)),
            "¹" => ast::Precedence::new_expr(span, Box::new(primary), 1),
            "²" => ast::Precedence::new_expr(span, Box::new(primary), 2),
            "³" => ast::Precedence::new_expr(span, Box::new(primary), 3),
            "⁴" => ast::Precedence::new_expr(span, Box::new(primary), 4),
            "⁵" => ast::Precedence::new_expr(span, Box::new(primary), 5),
            "⁶" => ast::Precedence::new_expr(span, Box::new(primary), 6),
            "⁷" => ast::Precedence::new_expr(span, Box::new(primary), 7),
            "⁸" => ast::Precedence::new_expr(span, Box::new(primary), 8),
            "⁹" => ast::Precedence::new_expr(span, Box::new(primary), 9),
            _ => primary,
        })
    }

    // GR: Primary <- Identifier !(LEFTARROW / (Identifier EQ))
    // GR:          / OPEN Expression CLOSE
    // GR:          / Node / List / Literal / Class / DOT
    fn parse_primary(&mut self) -> Result<ast::Expression, Error> {
        self.parse_spacing()?;
        self.choice(vec![
            |p| {
                let start = p.pos();
                let id = p.parse_identifier()?;
                p.not(|p| {
                    p.parse_spacing()?;
                    p.expect_str("<-")
                })?;
                let span = p.span_from(start);
                Ok(ast::Identifier::new_expr(span, id))
            },
            |p| {
                p.parse_spacing()?;
                p.expect('(')?;
                let expr = p.parse_expression()?;
                p.parse_spacing()?;
                p.expect(')')?;
                Ok(expr)
            },
            |p| p.parse_node(),
            |p| p.parse_list(),
            |p| p.parse_literal(),
            |p| p.parse_class(),
            |p| p.parse_dot(),
        ])
    }

    // GR: Node <- OPENC (!CLOSEC Expression)* CLOSEC
    fn parse_node(&mut self) -> Result<ast::Expression, Error> {
        self.parse_spacing()?;
        let start = self.pos();
        self.expect('{')?;

        let name = self.parse_identifier()?;
        self.parse_spacing()?;
        self.expect(':')?;

        let expr = self.parse_expression()?;
        self.parse_spacing()?;
        self.expect('}')?;
        let span = self.span_from(start);
        Ok(ast::Node::new_expr(span, name, Box::new(expr)))
    }

    // GR: List <- OPENC (!CLOSEC Expression)* CLOSEC
    fn parse_list(&mut self) -> Result<ast::Expression, Error> {
        self.parse_spacing()?;
        let start = self.pos();
        self.expect('{')?;
        let exprs = self.zero_or_more(|p| {
            p.not(|p| p.expect('}'))?;
            p.parse_expression()
        })?;
        self.parse_spacing()?;
        self.expect('}')?;
        let span = self.span_from(start);
        Ok(ast::List::new_expr(span, exprs))
    }

    // GR: Identifier <- IdentStart IdentCont* Spacing
    // GR: IdentStart <- [a-zA-Z_]
    // GR: IdentCont <- IdentStart / [0-9]
    fn parse_identifier(&mut self) -> Result<String, Error> {
        self.parse_spacing()?;
        let ident_start = self.choice(vec![
            |p| p.expect_range('a', 'z'),
            |p| p.expect_range('A', 'Z'),
            |p| p.expect('_'),
        ])?;
        let ident_cont = self.zero_or_more(|p| {
            p.choice(vec![
                |p| p.expect_range('a', 'z'),
                |p| p.expect_range('A', 'Z'),
                |p| p.expect_range('0', '9'),
                |p| p.expect('_'),
            ])
        })?;
        let cont_str: String = ident_cont.into_iter().collect();
        let id = format!("{}{}", ident_start, cont_str);
        Ok(id)
    }

    // GR: Literal <- [’] (![’]Char)* [’] Spacing
    // GR:          / ["] (!["]Char)* ["] Spacing
    fn parse_literal(&mut self) -> Result<ast::Expression, Error> {
        self.parse_spacing()?;
        let start = self.pos();
        let value = self.parse_literal_string()?;
        let span = self.span_from(start);
        Ok(ast::String::new_expr(span, value))
    }

    fn parse_literal_string(&mut self) -> Result<String, Error> {
        self.choice(vec![|p| p.parse_simple_quote(), |p| p.parse_double_quote()])
    }

    fn parse_simple_quote(&mut self) -> Result<String, Error> {
        self.expect('\'')?;
        let r = self
            .zero_or_more(|p| {
                p.not(|p| p.expect('\''))?;
                p.parse_char()
            })?
            .into_iter()
            .collect();
        self.expect('\'')?;
        Ok(r)
    }

    // TODO: duplicated the above code as I can't pass the quote as a
    // parameter to a more generic function. The `zero_or_more` parser
    // and all the other parsers expect a function pointer, not a
    // closure, and ~const Q: &'static str~ isn't allowed by default.
    fn parse_double_quote(&mut self) -> Result<String, Error> {
        self.expect('"')?;
        let r = self
            .zero_or_more(|p| {
                p.not(|p| p.expect('"'))?;
                p.parse_char()
            })?
            .into_iter()
            .collect();
        self.expect('"')?;
        Ok(r)
    }

    // GR: Class <- ’[’ (!’]’Range)* ’]’ Spacing
    fn parse_class(&mut self) -> Result<ast::Expression, Error> {
        self.parse_spacing()?;
        let start = self.pos();
        self.expect('[')?;
        let ranges = self.zero_or_more::<ast::Literal, _>(|p| {
            p.not(|pp| pp.expect(']'))?;
            p.parse_range()
        })?;
        self.expect(']')?;
        let span = self.span_from(start);
        Ok(ast::Class::new_expr(span, ranges))
    }

    // GR: Range <- Char ’-’ Char / Char
    fn parse_range(&mut self) -> Result<ast::Literal, Error> {
        self.choice(vec![
            |p| {
                let start = p.pos();
                let left = p.parse_char()?;
                p.expect('-')?;
                let right = p.parse_char()?;
                let span = p.span_from(start);
                Ok(ast::Literal::Range(ast::Range::new(span, left, right)))
            },
            |p| {
                let start = p.pos();
                let c = p.parse_char()?;
                let s = p.span_from(start);
                Ok(ast::Literal::Char(ast::Char::new(s, c)))
            },
        ])
    }

    // GR: Char <- ’\\’ [nrt’"\[\]\\]
    // GR:       / ’\\’ [0-2][0-7][0-7]
    // GR:       / ’\\’ [0-7][0-7]?
    // GR:       / !’\\’ .
    fn parse_char(&mut self) -> Result<char, Error> {
        self.choice(vec![|p| p.parse_char_escaped(), |p| {
            p.parse_char_non_escaped()
        }])
    }

    // ’\\’ [nrt’"\[\]\\]
    fn parse_char_escaped(&mut self) -> Result<char, Error> {
        self.expect('\\')?;
        self.choice(vec![
            |p| {
                p.expect('n')?;
                Ok('\n')
            },
            |p| {
                p.expect('r')?;
                Ok('\r')
            },
            |p| {
                p.expect('t')?;
                Ok('\t')
            },
            |p| {
                p.expect('\'')?;
                Ok('\'')
            },
            |p| {
                p.expect('"')?;
                Ok('\"')
            },
            |p| {
                p.expect(']')?;
                Ok(']')
            },
            |p| {
                p.expect('[')?;
                Ok('[')
            },
            |p| {
                p.expect('\\')?;
                Ok('\\')
            },
            |p| {
                p.expect('\'')?;
                Ok('\'')
            },
            |p| {
                p.expect('"')?;
                Ok('"')
            },
        ])
    }

    // !’\\’ .
    fn parse_char_non_escaped(&mut self) -> Result<char, Error> {
        self.not(|p| p.expect('\\'))?;
        self.any()
    }

    // GR: DOT <- '.' Spacing
    fn parse_dot(&mut self) -> Result<ast::Expression, Error> {
        self.parse_spacing()?;
        let start = self.pos();
        self.expect('.')?;
        let span = self.span_from(start);
        Ok(ast::Any::new_expr(span))
    }

    // GR: Spacing <- (Space/ Comment)*
    fn parse_spacing(&mut self) -> Result<(), Error> {
        self.zero_or_more(|p| p.choice(vec![|p| p.parse_space(), |p| p.parse_comment()]))?;
        Ok(())
    }

    // GR: Comment <- ’//’ (!EndOfLine.)* EndOfLine
    fn parse_comment(&mut self) -> Result<(), Error> {
        self.expect('/')?;
        self.expect('/')?;
        self.zero_or_more(|p| {
            p.not(|p| p.parse_eol())?;
            p.any()
        })?;
        self.parse_eol()
    }

    // GR: Space <- ’ ’ / ’\t’ / EndOfLine
    fn parse_space(&mut self) -> Result<(), Error> {
        self.choice(vec![
            |p| {
                p.expect(' ')?;
                Ok(())
            },
            |p| {
                p.expect('\t')?;
                Ok(())
            },
            |p| p.parse_eol(),
        ])
    }

    // EndOfLine <- ’\r\n’ / ’\n’ / ’\r’
    fn parse_eol(&mut self) -> Result<(), Error> {
        self.choice(vec![
            |p| {
                p.expect('\r')?;
                p.expect('\n')
            },
            |p| p.expect('\n'),
            |p| p.expect('\r'),
        ])?;
        Ok(())
    }

    // EndOfFile <- !.
    fn parse_eof(&mut self) -> Result<(), Error> {
        self.not(|p| p.current())?;
        Ok(())
    }

    fn choice<T>(&mut self, funcs: Vec<ParseFn<T>>) -> Result<T, Error> {
        let cursor = self.cursor;
        let column = self.column;
        let line = self.line;
        for func in &funcs {
            match func(self) {
                Ok(o) => return Ok(o),
                Err(_) => {
                    self.cursor = cursor;
                    self.column = column;
                    self.line = line;
                }
            }
        }
        Err(self.err("CHOICE".to_string()))
    }

    fn not<T>(&mut self, func: ParseFn<T>) -> Result<(), Error> {
        let cursor = self.cursor;
        let out = func(self);
        self.cursor = cursor;
        match out {
            Err(_) => Ok(()),
            Ok(_) => Err(self.err("NOT".to_string())),
        }
    }

    // fn one_or_more<T>(&mut self, func: ParseFn<T>) -> Result<Vec<T>, Error> {
    //     let mut output = vec![func(self)?];
    //     output.append(&mut self.zero_or_more::<T>(func)?);
    //     Ok(output)
    // }

    fn zero_or_more<T, P>(&mut self, mut func: P) -> Result<Vec<T>, Error>
    where
        P: FnMut(&mut Parser) -> Result<T, Error>,
    {
        let mut output = vec![];
        loop {
            match func(self) {
                Ok(ch) => output.push(ch),
                Err(e) => match e {
                    Error::BacktrackError(..) => break,
                },
            }
        }
        Ok(output)
    }

    /// If the character under the cursor isn't between `a` and `b`,
    /// return an error, otherwise return the current char and move
    /// the read cursor forward
    fn expect_range(&mut self, a: char, b: char) -> Result<char, Error> {
        let current = self.current()?;
        if current >= a && current <= b {
            self.next()?;
            return Ok(current);
        }
        Err(self.err(format!(
            "Expected char between `{}' and `{}' but got `{}' instead",
            a, b, current
        )))
    }

    /// Tries to match each character within `expected` against the
    /// input source.  It starts from where the read cursor currently is.
    fn expect_str(&mut self, expected: &str) -> Result<String, Error> {
        for c in expected.chars() {
            self.expect(c)?;
        }
        Ok(expected.to_string())
    }

    /// Compares `expected` to the current character under the cursor,
    /// and advance the cursor if they match.  Returns an error
    /// otherwise.
    fn expect(&mut self, expected: char) -> Result<char, Error> {
        let current = self.current()?;
        if current == expected {
            self.next()?;
            return Ok(current);
        }
        Err(self.err(format!(
            "Expected `{}' but got `{}' instead",
            expected, current
        )))
    }

    /// If it's not the end of the input, return the current char and
    /// increment the read cursor
    fn any(&mut self) -> Result<char, Error> {
        let current = self.current()?;
        self.next()?;
        Ok(current)
    }

    /// Retrieve the character within source under the read cursor, or
    /// EOF it it's the end of the input source stream
    fn current(&mut self) -> Result<char, Error> {
        if !self.eof() {
            return Ok(self.source[self.cursor]);
        }
        Err(self.err("EOF".to_string()))
    }

    /// Returns true if the cursor equals the length of the input source
    fn eof(&self) -> bool {
        self.cursor == self.source.len()
    }

    /// Increments the read cursor, and the farther failure position
    /// if it's farther than before the last call
    fn next(&mut self) -> Result<(), Error> {
        let c = self.current()?;
        self.cursor += 1;
        self.column += 1;
        if c == '\n' {
            self.column = 0;
            self.line += 1;
        }
        if self.cursor > self.ffp {
            self.ffp = self.cursor;
        }
        Ok(())
    }

    fn span_from(&self, start: Position) -> Span {
        Span::new(start, self.pos())
    }

    fn pos(&self) -> Position {
        Position::new(self.cursor, self.line, self.column)
    }

    /// produce a backtracking error with `message` attached to it
    fn err(&mut self, msg: String) -> Error {
        Error::BacktrackError(self.ffp, msg)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn roundtrip_via_stringify() {
        let tests = [
            ("A <- .", "A <- .\n"),
            ("A <- .\n", "A <- .\n"),
            ("A <- 'a'\n", "A <- \"a\"\n"),
            ("A <- [a-z]\n", "A <- [a-z]\n"),
            ("A <- 'a' / [b-e]\n", "A <- (\"a\" / [b-e])\n"),
        ];
        for (input, expected) in &tests {
            let output = parse(input);
            assert!(output.is_ok());
            assert_eq!(expected, &output.unwrap().to_string());
        }
    }

    // #[test]
    // fn test_precedence_syntax() {
    //     let mut p = Parser::new(
    //         "
    //         A <- A¹ '+' A² / 'n'
    //         ",
    //     );
    //     let ast = p.parse_grammar();
    //     assert!(ast.is_ok());
    //     assert_eq!(
    //         AST::Grammar(vec![AST::Definition(
    //             "A".to_string(),
    //             Box::new(AST::Choice(vec![
    //                 AST::Sequence(vec![
    //                     AST::Precedence(Box::new(AST::Identifier("A".to_string())), 1),
    //                     AST::String("+".to_string()),
    //                     AST::Precedence(Box::new(AST::Identifier("A".to_string())), 2),
    //                 ]),
    //                 AST::Sequence(vec![AST::String("n".to_string()),])
    //             ])),
    //         ),]),
    //         ast.unwrap(),
    //     );
    // }

    // #[test]
    // fn structure_empty() {
    //     let mut p = Parser::new(
    //         "A <- 'a' /
    //          B <- 'b'
    //         ",
    //     );
    //     let out = p.parse_grammar();

    //     assert!(out.is_ok());
    //     assert_eq!(
    //         AST::Grammar(vec![
    //             AST::Definition(
    //                 "A".to_string(),
    //                 Box::new(AST::Choice(vec![
    //                     AST::Sequence(vec![AST::String("a".to_string())]),
    //                     AST::Sequence(vec![AST::Empty])
    //                 ]))
    //             ),
    //             AST::Definition(
    //                 "B".to_string(),
    //                 Box::new(AST::Sequence(vec![AST::String("b".to_string())])),
    //             ),
    //         ]),
    //         out.unwrap()
    //     );
    // }

    #[test]
    fn parse_range_char() {
        let mut parser = Parser::new("a");
        let out = parser.parse_range();
        let span = Span::new(Position::new(0, 0, 0), Position::new(1, 0, 1));
        let expected = ast::Literal::Char(ast::Char::new(span, 'a'));

        assert!(out.is_ok());
        assert_eq!(expected, out.unwrap());
    }

    #[test]
    fn choice_pick_none() -> Result<(), Error> {
        let mut parser = Parser::new("e");
        let out = parser.choice(vec![
            |p| p.expect('a'),
            |p| p.expect('b'),
            |p| p.expect('c'),
            |p| p.expect('d'),
        ]);

        assert!(out.is_err());
        assert_eq!(0, parser.cursor);

        Ok(())
    }

    #[test]
    fn choice_pick_last() -> Result<(), Error> {
        let mut parser = Parser::new("d");
        let out = parser.choice(vec![
            |p| p.expect('a'),
            |p| p.expect('b'),
            |p| p.expect('c'),
            |p| p.expect('d'),
        ]);

        assert!(out.is_ok());
        assert_eq!(1, parser.cursor);

        Ok(())
    }

    #[test]
    fn choice_pick_first() -> Result<(), Error> {
        let mut parser = Parser::new("a");
        let out = parser.choice(vec![|p| p.expect('a')]);

        assert!(out.is_ok());
        assert_eq!(1, parser.cursor);

        Ok(())
    }

    #[test]
    fn not_success_on_err() -> Result<(), Error> {
        let mut parser = Parser::new("a");
        let out = parser.not(|p| p.expect('b'));

        assert!(out.is_ok());
        assert_eq!(0, parser.cursor);

        Ok(())
    }

    #[test]
    fn not_err_on_match() -> Result<(), Error> {
        let mut parser = Parser::new("a");
        let out = parser.not(|p| p.expect('a'));

        assert!(out.is_err());
        assert_eq!(0, parser.cursor);

        Ok(())
    }

    #[test]
    fn zero_or_more() -> Result<(), Error> {
        let mut parser = Parser::new("ab2");

        let prefix = parser.zero_or_more::<char, _>(|p| p.expect_range('a', 'z'))?;

        assert_eq!(vec!['a', 'b'], prefix);
        assert_eq!(2, parser.cursor);

        Ok(())
    }
}
