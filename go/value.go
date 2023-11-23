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
	VisitValueString(n *ValueString) error
	VisitValueSequence(n *ValueSequence) error
	VisitValueNode(n *ValueNode) error
	VisitValueError(n *ValueError) error
}

// String Value

type ValueString struct {
	span  Span
	Value string
}

func NewValueString(value string, span Span) *ValueString {
	return &ValueString{span: span, Value: value}
}

func (n ValueString) Type() string          { return "string" }
func (n ValueString) Span() Span            { return n.span }
func (n ValueString) String() string        { return fmt.Sprintf(`"%s" @ %s`, n.Value, n.Span()) }
func (n ValueString) Text() string          { return n.Value }
func (n ValueString) Accept(v ValueVisitor) { v.VisitValueString(&n) }

// Sequence Value

type ValueSequence struct {
	span  Span
	Items []Value
}

func NewValueSequence(items []Value, span Span) *ValueSequence {
	return &ValueSequence{Items: items, span: span}
}

func (n ValueSequence) Type() string          { return "sequence" }
func (n ValueSequence) Span() Span            { return n.span }
func (n ValueSequence) Accept(v ValueVisitor) { v.VisitValueSequence(&n) }
func (n ValueSequence) String() string {
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

func (n ValueSequence) Text() string {
	var s strings.Builder
	for _, expr := range n.Items {
		s.WriteString(expr.Text())
	}
	return s.String()
}

// Node Value

type ValueNode struct {
	span Span
	Name string
	Expr Value
}

func NewValueNode(name string, expr Value, span Span) *ValueNode {
	return &ValueNode{Name: name, Expr: expr, span: span}
}

func (n ValueNode) Type() string          { return "node" }
func (n ValueNode) Span() Span            { return n.span }
func (n ValueNode) Accept(v ValueVisitor) { v.VisitValueNode(&n) }

func (n ValueNode) Text() string {
	if n.Expr == nil {
		return "???"
	}
	return n.Expr.Text()
}

func (n ValueNode) String() string {
	return fmt.Sprintf("%s(%s) @ %s", n.Name, n.Expr, n.Span())
}

// Node Error

type ValueError struct {
	span  Span
	Label string
	Expr  Value
}

func NewValueError(label string, expr Value, span Span) *ValueError {
	return &ValueError{Label: label, Expr: expr, span: span}
}

func (n ValueError) Type() string          { return "error" }
func (n ValueError) Span() Span            { return n.span }
func (n ValueError) Accept(v ValueVisitor) { v.VisitValueError(&n) }

func (n ValueError) Text() string {
	if n.Expr == nil {
		return "error[" + n.Label + "]"
	}
	return fmt.Sprintf("error[%s: %s]", n.Label, n.Expr.Text())
}

func (n ValueError) String() string {
	if n.Expr == nil {
		return fmt.Sprintf(`Error("%s") @ %s`, n.Label, n.Span())
	}
	return fmt.Sprintf(`Error("%s", %s) @ %s`, n.Label, n.Expr, n.Span())
}
