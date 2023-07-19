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
}

// String Value

type ValueString struct {
	span  Span
	Value string
}

func NewValueString(value string, span Span) *ValueString {
	return &ValueString{span: span, Value: value}
}

func (n ValueString) Type() string   { return "string" }
func (n ValueString) Span() Span     { return n.span }
func (n ValueString) String() string { return fmt.Sprintf(`"%s" @ %s`, n.Value, n.Span()) }
func (n ValueString) Text() string   { return n.Value }

// Sequence Value

type ValueSequence struct {
	span  Span
	Items []Value
}

func NewValueSequence(items []Value, span Span) *ValueSequence {
	return &ValueSequence{Items: items, span: span}
}

func (n ValueSequence) Type() string { return "sequence" }
func (n ValueSequence) Span() Span   { return n.span }
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

func (n ValueNode) Type() string { return "node" }
func (n ValueNode) Span() Span   { return n.span }
func (n ValueNode) Text() string {
	if n.Expr == nil {
		return "???"
	}
	return n.Expr.Text()
}

func (n ValueNode) String() string {
	return fmt.Sprintf(`Node("%s", %s) @ %s`, n.Name, n.Expr, n.Span())
}
