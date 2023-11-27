package langlang

import (
	"fmt"
	"strings"
)

const eof = -1

// BaseParser keeps the state necessary to build parsing expressions
// on top of the basic parsing expressions available, like Choice,
// ZeroOrMore, OneOrMore, Options, etc.
type BaseParser struct {
	ffp    int
	cursor int
	line   int
	column int
	input  []rune

	inputFile   string
	grammarFile string

	lastErr    error
	lastErrFFP int
	predStkCnt int
	labelMsgs  map[string]string
	stacktrace []TracerSpan
	actionFns  map[string]func(*ValueNode) (Value, error)
}

// Location returns in which line/column/cursor the parser's input is currently in
func (p BaseParser) Location() Location {
	return Location{
		Line:   p.line,
		Column: p.column,
		Cursor: p.cursor,
		File:   p.inputFile,
	}
}

// SetFile allows users of the base parser to define the path of the
// input file.  That is used in error messages.
func (p *BaseParser) SetInputFile(file string) {
	p.inputFile = file
}

// SetGrammarFile allows users of the base parser to define the path
// of the grammar file.  That might be used in error messages.
func (p *BaseParser) SetGrammarFile(f string) {
	p.grammarFile = f
}

// SetInput associates an input to the parser struct.  The state of
// the parser is *partially* reset.  This doesn't reset the map
// between labels and error messages.
func (p *BaseParser) SetInput(input string) {
	p.ffp = 0
	p.cursor = 0
	p.line = 0
	p.column = 0
	p.input = []rune(input)

	p.lastErr = nil
	p.lastErrFFP = 0
	p.stacktrace = []TracerSpan{}
	p.predStkCnt = 0
}

// SetLabelMessages associates messages to labels, so `Throw` can pick it up and feed the error message it is
// generating with a user picked message
func (p *BaseParser) SetLabelMessages(m map[string]string) {
	p.labelMsgs = m
}

func (p *BaseParser) SetAction(name string, fn func(*ValueNode) (Value, error)) {
	if p.actionFns == nil {
		p.actionFns = map[string]func(*ValueNode) (Value, error){}
	}
	p.actionFns[name] = fn
}

func (p *BaseParser) RunAction(name string, node *ValueNode) (Value, error) {
	action, ok := p.actionFns[name]
	if !ok {
		return node, nil
	}
	return action(node)
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

	exp := fmt.Sprintf("`%c`", v)
	msg := fmt.Sprintf("Expected `%c` but got `%c`", v, c)
	err := p.NewError(exp, msg, NewSpan(start, p.Location()))
	return 0, err
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

	exp := fmt.Sprintf("`%c-%c`", l, r)
	msg := fmt.Sprintf("Expected `%c-%c` but got `%c`", l, r, c)
	err := p.NewError(exp, msg, NewSpan(start, p.Location()))
	return 0, err
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

		span := NewSpan(state.Location, p.Location())
		err = p.NewError(fmt.Sprintf("`%s`", literal), fmt.Sprintf("Missing `%s`", literal), span)
		return "", err
	}
	return literal, nil

}

// Any matches any rune under the input cursor, and will throw an error on EOF
func (p *BaseParser) Any() (rune, error) {
	pos := p.Location()
	c := p.Peek()
	if c == eof {
		return 0, p.NewError(".", "EOF", NewSpan(pos, p.Location()))
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
func (p *BaseParser) NewError(exp, msg string, span Span) error {
	n := p.PeekTraceSpan().Name
	e := &backtrackingError{
		Production: n,
		Expected:   exp,
		Message:    msg,
		// ErrSpan:    errSpan,
		Span:       span,
	}
	return e
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

func (p *BaseParser) PrintStackTrace() string {
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
		return p.NewError(label, label, span)
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
