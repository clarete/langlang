package langlang

import (
	"fmt"
)

type AstFormatToken int

const (
	AstFormatToken_None AstFormatToken = iota
	AstFormatToken_Span
	AstFormatToken_Literal
	AstFormatToken_Operator
	AstFormatToken_Operand
)

// astPrinterTheme is a map from the tokens available for pretty
// printing a grammar to an ASCII color.  These colors are supposed to
// fair well on both dark and light terminal settings
var astPrinterTheme = map[AstFormatToken]string{
	AstFormatToken_None:     "\033[0m",          // reset
	AstFormatToken_Span:     "\033[1;31;5;228m", // orange
	AstFormatToken_Literal:  "\033[1;38;5;245m", // gray
	AstFormatToken_Operator: "\033[1;38;5;99m",  // purple
	AstFormatToken_Operand:  "\033[1;38;5;127m", // pink
}

func ppAstNode(n AstNode) string {
	pp := newTreePrinter(func(input string, token AstFormatToken) string {
		return astPrinterTheme[token] + input + astPrinterTheme[AstFormatToken_None]
	})
	gp := &grammarPrinter{pp}
	n.Accept(gp)
	return gp.output.String()
}

type grammarPrinter struct {
	*treePrinter[AstFormatToken]
}

func (gp *grammarPrinter) VisitGrammarNode(n *GrammarNode) error {
	gp.writeOperator("Grammar")
	gp.writeSpanl(n)

	for i, item := range n.Definitions {
		switch {
		case i == len(n.Definitions)-1:
			gp.pwrite("└── ")
			gp.indent("    ")
			item.Accept(gp)
			gp.unindent()
		default:
			gp.pwrite("├── ")
			gp.indent("│   ")
			item.Accept(gp)
			gp.unindent()
			gp.write("\n")
		}
	}

	return nil
}

func (gp *grammarPrinter) VisitImportNode(*ImportNode) error {
	return nil
}

func (gp *grammarPrinter) VisitDefinitionNode(n *DefinitionNode) error {
	gp.writeOperatorWithOneRand("Definition", n.Name)
	gp.writeSpanl(n)
	gp.pwrite("└── ")
	gp.indent("    ")
	n.Expr.Accept(gp)
	gp.unindent()
	return nil
}

func (gp *grammarPrinter) VisitCaptureNode(n *CaptureNode) error {
	gp.writeOperatorWithOneRand("Capture", n.Name)
	gp.writeSpanl(n)
	gp.pwrite("└── ")
	gp.indent("    ")
	n.Expr.Accept(gp)
	gp.unindent()
	return nil
}

func (gp *grammarPrinter) VisitSequenceNode(n *SequenceNode) error {
	gp.writeOperator("Sequence")

	if len(n.Items) > 0 {
		gp.writeSpanl(n)
	} else {
		gp.writeSpan(n)
	}

	for i, item := range n.Items {
		switch {
		case i == len(n.Items)-1:
			gp.pwrite("└── ")
			gp.indent("    ")
			item.Accept(gp)
			gp.unindent()
		default:
			gp.pwrite("├── ")
			gp.indent("│   ")
			item.Accept(gp)
			gp.unindent()
			gp.write("\n")
		}
	}
	return nil
}

func (gp *grammarPrinter) VisitOneOrMoreNode(n *OneOrMoreNode) error {
	gp.writeOperator("OneOrMore")
	gp.writeSpanl(n)
	gp.pwrite("└── ")
	gp.indent("    ")
	n.Expr.Accept(gp)
	gp.unindent()
	return nil
}

func (gp *grammarPrinter) VisitZeroOrMoreNode(n *ZeroOrMoreNode) error {
	gp.writeOperator("ZeroOrMore")
	gp.writeSpanl(n)
	gp.pwrite("└── ")
	gp.indent("    ")
	n.Expr.Accept(gp)
	gp.unindent()
	return nil
}

func (gp *grammarPrinter) VisitOptionalNode(n *OptionalNode) error {
	gp.writeOperator("Optional")
	gp.writeSpanl(n)
	gp.pwrite("└── ")
	gp.indent("    ")
	n.Expr.Accept(gp)
	gp.unindent()
	return nil
}

