package langlang

import (
	"fmt"
	"strconv"
	"strings"
)

type AsmFormatToken int

const (
	AsmFormatToken_None AsmFormatToken = iota
	AsmFormatToken_Comment
	AsmFormatToken_Label
	AsmFormatToken_Literal
	AsmFormatToken_Operator
	AsmFormatToken_Operand
)

// asmPrinterTheme is a map from the tokens available for pretty
// printing the ASM grammar to an ASCII color.  These colors are
// supposed to fair well on both dark and light terminal settings
var asmPrinterTheme = map[AsmFormatToken]string{
	AsmFormatToken_None:     "\033[0m",          // reset
	AsmFormatToken_Comment:  "\033[1;38;5;245m", // gray
	AsmFormatToken_Label:    "\033[1;31m",       // red
	AsmFormatToken_Literal:  "\033[1;34m",       // blue
	AsmFormatToken_Operator: "\033[1;38;5;99m",  // purple
	AsmFormatToken_Operand:  "\033[1;38;5;127m", // pink
}

type recoveryEntry struct {
	label      ILabel
	precedence int
}

type Program struct {
	// identifiers is a map with keys as the position of the first
	// instruction of each production in the source code, and
	// values as the index in the strings table where the name of
	// the production can be found.
	identifiers map[int]int

	// recovery is a map from label IDs to tuples with two things:
	// address of the recovery expression and its precedence level
	recovery map[int]recoveryEntry

	// strings is a table with strings that refer to either error
	// labels or production identifiers.  IDs are assigned in the
	// order they are requested.
	strings []string

	stringsMap map[string]int

	// code is an array of instructions that get executed by the
	// virtual machine
	code []Instruction
}

func (p Program) StringID(n string) int {
	return p.stringsMap[n]
}

func (p Program) PrettyString() string {
	return p.prettyString(func(input string, _ AsmFormatToken) string {
		return input
	})
}

func (p Program) HighlightPrettyString() string {
	return p.prettyString(func(input string, token AsmFormatToken) string {
		return asmPrinterTheme[token] + input + asmPrinterTheme[AsmFormatToken_None]
	})
}

func (p Program) prettyString(format FormatFunc[AsmFormatToken]) string {
	var (
		s                strings.Builder
		previousWasLabel bool
		index            = 0
	)

	// fmt.Printf("strings: %#v\n", p.strings)
	// fmt.Printf("identifiers: %#v\n", p.identifiers)

	writeComment := func(i string) {
		s.WriteString(format(i, AsmFormatToken_Comment))
	}

	writeName := func(name string) {
		if !previousWasLabel {
			writeComment(fmt.Sprintf("%06d  ", index))
			s.WriteString("        ")
		}

		s.WriteString(format(name, AsmFormatToken_Operand))

		previousWasLabel = false
	}

	writeLabel := func(id int) {
		s.WriteString(format(fmt.Sprintf(" l%d", id), AsmFormatToken_Label))
	}

	writeInt := func(id int) {
		s.WriteString(format(fmt.Sprintf(" %d", id), AsmFormatToken_Literal))
	}

	writeString := func(id int) {
		s.WriteString(format(fmt.Sprintf(" '%v'", p.strings[id]), AsmFormatToken_Literal))
	}

	writeRune := func(r rune) {
		lit := fmt.Sprintf(" '%s'", escapeLiteral(string(r)))
		s.WriteString(format(lit, AsmFormatToken_Literal))
	}

	for cursor, instruction := range p.code {
		if idx, ok := p.identifiers[cursor]; ok {
			writeComment(fmt.Sprintf("\n;; %s\n", p.strings[idx]))
		}

		switch ii := instruction.(type) {
		case ILabel:
			if previousWasLabel {
				s.WriteString("\n")
			}
			writeComment(fmt.Sprintf("%06d  ", index))
			lb := fmt.Sprintf("l%d:%*s", ii.ID, 6-len(strconv.Itoa(ii.ID)), " ")
			s.WriteString(format(lb, AsmFormatToken_Label))
			previousWasLabel = true

		case ICall:
			writeName(instruction.Name())
			writeLabel(ii.Label.ID)
			writeInt(ii.Precedence)
			s.WriteString("\n")

		case IThrow:
			writeName(instruction.Name())
			writeString(ii.ErrorLabel)
			s.WriteString("\n")

		case IChoice:
			writeName(instruction.Name())
			writeLabel(ii.Label.ID)
			s.WriteString("\n")

		case IChoicePred:
			writeName(instruction.Name())
			writeLabel(ii.Label.ID)
			s.WriteString("\n")

		case ICommit:
			writeName(instruction.Name())
			writeLabel(ii.Label.ID)
			s.WriteString("\n")

		case IBackCommit:
			writeName(instruction.Name())
			writeLabel(ii.Label.ID)
			s.WriteString("\n")

		case IPartialCommit:
			writeName(instruction.Name())
			writeLabel(ii.Label.ID)
			s.WriteString("\n")

		case IChar:
			writeName(instruction.Name())
			writeRune(ii.Char)
			s.WriteString("\n")

		case IRange:
			writeName(instruction.Name())
			writeRune(ii.Lo)
			s.WriteString("")
			writeRune(ii.Hi)
			s.WriteString("\n")

		case ISet:
			writeName(instruction.Name())
			s.WriteString(" ")
			s.WriteString(format(escapeLiteral(ii.cs.String()), AsmFormatToken_Literal))
			s.WriteString("\n")

		case ISpan:
			writeName(instruction.Name())
			s.WriteString(" ")
			s.WriteString(format(escapeLiteral(ii.cs.String()), AsmFormatToken_Literal))
			s.WriteString("\n")

		case ICapBegin:
			writeName(instruction.Name())
			writeString(ii.ID)
			s.WriteString("\n")

		default:
			writeName(instruction.Name())
			s.WriteString("\n")
		}
		index += instruction.SizeInBytes()
	}
	return s.String()
}
