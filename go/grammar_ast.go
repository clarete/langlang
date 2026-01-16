package langlang

import (
	"fmt"
	"strings"
)

// AstNode is the interface that defines all behavior needed by the
// values output by the Grammar parser
type AstNode interface {
	// String returns the representation of the node recursively
	String() string

	// SourceLocation returns where the node was found in a
	// in a source file.  If the node was not found in a
	// source file, the FileID is -1.
	SourceLocation() SourceLocation

	// PrettyString returns the hierarchical structure of the node
	// recursively
	PrettyString() string

	// HighlightPrettyString returns the hierarchical structure of
	// the node recursively, highlighting different node types
	// with different ASCII colors
	HighlightPrettyString() string

	// Accept is an entrypoint for each node into the visitor
	Accept(AstNodeVisitor) error

	// Equal compares two AstNodes and returns true if they're
	// considered equal
	Equal(AstNode) bool
}

// Node Type: Error

type ErrorNode struct {
	src SourceLocation

	// Code is a stable, machine-usable category good for code
	// actions / filtering. "parse", "missing-closing-paren",
	// "missing-import-src", etc.
	Code string

	// Message is a human readable summary of what happened
	Message string

	// Optional recovery payload: "best-effort" subtree that was
	// parsed anyway
	Child AstNode

	// Expected is a human readable summary of what was expected
	Expected []ErrHint
}

func (n ErrorNode) SourceLocation() SourceLocation { return n.src }
func (n ErrorNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n ErrorNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n ErrorNode) Accept(v AstNodeVisitor) error  { return v.VisitErrorNode(&n) }

func (n ErrorNode) String() string {
	return fmt.Sprintf("Error(%s): %s", n.Code, n.Message)
}

func (n ErrorNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *ErrorNode:
		if len(n.Expected) != len(other.Expected) {
			return false
		}
		for i, item := range n.Expected {
			if !item.eq(&other.Expected[i]) {
				return false
			}
		}
		return n.Code == other.Code &&
			n.Message == other.Message &&
			n.Child.Equal(other.Child)
	default:
		return false
	}
}

// Node Type: Any

type AnyNode struct{ src SourceLocation }

func NewAnyNode(s SourceLocation) *AnyNode {
	n := &AnyNode{}
	n.src = s
	return n
}

func (n AnyNode) SourceLocation() SourceLocation { return n.src }
func (n AnyNode) String() string                 { return "." }
func (n AnyNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n AnyNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n AnyNode) Accept(v AstNodeVisitor) error  { return v.VisitAnyNode(&n) }

func (n AnyNode) Equal(o AstNode) bool {
	switch o.(type) {
	case *AnyNode:
		return true
	default:
		return false
	}
}

// Node Type: Literal

type LiteralNode struct {
	src   SourceLocation
	Value string
}

func NewLiteralNode(v string, s SourceLocation) *LiteralNode {
	n := &LiteralNode{Value: v}
	n.src = s
	return n
}

func (n LiteralNode) SourceLocation() SourceLocation { return n.src }
func (n LiteralNode) String() string                 { return fmt.Sprintf("'%s'", n.Value) }
func (n LiteralNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n LiteralNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n LiteralNode) Accept(v AstNodeVisitor) error  { return v.VisitLiteralNode(&n) }

func (n LiteralNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *LiteralNode:
		return n.Value == other.Value
	default:
		return false
	}
}

// Node Type: Identifier

type IdentifierNode struct {
	src   SourceLocation
	Value string
}

func NewIdentifierNode(v string, s SourceLocation) *IdentifierNode {
	n := &IdentifierNode{Value: v}
	n.src = s
	return n
}

func (n IdentifierNode) SourceLocation() SourceLocation { return n.src }
func (n IdentifierNode) String() string                 { return n.Value }
func (n IdentifierNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n IdentifierNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n IdentifierNode) Accept(v AstNodeVisitor) error  { return v.VisitIdentifierNode(&n) }

func (n IdentifierNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *IdentifierNode:
		return n.Value == other.Value
	default:
		return false
	}
}

// Node Type: Range

type RangeNode struct {
	src   SourceLocation
	Left  rune
	Right rune
}

func NewRangeNode(left, right rune, s SourceLocation) *RangeNode {
	n := &RangeNode{Left: left, Right: right}
	n.src = s
	return n
}

func (n RangeNode) SourceLocation() SourceLocation { return n.src }
func (n RangeNode) String() string                 { return fmt.Sprintf("%c-%c", n.Left, n.Right) }
func (n RangeNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n RangeNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n RangeNode) Accept(v AstNodeVisitor) error  { return v.VisitRangeNode(&n) }

