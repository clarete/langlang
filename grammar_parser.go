package parsing

type GrammarParser struct {
	BaseParser
}

func NewGrammarParser(grammar string) *GrammarParser {
	return &GrammarParser{BaseParser{input: []rune(grammar)}}
}

// Parse kicks off parsing the input string and generates an AST
// describing a grammar
func (p *GrammarParser) Parse() (Node, error) {
	return p.ParseGrammar()
}

// GR: Grammar <- Spacing Definition+ EndOfFile
func (p *GrammarParser) ParseGrammar() (Node, error) {
	p.ParseSpacing()
	start := p.Location()
	defs, err := OneOrMore(p, func(p Parser) (Node, error) {
		return p.(*GrammarParser).ParseDefinition()
	})
	if err != nil {
		return nil, err
	}
	if _, err := Not(p, p.ExpectRuneFn('.')); err != nil {
		return nil, err
	}
	return NewGrammarNode(defs, NewSpan(start, p.Location())), nil
}

// GR: Definition <- Identifier LEFTARROW Expression
func (p *GrammarParser) ParseDefinition() (Node, error) {
	start := p.Location()
	identifier, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	p.ParseSpacing()
	if err := p.ParseLeftArrow(); err != nil {
		return nil, err
	}
	expr, err := p.ParseExpression()
	if err != nil {
		return nil, err
	}
	return NewDefinitionNode(identifier, expr, NewSpan(start, p.Location())), nil
}

// GR: Expression <- Sequence (SLASH Sequence)*
func (p *GrammarParser) ParseExpression() (Node, error) {
	start := p.Location()
	head, err := p.ParseSequence()
	if err != nil {
		return nil, err
	}
	tail, err := ZeroOrMore(p, func(p Parser) (Node, error) {
		if _, err := p.ExpectRune('/'); err != nil {
			return nil, err
		}
		p.(*GrammarParser).ParseSpacing()

		return p.(*GrammarParser).ParseSequence()
	})
	if err != nil {
		return nil, err
	}
	if len(tail) == 0 {
		return head, nil
	}
	items := append([]Node{head}, tail...)
	return NewChoiceNode(items, NewSpan(start, p.Location())), nil
}

