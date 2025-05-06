package langlang

import (
	"fmt"
	"strconv"
	"strings"
)

type recoveryEntry struct {
	expression int
	precedence int
}

type Program struct {
	// identifiers is a map with keys as the position of the first
	// instruction of each production in the source code, and
	// values as the index in the strings table where the name of
	// the production can be found.
	identifiers map[int]int

	// labels is a map with IDs of labels as keys and the ID of
	// the messages associated with the labels as values
	labels map[int]int

	// recovery is a map from label IDs to tuples with two things:
	// address of the recovery expression and its precedence level
	recovery map[int]recoveryEntry

	// strings is a table with strings that refer to either error
	// labels or production identifiers.  IDs are assigned in the
	// order they are requested.
	strings []string

	// code is an array of instructions that get executed by the
	// virtual machine
	code []Instruction
}

func (p Program) PrettyPrint() string {
	var (
		s strings.Builder

		previousWasLabel bool
	)

	// fmt.Printf("strings: %#v\n", p.strings)
	// fmt.Printf("identifiers: %#v\n", p.identifiers)

	writeName := func(name string) {
		if !previousWasLabel {
			s.WriteString("        ")
		}

		s.WriteString(name)

		previousWasLabel = false
	}

	writeLabel := func(id int) {
		s.WriteString(fmt.Sprintf(" l%d", id))
	}

	for cursor, instruction := range p.code {
		if idx, ok := p.identifiers[cursor]; ok {
			s.WriteString("\n;; ")
			s.WriteString(p.strings[idx])
			s.WriteString("\n")
		}

		switch ii := instruction.(type) {
		case ILabel:
			s.WriteString(fmt.Sprintf("l%d:%*s", ii.ID, 6-len(strconv.Itoa(ii.ID)), " "))
			previousWasLabel = true

		case ICall:
			writeName(instruction.Name())
			writeLabel(ii.Label.ID)
			//s.WriteString(fmt.Sprintf(" %s @ %d", p.strings[p.identifiers[ii.Address]], ii.Address))
			s.WriteString("\n")

		case IChoice:
			writeName(instruction.Name())
			writeLabel(ii.Label.ID)
			s.WriteString("\n")

		case IChoiceP:
			writeName(instruction.Name())
			writeLabel(ii.Label.ID)
			s.WriteString("\n")

		case ICommit:
			writeName(instruction.Name())
			writeLabel(ii.Label.ID)
			s.WriteString("\n")

		case IPartialCommit:
			writeName(instruction.Name())
			writeLabel(ii.Label.ID)
			s.WriteString("\n")

		case IString:
			writeName(instruction.Name())
			s.WriteString(fmt.Sprintf(" '%v'", p.strings[ii.ID]))
			s.WriteString("\n")

		default:
			writeName(instruction.Name())
			s.WriteString("\n")
		}
	}

	return s.String()
}
