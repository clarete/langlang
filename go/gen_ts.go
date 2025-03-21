package langlang

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

type tsCodeEmitter struct {
	options     GenTsOptions
	parser      *outputWriter
	indentLevel int
	lexLevel    int
	labelsMap   map[string]struct{}
	labels      []string
	grammarNode *GrammarNode
}

type GenTsOptions struct {
	GrammarPath string
	RemoveLib   bool
}

func GenTs(node AstNode, opt GenTsOptions) (string, error) {
	g := newTsCodeEmitter(opt)
	g.writePrelude()
	g.parser.indent()

	if err := node.Accept(g); err != nil {
		return "", err
	}

	g.writeConstructor()
	g.parser.unindent()
	g.parser.write("}\n")
	return g.output()
}

//go:embed prelude.ts.in
var tsPrelude string

func newTsCodeEmitter(opt GenTsOptions) *tsCodeEmitter {
	return &tsCodeEmitter{
		options:   opt,
		parser:    newOutputWriter("  "),
		labelsMap: map[string]struct{}{},
		labels:    []string{},
	}
}

func (g *tsCodeEmitter) VisitImportNode(n *ImportNode) error {
	return fmt.Errorf("unreachable")
}

func (g *tsCodeEmitter) VisitGrammarNode(n *GrammarNode) error {
	g.grammarNode = n
	return WalkGrammarNode(g, n)
}

func (g *tsCodeEmitter) VisitDefinitionNode(n *DefinitionNode) error {
	g.parser.write("\n")
	g.parser.writeil(fmt.Sprintf("parse%s(): Value {", n.Name))
	g.parser.indent()

	g.parser.writeil("const start = this.location()")
	g.parser.writeil(fmt.Sprintf("const key = `%s_${start.cursor}`", n.Name))
	g.parser.writei(fmt.Sprintf(`return this.mknode("%s", start, `, n.Name))

	if err := n.Expr.Accept(g); err != nil {
		return err
	}

	g.parser.writel(")")
	g.parser.unindent()
	g.parser.writeil("}")
	return nil
}

func (g *tsCodeEmitter) VisitSequenceNode(n *SequenceNode) error {
	shouldConsumeSpaces := g.lexLevel == 0 && g.isUnderRuleLevel() && !n.IsSyntactic()

	if len(n.Items) == 1 {
		return n.Items[0].Accept(g)
	}

	g.parser.writel("((): Value => {")
	g.parser.indent()

	g.parser.writeil("const start = this.location();")
	g.parser.writeil("return wrapSeq([")
	g.parser.indent()

	for i, item := range n.Items {
		_, isLexNode := item.(*LexNode)
		if shouldConsumeSpaces && !isLexNode {
			g.parser.writeil("this.parseSpacing(),")
		}

		g.parser.writei("")
		if err := item.Accept(g); err != nil {
			return err
		}

		if i < len(n.Items)-1 {
			g.parser.writel(",")
		} else {
			g.parser.writel("")
		}
	}

	g.parser.unindent()
	g.parser.writeil("], this.span(start))")
	g.parser.unindent()
	g.parser.writei("})()")

	return nil
}

func (g *tsCodeEmitter) VisitOneOrMoreNode(n *OneOrMoreNode) error {
	g.parser.writel("(() => {")
	g.parser.indent()
	g.parser.writeil("const start = this.location();")
	g.parser.writei("return wrapSeq(this.oneOrMore(")
	g.writeExprFn(n.Expr)
	g.parser.writel("), new Span(start, this.location()))")
	g.parser.unindent()
	g.parser.writei("})()")
	return nil
}

func (g *tsCodeEmitter) VisitZeroOrMoreNode(n *ZeroOrMoreNode) error {
	g.parser.writel("(() => {")
	g.parser.indent()
	g.parser.writeil("const start = this.location();")
	g.parser.writei("return wrapSeq(this.zeroOrMore(")
	g.writeExprFn(n.Expr)
	g.parser.writel("), new Span(start, this.location()))")
	g.parser.unindent()
	g.parser.writei("})()")
	return nil
}

