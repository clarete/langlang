package langlang

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
)

type goCodeEmitter struct {
	options     GenGoOptions
	parser      *outputWriter
	indentLevel int
	lexLevel    int
	labels      map[string]struct{}
	productions map[string]struct{}
}

type GenGoOptions struct {
	PackageName string
	Prefix      string
	ParserBase  string
}

func GenGo(node AstNode, opt GenGoOptions) (string, error) {
	g := newGoCodeEmitter(opt)
	g.writePrelude()
	g.visit(node)
	g.writeConstructor()
	return g.output()
}

type tmplRenderOpts struct {
	PackageName string
	ParserName  string
}

//go:embed prelude.go.in
var goPrelude string

func newGoCodeEmitter(opt GenGoOptions) *goCodeEmitter {
	return &goCodeEmitter{
		options:     opt,
		parser:      newOutputWriter(),
		labels:      map[string]struct{}{},
		productions: map[string]struct{}{},
	}
}

func (g *goCodeEmitter) visit(node AstNode) {
	switch n := node.(type) {
	case *GrammarNode:
		g.visitGrammarNode(n)
	case *DefinitionNode:
		g.visitDefinitionNode(n)
	case *SequenceNode:
		g.visitSequenceNode(n)
	case *OneOrMoreNode:
		g.visitOneOrMoreNode(n)
	case *ZeroOrMoreNode:
		g.visitZeroOrMoreNode(n)
	case *OptionalNode:
		g.visitOptionalNode(n)
	case *ChoiceNode:
		g.visitChoiceNode(n)
	case *AndNode:
		g.visitAndNode(n)
	case *NotNode:
		g.visitNotNode(n)
	case *LexNode:
		g.visitLexNode(n)
	case *LabeledNode:
		g.visitLabeledNode(n)
	case *IdentifierNode:
		g.visitIdentifierNode(n)
	case *LiteralNode:
		g.visitLiteralNode(n)
	case *ClassNode:
		g.visitClassNode(n)
	case *RangeNode:
		g.visitRangeNode(n)
	case *AnyNode:
		g.visitAnyNode()
	default:
		panic(fmt.Sprintf("Unknown Grammar AST node: %s", n))
	}
}

func (g *goCodeEmitter) visitGrammarNode(n *GrammarNode) {
	for _, item := range n.GetItems() {
		g.visit(item)
	}
}

func (g *goCodeEmitter) visitDefinitionNode(n *DefinitionNode) {
	g.productions[n.Name] = struct{}{}

	g.parser.write("\nfunc (p *{{.ParserName}}) Parse")
	g.parser.write(n.Name)
	g.parser.write("() (langlang.Value, error) {\n")
	g.parser.indent()

	g.parser.writei("p.PushTraceSpan")
	fmt.Fprintf(g.parser.buffer, `(langlang.TracerSpan{Name: "%s"})`, n.Name)
	g.parser.write("\n")
	g.parser.writei("defer p.PopTraceSpan()\n")
	g.parser.writei("if p.printTraceback {\n")
	g.parser.indent()
	g.parser.writei("fmt.Printf(\"%s; %s\\n\", p.Location(), p.PrintStackTrace())\n")
	g.parser.unindent()
	g.parser.writei("}\n")

	g.parser.writei("var (\n")
	g.parser.indent()
	g.parser.writei("start = p.Location()\n")
	g.parser.writei("item  langlang.Value\n")
	g.parser.writei("err   error\n")
	g.parser.unindent()
	g.parser.writei(")\n")

	g.parser.writei("item, err = ")
	g.visit(n.Expr)
	g.parser.write("\n")
	g.writeIfErr()
	g.parser.writei("if item == nil {\n")
	g.parser.indent()
	g.parser.writei("return nil, nil\n")
	g.parser.unindent()
	g.parser.writei("}\n")

	g.parser.writei("return p.RunAction(\n")
	g.parser.indent()

	g.parser.writei(fmt.Sprintf("\"%s\",\n", n.Name))
	g.parser.writei(fmt.Sprintf("langlang.NewNode(\"%s\", item, langlang.NewSpan(start, p.Location())),\n", n.Name))

	g.parser.unindent()
	g.parser.writei(")\n")

	g.parser.unindent()
	g.parser.write("\n}\n")
}

