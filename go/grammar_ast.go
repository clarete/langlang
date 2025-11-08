package langlang

import (
	"fmt"
	"strings"
)

// AstNode is the interface that defines all behavior needed by the
// values output by the Grammar parser
type AstNode interface {
	// Range returns the location rg in which the node was found
	// within the input text
	Range() Range

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
}

// Node Type: Any

type AnyNode struct{ rg Range }

func NewAnyNode(s Range) *AnyNode {
	n := &AnyNode{}
	n.rg = s
	return n
}

func (n AnyNode) Range() Range                  { return n.rg }
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
	rg    Range
	Value string
}

func NewLiteralNode(v string, s Range) *LiteralNode {
	n := &LiteralNode{Value: v}
	n.rg = s
	return n
}

func (n LiteralNode) Range() Range                  { return n.rg }
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
	rg    Range
	Value string
}

func NewIdentifierNode(v string, s Range) *IdentifierNode {
	n := &IdentifierNode{Value: v}
	n.rg = s
	return n
}

func (n IdentifierNode) Range() Range                  { return n.rg }
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
	rg    Range
	Left  rune
	Right rune
}

func NewRangeNode(left, right rune, s Range) *RangeNode {
	n := &RangeNode{Left: left, Right: right}
	n.rg = s
	return n
}

func (n RangeNode) Range() Range                  { return n.rg }
func (n RangeNode) String() string                { return fmt.Sprintf("%c-%c", n.Left, n.Right) }
func (n RangeNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n RangeNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n RangeNode) Accept(v AstNodeVisitor) error { return v.VisitRangeNode(&n) }

func (n RangeNode) Equal(o AstNode) bool {
	if other, ok := o.(*RangeNode); ok {
		return n.Left == other.Left && n.Right == other.Right
	}
	return false
}

// Node Type: Charset

type CharsetNode struct {
	rg Range
	cs *charset
}

func NewCharsetNode(cs *charset, s Range) *CharsetNode {
	n := &CharsetNode{cs: cs}
	n.rg = s
	return n
}

func (n CharsetNode) Range() Range                  { return n.rg }
func (n CharsetNode) String() string                { return n.cs.String() }
func (n CharsetNode) PrettyString() string          { return ppAstNode(&n, formatNodePlain) }
func (n CharsetNode) HighlightPrettyString() string { return ppAstNode(&n, formatNodeThemed) }
func (n CharsetNode) Accept(v AstNodeVisitor) error { return v.VisitCharsetNode(&n) }

func (n CharsetNode) Equal(o AstNode) bool {
	if other, ok := o.(*CharsetNode); ok {
		return n.rg == other.rg && n.cs.eq(other.cs)
	}
	return false
}

// Node Type: Class

type ClassNode struct {
	rg    Range
	Items []AstNode
}

func NewClassNode(items []AstNode, s Range) *ClassNode {
	n := &ClassNode{Items: items}
	n.rg = s
	return n
}

func (n ClassNode) Range() Range                  { return n.rg }
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
	rg   Range
	Expr AstNode
}

func NewOptionalNode(expr AstNode, s Range) *OptionalNode {
	n := &OptionalNode{Expr: expr}
	n.rg = s
	return n
}

func (n OptionalNode) Range() Range                  { return n.rg }
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
	rg   Range
	Expr AstNode
}

func NewZeroOrMoreNode(expr AstNode, s Range) *ZeroOrMoreNode {
	n := &ZeroOrMoreNode{Expr: expr}
	n.rg = s
	return n
}

func (n ZeroOrMoreNode) Range() Range                  { return n.rg }
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
	rg   Range
	Expr AstNode
}

func NewOneOrMoreNode(expr AstNode, s Range) *OneOrMoreNode {
	n := &OneOrMoreNode{Expr: expr}
	n.rg = s
	return n
}

func (n OneOrMoreNode) Range() Range                  { return n.rg }
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
	rg   Range
	Expr AstNode
}

func NewAndNode(expr AstNode, s Range) *AndNode {
	n := &AndNode{Expr: expr}
	n.rg = s
	return n
}

func (n AndNode) Range() Range                  { return n.rg }
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
	rg   Range
	Expr AstNode
}

func NewNotNode(expr AstNode, s Range) *NotNode {
	n := &NotNode{Expr: expr}
	n.rg = s
	return n
}

func (n NotNode) Range() Range                  { return n.rg }
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
	rg   Range
	Expr AstNode
}

func NewLexNode(expr AstNode, s Range) *LexNode {
	n := &LexNode{Expr: expr}
	n.rg = s
	return n
}

func (n LexNode) Range() Range                  { return n.rg }
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
	rg    Range
	Label string
	Expr  AstNode
}

func NewLabeledNode(label string, expr AstNode, s Range) *LabeledNode {
	n := &LabeledNode{Label: label, Expr: expr}
	n.rg = s
	return n
}

func (n LabeledNode) Range() Range                  { return n.rg }
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
	rg    Range
	Items []AstNode
}

func NewSequenceNode(items []AstNode, s Range) *SequenceNode {
	n := &SequenceNode{Items: items}
	n.rg = s
	return n
}

func (n SequenceNode) Range() Range                  { return n.rg }
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
	rg    Range
	Left  AstNode
	Right AstNode
}

func NewChoiceNode(left, right AstNode, s Range) *ChoiceNode {
	n := &ChoiceNode{Left: left, Right: right}
	n.rg = s
	return n
}

func (n ChoiceNode) Range() Range                  { return n.rg }
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
	rg   Range
	Name string
	Expr AstNode
}

func NewCaptureNode(name string, expr AstNode, s Range) *CaptureNode {
	return &CaptureNode{
		rg:   s,
		Name: name,
		Expr: expr,
	}
}

func (n CaptureNode) Range() Range                  { return n.rg }
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
	rg   Range
	Name string
	Expr AstNode
}

func NewDefinitionNode(name string, expr AstNode, s Range) *DefinitionNode {
	n := &DefinitionNode{Name: name, Expr: expr}
	n.rg = s
	return n
}

func (n DefinitionNode) Range() Range                  { return n.rg }
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
	rg    Range
	Path  *LiteralNode
	Names []*LiteralNode
}

func NewImportNode(path *LiteralNode, names []*LiteralNode, s Range) *ImportNode {
	n := &ImportNode{Path: path, Names: names}
	n.rg = s
	return n
}

func (n ImportNode) Range() Range                  { return n.rg }
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
	rg          Range
	Imports     []*ImportNode
	Definitions []*DefinitionNode
	DefsByName  map[string]*DefinitionNode
}

func NewGrammarNode(
	imps []*ImportNode,
	defs []*DefinitionNode,
	defsByName map[string]*DefinitionNode,
	s Range,
) *GrammarNode {
	n := &GrammarNode{Imports: imps, Definitions: defs, DefsByName: defsByName}
	n.rg = s
	return n
}

func (n GrammarNode) Range() Range                  { return n.rg }
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
