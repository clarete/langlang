package langlang

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

type pyCodeEmitter struct {
	options     GenPyOptions
	parser      *outputWriter
	indentLevel int
	lexLevel    int
	labelsMap   map[string]struct{}
	labels      []string
	grammarNode *GrammarNode
}

type GenPyOptions struct {
	GrammarPath string
	RemoveLib   bool
}

func GenPy(node AstNode, opt GenPyOptions) (string, error) {
	g := newPyCodeEmitter(opt)
	g.writePrelude()
	node.Accept(g)
	g.writeConstructor()
	return g.output()
}

//go:embed prelude.py.in
var pyPrelude string

func newPyCodeEmitter(opt GenPyOptions) *pyCodeEmitter {
	return &pyCodeEmitter{
		options:   opt,
		parser:    newOutputWriter("    "),
		labelsMap: map[string]struct{}{},
		labels:    []string{},
	}
}

func (g *pyCodeEmitter) VisitGrammarNode(n *GrammarNode) error { return WalkGrammarNode(g, n) }

func (g *pyCodeEmitter) VisitDefinitionNode(n *DefinitionNode) error {
	g.parser.write("\n")
	g.parser.indent()
	g.parser.writeil(fmt.Sprintf("def parse_%s(self) -> Value:", n.Name))
	g.parser.indent()
	g.parser.writeil("start = self.location()")
	g.parser.writei(fmt.Sprintf(`return self.mknode("%s", start, `, n.Name))

	if err := n.Expr.Accept(g); err != nil {
		return err
	}

	g.parser.writel(")")
	g.parser.unindent()
	g.parser.unindent()
	return nil
}

func (g *pyCodeEmitter) VisitSequenceNode(n *SequenceNode) error {
	shouldConsumeSpaces := g.lexLevel == 0 && g.isUnderRuleLevel() && !n.IsSyntactic()

	if len(n.Items) == 1 {
		n.Items[0].Accept(g)
		return nil
	}

	g.parser.writel("self.mk(self.location(), narrow([")
	g.parser.indent()

	for i, item := range n.Items {
		_, isLexNode := item.(*LexNode)
		if shouldConsumeSpaces && !isLexNode {
			g.parser.writeil("self.parse_spacing(),")
		}

		g.parser.writei("")
		g.writeExprFn(item)

		if i < len(n.Items)-1 {
			g.parser.writel(",")
		}
	}

	g.parser.write("]))")
	g.parser.unindent()
	return nil
}

func (g *pyCodeEmitter) VisitOneOrMoreNode(n *OneOrMoreNode) error {
	g.parser.write("self.one_or_more(")
	g.writeExprFn(n.Expr)
	g.parser.write(")")
	return nil
}

func (g *pyCodeEmitter) VisitZeroOrMoreNode(n *ZeroOrMoreNode) error {
	g.parser.write("self.zero_or_more(")
	g.writeExprFn(n.Expr)
	g.parser.write(")")
	return nil
}

func (g *pyCodeEmitter) VisitOptionalNode(n *OptionalNode) error {
	g.parser.write("self.optional(")
	g.writeExprFn(n.Expr)
	g.parser.write(")")
	return nil
}

func (g *pyCodeEmitter) VisitChoiceNode(n *ChoiceNode) error {
	switch len(n.Items) {
	case 0:
		return nil
	case 1:
		return n.Items[0].Accept(g)
	default:
		g.parser.write("self.choice([\n")
		g.parser.indent()

		for _, expr := range n.Items {
			g.parser.writei("")
			g.writeExprFn(expr)
			g.parser.write(",\n")
		}

		g.parser.unindent()
		g.parser.writei("])")
	}
	return nil
}

func (g *pyCodeEmitter) VisitAndNode(n *AndNode) error {
	g.parser.write("self.and_(")
	g.writeExprFn(n.Expr)
	g.parser.write(")")
	return nil
}

