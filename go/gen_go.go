package langlang

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type goCodeEmitter struct {
	options     GenGoOptions
	buffer      *strings.Builder
	indentLevel int
	lexLevel    int
}

type GenGoOptions struct {
	PackageName   string
	StructSuffix  string
	CaptureSpaces bool
}

func DefaultGenGoOptions() GenGoOptions {
	return GenGoOptions{
		PackageName:   "parser",
		StructSuffix:  "",
	}
}

func newGoCodeEmitter(opt GenGoOptions) *goCodeEmitter {
	emitter := &goCodeEmitter{options: opt, buffer: &strings.Builder{}}
	emitter.write(`package {{.PackageName}}

import (
	"github.com/clarete/langlang/go"
)

type Parser{{.StructSuffix}} struct {
	langlang.BaseParser
	captureSpaces bool
}

func NewParser{{.StructSuffix}}(input string) *Parser{{.StructSuffix}} {
	p := &Parser{{.StructSuffix}}{
		captureSpaces: true,
	}
	p.SetInput([]rune(input))
	return p
}

func (p *Parser{{.StructSuffix}}) SetCaptureSpaces(v bool) {
	p.captureSpaces = v
}

func (p *Parser{{.StructSuffix}}) ParseAny() (langlang.Value, error) {
	start := p.Location()
	r, err := p.Any()
	if err != nil {
		var zero langlang.Value
		return zero, err
	}
	return langlang.NewValueString(string(r), langlang.NewSpan(start, p.Location())), nil
}

func (p *Parser{{.StructSuffix}}) ParseRange(left, right rune) (langlang.Value, error) {
	start := p.Location()
	r, err := p.ExpectRange(left, right)
	if err != nil {
		var zero langlang.Value
		return zero, err
	}
	return langlang.NewValueString(string(r), langlang.NewSpan(start, p.Location())), nil
}

func (p *Parser{{.StructSuffix}}) ParseLiteral(literal string) (langlang.Value, error) {
	start := p.Location()
	r, err := p.ExpectLiteral(literal)
	if err != nil {
		var zero langlang.Value
		return zero, err
	}
	return langlang.NewValueString(r, langlang.NewSpan(start, p.Location())), nil
}

func (p *Parser{{.StructSuffix}}) ParseSpacing() (langlang.Value, error) {
	start := p.Location()
	v, err := langlang.ZeroOrMore(p, func(p langlang.Parser) (rune, error) {
		return langlang.ChoiceRune(p, []rune{' ', '\t', '\r', '\n'})
	})
	if err != nil {
		return nil, err
	}
	if !p.captureSpaces {
		return nil, nil
	}
	r := string(v)
	if len(r) == 0 {
		return nil, nil
	}
	s := langlang.NewValueString(r, langlang.NewSpan(start, p.Location()))
	return langlang.NewValueNode("Spacing", s, langlang.NewSpan(start, p.Location())), nil
}

func (p *Parser{{.StructSuffix}}) ParseEOF() (langlang.Value, error) {
	return (func(p langlang.Parser) (langlang.Value, error) {
		var (
			start = p.Location()
			items []langlang.Value
			item  langlang.Value
			err   error
		)
		item, err = langlang.Not(p, func(p langlang.Parser) (langlang.Value, error) {
			return p.(*Parser{{.StructSuffix}}).ParseAny()
		})
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = append(items, item)
		}
		return p.(*Parser{{.StructSuffix}}).wrapSeq(items, langlang.NewSpan(start, p.Location())), nil
	}(p))
}

func (p *Parser{{.StructSuffix}}) wrapSeq(items []langlang.Value, span langlang.Span) langlang.Value {
	switch len(items) {
	case 0:
		return nil
	case 1:
		return items[0]
	default:
		return langlang.NewValueSequence(items, span)
	}
}

`)
	return emitter
}

func (g *goCodeEmitter) visit(node Node) {
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
		panic(fmt.Sprintf("Unknown Grammar AST node: %#v", n))
	}
}

func (g *goCodeEmitter) visitGrammarNode(n *GrammarNode) {
	for _, item := range n.Items {
		g.visit(item)
	}
}

func (g *goCodeEmitter) visitDefinitionNode(n *DefinitionNode) {
	g.writeIndent()
	g.write("\nfunc (p *Parser{{.StructSuffix}}) Parse")
	g.write(n.Name)
	g.write("() (langlang.Value, error) {\n")
	g.indent()

	g.writei("p.PushTraceSpan")
	fmt.Fprintf(g.buffer, `(langlang.TracerSpan{Name: "%s"})`, n.Name)
	g.write("\n")
	g.writei("defer p.PopTraceSpan()\n")

	g.writei("var (\n")
	g.indent()
	g.writei("start = p.Location()\n")
	g.writei("item  langlang.Value\n")
	g.writei("err   error\n")
	g.unindent()
	g.writei(")\n")

	g.writei("item, err = ")
	g.visit(n.Expr)
	g.write("\n")
	g.writeIfErr()
	g.writei("if item == nil {\n")
	g.indent()
	g.writei("return nil, nil")
	g.unindent()
	g.writei("}\n")

	g.writei("return langlang.NewValueNode")
	fmt.Fprintf(g.buffer, `("%s", item, langlang.NewSpan(start, p.Location())), nil`, n.Name)

	g.unindent()
	g.write("\n}\n")
}

