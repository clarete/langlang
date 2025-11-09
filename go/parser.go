package langlang

import (
	"fmt"
	"strings"
)

// Parser keeps the state necessary to build parsing expressions on
// top of the basic parsing expressions available, like Choice,
// ZeroOrMore, OneOrMore, Options, etc.
type Parser struct {
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
	actionFns  map[string]func(*Node) (Value, error)

	captureSpaces bool

	mtable map[string]mentry
}

type mentry struct {
	val Value
	err error
	end int
}

type Backtrackable interface {
	// SetInput associates input to a concrete parser struct
	SetInput(input string)

	// Peek returns the rune within the input that is under the
	// parser cursor.  It does not change the cursor.
	Peek() rune

	// Any returns the current rune and advances the cursor.  It
	// returns the EOF error if the cursor is beyond the input
	// length.
	Any() (rune, error)

	// Backtrack resets the parser's cursor
	Backtrack(cursor int)

	Cursor() int

	// NewError creates a new error message
	NewError(expected, msg string, rg Range) error

	// SetLabelMessages associates a message to a label, so when a
	// given label is thrown by the `Throw()` module
	SetLabelMessages(map[string]string)

	// SetAction registers `action` as a function to be called
	SetAction(string, func(*Node) (Value, error))
	RunAction(string, *Node) (Value, error)

	// Throw creates an error that can't be handled by backtracking
	Throw(label string, rg Range) error

	// WithinPredicate returns true if the parser is currently
	// executing a predicate expression.  This is used to prevent
	// generating exceptions from within the ~Throw~ operator so
	// it generates backtracking errors instead
	WithinPredicate() bool

	// EnterPredicate is called by the `Not` operator to inform
	// the parser that a predicate evaluation has started.  This
	// function is reentrant.
	EnterPredicate()

	// LeavePredicate is also called by the `Not` operator to
	// inform the parser that a predicate evaluation has ended.
	// This function is reentrant.
	LeavePredicate()

	// ExpectRune returns `r` if it's the same rune that's under
	// the cursor, or errors otherwise.
	ExpectRune(r rune) (rune, error)

	// ExpectRange returns the rune under the cursor if it's
	// between runes `l` and `r`, or errors otherwise.
	ExpectRange(l, r rune) (rune, error)

	// ExpectRangeFn returns a function wrapping a `ExpectRange` call.
	ExpectRangeFn(l, r rune) ParserFn[rune]

	// ExpectRuneFn returns a function wrapping an `ExpectRune` call.
	ExpectRuneFn(r rune) ParserFn[rune]

	// ExpectLiteral returns `l` if it's the same rune that's under
	// the cursor, or errors otherwise.
	ExpectLiteral(l string) (string, error)
}

// SetFile allows users of the base parser to define the path of the
// input file.  That is used in error messages.
func (p *Parser) SetInputFile(file string) {
	p.inputFile = file
}

// SetGrammarFile allows users of the base parser to define the path
// of the grammar file.  That might be used in error messages.
func (p *Parser) SetGrammarFile(f string) {
	p.grammarFile = f
}

// SetInput associates an input to the parser struct.  The state of
// the parser is *partially* reset.  This doesn't reset the map
// between labels and error messages.
func (p *Parser) SetInput(input string) {
	p.ffp = 0
	p.cursor = 0
	p.input = []rune(input)

	p.lastErr = nil
	p.lastErrFFP = 0
	p.mtable = map[string]mentry{}
}

// SetLabelMessages associates messages to labels, so `Throw` can pick
// it up and feed the error message it is generating with a user
// picked message
func (p *Parser) SetLabelMessages(m map[string]string) {
	p.labelMsgs = m
}

func (p *Parser) SetCaptureSpaces(v bool) {
	p.captureSpaces = v
}

func (p *Parser) SetAction(name string, fn func(*Node) (Value, error)) {
	if p.actionFns == nil {
		p.actionFns = map[string]func(*Node) (Value, error){}
	}
	p.actionFns[name] = fn
}

func (p *Parser) RunAction(name string, node *Node) (Value, error) {
	action, ok := p.actionFns[name]
	if !ok {
		return node, nil
	}
	return action(node)
}

func (p *Parser) Cursor() int {
	return p.cursor
}

// Peek returns the character under the input cursor, or eof if the entire input has been consumed
func (p *Parser) Peek() rune {
	if p.cursor >= len(p.input) {
		return eof
	}
	return p.input[p.cursor]
}

// Backtrack resets the internal parser state to the Location l
func (p *Parser) Backtrack(cursor int) {
	p.cursor = cursor
}