func (g *pyCodeEmitter) VisitNotNode(n *NotNode) error {
	g.parser.write("self.not_(")
	g.writeExprFn(n.Expr)
	g.parser.write(")")
	return nil
}

func (g *pyCodeEmitter) VisitLexNode(n *LexNode) error {
	g.lexLevel++
	if err := n.Expr.Accept(g); err != nil {
		return err
	}
	g.lexLevel--
	return nil
}

func (g *pyCodeEmitter) VisitLabeledNode(n *LabeledNode) error {
	g.labelsMap[n.Label] = struct{}{}

	panic("visitLabeledNode")

	// g.parser.write("func(p langlang.Parser) (langlang.Value, error) {\n")
	// g.parser.indent()
	// g.parser.writei("start := p.Location()\n")

	// g.parser.writei("return langlang.Choice(p, []langlang.ParserFn[langlang.Value]{\n")
	// g.parser.indent()

	// // Write the expression as the first option
	// g.writeExprFn(n.Expr)
	// g.parser.write(",\n")

	// // if the expression failed, throw an error
	// g.parser.writei("func(p langlang.Parser) (langlang.Value, error) {\n")
	// g.parser.indent()

	// g.parser.writei("if p.(*{{.ParserName}}).predicateLevel > 0 {\n")
	// g.parser.indent()
	// g.parser.writei("return nil, p.NewError")
	// fmt.Fprintf(g.parser.buffer, "(\"%s\", \"%s\", langlang.NewSpan(start, p.Location()))\n", n.Label, n.Label)

	// g.parser.unindent()
	// g.parser.writei("}\n")

	// g.parser.writeIndent()
	// fmt.Fprintf(g.parser.buffer, "if fn, ok := p.(*{{.ParserName}}).recoveryTable[\"%s\"]; ok {\n", n.Label)
	// g.parser.indent()
	// g.parser.writei("return fn(p)\n")
	// g.parser.unindent()
	// g.parser.writei("}\n")

	// g.parser.writei("return nil, p.Throw")
	// g.parser.write(fmt.Sprintf(`("%s", langlang.NewSpan(start, p.Location()))`, n.Label))
	// g.parser.write("\n")

	// g.parser.unindent()
	// g.parser.writei("},\n")

	// g.parser.unindent()
	// g.parser.writei("})\n")

	// g.parser.unindent()
	// g.parser.writei("}(p)\n")
	return nil
}

func (g *pyCodeEmitter) VisitIdentifierNode(n *IdentifierNode) error {
	g.parser.write(fmt.Sprintf("self.parse_%s()", n.Value))
	return nil
}

func (g *pyCodeEmitter) VisitImportNode(n *ImportNode) error {
	return fmt.Errorf("Not Implementated")
}

func (g *pyCodeEmitter) VisitLiteralNode(n *LiteralNode) error {
	s := `self.parse_literal("%s")`
	v := fmt.Sprintf(s, quoteSanitizer.Replace(n.Value))
	g.parser.write(v)
	return nil
}

func (g *pyCodeEmitter) VisitClassNode(n *ClassNode) error {
	switch len(n.Items) {
	case 0:
	case 1:
		return n.Items[0].Accept(g)
	default:
		g.parser.write("self.choice([\n")
		g.parser.indent()

		for _, expr := range n.Items {
			g.parser.writei("")
			g.writeExprFn(expr)
			g.parser.write(",\n")
		}

		g.parser.unindent()
		g.parser.writei("])")
	}
	return nil
}

func (g *pyCodeEmitter) VisitRangeNode(n *RangeNode) error {
	s := `self.parse_range("%s", "%s")`
	g.parser.write(fmt.Sprintf(s, n.Left, n.Right))
	return nil
}

func (g *pyCodeEmitter) VisitAnyNode(_ *AnyNode) error {
	g.parser.write("self.parse_any()")
	return nil
}

// Utilities to write data into the output buffer

func (g *pyCodeEmitter) writePrelude() {
	g.parser.writei(pyPrelude)
}