func (gp *grammarPrinter) VisitChoiceNode(n *ChoiceNode) error {
	gp.writeOperator("Choice")
	gp.writeSpanl(n)

	for i, item := range n.Items {
		switch {
		case i == len(n.Items)-1:
			gp.pwrite("└── ")
			gp.indent("    ")
			item.Accept(gp)
			gp.unindent()
		default:
			gp.pwrite("├── ")
			gp.indent("│   ")
			item.Accept(gp)
			gp.unindent()
			gp.write("\n")
		}
	}
	return nil
}

func (gp *grammarPrinter) VisitAndNode(n *AndNode) error {
	gp.writeOperator("And")
	gp.writeSpanl(n)
	gp.pwrite("└── ")
	gp.indent("    ")
	n.Expr.Accept(gp)
	gp.unindent()
	return nil
}

func (gp *grammarPrinter) VisitNotNode(n *NotNode) error {
	gp.writeOperator("Not")
	gp.writeSpanl(n)
	gp.pwrite("└── ")
	gp.indent("    ")
	n.Expr.Accept(gp)
	gp.unindent()
	return nil
}

func (gp *grammarPrinter) VisitLexNode(n *LexNode) error {
	gp.writeOperator("Lex")
	gp.writeSpanl(n)
	gp.pwrite("└── ")
	gp.indent("    ")
	n.Expr.Accept(gp)
	gp.unindent()
	return nil
}

func (gp *grammarPrinter) VisitLabeledNode(n *LabeledNode) error {
	gp.writeOperatorWithOneRand("Throw", n.Label)
	gp.writeSpanl(n)
	gp.pwrite("└── ")
	gp.indent("    ")
	n.Expr.Accept(gp)
	gp.unindent()
	return nil
}

func (gp *grammarPrinter) VisitLiteralNode(n *LiteralNode) error {
	gp.writeOperatorWithOneRand("Literal", n.Value)
	gp.writeSpan(n)
	return nil
}

func (gp *grammarPrinter) VisitClassNode(n *ClassNode) error {
	gp.writeOperator("Class")
	gp.writeSpanl(n)
	for i, item := range n.Items {
		switch {
		case i == len(n.Items)-1:
			gp.pwrite("└── ")
			gp.indent("    ")
			item.Accept(gp)
			gp.unindent()
		default:
			gp.pwrite("├── ")
			gp.indent("│   ")
			item.Accept(gp)
			gp.unindent()
			gp.write("\n")
		}
	}
	return nil
}

func (gp *grammarPrinter) VisitRangeNode(n *RangeNode) error {
	gp.writeOperator("Range[")
	gp.writeOperand(string(n.Left))
	gp.writeOperator(", ")
	gp.writeOperand(string(n.Right))
	gp.writeOperator("]")
	gp.writeSpan(n)
	return nil
}

func (gp *grammarPrinter) VisitAnyNode(n *AnyNode) error {
	gp.writeOperator("Any")
	gp.writeSpan(n)
	return nil
}

func (gp *grammarPrinter) VisitIdentifierNode(n *IdentifierNode) error {
	gp.writeOperatorWithOneRand("Identifier", n.Value)
	gp.writeSpan(n)
	return nil
}

func (gp *grammarPrinter) writeOperator(op string) {
	gp.write(gp.format(op, AstFormatToken_Operator))
}

func (gp *grammarPrinter) writeOperatorWithOneRand(rator, rand string) {
	gp.write(gp.format(rator, AstFormatToken_Operator))
	gp.write(gp.format("[", AstFormatToken_Operator))
	gp.write(gp.format(rand, AstFormatToken_Operand))
	gp.write(gp.format("]", AstFormatToken_Operator))
}

func (gp *grammarPrinter) writeOperand(op string) {
	gp.write(gp.format(op, AstFormatToken_Operand))
}

func (gp *grammarPrinter) writeSpanl(n AstNode) {
	gp.writeSpan(n)
	gp.write("\n")
}

func (gp *grammarPrinter) writeSpan(n AstNode) {
	gp.write(gp.format(fmt.Sprintf(" (%s)", n.Span()), AstFormatToken_Span))
}
