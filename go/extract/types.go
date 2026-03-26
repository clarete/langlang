package extract

import (
	"reflect"
	"strings"
)

// FieldKind classifies a struct field's extraction strategy.
type FieldKind int

const (
	FieldText      FieldKind = iota // terminal rule -> string
	FieldNamedRule                  // rule reference -> struct with ll: tags
	FieldOptional                   // grammar ? or pointer choice branch
	FieldSlice                      // grammar * or + -> slice
	FieldChoice                     // grammar / -> struct with pointer fields
)

// FieldInfo describes a single struct field tagged with ll:"RuleName".
type FieldInfo struct {
	GoName   string      // Go field name
	LLTag    string      // rule name from ll:"..." tag
	Kind     FieldKind   // classification
	GoType   string      // Go type name (e.g., "string", "*JSONObject")
	ElemType string      // for slices: element type name
	Inner    *StructInfo // for nested structs with ll: tags
	NameID   int32       // resolved from bytecode smap (set during validation)
}

// StructInfo describes a struct with ll:-tagged fields.
type StructInfo struct {
	Name   string
	Fields []FieldInfo
}

// RuleKind classifies a grammar rule's expression structure.
type RuleKind int

const (
	RuleLeaf     RuleKind = iota // terminal (literal, charset, range, any)
	RuleSequence                 // SequenceNode
	RuleChoice                   // ChoiceNode (possibly nested)
	RuleRepeat                   // ZeroOrMore or OneOrMore
	RuleOptional                 // OptionalNode
	RuleAlias                    // single IdentifierNode (rule reference)
)

// RuleInfo describes a grammar rule's structure for extraction purposes.
type RuleInfo struct {
	Name     string
	Kind     RuleKind
	NameID   int32       // from Bytecode.smap
	Children []RuleChild // for sequences: ordered children with metadata
	Choices  []string    // for choices: rule names of alternatives
	Inner    string      // for alias/optional/repeat: the referenced rule
}

// RuleChild describes one item in a sequence rule.
type RuleChild struct {
	RuleName  string // rule or capture name; empty for literals
	IsLiteral bool   // true for structural punctuation (dead child)
	Repeated  bool   // true when child comes from a ZeroOrMore or OneOrMore
	Index     int    // position in SequenceNode.Items
}

// extractLLTag parses an ll:"RuleName" tag from a raw struct tag string.
// Returns the rule name and true if found, or ("", false) if absent or "-".
func extractLLTag(raw string) (string, bool) {
	tag := reflect.StructTag(strings.Trim(raw, "`"))
	val, ok := tag.Lookup("ll")
	if !ok || val == "" || val == "-" {
		return "", false
	}
	if idx := strings.Index(val, ","); idx >= 0 {
		val = val[:idx]
	}
	return val, true
}