func (p *Parser) ExpectRune(v rune) (rune, error) {
	start := p.Cursor()
	c := p.Peek()
	if c == v {
		return p.Any()
	}

	exp := "`" + string(v) + "`"
	msg := "Expected " + exp + " but got `" + string(c) + "`"
	err := p.NewError(exp, msg, NewRange(start, p.Cursor()))
	return 0, err
}

func (p *Parser) ExpectRuneFn(v rune) ParserFn[rune] {
	return func(p Backtrackable) (rune, error) { return p.ExpectRune(v) }
}

func (p *Parser) ExpectRange(l, r rune) (rune, error) {
	start := p.Cursor()
	c := p.Peek()
	if c >= l && c <= r {
		return p.Any()
	}

	exp := "`" + string(l) + "-" + string(r) + "`"
	msg := "Expected " + exp + " but got `" + string(c) + "`"
	err := p.NewError(exp, msg, NewRange(start, p.Cursor()))
	return 0, err
}

func (p *Parser) ExpectRangeFn(l, r rune) ParserFn[rune] {
	return func(p Backtrackable) (rune, error) { return p.ExpectRange(l, r) }
}

func (p *Parser) ExpectLiteral(literal string) (string, error) {
	start := p.Cursor()

	for _, v := range literal {
		c, err := p.Any()
		if err != nil {
			return "", err
		}
		if c == v {
			continue
		}

		exp := "`" + literal + "`"
		msg := "Missing " + exp
		return "", p.NewError(exp, msg, NewRange(start, p.Cursor()))
	}
	return literal, nil

}

// Any matches any rune under the input cursor, and will throw an error on EOF
func (p *Parser) Any() (rune, error) {
	pos := p.Cursor()
	c := p.Peek()
	if c == eof {
		return 0, p.NewError(".", "EOF", NewRange(pos, p.Cursor()))
	}
	p.cursor++
	p.column++
	if p.cursor > p.ffp {
		p.ffp = p.cursor
	}
	return c, nil
}

// NewError creates a type of error that is handled and discarded when
// the parser backtracks the input position
func (p *Parser) NewError(exp, msg string, rg Range) error {
	return &backtrackingError{
		Expected: exp,
		Message:  msg,
		Range:    rg,
	}
}

func (p *Parser) parseRange(left, right rune) (Value, error) {
	start := p.Cursor()
	_, err := p.ExpectRange(left, right)
	if err != nil {
		var zero Value
		return zero, err
	}
	return NewString(NewRange(start, p.Cursor())), nil
}

func (p *Parser) parseAny() (Value, error) {
	start := p.Cursor()
	_, err := p.Any()
	if err != nil {
		var zero Value
		return zero, err
	}
	return NewString(NewRange(start, p.Cursor())), nil
}

func (p *Parser) parseLiteral(literal string) (Value, error) {
	start := p.Cursor()
	_, err := p.ExpectLiteral(literal)
	if err != nil {
		var zero Value
		return zero, err
	}
	return NewString(NewRange(start, p.Cursor())), nil
}

var spacingRunes = map[rune]struct{}{
	' ':  {},
	'\t': {},
	'\r': {},
	'\n': {},
}

func (p *Parser) parseSpacingChar() (rune, error) {
	r := p.Peek()
	start := p.Cursor()

	if _, ok := spacingRunes[r]; ok {
		return p.Any()
	}

	exp := "` `, `\t`, `\n`, `\r`"
	msg := "Expected " + exp + " but got `" + string(r) + "`"
	return ' ', p.NewError(msg, msg, NewRange(start, p.Cursor()))
}

func (p *Parser) parseSpacing() (Value, error) {
	start := p.Cursor()
	_, err := ZeroOrMore(p, func(p Backtrackable) (rune, error) {
		return p.(*Parser).parseSpacingChar()
	})
	if err != nil {
		return nil, err
	}
	if !p.captureSpaces {
		return nil, nil
	}
	end := p.Cursor()
	if end-start == 0 {
		return nil, nil
	}
	s := NewString(NewRange(start, p.Cursor()))
	return NewNode("Spacing", s, NewRange(start, p.Cursor())), nil
}

// Throw returns an error that can't be caught by the backtrack system
// and will error right away
func (p *Parser) Throw(label string, rg Range) error {
	if p.WithinPredicate() {
		return p.NewError(label, label, rg)
	}
	message := ""
	if m, ok := p.labelMsgs[label]; ok {
		message = m
	}
	e := ParsingError{
		Label:   label,
		Message: message,
		Range:   rg,
	}
	p.lastErr = e
	p.lastErrFFP = p.ffp
	return e
}

func (p *Parser) WithinPredicate() bool { return p.predStkCnt > 0 }
func (p *Parser) EnterPredicate()       { p.predStkCnt++ }
func (p *Parser) LeavePredicate()       { p.predStkCnt-- }