func (g *goCodeEmitter) visitSequenceNode(n *SequenceNode) {
	shouldConsumeSpaces := g.lexLevel == 0 && g.isUnderRuleLevel() && !n.IsSyntactic()
	g.parser.write("(func(p langlang.Parser) (langlang.Value, error) {\n")
	g.parser.indent()

	g.parser.writei("var (\n")
	g.parser.indent()
	g.parser.writei("start = p.Location()\n")
	g.parser.writei("items []langlang.Value\n")

	if len(n.Items) > 0 {
		g.parser.writei("item  langlang.Value\n")
		g.parser.writei("err   error\n")
	}

	g.parser.unindent()
	g.parser.writei(")\n")

	for _, item := range n.Items {
		_, isLexNode := item.(*LexNode)
		if shouldConsumeSpaces && !isLexNode {
			g.parser.writei("item, err = p.(*{{.ParserName}}).parseSpacing()\n")
			g.writeIfErr()
			g.parser.writei("if item != nil {\n")
			g.parser.indent()
			g.parser.writei("items = append(items, item)\n")
			g.parser.unindent()
			g.parser.writei("}\n")
		}
		g.parser.writei("item, err = ")
		g.visit(item)
		g.parser.write("\n")
		g.writeIfErr()

		g.parser.writei("if item != nil {\n")
		g.parser.indent()
		g.parser.writei("items = append(items, item)\n")
		g.parser.unindent()
		g.parser.writei("}\n")
	}

	g.writeSeqOrNode()

	g.parser.unindent()
	g.parser.writei("}(p))")
}

func (g *goCodeEmitter) visitOneOrMoreNode(n *OneOrMoreNode) {
	g.parser.write("(func(p langlang.Parser) (langlang.Value, error) {\n")
	g.parser.indent()

	g.parser.writei("start := p.Location()\n")
	g.parser.writei("items, err := langlang.OneOrMore(p, func(p langlang.Parser) (langlang.Value, error) {\n")
	g.parser.indent()

	g.parser.writei("return ")
	g.visit(n.Expr)
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("})\n")
	g.writeIfErr()
	g.writeSeqOrNode()

	g.parser.unindent()
	g.parser.writei("}(p))")
}

func (g *goCodeEmitter) visitZeroOrMoreNode(n *ZeroOrMoreNode) {
	g.parser.write("(func(p langlang.Parser) (langlang.Value, error) {\n")
	g.parser.indent()

	g.parser.writei("start := p.Location()\n")
	g.parser.writei("items, err := langlang.ZeroOrMore(p, func(p langlang.Parser) (langlang.Value, error) {\n")
	g.parser.indent()

	g.parser.writei("return ")
	g.visit(n.Expr)
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("})\n")
	g.writeIfErr()
	g.writeSeqOrNode()

	g.parser.unindent()
	g.parser.writei("}(p))")
}

func (g *goCodeEmitter) visitOptionalNode(n *OptionalNode) {
	g.parser.write("langlang.Choice(p, []langlang.ParserFn[langlang.Value]{\n")
	g.parser.indent()

	g.writeExprFn(n.Expr)
	g.parser.write(",\n")

	g.parser.writei("func(p langlang.Parser) (langlang.Value, error) {\n")
	g.parser.indent()
	g.parser.writei("return nil, nil\n")
	g.parser.unindent()
	g.parser.writei("},\n")

	g.parser.unindent()
	g.parser.writei("})")
}

func (g *goCodeEmitter) visitChoiceNode(n *ChoiceNode) {
	switch len(n.Items) {
	case 0:
		return
	case 1:
		g.visit(n.Items[0])
	default:
		g.parser.write("langlang.Choice(p, []langlang.ParserFn[langlang.Value]{\n")
		g.parser.indent()

		for _, expr := range n.Items {
			g.writeExprFn(expr)
			g.parser.write(",\n")
		}

		g.parser.unindent()
		g.parser.writei("})")
	}
}

