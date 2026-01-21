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

// String returns a human-readable representation of the hint.
func (eh ErrHint) String() string {
	switch eh.Type {
	case ErrHintType_EOF:
		return "end of input"
	case ErrHintType_Char:
		return formatChar(eh.Char)
	case ErrHintType_Range:
		return fmt.Sprintf("'%c'-'%c'", eh.Range[0], eh.Range[1])
	default:
		return "?"
	}
}

// formatChar returns a human-readable representation of a character,
// handling control characters and special cases.
func formatChar(c rune) string {
	switch {
	case c == 0:
		return "end of input"
	case c == '\n':
		return "newline"
	case c == '\r':
		return "carriage return"
	case c == '\t':
		return "tab"
	case c == ' ':
		return "space"
	case c < 32 || c == 127:
		// Control characters - show as escape sequence
		return fmt.Sprintf("'\\x%02x'", c)
	default:
		return fmt.Sprintf("'%c'", c)
	}
}

// FormatExpectedMessage generates a human readable message from expected
// hints and the actual input at the error position.
func FormatExpectedMessage(hints []ErrHint, input []byte, pos int) string {
	if len(hints) == 0 {
		return ""
	}
	parts := make([]string, len(hints))
	for i, h := range hints {
		parts[i] = h.String()
	}
	got := "end of input"
	if pos < len(input) {
		got = fmt.Sprintf("'%c'", input[pos])
	}
	if len(parts) == 1 {
		return fmt.Sprintf("Expected %s but got %s", parts[0], got)
	}
	return fmt.Sprintf("Expected %s but got %s",
		joinWithOr(parts), got)
}

// joinWithOr joins strings with commas and "or" for the last item.
func joinWithOr(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " or " + items[1]
	}
	// For 3+: "a, b, c, or d"
	result := ""
	for i, item := range items {
		if i == len(items)-1 {
			result += "or " + item
		} else {
			result += item + ", "
		}
	}
	return result
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