func (n RangeNode) Equal(o AstNode) bool {
	if other, ok := o.(*RangeNode); ok {
		return n.Left == other.Left && n.Right == other.Right
	}
	return false
}

// Node Type: Charset

type CharsetNode struct {
	src SourceLocation
	cs  *charset
}

func NewCharsetNode(cs *charset, s SourceLocation) *CharsetNode {
	n := &CharsetNode{cs: cs}
	n.src = s
	return n
}

func (n CharsetNode) SourceLocation() SourceLocation { return n.src }
func (n CharsetNode) String() string                 { return n.cs.String() }
func (n CharsetNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n CharsetNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n CharsetNode) Accept(v AstNodeVisitor) error  { return v.VisitCharsetNode(&n) }

func (n CharsetNode) Equal(o AstNode) bool {
	if other, ok := o.(*CharsetNode); ok {
		return n.src == other.src && n.cs.eq(other.cs)
	}
	return false
}

// Node Type: Class

type ClassNode struct {
	src   SourceLocation
	Items []AstNode
}

func NewClassNode(items []AstNode, s SourceLocation) *ClassNode {
	n := &ClassNode{Items: items}
	n.src = s
	return n
}

func (n ClassNode) SourceLocation() SourceLocation { return n.src }
func (n ClassNode) String() string                 { return fmt.Sprintf("[%s]", nodesString(n.Items, "")) }
func (n ClassNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n ClassNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n ClassNode) Accept(v AstNodeVisitor) error  { return v.VisitClassNode(&n) }

func (n ClassNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *ClassNode:
		if len(n.Items) != len(other.Items) {
			return false
		}
		for i, item := range n.Items {
			if !item.Equal(other.Items[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Node Type: Optional

type OptionalNode struct {
	src  SourceLocation
	Expr AstNode
}

func NewOptionalNode(expr AstNode, s SourceLocation) *OptionalNode {
	n := &OptionalNode{Expr: expr}
	n.src = s
	return n
}

func (n OptionalNode) SourceLocation() SourceLocation { return n.src }
func (n OptionalNode) String() string                 { return fmt.Sprintf("%s?", n.Expr.String()) }
func (n OptionalNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n OptionalNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n OptionalNode) Accept(v AstNodeVisitor) error  { return v.VisitOptionalNode(&n) }

func (n OptionalNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *OptionalNode:
		return n.Expr.Equal(other.Expr)
	default:
		return false
	}
}

// Node Type: ZeroOrMore

type ZeroOrMoreNode struct {
	src  SourceLocation
	Expr AstNode
}

func NewZeroOrMoreNode(expr AstNode, s SourceLocation) *ZeroOrMoreNode {
	n := &ZeroOrMoreNode{Expr: expr}
	n.src = s
	return n
}

func (n ZeroOrMoreNode) SourceLocation() SourceLocation { return n.src }
func (n ZeroOrMoreNode) String() string                 { return fmt.Sprintf("%s*", n.Expr.String()) }
func (n ZeroOrMoreNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n ZeroOrMoreNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n ZeroOrMoreNode) Accept(v AstNodeVisitor) error  { return v.VisitZeroOrMoreNode(&n) }

func (n ZeroOrMoreNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *ZeroOrMoreNode:
		return n.Expr.Equal(other.Expr)
	default:
		return false
	}
}

// Node Type: OneOrMore

type OneOrMoreNode struct {
	src  SourceLocation
	Expr AstNode
}

func NewOneOrMoreNode(expr AstNode, s SourceLocation) *OneOrMoreNode {
	n := &OneOrMoreNode{Expr: expr}
	n.src = s
	return n
}

func (n OneOrMoreNode) SourceLocation() SourceLocation { return n.src }
func (n OneOrMoreNode) String() string                 { return fmt.Sprintf("%s+", n.Expr.String()) }
func (n OneOrMoreNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n OneOrMoreNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n OneOrMoreNode) Accept(v AstNodeVisitor) error  { return v.VisitOneOrMoreNode(&n) }

func (n OneOrMoreNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *OneOrMoreNode:
		return n.Expr.Equal(other.Expr)
	default:
		return false
	}
}

// Node Type: And

type AndNode struct {
	src  SourceLocation
	Expr AstNode
}

func NewAndNode(expr AstNode, s SourceLocation) *AndNode {
	n := &AndNode{Expr: expr}
	n.src = s
	return n
}

func (n AndNode) SourceLocation() SourceLocation { return n.src }
func (n AndNode) String() string                 { return fmt.Sprintf("&%s", n.Expr.String()) }
func (n AndNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n AndNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n AndNode) Accept(v AstNodeVisitor) error  { return v.VisitAndNode(&n) }

func (n AndNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *AndNode:
		return n.Expr.Equal(other.Expr)
	default:
		return false
	}
}