func (g *goCodeEmitter) visitAndNode(n *AndNode) {
	g.parser.write("langlang.And(p, func(p langlang.Parser) (langlang.Value, error) {\n")
	g.parser.indent()

	g.parser.writei("p.(*{{.ParserName}}).predicateLevel++\n")
	g.parser.writei("defer func() { p.(*{{.ParserName}}).predicateLevel-- }()\n")

	g.parser.writei("return ")
	g.visit(n.Expr)
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("})")
}

func (g *goCodeEmitter) visitNotNode(n *NotNode) {
	g.parser.write("langlang.Not(p, func(p langlang.Parser) (langlang.Value, error) {\n")
	g.parser.indent()

	g.parser.writei("p.(*{{.ParserName}}).predicateLevel++\n")
	g.parser.writei("defer func() { p.(*{{.ParserName}}).predicateLevel-- }()\n")

	g.parser.writei("return ")
	g.visit(n.Expr)
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("})")
}

func (g *goCodeEmitter) visitLexNode(n *LexNode) {
	g.lexLevel++
	g.visit(n.Expr)
	g.parser.write("\n")
	g.lexLevel--
}

func (g *goCodeEmitter) visitLabeledNode(n *LabeledNode) {
	g.labels[n.Label] = struct{}{}

	g.parser.write("func(p langlang.Parser) (langlang.Value, error) {\n")
	g.parser.indent()
	g.parser.writei("start := p.Location()\n")

	g.parser.writei("return langlang.Choice(p, []langlang.ParserFn[langlang.Value]{\n")
	g.parser.indent()

	// Write the expression as the first option
	g.writeExprFn(n.Expr)
	g.parser.write(",\n")

	// if the expression failed, throw an error
	g.parser.writei("func(p langlang.Parser) (langlang.Value, error) {\n")
	g.parser.indent()

	g.parser.writei("if p.(*{{.ParserName}}).predicateLevel > 0 {\n")
	g.parser.indent()
	g.parser.writei("return nil, p.NewError")
	fmt.Fprintf(g.parser.buffer, "(\"%s\", \"%s\", langlang.NewSpan(start, p.Location()))\n", n.Label, n.Label)

	g.parser.unindent()
	g.parser.writei("}\n")

	g.parser.writeIndent()
	fmt.Fprintf(g.parser.buffer, "if fn, ok := p.(*{{.ParserName}}).recoveryTable[\"%s\"]; ok {\n", n.Label)
	g.parser.indent()
	g.parser.writei("return fn(p)\n")
	g.parser.unindent()
	g.parser.writei("}\n")

	g.parser.writei("return nil, p.Throw")
	g.parser.write(fmt.Sprintf(`("%s", langlang.NewSpan(start, p.Location()))`, n.Label))
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("},\n")

	g.parser.unindent()
	g.parser.writei("})\n")

	g.parser.unindent()
	g.parser.writei("}(p)\n")
}

func (g *goCodeEmitter) visitIdentifierNode(n *IdentifierNode) {
	s := "p.(*{{.ParserName}}).Parse%s()"
	if g.isAtRuleLevel() {
		s = "p.Parse%s()"
	}
	g.parser.write(fmt.Sprintf(s, n.Value))
}

var quoteSanitizer = strings.NewReplacer(`"`, `\"`)

func (g *goCodeEmitter) visitLiteralNode(n *LiteralNode) {
	s := `p.(*{{.ParserName}}).parseLiteral("%s")`
	if g.isAtRuleLevel() {
		s = `p.parseLiteral("%s")`
	}
	g.parser.write(fmt.Sprintf(s, quoteSanitizer.Replace(n.Value)))
}

func (g *goCodeEmitter) visitClassNode(n *ClassNode) {
	switch len(n.Items) {
	case 0:
	case 1:
		g.visit(n.Items[0])
	default:
		g.parser.write("langlang.Choice(p, []langlang.ParserFn[langlang.Value]{\n")
		g.parser.indent()

		for _, expr := range n.Items {
			g.writeExprFn(expr)
			g.parser.write(",\n")
		}

		g.parser.unindent()
		g.parser.writei("})")
	}
}

func (g *goCodeEmitter) visitRangeNode(n *RangeNode) {
	s := "p.(*{{.ParserName}}).parseRange('%s', '%s')"
	if g.isAtRuleLevel() {
		s = "p.parseRange('%s', '%s')"
	}
	g.parser.write(fmt.Sprintf(s, n.Left, n.Right))
}

