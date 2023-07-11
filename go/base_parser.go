package langlang

import (
	"fmt"
	"strings"
)

const eof = -1

type BaseParser struct {
	ffp    int
	cursor int
	line   int
	column int
	input  []rune

	lastErr    error
	lastErrFFP int
	stacktrace []TracerSpan
	labelMsgs  map[string]string
	predStkCnt int
}

// Location returns in which line/column/cursor the parser's input is currently in
func (p BaseParser) Location() Location {
	return Location{
		Line:   p.line,
		Column: p.column,
		Cursor: p.cursor,
	}
}

// SetInput associates an input to the parser struct.  This should only be called once, obviously before parsing.
func (p *BaseParser) SetInput(input []rune) {
	p.input = input
}

// SetLabelMessages associates messages to labels, so `Throw` can pick it up and feed the error message it is
// generating with a user picked message
func (p *BaseParser) SetLabelMessages(m map[string]string) {
	p.labelMsgs = m
}

// Peek returns the character under the input cursor, or eof if the entire input has been consumed
func (p *BaseParser) Peek() rune {
	if p.cursor >= len(p.input) {
		return eof
	}
	return p.input[p.cursor]
}

// Backtrack resets the internal parser state to the Location l
func (p *BaseParser) Backtrack(state ParserState) {
	p.cursor = state.Location.Cursor
	p.line = state.Location.Line
	p.column = state.Location.Column
	p.stacktrace = state.StackTrace
}

func (p BaseParser) State() ParserState {
	return ParserState{
		Location:   p.Location(),
		StackTrace: p.stacktrace,
	}
}

func (p *BaseParser) ExpectRune(v rune) (rune, error) {
	start := p.Location()
	c := p.Peek()
	if c == v {
		return p.Any()
	}

	msg := fmt.Sprintf("Expected char `%c', got `%c'", v, c)
	return 0, p.NewError(msg, NewSpan(start, p.Location()))
}

func (p *BaseParser) ExpectRuneFn(v rune) ParserFn[rune] {
	return func(p Parser) (rune, error) { return p.ExpectRune(v) }
}

func (p *BaseParser) ExpectRange(l, r rune) (rune, error) {
	start := p.Location()
	c := p.Peek()
	if c >= l && c <= r {
		return p.Any()
	}

	msg := fmt.Sprintf("Expected char between `%c' and `%c', got `%c'", l, r, c)
	return 0, p.NewError(msg, NewSpan(start, p.Location()))
}

func (p *BaseParser) ExpectRangeFn(l, r rune) ParserFn[rune] {
	return func(p Parser) (rune, error) { return p.ExpectRange(l, r) }
}

func (p *BaseParser) ExpectLiteral(literal string) (string, error) {
	state := p.State()

	for _, v := range literal {
		c, err := p.Any()
		if err != nil {
			p.Backtrack(state)
			return "", err
		}
		if c == v {
			continue
		}
		return "", p.NewError(fmt.Sprintf("Missing `%s`", literal),
			NewSpan(state.Location, p.Location()))
	}
	return literal, nil

}

// Any matches any rune under the input cursor, and will throw an error on EOF
func (p *BaseParser) Any() (rune, error) {
	pos := p.Location()
	c := p.Peek()
	if c == eof {
		return 0, p.NewError("EOF", NewSpan(pos, p.Location()))
	}
	p.cursor++
	p.column++
	if c == '\n' {
		p.column = 0
		p.line++
	}
	if p.cursor > p.ffp {
		p.ffp = p.cursor
	}
	return c, nil
}

// NewError creates a type of error that is handled and discarded when
// the parser backtracks the input position
func (p *BaseParser) NewError(msg string, span Span) error {
	if p.lastErr == nil || p.ffp > p.lastErrFFP {
		n := p.PeekTraceSpan().Name
		e := backtrackingError{Production: n, Message: msg, Span: span}
		p.lastErr = e
		p.lastErrFFP = p.ffp
		return e
	}
	return p.lastErr
}

func (p *BaseParser) PushTraceSpan(ts TracerSpan) {
	p.stacktrace = append(p.stacktrace, ts)
}

func (p *BaseParser) PeekTraceSpan() *TracerSpan {
	idx := len(p.stacktrace) - 1
	if idx < 0 {
		return nil
	}
	return &p.stacktrace[idx]
}

func (p *BaseParser) PopTraceSpan() TracerSpan {
	idx := len(p.stacktrace) - 1
	top := p.stacktrace[idx]
	p.stacktrace = p.stacktrace[:idx]
	return top
}

func (p *BaseParser) StackTrace() []TracerSpan {
	return p.stacktrace
}

func (p *BaseParser) printStackTrace() string {
	var (
		s     strings.Builder
		stack = p.StackTrace()
		ln    = len(stack) - 1
	)
	for i, span := range stack {
		s.WriteString(span.String())

		if i < ln {
			s.WriteString(" > ")
		}
	}
	return s.String()
}

// Throw returns an error that can't be caught by the backtrack system
// and will error right away
func (p *BaseParser) Throw(label string, span Span) error {
	if p.WithinPredicate() {
		return p.NewError(label, span)
	}
	production := p.PeekTraceSpan().Name
	message := ""
	if m, ok := p.labelMsgs[label]; ok {
		message = m
	}
	e := ParsingError{
		Production: production,
		Label:      label,
		Message:    message,
		Span:       span,
	}
	p.lastErr = e
	p.lastErrFFP = p.ffp
	return e
}

func (p *BaseParser) WithinPredicate() bool { return p.predStkCnt > 0 }
func (p *BaseParser) EnterPredicate()       { p.predStkCnt++ }
func (p *BaseParser) LeavePredicate()       { p.predStkCnt-- }