func (g *goCodeEmitter) visitSequenceNode(n *SequenceNode) {
	shouldConsumeSpaces := g.lexLevel == 0 && g.isUnderRuleLevel() && !n.IsSyntactic()
	g.write("(func(p langlang.Parser) (langlang.Value, error) {\n")
	g.indent()

	g.writei("var (\n")
	g.indent()
	g.writei("start = p.Location()\n")
	g.writei("items []langlang.Value\n")
	g.writei("item  langlang.Value\n")
	g.writei("err   error\n")
	g.unindent()
	g.writei(")\n")

	for _, item := range n.Items {
		_, isLexNode := item.(*LexNode)
		if shouldConsumeSpaces && !isLexNode {
			g.writei("item, err = p.(*Parser{{.StructSuffix}}).ParseSpacing()\n")
			g.writeIfErr()
			g.writei("if item != nil {\n")
			g.indent()
			g.writei("items = append(items, item)\n")
			g.unindent()
			g.writei("}\n")
		}
		g.writei("item, err = ")
		g.visit(item)
		g.write("\n")
		g.writeIfErr()

		g.writei("if item != nil {\n")
		g.indent()
		g.writei("items = append(items, item)\n")
		g.unindent()
		g.writei("}\n")
	}

	g.writeSeqOrNode()

	g.unindent()
	g.writei("}(p))")
}

func (g *goCodeEmitter) visitOneOrMoreNode(n *OneOrMoreNode) {
	g.write("(func(p langlang.Parser) (langlang.Value, error) {\n")
	g.indent()

	g.writei("start := p.Location()\n")
	g.writei("items, err := langlang.OneOrMore(p, func(p langlang.Parser) (langlang.Value, error) {\n")
	g.indent()

	g.writei("return ")
	g.visit(n.Expr)
	g.write("\n")

	g.unindent()
	g.writei("})\n")
	g.writeIfErr()
	g.writeSeqOrNode()

	g.unindent()
	g.writei("}(p))")
}

func (g *goCodeEmitter) visitZeroOrMoreNode(n *ZeroOrMoreNode) {
	g.write("(func(p langlang.Parser) (langlang.Value, error) {\n")
	g.indent()

	g.writei("start := p.Location()\n")
	g.writei("items, err := langlang.ZeroOrMore(p, func(p langlang.Parser) (langlang.Value, error) {\n")
	g.indent()

	g.writei("return ")
	g.visit(n.Expr)
	g.write("\n")

	g.unindent()
	g.writei("})\n")
	g.writeIfErr()
	g.writeSeqOrNode()

	g.unindent()
	g.writei("}(p))")
}

func (g *goCodeEmitter) visitOptionalNode(n *OptionalNode) {
	g.write("langlang.Choice(p, []langlang.ParserFn[langlang.Value]{\n")
	g.indent()

	g.wirteExprFn(n.Expr)
	g.write(",\n")

	g.writei("func(p langlang.Parser) (langlang.Value, error) {\n")
	g.indent()
	g.writei("return nil, nil\n")
	g.unindent()
	g.writei("},\n")

	g.unindent()
	g.writei("})")
}

func (g *goCodeEmitter) visitChoiceNode(n *ChoiceNode) {
	switch len(n.Items) {
	case 0:
		return
	case 1:
		g.visit(n.Items[0])
	default:
		g.write("langlang.Choice(p, []langlang.ParserFn[langlang.Value]{\n")
		g.indent()

		for _, expr := range n.Items {
			g.wirteExprFn(expr)
			g.write(",\n")
		}

		g.unindent()
		g.writei("})")
	}
}

func (g *goCodeEmitter) visitAndNode(n *AndNode) {
	g.write("langlang.And(p, func(p langlang.Parser) (langlang.Value, error) {\n")
	g.indent()

	g.writei("return ")
	g.visit(n.Expr)
	g.write("\n")

	g.unindent()
	g.writei("})")
}

func (g *goCodeEmitter) visitNotNode(n *NotNode) {
	g.write("langlang.Not(p, func(p langlang.Parser) (langlang.Value, error) {\n")
	g.indent()

	g.writei("return ")
	g.visit(n.Expr)
	g.write("\n")

	g.unindent()
	g.writei("})")
}

func (g *goCodeEmitter) visitLexNode(n *LexNode) {
	g.lexLevel++
	g.visit(n.Expr)
	g.write("\n")
	g.lexLevel--
}

