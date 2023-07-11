package langlang

import "fmt"

// ParsingError is the error thrown when the parser can't finish successfuly
type ParsingError struct {
	Message    string
	Label      string
	Production string
	Span       Span
}

// String returns the human readable representation of a parsing error
func (e ParsingError) Error() string {
	message := e.Label
	if e.Message != "" {
		message = e.Message
	}
	return fmt.Sprintf("%s: %s @ %s", e.Production, message, e.Span)
}

// backtrackingError is an internal error type that is captured by the
// Choice operator
type backtrackingError struct {
	Message    string
	Production string
	Span       Span
}

// String returns the human readable representation of a parsing error
func (e backtrackingError) Error() string {
	return fmt.Sprintf("%s: %s @ %s", e.Production, e.Message, e.Span)
}

func isthrown(err error) bool {
	_, ok := err.(ParsingError)
	return ok
}