// GR: Sequence <- Prefix*
func (p *GrammarParser) ParseSequence() (Node, error) {
	start := p.Location()
	items, err := ZeroOrMore(p, func(p Parser) (Node, error) {
		return p.(*GrammarParser).ParsePrefix()
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 1 {
		return items[0], nil
	}
	return NewSequenceNode(items, NewSpan(start, p.Location())), nil
}

// GR: Prefix <- (AND / NOT)? Suffix
func (p *GrammarParser) ParsePrefix() (Node, error) {
	start := p.Location()
	prefix, err := Choice(p, []ParserFn[rune]{
		p.ExpectRuneFn('&'),
		p.ExpectRuneFn('!'),
		func(p Parser) (rune, error) { return 0, nil },
	})
	if err != nil {
		return nil, err
	}
	suffix, err := p.ParseSuffix()
	if err != nil {
		return nil, err
	}
	switch prefix {
	case '&':
		return NewAndNode(suffix, NewSpan(start, p.Location())), nil
	case '!':
		return NewNotNode(suffix, NewSpan(start, p.Location())), nil
	default:
		return suffix, nil
	}
}

// GR: Suffix <- Primary (QUESTION / STAR / PLUS)?
func (p *GrammarParser) ParseSuffix() (Node, error) {
	start := p.Location()
	primary, err := p.ParsePrimary()
	if err != nil {
		return nil, err
	}
	suffix, err := Choice(p, []ParserFn[rune]{
		p.ExpectRuneFn('?'),
		p.ExpectRuneFn('*'),
		p.ExpectRuneFn('+'),
		func(p Parser) (rune, error) { return 0, nil },
	})
	if err != nil {
		return nil, err
	}

	p.ParseSpacing()

	switch suffix {
	case '?':
		return NewOptionalNode(primary, NewSpan(start, p.Location())), nil
	case '*':
		return NewZeroOrMoreNode(primary, NewSpan(start, p.Location())), nil
	case '+':
		return NewOneOrMoreNode(primary, NewSpan(start, p.Location())), nil
	default:
		return primary, nil
	}
}

// GR: Primary <- Identifier !LEFTARROW
// GR:          / OPEN Expression CLOSE
// GR:          / Literal / Class / DOT
func (p *GrammarParser) ParsePrimary() (Node, error) {
	return Choice(p, []ParserFn[Node]{
		func(p Parser) (Node, error) { return p.(*GrammarParser).ParseIdentifier() },
		func(p Parser) (Node, error) { return p.(*GrammarParser).ParseParenExpression() },
		func(p Parser) (Node, error) { return p.(*GrammarParser).ParseLiteral() },
		func(p Parser) (Node, error) { return p.(*GrammarParser).ParseClass() },
		func(p Parser) (Node, error) { return p.(*GrammarParser).ParseDot() },
	})
}

// GR: Identifier <- IdentStart IdentCont* Spacing
// GR: IdentStart <- [a-zA-Z_]
// GR: IdentCont  <- IdentStart / [0-9]
func (p *GrammarParser) ParseIdentifier() (Node, error) {
	start := p.Location()
	value, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	end := p.Location()
	p.ParseSpacing()

	if _, err := Not(p, func(p Parser) (Node, error) {
		return nil, p.(*GrammarParser).ParseLeftArrow()
	}); err != nil {
		return nil, err
	}

	return NewIdentifierNode(value, NewSpan(start, end)), nil
}

func (p *GrammarParser) parseIdentifier() (string, error) {
	head, err := Choice(p, []ParserFn[rune]{
		p.ExpectRangeFn('a', 'z'),
		p.ExpectRangeFn('A', 'Z'),
		p.ExpectRuneFn('_'),
	})
	if err != nil {
		return "", err
	}
	tail, err := ZeroOrMore(p, func(p Parser) (rune, error) {
		return Choice(p, []ParserFn[rune]{
			p.ExpectRangeFn('a', 'z'),
			p.ExpectRangeFn('A', 'Z'),
			p.ExpectRuneFn('_'),
		})
	})
	if err != nil {
		return "", err
	}

	return string(append([]rune{head}, tail...)), nil
}

func (p *GrammarParser) ParseLeftArrow() error {
	if _, err := p.ExpectRune('<'); err != nil {
		return err
	}
	if _, err := p.ExpectRune('-'); err != nil {
		return err
	}
	p.ParseSpacing()
	return nil
}

func (p *GrammarParser) ParseParenExpression() (Node, error) {
	if _, err := p.ExpectRune('('); err != nil {
		return nil, err
	}

	expr, err := p.ParseExpression()
	if err != nil {
		return nil, err
	}

	if _, err := p.ExpectRune(')'); err != nil {
		return nil, err
	}

	return expr, nil
}

// GR: Class <- '[' (!']' Range)* ']' Spacing
func (p *GrammarParser) ParseClass() (Node, error) {
	start := p.Location()
	if _, err := p.ExpectRune('['); err != nil {
		return nil, err
	}
	ranges, err := ZeroOrMore(p, func(p Parser) (Node, error) {
		if _, err := Not(p, p.ExpectRuneFn(']')); err != nil {
			return nil, err
		}
		return p.(*GrammarParser).ParseRange()
	})
	if err != nil {
		return nil, err
	}
	if _, err := p.ExpectRune(']'); err != nil {
		return nil, err
	}
	p.ParseSpacing()
	return NewClassNode(ranges, NewSpan(start, p.Location())), nil
}

// GR: Range <- Char '-' Char / Char
func (p *GrammarParser) ParseRange() (Node, error) {
	return Choice(p, []ParserFn[Node]{
		func(p Parser) (Node, error) {
			start := p.Location()
			left, err := p.(*GrammarParser).parseChar()
			if err != nil {
				return nil, err
			}
			if _, err := p.ExpectRune('-'); err != nil {
				return nil, err
			}
			right, err := p.(*GrammarParser).parseChar()
			if err != nil {
				return nil, err
			}
			return NewRangeNode(left, right, NewSpan(start, p.Location())), nil
		},
		func(p Parser) (Node, error) {
			return p.(*GrammarParser).ParseChar()
		},
	})
}

func (p *GrammarParser) ParseDot() (Node, error) {
	start := p.Location()
	_, err := p.ExpectRune('.')
	if err != nil {
		return nil, err
	}
	p.ParseSpacing()
	return NewAnyNode(NewSpan(start, p.Location())), nil
}

// GR: Literal <- ['] (!['] Char)* ['] Spacing
// GR:          / ["] (!["] Char)* ["] Spacing
func (p *GrammarParser) ParseLiteral() (Node, error) {
	start := p.Location()
	value, err := Choice(p, []ParserFn[string]{
		func(p Parser) (string, error) {
			if _, err := p.ExpectRune('\''); err != nil {
				return "", err
			}
			s, err := ZeroOrMore(p, func(p Parser) (rune, error) {
				if _, err := Not(p, p.ExpectRuneFn('\'')); err != nil {
					return 0, err
				}
				return p.Any()
			})
			if err != nil {
				return "", err
			}
			if _, err := p.ExpectRune('\''); err != nil {
				return "", err
			}
			return string(s), nil
		},
		func(p Parser) (string, error) {
			if _, err := p.ExpectRune('"'); err != nil {
				return "", err
			}
			s, err := ZeroOrMore(p, func(p Parser) (rune, error) {
				if _, err := Not(p, p.ExpectRuneFn('"')); err != nil {
					return 0, err
				}
				return p.Any()
			})
			if err != nil {
				return "", err
			}
			if _, err := p.ExpectRune('"'); err != nil {
				return "", err
			}
			return string(s), nil
		},
	})
	if err != nil {
		return nil, err
	}
	span := NewSpan(start, p.Location())
	p.ParseSpacing()
	return NewLiteralNode(value, span), nil
}

// !'\\' .
func (p *GrammarParser) ParseChar() (Node, error) {
	start := p.Location()
	value, err := p.parseChar()
	if err != nil {
		return nil, err
	}
	return NewLiteralNode(value, NewSpan(start, p.Location())), nil
}

func (p *GrammarParser) parseChar() (string, error) {
	if _, err := Not(p, p.ExpectRuneFn('\\')); err != nil {
		return "", err
	}
	value, err := p.Any()
	if err != nil {
		return "", err
	}
	return string(value), nil
}

// GR: ParseSpacing <- ' ' / '\t' / '\r' / '\n'
func (p *GrammarParser) ParseSpacing() {
	ZeroOrMore(p, func(p Parser) (rune, error) {
		return ChoiceRune(p, []rune{' ', '\t', '\r', '\n'})
	})
}
