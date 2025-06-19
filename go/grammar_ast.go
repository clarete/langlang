package langlang

import (
	"fmt"
	"strings"
)

// AstNode is the interface that defines all behavior needed by the
// values output by the Grammar parser
type AstNode interface {
	// Span returns the location span in which the node was found
	// within the input text
	Span() Span

	// String returns the representation of the node recursively
	String() string

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

	// IsSyntactic returns true only for nodes that are considered
	// syntactical rules.  Outside this module, it makes sense to
	// call this method on a `DefinitionNode`, but it'd then
	// trigger the recursive call needed to answer that question
	// in such level
	IsSyntactic() bool
}

// Node Type: Any

type AnyNode struct{ span Span }

func NewAnyNode(s Span) *AnyNode {
	n := &AnyNode{}
	n.span = s
	return n
}

func (n AnyNode) Span() Span                    { return n.span }
func (n AnyNode) IsSyntactic() bool             { return true }
func (n AnyNode) String() string                { return "." }
func (n AnyNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n AnyNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n AnyNode) Accept(v AstNodeVisitor) error { return v.VisitAnyNode(&n) }

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
	span  Span
	Value string
}

func NewLiteralNode(v string, s Span) *LiteralNode {
	n := &LiteralNode{Value: v}
	n.span = s
	return n
}

func (n LiteralNode) Span() Span                    { return n.span }
func (n LiteralNode) IsSyntactic() bool             { return true }
func (n LiteralNode) String() string                { return fmt.Sprintf("'%s'", n.Value) }
func (n LiteralNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n LiteralNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n LiteralNode) Accept(v AstNodeVisitor) error { return v.VisitLiteralNode(&n) }

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
	span  Span
	Value string
}

func NewIdentifierNode(v string, s Span) *IdentifierNode {
	n := &IdentifierNode{Value: v}
	n.span = s
	return n
}

func (n IdentifierNode) Span() Span                    { return n.span }
func (n IdentifierNode) IsSyntactic() bool             { return false }
func (n IdentifierNode) String() string                { return n.Value }
func (n IdentifierNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n IdentifierNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n IdentifierNode) Accept(v AstNodeVisitor) error { return v.VisitIdentifierNode(&n) }

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
	span  Span
	Left  rune
	Right rune
}

func NewRangeNode(left, right rune, s Span) *RangeNode {
	n := &RangeNode{Left: left, Right: right}
	n.span = s
	return n
}

func (n RangeNode) Span() Span                    { return n.span }
func (n RangeNode) IsSyntactic() bool             { return true }
func (n RangeNode) String() string                { return fmt.Sprintf("%c-%c", n.Left, n.Right) }
func (n RangeNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n RangeNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n RangeNode) Accept(v AstNodeVisitor) error { return v.VisitRangeNode(&n) }

func (n RangeNode) Equal(o AstNode) bool {
	switch other := o.(type) {
	case *RangeNode:
		return n.Left == other.Left && n.Right == other.Right
	default:
		return false
	}
}

// Node Type: Class

type ClassNode struct {
	span  Span
	Items []AstNode
}

func NewClassNode(items []AstNode, s Span) *ClassNode {
	n := &ClassNode{Items: items}
	n.span = s
	return n
}

func (n ClassNode) Span() Span                    { return n.span }
func (n ClassNode) IsSyntactic() bool             { return true }
func (n ClassNode) String() string                { return fmt.Sprintf("[%s]", nodesString(n.Items, "")) }
func (n ClassNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n ClassNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n ClassNode) Accept(v AstNodeVisitor) error { return v.VisitClassNode(&n) }

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
	span Span
	Expr AstNode
}

func NewOptionalNode(expr AstNode, s Span) *OptionalNode {
	n := &OptionalNode{Expr: expr}
	n.span = s
	return n
}

