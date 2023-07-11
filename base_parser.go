package parsing

import "fmt"

const eof = -1

type BaseParser struct {
	cursor int
	line   int
	column int
	input  []rune
}

// Location returns in which line/column/cursor the parser's input is currently in
func (p BaseParser) Location() Location {
	return Location{
		Line:   p.line,
		Column: p.column,
		Cursor: p.cursor,
	}
}

// Peek returns the character under the input cursor, or eof if the entire input has been consumed
func (p *BaseParser) Peek() rune {
	if p.cursor >= len(p.input) {
		return eof
	}
	return p.input[p.cursor]
}

// Backtrack resets the internal parser state to the Location l
func (p *BaseParser) Backtrack(l Location) {
	p.cursor = l.Cursor
	p.line = l.Line
	p.column = l.Column
}

func (p *BaseParser) ExpectRune(v rune) (rune, error) {
	c := p.Peek()
	if c == v {
		return p.Any()
	}
	return 0, p.NewError(fmt.Sprintf("Expected %c, got %c", v, c))
}

func (p *BaseParser) ExpectRuneFn(v rune) ParserFn[rune] {
	return func(p Parser) (rune, error) { return p.ExpectRune(v) }
}

func (p *BaseParser) ExpectRange(l, r rune) (rune, error) {
	c := p.Peek()
	if c >= l && c <= r {
		return p.Any()
	}
	return 0, p.NewError(fmt.Sprintf("Expected char between %c and %c, got %c", l, r, c))
}

func (p *BaseParser) ExpectRangeFn(l, r rune) ParserFn[rune] {
	return func(p Parser) (rune, error) { return p.ExpectRange(l, r) }
}

func (p *BaseParser) NewError(msg string) error {
	return backtrackingError{Message: msg}
}

// Any matches any rune under the input cursor, and will throw an error on EOF
func (p *BaseParser) Any() (rune, error) {
	c := p.Peek()
	if c == eof {
		return 0, p.NewError("EOF")
	}
	p.cursor++
	p.column++
	if c == '\n' {
		p.column = 0
		p.line++
	}
	return c, nil
}

// // throw returns an error that can't be caught by the backtrack system
// // and will error right away
// func (p *Parser) throw(message string, span Span) ParsingError {
// 	return ParsingError{Message: message, Span: span}
// }
