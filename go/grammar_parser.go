package langlang

type GrammarParser struct {
	Parser
}

func NewGrammarParser(grammar string) *GrammarParser {
	return &GrammarParser{Parser{input: []rune(grammar)}}
}

// Parse kicks off parsing the input string and generates an AST
// describing a grammar
func (p *GrammarParser) Parse() (AstNode, error) {
	return p.ParseGrammar()
}

// GR: Grammar <- Import* Definition+ EndOfFile
func (p *GrammarParser) ParseGrammar() (AstNode, error) {
	p.ParseSpacing()
	start := p.Location()
	imports, err := ZeroOrMore(p, func(p Backtrackable) (*ImportNode, error) {
		return p.(*GrammarParser).ParseImport()
	})
	if err != nil {
		return nil, err
	}
	defsByName := map[string]*DefinitionNode{}
	defs, err := OneOrMore(p, func(p Backtrackable) (*DefinitionNode, error) {
		d, err := p.(*GrammarParser).ParseDefinition()
		if err != nil {
			return nil, err
		}
		defsByName[d.Name] = d
		return d, nil
	})
	if err != nil {
		return nil, err
	}

	p.ParseSpacing()
	if _, err := Not(p, p.ExpectRuneFn('.')); err != nil {
		return nil, err
	}
	return NewGrammarNode(imports, defs, defsByName, NewSpan(start, p.Location())), nil
}

// GR: Import <- '@import' Identifier ("," Identifier)* 'from' Literal
func (p *GrammarParser) ParseImport() (*ImportNode, error) {
	p.ParseSpacing()
	start := p.Location()

	if _, err := p.ExpectLiteral("@import"); err != nil {
		return nil, err
	}

	names, err := p.parseImportNames()
	if err != nil {
		return nil, err
	}

	p.ParseSpacing()
	if _, err := p.ExpectLiteral("from"); err != nil {
		return nil, err
	}

	p.ParseSpacing()
	pathStart := p.Location()
	path, err := p.parseLiteral()
	if err != nil {
		return nil, err
	}
	end := p.Location()
	pathLiteral := NewLiteralNode(path, NewSpan(pathStart, end))
	return NewImportNode(pathLiteral, names, NewSpan(start, end)), nil
}

func (p *GrammarParser) parseImportNames() ([]*LiteralNode, error) {
	p.ParseSpacing()
	start := p.Location()
	headId, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	head := NewLiteralNode(headId, NewSpan(start, p.Location()))
	tail, err := ZeroOrMore(p, func(p Backtrackable) (*LiteralNode, error) {
		p.(*GrammarParser).ParseSpacing()
		if _, err := p.(*GrammarParser).ExpectRune(','); err != nil {
			return nil, err
		}

		p.(*GrammarParser).ParseSpacing()
		start := p.Location()
		id, err := p.(*GrammarParser).parseIdentifier()
		if err != nil {
			return nil, err
		}
		return NewLiteralNode(id, NewSpan(start, p.Location())), nil
	})
	if err != nil {
		return nil, err
	}
	return append([]*LiteralNode{head}, tail...), nil
}

