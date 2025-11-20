package langlang

import (
	"fmt"
	"strconv"
	"strings"
)

type FormatToken int

const (
	FormatToken_None FormatToken = iota
	FormatToken_Range
	FormatToken_Literal
	FormatToken_Error
)

var treePrinterTheme = map[FormatToken]string{
	FormatToken_None:    "\033[0m",          // reset
	FormatToken_Range:   "\033[1;31;5;228m", // orange
	FormatToken_Literal: "\033[1;38;5;245m", // gray
	FormatToken_Error:   "\033[1;38;5;127m", // pink
}

const eof = -1

// Range takes as little as possible (8 bytes in 64bit systems) to
// represent a position within the input.
type Range struct{ Start, End int }

func (r Range) String() string {
	if r.Start == r.End {
		return fmt.Sprintf("%d", r.Start)
	}
	return fmt.Sprintf("%d..%d", r.Start, r.End)
}

func NewRange(start, end int) Range {
	return Range{Start: start, End: end}
}

func (r Range) Str(v []byte) string {
	return string(v[r.Start:r.End])
}

type Value interface {
	Type() string
	Range() Range
	String(input []byte) string
	Accept(ValueVisitor) error
}

type ValueVisitor interface {
	VisitString(n *String) error
	VisitSequence(n *Sequence) error
	VisitNode(n *Node) error
	VisitError(n *Error) error
}

// String Value

type String struct {
	rg Range
}

func NewString(rg Range) *String {
	return &String{rg: rg}
}

func (n String) Type() string                 { return "string" }
func (n String) Range() Range                 { return n.rg }
func (n String) String(input []byte) string   { return n.Range().Str(input) }
func (n *String) Accept(v ValueVisitor) error { return v.VisitString(n) }

// Sequence Value

type Sequence struct {
	rg    Range
	Items []Value
}

func NewSequence(items []Value, rg Range) *Sequence {
	return &Sequence{Items: items, rg: rg}
}

func (n Sequence) Type() string                 { return "sequence" }
func (n Sequence) Range() Range                 { return n.rg }
func (n *Sequence) Accept(v ValueVisitor) error { return v.VisitSequence(n) }
func (n Sequence) String(input []byte) string {
	var s strings.Builder
	s.WriteString("Sequence(")
	for i, expr := range n.Items {
		s.WriteString(expr.String(input))
		if i < len(n.Items)-1 {
			s.WriteString(", ")
		}
	}
	fmt.Fprintf(&s, ")")
	return s.String()
}

// Node Value

type Node struct {
	rg   Range
	Name string
	Expr Value
}

func NewNode(name string, expr Value, rg Range) *Node {
	return &Node{Name: name, Expr: expr, rg: rg}
}

func (n Node) Type() string                 { return "node" }
func (n Node) Range() Range                 { return n.rg }
func (n *Node) Accept(v ValueVisitor) error { return v.VisitNode(n) }
func (n Node) String(input []byte) string   { return n.Range().Str(input) }

// Node Error

type Error struct {
	rg      Range
	Label   string
	Message string
	Expr    Value
}

func NewError(label, message string, expr Value, rg Range) *Error {
	return &Error{Label: label, Message: message, Expr: expr, rg: rg}
}

func (n Error) Type() string                 { return "error" }
func (n Error) Range() Range                 { return n.rg }
func (n *Error) Accept(v ValueVisitor) error { return v.VisitError(n) }

func (n Error) String(input []byte) string {
	return fmt.Sprintf(`Error("%s", "%s")`, n.Label, n.Range().Str(input))
}

func (n Error) AsError() ParsingError {
	return ParsingError{
		Label:   n.Label,
		Message: n.Message,
		Range:   n.Range(),
	}
}

// ---- Text Printer ----

func Text(input []byte, node Value) string {
	var (
		output  strings.Builder
		visitor = &TextVisitor{input: input, output: &output}
	)
	_ = node.Accept(visitor)
	return output.String()
}

type TextVisitor struct {
	input  []byte
	output *strings.Builder
}

