package langlang

import "fmt"

// ParsingError is the error thrown when the parser can't finish successfuly
type ParsingError struct {
	Message    string
	Label      string
	Start, End int
}

// String returns the human readable representation of a parsing error
func (e ParsingError) Error() string {
	message := e.Label
	if e.Message != "" {
		message = e.Message
	}
	// FIXME: Find a way to show the right line/col
	span := fmt.Sprintf("%d", e.Start+1)
	if e.Start != e.End {
		span += fmt.Sprintf("..%d", e.End+1)
	}
	return fmt.Sprintf("%s @ %s", message, span)
}

func isthrown(err error) bool {
	_, ok := err.(ParsingError)
	return ok
}