func (g *pyCodeEmitter) writeConstructor() {
	// g.parser.writei("\nfunc New{{.ParserName}}() *{{.ParserName}} {\n")
	// g.parser.indent()

	// g.parser.writei("p := &{{.ParserName}}{\n")
	// g.parser.indent()
	// g.parser.writei("captureSpaces: true,\n")
	// g.parser.writei("recoveryTable: map[string]langlang.ParserFn[langlang.Value]{},\n")
	// g.parser.unindent()
	// g.parser.writei("}\n")

	// for label := range g.labels {
	// 	if _, ok := g.productions[label]; ok {
	// 		g.parser.writei("p.recoveryTable[\"")
	// 		g.parser.write(label)

	// 		g.parser.write("\"] = func(p langlang.Parser) (langlang.Value, error) {\n")
	// 		g.parser.indent()

	// 		g.parser.writei("start := p.Location()\n")
	// 		g.parser.writei("item, err := p.(*{{.ParserName}})")
	// 		fmt.Fprintf(g.parser.buffer, ".Parse%s()\n", label)
	// 		g.writeIfErr()
	// 		g.parser.writei("return langlang.NewValueError")
	// 		fmt.Fprintf(g.parser.buffer, "(\"%s\", item, langlang.NewSpan(start, p.Location())), nil\n", label)

	// 		g.parser.unindent()
	// 		g.parser.writei("}\n")
	// 	}
	// }
	// g.parser.writei("return p\n")

	// g.parser.unindent()
	// g.parser.writei("}\n")
}

func (g *pyCodeEmitter) writeSeqOrNode() {
	// 	g.parser.writei("if len(items) == 0:\n")
	// 	g.parser.indent()
	// 	g.parser.writei("return None\n")
	// 	g.parser.unindent()

	// 	g.parser.writei("elif len(items) == 1:\n")
	// 	g.parser.indent()
	// 	g.parser.writei("return items[0]\n")
	// 	g.parser.unindent()

	// g.parser.writei("else:\n")
	// g.parser.indent()
	// g.parser.writei("return List(items, Span(start, p.location()))\n")
	// g.parser.unindent()
}

func (g *pyCodeEmitter) writeExprFn(expr AstNode) error {
	switch n := expr.(type) {
	case *IdentifierNode:
		// This avids wrapping a function call unnecessarily
		// in a lambda, when it could be passed as an argument
		g.parser.write(fmt.Sprintf("self.parse_%s", n.Value))
		return nil

	case *AnyNode:
		g.parser.write("self.parse_any")
		return nil
	}

	g.parser.write("(lambda: ")
	if err := expr.Accept(g); err != nil {
		return err
	}
	g.parser.write(")")

	return nil
}

func (g *pyCodeEmitter) writeIfErr() {
	g.parser.writei("if err != nil {\n")
	g.parser.indent()
	g.parser.writei("return nil, err\n")
	g.parser.unindent()
	g.parser.writei("}\n")
}

// other helpers

// isAtRuleLevel returns true exclusively if the traversal is exactly
// one indent within the `DefinitionNode` traversal.  That's useful to
// know because that's the only level in the generated parser that
// doesn't need type casting the variable `p` from `parsing.Parser`
// into the local concrete `Parser`.
func (g *pyCodeEmitter) isAtRuleLevel() bool {
	return g.parser.indentLevel == 1
}

// isUnderRuleLevel returns true when the traversal is any level
// within the `DefinitionNode`.  It's only in that level that we
// should be automatically handling spaces.
func (g *pyCodeEmitter) isUnderRuleLevel() bool {
	return g.parser.indentLevel >= 1
}

func (g *pyCodeEmitter) output() (string, error) {
	parserTmpl, err := template.New("parser").Parse(g.parser.buffer.String())
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	vv := tmplRenderOpts{
		GrammarPath: g.options.GrammarPath,
	}
	if err = parserTmpl.Execute(&output, vv); err != nil {
		return "", err
	}
	return output.String(), nil
}