func (g *goCodeEmitter) visitAnyNode() {
	s := "p.(*{{.ParserName}}).parseAny()"
	if g.isAtRuleLevel() {
		s = "p.parseAny()"
	}
	g.parser.write(s)
}

// Utilities to write data into the output buffer

func (g *goCodeEmitter) writePrelude() {
	g.parser.write(goPrelude)
}

func (g *goCodeEmitter) writeConstructor() {
	g.parser.writei("\nfunc New{{.ParserName}}() *{{.ParserName}} {\n")
	g.parser.indent()

	g.parser.writei("p := &{{.ParserName}}{\n")
	g.parser.indent()
	g.parser.writei("captureSpaces: true,\n")
	g.parser.writei("recoveryTable: map[string]langlang.ParserFn[langlang.Value]{},\n")
	g.parser.unindent()
	g.parser.writei("}\n")

	for label := range g.labels {
		if _, ok := g.productions[label]; ok {
			g.parser.writei("p.recoveryTable[\"")
			g.parser.write(label)

			g.parser.write("\"] = func(p langlang.Parser) (langlang.Value, error) {\n")
			g.parser.indent()

			g.parser.writei("start := p.Location()\n")
			g.parser.writei("item, err := p.(*{{.ParserName}})")
			fmt.Fprintf(g.parser.buffer, ".Parse%s()\n", label)
			g.writeIfErr()
			g.parser.writei("return langlang.NewError")
			fmt.Fprintf(g.parser.buffer, "(\"%s\", item, langlang.NewSpan(start, p.Location())), nil\n", label)

			g.parser.unindent()
			g.parser.writei("}\n")
		}
	}
	g.parser.writei("return p\n")

	g.parser.unindent()
	g.parser.writei("}\n")
}

func (g *goCodeEmitter) writeSeqOrNode() {
	g.parser.writei("switch len(items) {\n")
	g.parser.writei("case 0:\n")
	g.parser.indent()
	g.parser.writei("return nil, nil\n")
	g.parser.unindent()

	g.parser.writei("case 1:\n")
	g.parser.indent()
	g.parser.writei("return items[0], nil\n")
	g.parser.unindent()

	g.parser.writei("default:\n")
	g.parser.indent()
	g.parser.writei("return langlang.NewSequence(items, langlang.NewSpan(start, p.Location())), nil\n")
	g.parser.unindent()
	g.parser.writei("}\n")
}

func (g *goCodeEmitter) writeExprFn(expr AstNode) {
	g.parser.writei("func(p langlang.Parser) (langlang.Value, error) {\n")
	g.parser.indent()

	g.parser.writei("return ")
	g.visit(expr)
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("}")
}

func (g *goCodeEmitter) writeIfErr() {
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
func (g *goCodeEmitter) isAtRuleLevel() bool {
	return g.parser.indentLevel == 1
}

// isUnderRuleLevel returns true when the traversal is any level
// within the `DefinitionNode`.  It's only in that level that we
// should be automatically handling spaces.
func (g *goCodeEmitter) isUnderRuleLevel() bool {
	return g.parser.indentLevel >= 1
}

func (g *goCodeEmitter) output() (string, error) {
	parserTmpl, err := template.New("parser").Parse(g.parser.buffer.String())
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	vv := tmplRenderOpts{
		PackageName: g.options.PackageName,
		ParserName:  g.options.Prefix + g.options.ParserBase,
	}
	if err = parserTmpl.Execute(&output, vv); err != nil {
		return "", err
	}
	return output.String(), nil
}

// IO helper

type outputWriter struct {
	buffer      *strings.Builder
	indentLevel int
}

func newOutputWriter() *outputWriter {
	return &outputWriter{buffer: &strings.Builder{}}
}

func (o *outputWriter) indent() {
	o.indentLevel++
}

func (o *outputWriter) unindent() {
	o.indentLevel--
}

func (o *outputWriter) writeIndent() {
	for i := 0; i < o.indentLevel; i++ {
		o.buffer.WriteString("	")
	}
}

func (o *outputWriter) writei(s string) {
	o.writeIndent()
	o.write(s)
}

func (o *outputWriter) write(s string) {
	o.buffer.WriteString(s)
}
