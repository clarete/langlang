// Package ascii provides terminal ANSI color codes semantic names for
// colors so they can be grouped in themes.
package ascii

import "fmt"

const (
	Reset  = "\033[0m"
	Red    = "\033[1;31m"
	Yellow = "\033[1;33m"
	Green  = "\033[1;32m"
	Blue   = "\033[1;34m"
	Cyan   = "\033[1;36m"
	Gray   = "\033[90m" // Bright black, actually

	Magenta = "\033[1;35m"
	White   = "\033[1;37m"
	Bold    = "\033[1m"

	// 256-color palette
	Orange  = "\033[38;5;208m"
	Gray245 = "\033[1;38;5;245m" // Medium gray
	Purple  = "\033[1;38;5;99m"
	Pink    = "\033[1;38;5;127m"
)

// Theme defines semantic color mappings
type Theme struct {
	// Diagnostic levels
	Error   string
	Warning string
	Info    string
	Hint    string

	// UI elements
	Muted   string // secondary/dimmed text
	Accent  string // highlighted/emphasized text
	Success string

	// Syntax highlighting (AST/ASM printers, etc.)
	Operator string
	Operand  string
	Literal  string
	Span     string
	Comment  string
	Label    string
}

// DefaultTheme provides a sensible default color mapping.
var DefaultTheme = Theme{
	Error:   Red,
	Warning: Yellow,
	Info:    Cyan,
	Hint:    Gray,

	Muted:   Gray,
	Accent:  Cyan,
	Success: Green,

	Operator: Purple,
	Operand:  Pink,
	Literal:  Green,
	Span:     Orange,
	Comment:  Gray245,
	Label:    Red,
}

func Color(color, format string, args ...any) string {
	return fmt.Sprintf(color+format+Reset, args...)
}