func (g *goCodeEmitter) visitLabeledNode(n *LabeledNode) {
	g.write("func(p langlang.Parser) (langlang.Value, error) {\n")
	g.indent()
	g.writei("start = p.Location()\n")

	g.writei("return langlang.Choice(p, []langlang.ParserFn[langlang.Value]{\n")
	g.indent()

	// Write the expression as the first option
	g.wirteExprFn(n.Expr)
	g.write(",\n")

	// if the expression failed, throw an error
	g.writei("func(p langlang.Parser) (langlang.Value, error) {\n")
	g.indent()
	g.writei("return nil, p.Throw")
	g.write(fmt.Sprintf(`("%s", langlang.NewSpan(start, p.Location()))`, n.Label))
	g.write("\n")

	g.unindent()
	g.writei("},\n")

	g.unindent()
	g.writei("})\n")

	g.unindent()
	g.writei("}(p)\n")
}

func (g *goCodeEmitter) visitIdentifierNode(n *IdentifierNode) {
	s := "p.(*Parser{{.StructSuffix}}).Parse%s()"
	if g.isAtRuleLevel() {
		s = "p.Parse%s()"
	}
	g.write(fmt.Sprintf(s, n.Value))
}

var quoteSanitizer = strings.NewReplacer(`"`, `\"`)

func (g *goCodeEmitter) visitLiteralNode(n *LiteralNode) {
	s := `p.(*Parser{{.StructSuffix}}).ParseLiteral("%s")`
	if g.isAtRuleLevel() {
		s = `p.ParseLiteral("%s")`
	}
	g.write(fmt.Sprintf(s, quoteSanitizer.Replace(n.Value)))
}

func (g *goCodeEmitter) visitClassNode(n *ClassNode) {
	switch len(n.Items) {
	case 0:
	case 1:
		g.visit(n.Items[0])
	default:
		g.write("langlang.Choice(p, []langlang.ParserFn[langlang.Value]{\n")
		g.indent()

		for _, expr := range n.Items {
			g.wirteExprFn(expr)
			g.write(",\n")
		}

		g.unindent()
		g.writei("})")
	}
}

func (g *goCodeEmitter) visitRangeNode(n *RangeNode) {
	s := "p.(*Parser{{.StructSuffix}}).ParseRange('%s', '%s')"
	if g.isAtRuleLevel() {
		s = "p.ParseRange('%s', '%s')"
	}
	g.write(fmt.Sprintf(s, n.Left, n.Right))
}

func (g *goCodeEmitter) visitAnyNode() {
	s := "p.(*Parser{{.StructSuffix}}).ParseAny()"
	if g.isAtRuleLevel() {
		s = "p.ParseAny()"
	}
	g.write(s)
}

// Utilities to write data into the output buffer

func (g *goCodeEmitter) writeSeqOrNode() {
	g.writei("switch len(items) {\n")
	g.writei("case 0: return nil, nil\n")
	g.writei("case 1: return items[0], nil\n")
	g.writei("default:\n")
	g.indent()
	g.writei("return langlang.NewValueSequence(items, langlang.NewSpan(start, p.Location())), nil\n")
	g.unindent()
	g.writei("}\n")
}

func (g *goCodeEmitter) wirteExprFn(expr Node) {
	g.writei("func(p langlang.Parser) (langlang.Value, error) {\n")
	g.indent()

	g.writei("return ")
	g.visit(expr)
	g.write("\n")

	g.unindent()
	g.writei("}")
}

func (g *goCodeEmitter) writeIfErr() {
	g.writei("if err != nil {\n")
	g.indent()
	g.writei("return nil, err\n")
	g.unindent()
	g.writei("}\n")
}

func (g *goCodeEmitter) writei(s string) {
	g.writeIndent()
	g.write(s)
}

func (g *goCodeEmitter) write(s string) {
	g.buffer.WriteString(strings.ReplaceAll(s, "{{.StructSuffix}}", g.options.StructSuffix))
}

func (g *goCodeEmitter) writeIndent() {
	for i := 0; i < g.indentLevel; i++ {
		g.buffer.WriteString("	")
	}
}

// Indentation related utilities

func (g *goCodeEmitter) indent() {
	g.indentLevel++
}

func (g *goCodeEmitter) unindent() {
	g.indentLevel--
}

// other helpers

// isInRuleLevel returns true exclusively if the traversal is exactly
// one indent within the `DefinitionNode` traversal.  That's useful to
// know because that's the only level in the generated parser that
// doesn't need type casting the variable `p` from `parsing.Parser`
// into the local concrete `Parser`.
func (g *goCodeEmitter) isAtRuleLevel() bool {
	return g.indentLevel == 1
}

// isUnderRuleLevel returns true when the traversal is any level
// within the `DefinitionNode`.  It's only in that level that we
// should be automatically handling spaces.
func (g *goCodeEmitter) isUnderRuleLevel() bool {
	return g.indentLevel >= 1
}

func (g *goCodeEmitter) output() (string, error) {
	tmpl, err := template.New("gen_go").Parse(g.buffer.String())
	if err != nil {
		return "", err
	}

	var output bytes.Buffer
	err = tmpl.Execute(&output, g.options)
	return output.String(), nil
}

func GenGo(node Node, opt GenGoOptions) (string, error) {
	g := newGoCodeEmitter(opt)
	g.visit(node)
	return g.output()
}
