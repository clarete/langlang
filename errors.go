package parsing

import "fmt"

// ParsingError is the error thrown when the parser can't finish successfuly
type ParsingError struct {
	Message string
	Span    Span
}

// String returns the human readable representation of a parsing error
func (e ParsingError) Error() string {
	return fmt.Sprintf("%s @ %s", e.Message, e.Span)
}

// backtrackingError is an internal error type that is captured by the
// Choice operator
type backtrackingError struct {
	Message string
	Span    Span
}

// String returns the human readable representation of a parsing error
func (e backtrackingError) Error() string {
	return fmt.Sprintf("%s @ %s", e.Message, e.Span)
}

func isthrown(err error) bool {
	_, ok := err.(ParsingError)
	return ok
}
