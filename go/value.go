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

var valuePrinterTheme = map[FormatToken]string{
	FormatToken_None:    "\033[0m",          // reset
	FormatToken_Span:    "\033[1;31;5;228m", // orange
	FormatToken_Literal: "\033[1;38;5;245m", // gray
	FormatToken_Error:   "\033[1;38;5;127m", // pink
}

const eof = -1

type Location struct {
	Line   int
	Column int
	Cursor int
	File   string
}

func NewLocation(line, column, cursor int) Location {
	return Location{Line: line, Column: column, Cursor: cursor}
}

func (l Location) String() string {
	if l.Line == 0 {
		return fmt.Sprintf("%d", l.Column+1)
	}
	return fmt.Sprintf("%d:%d", l.Line+1, l.Column+1)
}

type Span struct {
	Start Location
	End   Location
}

func NewSpan(s, e Location) Span {
	return Span{s, e}
}

func (s Span) String() string {
	var prefix string
	if s.Start.File != "" {
		prefix = s.Start.File + ":"
	}
	if s.Start.Line == s.End.Line && s.Start.Column == s.End.Column {
		return s.Start.String()
	}
	if s.Start.Line == s.End.Line && s.Start.Line == 1 {
		if s.Start.Column == s.End.Column {
			return fmt.Sprintf("%s%d", prefix, s.Start.Column)
		}
		return fmt.Sprintf("%s%d..%d", prefix, s.Start.Column, s.End.Column)
	}
	return fmt.Sprintf("%s%s..%s", prefix, s.Start, s.End)
}

type Value interface {
	Span() Span
	String() string
	Text() string
	Type() string
	Accept(ValueVisitor) error
	PrettyString() string
	HighlightPrettyString() string
	Format(FormatFunc[FormatToken]) string
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

func (n String) Type() string                             { return "string" }
func (n String) Span() Span                               { return n.span }
func (n String) String() string                           { return fmt.Sprintf(`"%s" @ %s`, n.Value, n.Span()) }
func (n String) Text() string                             { return n.Value }
func (n String) Accept(v ValueVisitor) error              { return v.VisitString(&n) }
func (n String) PrettyString() string                     { return n.Format(formatValuePlain) }
func (n String) HighlightPrettyString() string            { return n.Format(formatValueHighlight) }
func (n String) Format(fn FormatFunc[FormatToken]) string { return formatValue(n, fn) }

// Sequence Value

type Sequence struct {
	span  Span
	Items []Value
}

func NewSequence(items []Value, span Span) *Sequence {
	return &Sequence{Items: items, span: span}
}

func (n Sequence) Type() string                             { return "sequence" }
func (n Sequence) Span() Span                               { return n.span }
func (n Sequence) Accept(v ValueVisitor) error              { return v.VisitSequence(&n) }
func (n Sequence) PrettyString() string                     { return n.Format(formatValuePlain) }
func (n Sequence) HighlightPrettyString() string            { return n.Format(formatValueHighlight) }
func (n Sequence) Format(fn FormatFunc[FormatToken]) string { return formatValue(n, fn) }
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

func (n Node) Type() string                             { return "node" }
func (n Node) Span() Span                               { return n.span }
func (n Node) Accept(v ValueVisitor) error              { return v.VisitNode(&n) }
func (n Node) String() string                           { return fmt.Sprintf("%s(%s) @ %s", n.Name, n.Expr, n.Span()) }
func (n Node) PrettyString() string                     { return n.Format(formatValuePlain) }
func (n Node) HighlightPrettyString() string            { return n.Format(formatValueHighlight) }
func (n Node) Format(fn FormatFunc[FormatToken]) string { return formatValue(n, fn) }

func (n Node) Text() string {
	if n.Expr == nil {
		return "???"
	}
	return n.Expr.Text()
}

// Node Error

type Error struct {
	span    Span
	Label   string
	Message string
	Expr    Value
}

func NewError(label, message string, expr Value, span Span) *Error {
	return &Error{Label: label, Message: message, Expr: expr, span: span}
}

func (n Error) Type() string                             { return "error" }
func (n Error) Span() Span                               { return n.span }
func (n Error) Accept(v ValueVisitor) error              { return v.VisitError(&n) }
func (n Error) PrettyString() string                     { return n.Format(formatValuePlain) }
func (n Error) HighlightPrettyString() string            { return n.Format(formatValueHighlight) }
func (n Error) Format(fn FormatFunc[FormatToken]) string { return formatValue(n, fn) }

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

func (n Error) AsError() ParsingError {
	return ParsingError{
		Label:   n.Label,
		Message: n.Message,
		Span:    n.Span(),
	}
}

type ValuePrinter struct {
	*treePrinter[FormatToken]
}

func NewValuePrinter(format FormatFunc[FormatToken]) *ValuePrinter {
	return &ValuePrinter{newTreePrinter(format)}
}

func formatValuePlain(input string, _ FormatToken) string {
	return input
}

func formatValueHighlight(input string, token FormatToken) string {
	return valuePrinterTheme[token] + input + valuePrinterTheme[FormatToken_None]
}

func formatValue(node Value, fmtFn FormatFunc[FormatToken]) string {
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
	v.write(v.format(fmt.Sprintf("Error<%s>", n.Label), FormatToken_Error))
	v.write(v.format(fmt.Sprintf(" (%s)", n.Span()), FormatToken_Span))
	if n.Expr != nil {
		v.writel("")
		v.pwrite("└── ")
		v.indent("    ")
		n.Expr.Accept(v)
		v.unindent()
	}
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