func wrapSeq(items []Value, rg Range) Value {
	switch len(items) {
	case 0:
		return nil
	case 1:
		return items[0]
	default:
		return NewSequence(items, rg)
	}
}

// ParserFn is the signature of a parser function.  It unfortunately
// can't be a method because of Go's generics limitations, but a
// closure will fit in just right.  By being generic on its return,
// all matching functions can be generic over this same `T`, which
// allow composing recursive parsers sharing the same tooling despite
// their different return types
type ParserFn[T any] func(p Backtrackable) (T, error)

// ZeroOrMore will call `fn` until it errors out, collecting and
// returning all the successful outputs.  Since we support any set of
// expressions within the closure `fn`, it will backtrack on error.
func ZeroOrMore[T any](p Backtrackable, fn ParserFn[T]) ([]T, error) {
	var output []T
	for {
		state := p.Cursor()
		item, err := fn(p)
		if err != nil {
			p.Backtrack(state)
			if isthrown(err) && !p.WithinPredicate() {
				return nil, err
			}
			break
		}
		output = append(output, item)
	}
	return output, nil
}

// OneOrMore will match `fn` once and then pass fn to ZeroOrMore
func OneOrMore[T any](p Backtrackable, fn ParserFn[T]) ([]T, error) {
	var output []T
	head, err := fn(p)
	if err != nil {
		return nil, err
	}
	output = append(output, head)
	tail, err := ZeroOrMore(p, fn)
	if err != nil {
		return nil, err
	}
	output = append(output, tail...)
	return output, nil
}

// ChoiceRune is a specialization of `Choice` that's less verbose for
// picking from a slice of runes
func ChoiceRune(p Backtrackable, runes map[rune]struct{}) (rune, error) {
	start := p.Cursor()
	r := p.Peek()
	if _, ok := runes[r]; ok {
		return p.Any()
	}

	expected := make([]string, len(runes))
	for k := range runes {
		expected = append(expected, string(k))
	}
	exp := strings.Join(expected, ", ")
	msg := fmt.Sprintf("Expected %s but got `%c`", exp, r)
	return ' ', p.NewError(exp, msg, NewRange(start, p.Cursor()))
}

// Choice walks through fns and return the first to succeed.  It will
// backtrack the parser cursor before each attempt, and it will fail
// if no alternatives match.
func Choice[T any](p Backtrackable, fns []ParserFn[T]) (T, error) {
	var (
		zero        T
		expected    []string
		expectedMap = map[string]struct{}{}
		start       = p.Cursor()
	)
	for _, fn := range fns {
		item, err := fn(p)
		if err == nil {
			return item, nil
		} else {
			p.Backtrack(start)
			if isthrown(err) && !p.WithinPredicate() {
				return zero, err
			}
			if berr, ok := err.(*backtrackingError); ok {
				if _, ok := expectedMap[berr.Expected]; !ok {
					expectedMap[berr.Expected] = struct{}{}
					expected = append(expected, berr.Expected)
				}
			}
		}
	}
	exp := strings.Join(expected, ", ")
	msg := "Expected " + exp + " but got `" + string(p.Peek()) + "`"
	return zero, p.NewError(exp, msg, NewRange(start, p.Cursor()))
}

// Optional is a syntax sugar for an ordered choice in which the
// second option is nil
func Optional[T any](p Backtrackable, fn ParserFn[T]) (T, error) {
	return Choice(p, []ParserFn[T]{
		fn,
		func(p Backtrackable) (T, error) {
			var zero T
			return zero, nil
		},
	})
}

// And returns an error if fn fails, or fails if fn doesn't succeed.
// This is the same as calling Not twice but here's a shortuct
func And[T any](p Backtrackable, fn ParserFn[T]) (T, error) {
	var zero T
	p.EnterPredicate()
	start := p.Cursor()
	_, err := fn(p)

	// unconditionally backtrack as the predicate never consumes any input
	p.Backtrack(start)
	p.LeavePredicate()

	if err != nil {
		return zero, p.NewError("&", "And Error", NewRange(start, p.Cursor()))
	}
	return zero, nil
}

// Not returns an error if fn succeeds, or succeed if fn doesn't succeed
func Not[T any](p Backtrackable, fn ParserFn[T]) (T, error) {
	var zero T
	p.EnterPredicate()
	start := p.Cursor()
	_, err := fn(p)

	// unconditionally backtrack as the predicate never consumes any input
	p.Backtrack(start)
	p.LeavePredicate()

	if err == nil {
		return zero, p.NewError("!", "Not Error", NewRange(start, p.Cursor()))
	}
	return zero, nil
}
