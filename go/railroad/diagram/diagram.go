package diagram

import (
	"fmt"
	"strings"
)

type diagram interface {
	String() string
	Height() float64
	Width() float64
	Baseline() float64
	Accept(visitor) error
}

type visitor interface {
	AcceptTerm(*term) error
	AcceptNonTerm(*nonterm) error
	AcceptSeq(*seq) error
	AcceptStack(*stack) error
	AcceptEmpty(*empty) error
}

type term struct {
	label    string
	height   float64
	width    float64
	baseline float64
}

func newterm(label string) *term        { return &term{label: label} }
func (v *term) String() string          { return `"` + v.label + `"` }
func (v *term) Height() float64         { return v.height }
func (v *term) Width() float64          { return v.width }
func (v *term) Baseline() float64       { return v.baseline }
func (v *term) Accept(vi visitor) error { return vi.AcceptTerm(v) }

type nonterm struct {
	label    string
	height   float64
	width    float64
	baseline float64
}

func newnonterm(label string) *nonterm     { return &nonterm{label: label} }
func (v *nonterm) String() string          { return `[` + v.label + `]` }
func (v *nonterm) Height() float64         { return v.height }
func (v *nonterm) Width() float64          { return v.width }
func (v *nonterm) Baseline() float64       { return v.baseline }
func (v *nonterm) Accept(vi visitor) error { return vi.AcceptNonTerm(v) }

type seq struct {
	items    []diagram
	height   float64
	width    float64
	baseline float64
}

func newseq(items []diagram) *seq      { return &seq{items: items} }
func (v *seq) Height() float64         { return v.height }
func (v *seq) Width() float64          { return v.width }
func (v *seq) Baseline() float64       { return v.baseline }
func (v *seq) Accept(vi visitor) error { return vi.AcceptSeq(v) }
func (v *seq) String() string {
	var (
		out     strings.Builder
		lastIdx = len(v.items) - 1
	)
	if lastIdx == 0 {
		return v.items[0].String()
	}
	out.WriteRune('(')
	for i, item := range v.items {
		out.WriteString(item.String())
		if i < lastIdx {
			out.WriteRune(' ')
		}
	}
	out.WriteRune(')')
	return out.String()
}

type polarity int

const (
	pol_unknown polarity = iota
	pol_plus
	pol_minus
)

func (v polarity) String() string {
	switch v {
	case pol_plus:
		return "+"
	case pol_minus:
		return "-"
	}
	panic("unknown polarity")
}

func newpolarity(in string) polarity {
	switch in {
	case "+":
		return pol_plus
	case "-":
		return pol_minus
	}
	panic("unknown polarity")
}

type stack struct {
	pol      polarity
	top      diagram
	bottom   diagram
	height   float64
	width    float64
	baseline float64
}

func newstack(pol polarity, top, bot diagram) *stack { return &stack{pol: pol, top: top, bottom: bot} }
func (v *stack) Height() float64                     { return v.height }
func (v *stack) Width() float64                      { return v.width }
func (v *stack) String() string                      { return fmt.Sprintf(`(%s %s %s)`, v.pol, v.top, v.bottom) }
func (v *stack) Baseline() float64                   { return v.baseline }
func (v *stack) Accept(vi visitor) error             { return vi.AcceptStack(v) }

type empty struct{}

func newempty() *empty                   { return &empty{} }
func (v *empty) String() string          { return "()" }
func (v *empty) Height() float64         { return 0 }
func (v *empty) Width() float64          { return 0 }
func (v *empty) Baseline() float64       { return 0 }
func (v *empty) Accept(vi visitor) error { return vi.AcceptEmpty(v) }