// GR: Definition <- Identifier LEFTARROW Expression
func (p *GrammarParser) ParseDefinition() (*DefinitionNode, error) {
	p.ParseSpacing()
	start := p.Location()
	identifier, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}

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
func (p *GrammarParser) ParseExpression() (AstNode, error) {
	p.ParseSpacing()
	start := p.Location()
	head, err := p.ParseSequence()
	if err != nil {
		return nil, err
	}
	tail, err := ZeroOrMore(p, func(p Backtrackable) (AstNode, error) {
		p.(*GrammarParser).ParseSpacing()
		if _, err := p.(*GrammarParser).ExpectRune('/'); err != nil {
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
	items := append([]AstNode{head}, tail...)
	return NewChoiceNode(items, NewSpan(start, p.Location())), nil
}

// GR: Sequence <- Prefix*
func (p *GrammarParser) ParseSequence() (AstNode, error) {
	start := p.Location()
	items, err := ZeroOrMore(p, func(p Backtrackable) (AstNode, error) {
		return p.(*GrammarParser).ParsePrefix()
	})
	if err != nil {
		return nil, err
	}

	// Note: Don't shorten the path when the sequence has a single
	// item.  We need a Sequence node with a single item in the
	// output tree.  That way, the code generator traversal can
	// properly decide introducing automatic space consumption.
	return NewSequenceNode(items, NewSpan(start, p.Location())), nil
}

// GR: Prefix <- (AND / NOT / LEX)? Labeled
func (p *GrammarParser) ParsePrefix() (AstNode, error) {
	p.ParseSpacing()
	start := p.Location()
	prefix, err := Choice(p, []ParserFn[rune]{
		p.ExpectRuneFn('&'),
		p.ExpectRuneFn('!'),
		p.ExpectRuneFn('#'),
		func(p Backtrackable) (rune, error) { return 0, nil },
	})
	if err != nil {
		return nil, err
	}
	expr, err := p.ParseLabeled()
	if err != nil {
		return nil, err
	}
	switch prefix {
	case '&':
		return NewAndNode(expr, NewSpan(start, p.Location())), nil
	case '!':
		return NewNotNode(expr, NewSpan(start, p.Location())), nil
	case '#':
		return NewLexNode(expr, NewSpan(start, p.Location())), nil
	default:
		return expr, nil
	}
}

// GR: Labeled <- Suffix (LABEL Identifier)?
func (p *GrammarParser) ParseLabeled() (AstNode, error) {
	p.ParseSpacing()
	start := p.Location()
	expr, err := p.ParseSuffix()
	if err != nil {
		return nil, err
	}
	return Choice(p, []ParserFn[AstNode]{
		func(p Backtrackable) (AstNode, error) {
			if _, err := p.(*GrammarParser).ExpectRune('^'); err != nil {
				return nil, err
			}
			label, err := p.(*GrammarParser).parseIdentifier()
			if err != nil {
				return nil, err
			}
			return NewLabeledNode(label, expr, NewSpan(start, p.Location())), nil
		},
		func(p Backtrackable) (AstNode, error) { return expr, nil },
	})
}

// GR: Suffix <- Primary (QUESTION / STAR / PLUS)?
func (p *GrammarParser) ParseSuffix() (AstNode, error) {
	p.ParseSpacing()
	start := p.Location()
	primary, err := p.ParsePrimary()
	if err != nil {
		return nil, err
	}

	p.ParseSpacing()
	suffix, err := Choice(p, []ParserFn[rune]{
		p.ExpectRuneFn('?'),
		p.ExpectRuneFn('*'),
		p.ExpectRuneFn('+'),
		func(p Backtrackable) (rune, error) { return 0, nil },
	})
	if err != nil {
		return nil, err
	}

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
func (p *GrammarParser) ParsePrimary() (AstNode, error) {
	return Choice(p, []ParserFn[AstNode]{
		func(p Backtrackable) (AstNode, error) { return p.(*GrammarParser).ParseIdentifier() },
		func(p Backtrackable) (AstNode, error) { return p.(*GrammarParser).ParseParenExpression() },
		func(p Backtrackable) (AstNode, error) { return p.(*GrammarParser).ParseLiteral() },
		func(p Backtrackable) (AstNode, error) { return p.(*GrammarParser).ParseClass() },
		func(p Backtrackable) (AstNode, error) { return p.(*GrammarParser).ParseDot() },
	})
}

// GR: Identifier <- IdentStart IdentCont*
// GR: IdentStart <- [a-zA-Z_]
// GR: IdentCont  <- IdentStart / [0-9]
func (p *GrammarParser) ParseIdentifier() (AstNode, error) {
	p.ParseSpacing()
	start := p.Location()
	value, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	end := p.Location()

	if _, err := Not(p, func(p Backtrackable) (AstNode, error) {
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
	tail, err := ZeroOrMore(p, func(p Backtrackable) (rune, error) {
		return Choice(p, []ParserFn[rune]{
			p.(*GrammarParser).ExpectRangeFn('a', 'z'),
			p.(*GrammarParser).ExpectRangeFn('A', 'Z'),
			p.(*GrammarParser).ExpectRangeFn('0', '9'),
			p.(*GrammarParser).ExpectRuneFn('_'),
		})
	})
	if err != nil {
		return "", err
	}

	return string(append([]rune{head}, tail...)), nil
}

func (p *GrammarParser) ParseLeftArrow() error {
	p.ParseSpacing()
	if _, err := p.ExpectRune('<'); err != nil {
		return err
	}
	if _, err := p.ExpectRune('-'); err != nil {
		return err
	}
	return nil
}

func (p *GrammarParser) ParseParenExpression() (AstNode, error) {
	p.ParseSpacing()
	if _, err := p.ExpectRune('('); err != nil {
		return nil, err
	}

	expr, err := p.ParseExpression()
	if err != nil {
		return nil, err
	}

	p.ParseSpacing()
	if _, err := p.ExpectRune(')'); err != nil {
		return nil, err
	}

	return expr, nil
}

// GR: Class <- '[' (!']' Range)* ']'
func (p *GrammarParser) ParseClass() (AstNode, error) {
	p.ParseSpacing()
	start := p.Location()
	if _, err := p.ExpectRune('['); err != nil {
		return nil, err
	}
	ranges, err := ZeroOrMore(p, func(p Backtrackable) (AstNode, error) {
		if _, err := Not(p, p.(*GrammarParser).ExpectRuneFn(']')); err != nil {
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
	return NewClassNode(ranges, NewSpan(start, p.Location())), nil
}

// GR: Range <- Char '-' Char / Char
func (p *GrammarParser) ParseRange() (AstNode, error) {
	return Choice(p, []ParserFn[AstNode]{
		func(p Backtrackable) (AstNode, error) {
			start := p.Location()
			left, err := p.(*GrammarParser).parseChar()
			if err != nil {
				return nil, err
			}
			if _, err := p.(*GrammarParser).ExpectRune('-'); err != nil {
				return nil, err
			}
			right, err := p.(*GrammarParser).parseChar()
			if err != nil {
				return nil, err
			}
			return NewRangeNode(left, right, NewSpan(start, p.Location())), nil
		},
		func(p Backtrackable) (AstNode, error) {
			return p.(*GrammarParser).ParseChar()
		},
	})
}

func (p *GrammarParser) ParseDot() (AstNode, error) {
	p.ParseSpacing()
	start := p.Location()
	if _, err := p.ExpectRune('.'); err != nil {
		return nil, err
	}
	return NewAnyNode(NewSpan(start, p.Location())), nil
}

// GR: Literal <- ['] (!['] Char)* [']
// GR:          / ["] (!["] Char)* ["]
func (p *GrammarParser) ParseLiteral() (AstNode, error) {
	p.ParseSpacing()
	start := p.Location()
	value, err := p.parseLiteral()
	if err != nil {
		return nil, err
	}
	return NewLiteralNode(value, NewSpan(start, p.Location())), nil
}

func (p *GrammarParser) parseLiteral() (string, error) {
	value, err := Choice(p, []ParserFn[string]{
		func(p Backtrackable) (string, error) {
			if _, err := p.(*GrammarParser).ExpectRune('\''); err != nil {
				return "", err
			}
			s, err := ZeroOrMore(p, func(p Backtrackable) (rune, error) {
				if _, err := Not(p, p.(*GrammarParser).ExpectRuneFn('\'')); err != nil {
					return 0, err
				}
				return p.(*GrammarParser).parseChr()
			})
			if err != nil {
				return "", err
			}
			if _, err := p.(*GrammarParser).ExpectRune('\''); err != nil {
				return "", err
			}
			return string(s), nil
		},
		func(p Backtrackable) (string, error) {
			if _, err := p.(*GrammarParser).ExpectRune('"'); err != nil {
				return "", err
			}
			s, err := ZeroOrMore(p, func(p Backtrackable) (rune, error) {
				if _, err := Not(p, p.(*GrammarParser).ExpectRuneFn('"')); err != nil {
					return 0, err
				}
				return p.(*GrammarParser).parseChr()
			})
			if err != nil {
				return "", err
			}
			if _, err := p.(*GrammarParser).ExpectRune('"'); err != nil {
				return "", err
			}
			return string(s), nil
		},
	})
	if err != nil {
		return "", err
	}
	return value, nil
}

// GR: Char <- '\\' [nrt’"\[\]\\]
//
//	/ !'\\' .
func (p *GrammarParser) ParseChar() (AstNode, error) {
	start := p.Location()
	value, err := p.parseChr()
	if err != nil {
		return nil, err
	}
	return NewLiteralNode(string(value), NewSpan(start, p.Location())), nil
}

func (p *GrammarParser) parseChr() (rune, error) {
	return Choice(p, []ParserFn[rune]{
		func(p Backtrackable) (rune, error) { return p.(*GrammarParser).parseEscapedChar() },
		func(p Backtrackable) (rune, error) { return p.(*GrammarParser).parseChar() },
	})
}

// '\\' [nrt'"\[\]\\]
func (p *GrammarParser) parseEscapedChar() (rune, error) {
	if _, err := p.ExpectRune('\\'); err != nil {
		return 0, err
	}
	var choices []ParserFn[rune]
	for _, choice := range []struct {
		In, Out rune
	}{
		{In: '-', Out: '-'},
		{In: 'n', Out: '\n'},
		{In: 'r', Out: '\r'},
		{In: 't', Out: '\t'},
		{In: '\'', Out: '\''},
		{In: '"', Out: '"'},
		{In: '[', Out: '['},
		{In: ']', Out: ']'},
		{In: '\\', Out: '\\'},
	} {
		c := choice
		choices = append(choices, func(p Backtrackable) (rune, error) {
			if _, err := p.(*GrammarParser).ExpectRune(c.In); err != nil {
				return 0, err
			}
			return c.Out, nil
		})
	}
	return Choice(p, choices)
}

// !'\\' .
func (p *GrammarParser) parseChar() (rune, error) {
	if _, err := Not(p, p.ExpectRuneFn('\\')); err != nil {
		return 0, err
	}
	value, err := p.Any()
	if err != nil {
		return 0, err
	}
	return value, nil
}

// GR: ParseSpacing <- ' ' / '\t' / '\r' / '\n'
func (p *GrammarParser) ParseSpacing() {
	ZeroOrMore(p, func(p Backtrackable) (rune, error) {
		return Choice(p, []ParserFn[rune]{
			func(p Backtrackable) (rune, error) {
				return 0, p.(*GrammarParser).ParseComment()
			},
			func(p Backtrackable) (rune, error) {
				return p.(*GrammarParser).parseSpacingChar()
			},
		})
	})
}

// GR: ParseComment <- '//' (!'\n' .)* '\n'
func (p *GrammarParser) ParseComment() error {
	if _, err := p.ExpectRune('/'); err != nil {
		return err
	}
	if _, err := p.ExpectRune('/'); err != nil {
		return err
	}

	ZeroOrMore(p, func(p Backtrackable) (rune, error) {
		if _, err := Not(p, p.(*GrammarParser).ExpectRuneFn('\n')); err != nil {
			return 0, err
		}
		return p.(*GrammarParser).Any()
	})

	p.ExpectRune('\n')

	return nil
}