// Node Type: Not

type NotNode struct {
	src  SourceLocation
	Expr AstNode
}

func NewNotNode(expr AstNode, s SourceLocation) *NotNode {
	n := &NotNode{Expr: expr}
	n.src = s
	return n
}

func (n NotNode) SourceLocation() SourceLocation { return n.src }
func (n NotNode) String() string                 { return fmt.Sprintf("!%s", n.Expr.String()) }
func (n NotNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n NotNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n NotNode) Accept(v AstNodeVisitor) error  { return v.VisitNotNode(&n) }

func (n NotNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *NotNode:
		return n.Expr.Equal(other.Expr)
	default:
		return false
	}
}

// Node Type: Lex

type LexNode struct {
	src  SourceLocation
	Expr AstNode
}

func NewLexNode(expr AstNode, s SourceLocation) *LexNode {
	n := &LexNode{Expr: expr}
	n.src = s
	return n
}

func (n LexNode) SourceLocation() SourceLocation { return n.src }
func (n LexNode) Accept(v AstNodeVisitor) error  { return v.VisitLexNode(&n) }
func (n LexNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n LexNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }

func (n LexNode) String() string {
	if _, ok := n.Expr.(SequenceNode); ok {
		return fmt.Sprintf("#(%s)", n.Expr.String())
	}
	return fmt.Sprintf("#%s", n.Expr)
}

func (n LexNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *LexNode:
		return n.Expr.Equal(other.Expr)
	default:
		return false
	}
}

// Node Type: Labeled

type LabeledNode struct {
	src   SourceLocation
	Label string
	Expr  AstNode
}

func NewLabeledNode(label string, expr AstNode, s SourceLocation) *LabeledNode {
	n := &LabeledNode{Label: label, Expr: expr}
	n.src = s
	return n
}

func (n LabeledNode) SourceLocation() SourceLocation { return n.src }
func (n LabeledNode) String() string                 { return fmt.Sprintf("%s^%s", n.Expr.String(), n.Label) }
func (n LabeledNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n LabeledNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n LabeledNode) Accept(v AstNodeVisitor) error  { return v.VisitLabeledNode(&n) }

func (n LabeledNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *LabeledNode:
		return n.Expr.Equal(other.Expr)
	default:
		return false
	}
}

// Node Type: Sequence

type SequenceNode struct {
	src   SourceLocation
	Items []AstNode
}

func NewSequenceNode(items []AstNode, s SourceLocation) *SequenceNode {
	n := &SequenceNode{Items: items}
	n.src = s
	return n
}

func (n SequenceNode) SourceLocation() SourceLocation { return n.src }
func (n SequenceNode) Accept(v AstNodeVisitor) error  { return v.VisitSequenceNode(&n) }
func (n SequenceNode) String() string                 { return nodesString(n.Items, " ") }
func (n SequenceNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n SequenceNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }

