package langlang

import (
	"fmt"
	"strconv"
	"strings"
)

type FormatToken int

const (
	FormatToken_None FormatToken = iota
	FormatToken_Span
	FormatToken_Literal
	FormatToken_Error
)

type FormatFn func(input string, token FormatToken) string

type Value interface {
	Span() Span
	String() string
	Text() string
	Type() string
	Accept(ValueVisitor) error
	Format(FormatFn) string
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

func (n String) Type() string                { return "string" }
func (n String) Span() Span                  { return n.span }
func (n String) String() string              { return fmt.Sprintf(`"%s" @ %s`, n.Value, n.Span()) }
func (n String) Text() string                { return n.Value }
func (n String) Accept(v ValueVisitor) error { return v.VisitString(&n) }
func (n String) Format(fn FormatFn) string   { return formatNode(n, fn) }

// Sequence Value

type Sequence struct {
	span  Span
	Items []Value
}

func NewSequence(items []Value, span Span) *Sequence {
	return &Sequence{Items: items, span: span}
}

func (n Sequence) Type() string                { return "sequence" }
func (n Sequence) Span() Span                  { return n.span }
func (n Sequence) Accept(v ValueVisitor) error { return v.VisitSequence(&n) }
func (n Sequence) Format(fn FormatFn) string   { return formatNode(n, fn) }
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

func (n Node) Type() string                { return "node" }
func (n Node) Span() Span                  { return n.span }
func (n Node) Accept(v ValueVisitor) error { return v.VisitNode(&n) }
func (n Node) Format(fn FormatFn) string   { return formatNode(n, fn) }

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

func (n Error) Type() string                { return "error" }
func (n Error) Span() Span                  { return n.span }
func (n Error) Accept(v ValueVisitor) error { return v.VisitError(&n) }
func (n Error) Format(fn FormatFn) string   { return formatNode(n, fn) }

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

type ValuePrinter struct {
	padStr *[]string
	output *strings.Builder
	format FormatFn
}

func NewValuePrinter(format FormatFn) *ValuePrinter {
	return &ValuePrinter{
		padStr: &[]string{},
		output: &strings.Builder{},
		format: format,
	}
}

func formatNode(node Value, fmtFn FormatFn) string {
	p := NewValuePrinter(fmtFn)
	node.Accept(p)
	return p.output.String()
}

func (v *ValuePrinter) VisitString(n *String) error {
	escaped := strconv.Quote(n.Value)
	v.write(v.format(escaped, FormatToken_Literal))
	v.write(v.format(fmt.Sprintf(" (%s)", n.Span()), FormatToken_Span))
	return nil
}

func (v *ValuePrinter) VisitError(n *Error) error {
	v.write(v.format("Error", FormatToken_Error))
	v.write(v.format(fmt.Sprintf(" (%s)", n.Span()), FormatToken_Span))
	return nil
}

func (v *ValuePrinter) VisitSequence(n *Sequence) error {
	seq := fmt.Sprintf("Sequence<%d> (%s)", len(n.Items), n.Span())
	v.writel(v.format(seq, FormatToken_Span))
	for i, item := range n.Items {
		switch {
		case i == len(n.Items)-1:
			v.pwrite("└── ")
			v.indent("    ")
			item.Accept(v)
			v.unindent()
		default:
			v.pwrite("├── ")
			v.indent("│   ")
			item.Accept(v)
			v.unindent()
			v.write("\n")
		}
	}
	return nil
}

func (v *ValuePrinter) VisitNode(n *Node) error {
	v.write(v.format(n.Name, FormatToken_Literal))
	v.writel(v.format(fmt.Sprintf(" (%s)", n.Span()), FormatToken_Span))
	v.pwrite("└── ")
	v.indent("    ")
	n.Expr.Accept(v)
	v.unindent()
	return nil
}

func (v *ValuePrinter) indent(s string) {
	*v.padStr = append(*v.padStr, s)
}

func (v *ValuePrinter) unindent() {
	index := len(*v.padStr) - 1
	*v.padStr = (*v.padStr)[:index]
}

func (v *ValuePrinter) padding() {
	for _, item := range *v.padStr {
		v.write(item)
	}
}

func (v *ValuePrinter) writel(s string) {
	v.write(s)
	v.output.WriteRune('\n')
}

func (v *ValuePrinter) pwritel(s string) {
	v.pwrite(s)
	v.output.WriteRune('\n')
}

func (v *ValuePrinter) write(s string) {
	v.output.WriteString(s)
}

func (v *ValuePrinter) pwrite(s string) {
	v.padding()
	v.write(s)
}