func (n OptionalNode) Span() Span                    { return n.span }
func (n OptionalNode) IsSyntactic() bool             { return n.Expr.IsSyntactic() }
func (n OptionalNode) String() string                { return fmt.Sprintf("%s?", n.Expr.String()) }
func (n OptionalNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n OptionalNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n OptionalNode) Accept(v AstNodeVisitor) error { return v.VisitOptionalNode(&n) }

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
	span Span
	Expr AstNode
}

func NewZeroOrMoreNode(expr AstNode, s Span) *ZeroOrMoreNode {
	n := &ZeroOrMoreNode{Expr: expr}
	n.span = s
	return n
}

func (n ZeroOrMoreNode) Span() Span                    { return n.span }
func (n ZeroOrMoreNode) IsSyntactic() bool             { return n.Expr.IsSyntactic() }
func (n ZeroOrMoreNode) String() string                { return fmt.Sprintf("%s*", n.Expr.String()) }
func (n ZeroOrMoreNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n ZeroOrMoreNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n ZeroOrMoreNode) Accept(v AstNodeVisitor) error { return v.VisitZeroOrMoreNode(&n) }

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
	span Span
	Expr AstNode
}

func NewOneOrMoreNode(expr AstNode, s Span) *OneOrMoreNode {
	n := &OneOrMoreNode{Expr: expr}
	n.span = s
	return n
}

func (n OneOrMoreNode) Span() Span                    { return n.span }
func (n OneOrMoreNode) IsSyntactic() bool             { return n.Expr.IsSyntactic() }
func (n OneOrMoreNode) String() string                { return fmt.Sprintf("%s+", n.Expr.String()) }
func (n OneOrMoreNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n OneOrMoreNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n OneOrMoreNode) Accept(v AstNodeVisitor) error { return v.VisitOneOrMoreNode(&n) }

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
	span Span
	Expr AstNode
}

func NewAndNode(expr AstNode, s Span) *AndNode {
	n := &AndNode{Expr: expr}
	n.span = s
	return n
}

func (n AndNode) Span() Span                    { return n.span }
func (n AndNode) IsSyntactic() bool             { return true }
func (n AndNode) String() string                { return fmt.Sprintf("&%s", n.Expr.String()) }
func (n AndNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n AndNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n AndNode) Accept(v AstNodeVisitor) error { return v.VisitAndNode(&n) }

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
	span Span
	Expr AstNode
}

func NewNotNode(expr AstNode, s Span) *NotNode {
	n := &NotNode{Expr: expr}
	n.span = s
	return n
}

func (n NotNode) Span() Span                    { return n.span }
func (n NotNode) IsSyntactic() bool             { return true }
func (n NotNode) String() string                { return fmt.Sprintf("!%s", n.Expr.String()) }
func (n NotNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n NotNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n NotNode) Accept(v AstNodeVisitor) error { return v.VisitNotNode(&n) }

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
	span Span
	Expr AstNode
}

func NewLexNode(expr AstNode, s Span) *LexNode {
	n := &LexNode{Expr: expr}
	n.span = s
	return n
}

func (n LexNode) Span() Span                    { return n.span }
func (n LexNode) IsSyntactic() bool             { return n.Expr.IsSyntactic() }
func (n LexNode) Accept(v AstNodeVisitor) error { return v.VisitLexNode(&n) }
func (n LexNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n LexNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }

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
	span  Span
	Label string
	Expr  AstNode
}

func NewLabeledNode(label string, expr AstNode, s Span) *LabeledNode {
	n := &LabeledNode{Label: label, Expr: expr}
	n.span = s
	return n
}

func (n LabeledNode) Span() Span                    { return n.span }
func (n LabeledNode) IsSyntactic() bool             { return false }
func (n LabeledNode) String() string                { return fmt.Sprintf("%s^%s", n.Expr.String(), n.Label) }
func (n LabeledNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n LabeledNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n LabeledNode) Accept(v AstNodeVisitor) error { return v.VisitLabeledNode(&n) }

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
	span  Span
	Items []AstNode
}

func NewSequenceNode(items []AstNode, s Span) *SequenceNode {
	n := &SequenceNode{Items: items}
	n.span = s
	return n
}