func (g *tsCodeEmitter) VisitOptionalNode(n *OptionalNode) error {
	g.parser.write("this.optional(")
	g.writeExprFn(n.Expr)
	g.parser.write(")")
	return nil
}

func (g *tsCodeEmitter) VisitChoiceNode(n *ChoiceNode) error {
	switch len(n.Items) {
	case 0:
		return nil
	case 1:
		return n.Items[0].Accept(g)
	default:
		g.parser.write("this.choice([\n")
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

func (g *tsCodeEmitter) VisitAndNode(n *AndNode) error {
	g.parser.write("this.and(")
	g.writeExprFn(n.Expr)
	g.parser.write(")")
	return nil
}

func (g *tsCodeEmitter) VisitNotNode(n *NotNode) error {
	g.parser.write("this.not(")
	g.writeExprFn(n.Expr)
	g.parser.write(") as unknown as Value")
	return nil
}

func (g *tsCodeEmitter) VisitLexNode(n *LexNode) error {
	g.lexLevel++
	if err := n.Expr.Accept(g); err != nil {
		return err
	}
	g.lexLevel--
	return nil
}

func (g *tsCodeEmitter) VisitLabeledNode(n *LabeledNode) error {
	g.labelsMap[n.Label] = struct{}{}

	if err := n.Expr.Accept(g); err != nil {
		return err
	}

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

func (g *tsCodeEmitter) VisitIdentifierNode(n *IdentifierNode) error {
	g.parser.write(fmt.Sprintf("this.parse%s()", n.Value))
	return nil
}

func (g *tsCodeEmitter) VisitLiteralNode(n *LiteralNode) error {
	s := `this.parseLiteral("%s")`
	v := fmt.Sprintf(s, quoteSanitizer.Replace(n.Value))
	g.parser.write(v)
	return nil
}

func (g *tsCodeEmitter) VisitClassNode(n *ClassNode) error {
	switch len(n.Items) {
	case 0:
	case 1:
		return n.Items[0].Accept(g)
	default:
		g.parser.write("this.choice([\n")
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

func (g *tsCodeEmitter) VisitRangeNode(n *RangeNode) error {
	s := `this.parseRange("%s", "%s")`
	g.parser.write(fmt.Sprintf(s, n.Left, n.Right))
	return nil
}

func (g *tsCodeEmitter) VisitAnyNode(_ *AnyNode) error {
	g.parser.write("this.parseAny()")
	return nil
}

// Utilities to write data into the output buffer

func (g *tsCodeEmitter) writePrelude() {
	g.parser.writei(tsPrelude)
}

func (g *tsCodeEmitter) writeConstructor() {
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

func (g *tsCodeEmitter) writeSeqOrNode() {
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

func (g *tsCodeEmitter) writeExprFn(expr AstNode) error {
	switch n := expr.(type) {
	case *IdentifierNode:
		// This avids wrapping a function call unnecessarily
		// in a lambda, when it could be passed as an argument
		g.parser.write(fmt.Sprintf("() => this.parse%s()", n.Value))
		return nil

	case *AnyNode:
		g.parser.write("this.parseAny")
		return nil
	}

	g.parser.write("() => ")
	if err := expr.Accept(g); err != nil {
		return err
	}

	return nil
}

func (g *tsCodeEmitter) writeIfErr() {
	g.parser.writei("if (err !== null) {\n")
	g.parser.indent()
	g.parser.writei("return [null, err]\n")
	g.parser.unindent()
	g.parser.writei("}\n")
}

// other helpers

// isAtRuleLevel returns true exclusively if the traversal is exactly
// one indent within the `DefinitionNode` traversal.  That's useful to
// know because that's the only level in the generated parser that
// doesn't need type casting the variable `p` from `parsing.Parser`
// into the local concrete `Parser`.
func (g *tsCodeEmitter) isAtRuleLevel() bool {
	return g.parser.indentLevel == 1
}

// isUnderRuleLevel returns true when the traversal is any level
// within the `DefinitionNode`.  It's only in that level that we
// should be automatically handling spaces.
func (g *tsCodeEmitter) isUnderRuleLevel() bool {
	return g.parser.indentLevel >= 1
}

func (g *tsCodeEmitter) output() (string, error) {
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