func (n SequenceNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *SequenceNode:
		if len(n.Items) != len(other.Items) {
			return false
		}
		for i, item := range n.Items {
			if !item.Equal(other.Items[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// Node Type: Choice

type ChoiceNode struct {
	src   SourceLocation
	Left  AstNode
	Right AstNode
}

func NewChoiceNode(left, right AstNode, s SourceLocation) *ChoiceNode {
	n := &ChoiceNode{Left: left, Right: right}
	n.src = s
	return n
}

func (n ChoiceNode) SourceLocation() SourceLocation { return n.src }
func (n ChoiceNode) Accept(v AstNodeVisitor) error  { return v.VisitChoiceNode(&n) }
func (n ChoiceNode) String() string                 { return fmt.Sprintf("%s / %s", n.Left.String(), n.Right.String()) }
func (n ChoiceNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n ChoiceNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }

func (n ChoiceNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *ChoiceNode:
		return n.Left.Equal(other.Left) && n.Right.Equal(other.Right)
	default:
		return false
	}
}

// Node Type: Capture

type CaptureNode struct {
	src  SourceLocation
	Name string
	Expr AstNode
}

func NewCaptureNode(name string, expr AstNode, s SourceLocation) *CaptureNode {
	return &CaptureNode{
		src:  s,
		Name: name,
		Expr: expr,
	}
}

func (n CaptureNode) SourceLocation() SourceLocation { return n.src }
func (n CaptureNode) Accept(v AstNodeVisitor) error  { return v.VisitCaptureNode(&n) }
func (n CaptureNode) String() string                 { return fmt.Sprintf("#%s{{ %s }}", n.Name, n.Expr) }
func (n CaptureNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n CaptureNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }

func (n CaptureNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *CaptureNode:
		return n.Expr.Equal(other.Expr)
	default:
		return false
	}
}

// Node Type: Definition

type DefinitionNode struct {
	src  SourceLocation
	Name string
	Expr AstNode
}

func NewDefinitionNode(name string, expr AstNode, s SourceLocation) *DefinitionNode {
	n := &DefinitionNode{Name: name, Expr: expr}
	n.src = s
	return n
}

func (n DefinitionNode) SourceLocation() SourceLocation { return n.src }
func (n DefinitionNode) String() string                 { return fmt.Sprintf("%s <- %s", n.Name, n.Expr.String()) }
func (n DefinitionNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n DefinitionNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n DefinitionNode) Accept(v AstNodeVisitor) error  { return v.VisitDefinitionNode(&n) }

func (n DefinitionNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *DefinitionNode:
		return n.Expr.Equal(other.Expr)
	default:
		return false
	}
}

// Node Type: Import

type ImportNode struct {
	src   SourceLocation
	Path  *LiteralNode
	Names []*LiteralNode
}

func NewImportNode(path *LiteralNode, names []*LiteralNode, s SourceLocation) *ImportNode {
	n := &ImportNode{Path: path, Names: names}
	n.src = s
	return n
}

func (n ImportNode) SourceLocation() SourceLocation { return n.src }
func (n ImportNode) Accept(v AstNodeVisitor) error  { return v.VisitImportNode(&n) }
func (n ImportNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n ImportNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }

func (n ImportNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *ImportNode:
		if len(n.Names) != len(other.Names) {
			return false
		}
		for i, name := range n.Names {
			if !name.Equal(other.Names[i]) {
				return false
			}
		}
		return n.Path.Equal(other.Path)
	default:
		return false
	}
}

func (n ImportNode) String() string {
	names := strings.Join(n.GetNames(), ", ")
	return fmt.Sprintf("import %s from \"%s\"", names, n.GetPath())
}

func (n ImportNode) GetPath() string {
	return n.Path.Value
}

func (n ImportNode) GetNames() []string {
	var names []string
	for _, name := range n.Names {
		names = append(names, name.Value)
	}
	return names
}

// Node Type: Grammar

type GrammarNode struct {
	src         SourceLocation
	Imports     []*ImportNode
	Definitions []*DefinitionNode
	DefsByName  map[string]*DefinitionNode
	SourceFiles []string
}

func NewGrammarNode(
	imps []*ImportNode,
	defs []*DefinitionNode,
	defsByName map[string]*DefinitionNode,
	s SourceLocation,
) *GrammarNode {
	n := &GrammarNode{Imports: imps, Definitions: defs, DefsByName: defsByName}
	n.src = s
	return n
}

func (n GrammarNode) SourceLocation() SourceLocation { return n.src }
func (n GrammarNode) String() string                 { return nodesString(n.GetItems(), "\n") }
func (n GrammarNode) PrettyString() string           { return ppAstNode(&n, formatNodePlain) }
func (n GrammarNode) HighlightPrettyString() string  { return ppAstNode(&n, formatNodeThemed) }
func (n GrammarNode) Accept(v AstNodeVisitor) error  { return v.VisitGrammarNode(&n) }

func (n GrammarNode) GetItems() []AstNode {
	var items []AstNode
	for _, imp := range n.Imports {
		items = append(items, imp)
	}
	for _, def := range n.Definitions {
		items = append(items, def)
	}
	return items
}

func (n GrammarNode) FirstDefinition() *DefinitionNode {
	if len(n.Definitions) == 0 {
		return nil
	}
	return n.Definitions[0]
}

func (n GrammarNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *GrammarNode:
		items := n.GetItems()
		otherItems := other.GetItems()
		if len(items) != len(otherItems) {
			return false
		}
		for i, item := range items {
			if !item.Equal(otherItems[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (n *GrammarNode) AddDefinition(def *DefinitionNode) {
	if _, ok := n.DefsByName[def.Name]; !ok {
		n.Definitions = append(n.Definitions, def)
		n.DefsByName[def.Name] = def
	}
}

// Helpers

type asString interface{ String() string }

func nodesString[T asString](items []T, sep string) string {
	var (
		s  strings.Builder
		ln = len(items) - 1
	)
	for i, child := range items {
		s.WriteString(child.String())

		if i < ln {
			s.WriteString(sep)
		}
	}
	return s.String()
}
