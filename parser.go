package parsing

type Parser interface {
	// Peek returns the rune within the input that is under the
	// parser cursor.  It does not change the cursor.
	Peek() rune

	// Any returns the current rune and advances the cursor.  It
	// returns the EOF error if the cursor is beyond the input
	// length.
	Any() (rune, error)

	// Backtrack resets the parser's cursor to `location`
	Backtrack(location Location)

	// Location returns the full location of the cursor within the
	// input.
	Location() Location

	// NewError creates a new error message
	NewError(msg string) error

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
}

// ParserFn is the signature of a parser function.  It unfortunately
// can't be a method because of Go's generics limitations, but a
// closure will fit in just right.  By being generic on its return,
// all matching functions can be generic over this same `T`, which
// allow composing recursive parsers sharing the same tooling despite
// their different return types
type ParserFn[T any] func(p Parser) (T, error)

// ZeroOrMore will call `fn` until it errors out, collecting and
// returning all the successful outputs.  Since we support any set of
// expressions within the closure `fn`, it will backtrack on error.
func ZeroOrMore[T any](p Parser, fn ParserFn[T]) ([]T, error) {
	var output []T
	for {
		pos := p.Location()
		item, err := fn(p)
		if err != nil {
			p.Backtrack(pos)
			if isthrown(err) {
				return nil, err
			}
			break
		}
		output = append(output, item)
	}
	return output, nil
}

// OneOrMore will match `fn` once and then pass fn to ZeroOrMore
func OneOrMore[T any](p Parser, fn ParserFn[T]) ([]T, error) {
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
func ChoiceRune(p Parser, runes []rune) (rune, error) {
	var fns []ParserFn[rune]
	for _, r := range runes {
		fns = append(fns, p.ExpectRuneFn(r))
	}
	return Choice(p, fns)
}

// Choice walks through fns and return the first to succeed.  It will
// backtrack the parser cursor before each attempt, and it will fail
// if no alternatives match.
func Choice[T any](p Parser, fns []ParserFn[T]) (T, error) {
	var zero T
	pos := p.Location()
	for _, fn := range fns {
		item, err := fn(p)
		if err == nil {
			return item, nil
		} else {
			p.Backtrack(pos)
			if isthrown(err) {
				return zero, err
			}
		}
	}
	return zero, p.NewError("Choice Error")
}

// Optional is a syntax sugar for an ordered choice in which the
// second option is nil
func Optional[T any](p Parser, fn ParserFn[T]) (T, error) {
	return Choice(p, []ParserFn[T]{
		fn,
		func(p Parser) (T, error) {
			var zero T
			return zero, nil
		},
	})
}

// And returns an error if fn fails, or fails if fn doesn't succeed.
// This is the same as calling Not twice but here's a shortuct
func And[T any](p Parser, fn ParserFn[T]) (T, error) {
	var zero T
	pos := p.Location()
	_, err := fn(p)

	// unconditionally backtrack as the predicate never consumes any input
	p.Backtrack(pos)

	if err != nil {
		return zero, p.NewError("And Error")
	}
	return zero, nil
}

// Not returns an error if fn succeeds, or succeed if fn doesn't succeed
func Not[T any](p Parser, fn ParserFn[T]) (T, error) {
	var zero T
	pos := p.Location()
	_, err := fn(p)

	// unconditionally backtrack as the predicate never consumes any input
	p.Backtrack(pos)

	if err == nil {
		return zero, p.NewError("Not Error")
	}
	return zero, nil
}
