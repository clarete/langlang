package langlang

import "fmt"

// ParsingError is the error thrown when the parser can't finish successfuly
type ParsingError struct {
	Message    string
	Label      string
	Start, End int
	Expected   []ErrHint
}

type ErrHintType uint8

const (
	ErrHintType_Unknown ErrHintType = iota
	ErrHintType_EOF
	ErrHintType_Char
	ErrHintType_Range
)

type ErrHint struct {
	Type  ErrHintType
	Char  rune
	Range [2]rune
}

func (eh *ErrHint) eq(oh *ErrHint) bool {
	if eh.Type != oh.Type {
		return false
	}
	switch eh.Type {
	case ErrHintType_Char:
		return eh.Char == oh.Char
	case ErrHintType_Range:
		return eh.Range[0] == oh.Range[0] && eh.Range[1] == oh.Range[1]
	case ErrHintType_EOF:
		return true
	}
	return false
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