func (n SequenceNode) Span() Span { return n.span }

func (n SequenceNode) IsSyntactic() bool {
	for _, expr := range n.Items {
		if !expr.IsSyntactic() {
			return false
		}
	}
	return true
}

func (n SequenceNode) Accept(v AstNodeVisitor) error { return v.VisitSequenceNode(&n) }
func (n SequenceNode) String() string                { return nodesString(n.Items, " ") }
func (n SequenceNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n SequenceNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }

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
	span  Span
	Left  AstNode
	Right AstNode
}

func NewChoiceNode(left, right AstNode, s Span) *ChoiceNode {
	n := &ChoiceNode{Left: left, Right: right}
	n.span = s
	return n
}

func (n ChoiceNode) Span() Span                    { return n.span }
func (n ChoiceNode) IsSyntactic() bool             { return n.Left.IsSyntactic() && n.Right.IsSyntactic() }
func (n ChoiceNode) Accept(v AstNodeVisitor) error { return v.VisitChoiceNode(&n) }
func (n ChoiceNode) String() string                { return fmt.Sprintf("%s / %s", n.Left.String(), n.Right.String()) }
func (n ChoiceNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n ChoiceNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }

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
	span Span
	Name string
	Expr AstNode
}

func NewCaptureNode(name string, expr AstNode, s Span) *CaptureNode {
	return &CaptureNode{
		span: s,
		Name: name,
		Expr: expr,
	}
}

func (n CaptureNode) Span() Span                    { return n.span }
func (n CaptureNode) IsSyntactic() bool             { return n.Expr.IsSyntactic() }
func (n CaptureNode) Accept(v AstNodeVisitor) error { return v.VisitCaptureNode(&n) }
func (n CaptureNode) String() string                { return fmt.Sprintf("#%s{{ %s }}", n.Name, n.Expr) }
func (n CaptureNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n CaptureNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }

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
	span Span
	Name string
	Expr AstNode
}

func NewDefinitionNode(name string, expr AstNode, s Span) *DefinitionNode {
	n := &DefinitionNode{Name: name, Expr: expr}
	n.span = s
	return n
}

func (n DefinitionNode) Span() Span                    { return n.span }
func (n DefinitionNode) IsSyntactic() bool             { return n.Expr.IsSyntactic() }
func (n DefinitionNode) String() string                { return fmt.Sprintf("%s <- %s", n.Name, n.Expr.String()) }
func (n DefinitionNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n DefinitionNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n DefinitionNode) Accept(v AstNodeVisitor) error { return v.VisitDefinitionNode(&n) }

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
	span  Span
	Path  *LiteralNode
	Names []*LiteralNode
}

func NewImportNode(path *LiteralNode, names []*LiteralNode, s Span) *ImportNode {
	n := &ImportNode{Path: path, Names: names}
	n.span = s
	return n
}

func (n ImportNode) Span() Span                    { return n.span }
func (n ImportNode) IsSyntactic() bool             { return false }
func (n ImportNode) Accept(v AstNodeVisitor) error { return v.VisitImportNode(&n) }
func (n ImportNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n ImportNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }

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
	span        Span
	Imports     []*ImportNode
	Definitions []*DefinitionNode
	DefsByName  map[string]*DefinitionNode
}

func NewGrammarNode(
	imps []*ImportNode,
	defs []*DefinitionNode,
	defsByName map[string]*DefinitionNode,
	s Span,
) *GrammarNode {
	n := &GrammarNode{Imports: imps, Definitions: defs, DefsByName: defsByName}
	n.span = s
	return n
}

func (n GrammarNode) Span() Span                    { return n.span }
func (n GrammarNode) IsSyntactic() bool             { return false }
func (n GrammarNode) String() string                { return nodesString(n.GetItems(), "\n") }
func (n GrammarNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n GrammarNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n GrammarNode) Accept(v AstNodeVisitor) error { return v.VisitGrammarNode(&n) }

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
