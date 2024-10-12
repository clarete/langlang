package langlang

import (
	"fmt"
	"strings"
)

type Value interface {
	Span() Span
	String() string
	Text() string
	Type() string
	Accept(ValueVisitor)
}

type ValueVisitor interface {
	VisitString(n *String) error
	VisitSequence(n *Sequence) error
	VisitNode(n *Node) error
	VisitError(n *Error) error
}

// String Value

type String struct {
	span  Span
	Value string
}

func NewString(value string, span Span) *String {
	return &String{span: span, Value: value}
}

func (n String) Type() string          { return "string" }
func (n String) Span() Span            { return n.span }
func (n String) String() string        { return fmt.Sprintf(`"%s" @ %s`, n.Value, n.Span()) }
func (n String) Text() string          { return n.Value }
func (n String) Accept(v ValueVisitor) { v.VisitString(&n) }

// Sequence Value

type Sequence struct {
	span  Span
	Items []Value
}

func NewSequence(items []Value, span Span) *Sequence {
	return &Sequence{Items: items, span: span}
}

func (n Sequence) Type() string          { return "sequence" }
func (n Sequence) Span() Span            { return n.span }
func (n Sequence) Accept(v ValueVisitor) { v.VisitSequence(&n) }
func (n Sequence) String() string {
	var s strings.Builder
	s.WriteString("Sequence(")
	for i, expr := range n.Items {
		s.WriteString(expr.String())
		if i < len(n.Items)-1 {
			s.WriteString(", ")
		}
	}
	fmt.Fprintf(&s, ") @ %s", n.Span())
	return s.String()
}

func (n Sequence) Text() string {
	var s strings.Builder
	for _, expr := range n.Items {
		s.WriteString(expr.Text())
	}
	return s.String()
}

// Node Value

type Node struct {
	span Span
	Name string
	Expr Value
}

func NewNode(name string, expr Value, span Span) *Node {
	return &Node{Name: name, Expr: expr, span: span}
}

func (n Node) Type() string          { return "node" }
func (n Node) Span() Span            { return n.span }
func (n Node) Accept(v ValueVisitor) { v.VisitNode(&n) }

func (n Node) Text() string {
	if n.Expr == nil {
		return "???"
	}
	return n.Expr.Text()
}

func (n Node) String() string {
	return fmt.Sprintf("%s(%s) @ %s", n.Name, n.Expr, n.Span())
}

// Node Error

type Error struct {
	span  Span
	Label string
	Expr  Value
}

func NewError(label string, expr Value, span Span) *Error {
	return &Error{Label: label, Expr: expr, span: span}
}

func (n Error) Type() string          { return "error" }
func (n Error) Span() Span            { return n.span }
func (n Error) Accept(v ValueVisitor) { v.VisitError(&n) }

func (n Error) Text() string {
	if n.Expr == nil {
		return "error[" + n.Label + "]"
	}
	return fmt.Sprintf("error[%s: %s]", n.Label, n.Expr.Text())
}

func (n Error) String() string {
	if n.Expr == nil {
		return fmt.Sprintf(`Error("%s") @ %s`, n.Label, n.Span())
	}
	return fmt.Sprintf(`Error("%s", %s) @ %s`, n.Label, n.Expr, n.Span())
}