func (v *TextVisitor) VisitString(n *String) error {
	r := n.Range()
	v.output.WriteString(string(v.input[r.Start:r.End]))
	return nil
}

func (v *TextVisitor) VisitSequence(n *Sequence) error {
	for _, expr := range n.Items {
		expr.Accept(v)
	}
	return nil
}

func (v *TextVisitor) VisitNode(n *Node) error {
	return n.Expr.Accept(v)
}

func (v *TextVisitor) VisitError(n *Error) error {
	if n.Expr == nil {
		v.output.WriteString("error[" + n.Label + "]")
		return nil
	}
	r := n.Expr.Range()
	v.output.WriteString("error[")
	v.output.WriteString(n.Label)
	v.output.WriteString(": ")
	v.output.WriteString(string(v.input[r.Start:r.End]))
	v.output.WriteString("]")
	return nil
}

// ---- Tree Printer ----

func PrettyString(input []byte, node Value) string {
	tp := NewTreePrinter(input, func(input string, _ FormatToken) string {
		return input
	})
	node.Accept(tp)
	return tp.output.String()
}

func HighlightPrettyString(input []byte, node Value) string {
	tp := NewTreePrinter(input, func(input string, token FormatToken) string {
		return treePrinterTheme[token] + input + treePrinterTheme[FormatToken_None]
	})
	node.Accept(tp)
	return tp.output.String()
}

type TreePrinter struct {
	input []byte
	*treePrinter[FormatToken]
}

func NewTreePrinter(input []byte, format FormatFunc[FormatToken]) *TreePrinter {
	return &TreePrinter{input: input, treePrinter: newTreePrinter(format)}
}

// posToLineCol converts a byte position in the input to line and
// column numbers (both 1-based)
func (v *TreePrinter) posToLineCol(pos int) (line, column int) {
	line, column = 1, 1
	data := v.input[0:pos]
	for _, ch := range data {
		if ch == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return line, column
}

// formatPosition formats a Range as "startLine:startCol-endLine:endCol"
func (v *TreePrinter) formatPosition(rg Range) string {
	startLine, startCol := v.posToLineCol(rg.Start)
	endLine, endCol := v.posToLineCol(rg.End)
	if startLine == endLine && startLine == 1 {
		if startCol == endCol {
			return fmt.Sprintf("%d", startCol)
		}
		return fmt.Sprintf("%d..%d", startCol, endCol)
	}
	if startLine == endLine && startCol == endCol {
		return fmt.Sprintf("%d:%d", startLine, startCol)
	}
	return fmt.Sprintf("%d:%d..%d:%d", startLine, startCol, endLine, endCol)
}

func (v *TreePrinter) VisitString(n *String) error {
	rg := n.Range()
	text := string(v.input[rg.Start:rg.End])
	escaped := strconv.Quote(text)
	v.write(v.format(escaped, FormatToken_Literal))
	v.write(v.format(fmt.Sprintf(" (%s)", v.formatPosition(rg)), FormatToken_Range))
	return nil
}

func (v *TreePrinter) VisitError(n *Error) error {
	v.write(v.format(fmt.Sprintf("Error<%s>", n.Label), FormatToken_Error))
	v.write(v.format(fmt.Sprintf(" (%s)", v.formatPosition(n.Range())), FormatToken_Range))
	if n.Expr != nil {
		v.writel("")
		v.pwrite("└── ")
		v.indent("    ")
		n.Expr.Accept(v)
		v.unindent()
	}
	return nil
}

func (v *TreePrinter) VisitSequence(n *Sequence) error {
	seq := fmt.Sprintf("Sequence<%d> (%s)", len(n.Items), v.formatPosition(n.Range()))
	v.writel(v.format(seq, FormatToken_Range))
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

func (v *TreePrinter) VisitNode(n *Node) error {
	v.write(v.format(n.Name, FormatToken_Literal))
	v.writel(v.format(fmt.Sprintf(" (%s)", v.formatPosition(n.Range())), FormatToken_Range))
	v.pwrite("└── ")
	v.indent("    ")
	n.Expr.Accept(v)
	v.unindent()
	return nil
}
