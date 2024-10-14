package langlang

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"

	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
)

type goCodeEmitter struct {
	options     GenGoOptions
	parser      *outputWriter
	indentLevel int
	lexLevel    int
	labels      map[string]struct{}
	grammarNode *GrammarNode
}

type GenGoOptions struct {
	PackageName string
	RemoveLib   bool
}

func GenGo(node AstNode, opt GenGoOptions) (string, error) {
	g := newGoCodeEmitter(opt)
	g.writePrelude()
	g.visit(node)
	g.writeConstructor()
	g.writeEmbeds()
	return g.output()
}

type tmplRenderOpts struct {
	PackageName string
}

//go:embed parser.go value.go errors.go
var content embed.FS

func newGoCodeEmitter(opt GenGoOptions) *goCodeEmitter {
	return &goCodeEmitter{
		options: opt,
		parser:  newOutputWriter(),
		labels:  map[string]struct{}{},
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
	g.grammarNode = n
	for _, item := range n.GetItems() {
		g.visit(item)
	}
}

func (g *goCodeEmitter) visitDefinitionNode(n *DefinitionNode) {
	g.parser.write("\nfunc (p *Parser) Parse")
	g.parser.write(n.Name)
	g.parser.write("() (Value, error) {\n")
	g.parser.indent()

	g.parser.writei("p.PushTraceSpan")
	fmt.Fprintf(g.parser.buffer, `(TracerSpan{Name: "%s"})`, n.Name)
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
	g.parser.writei("item  Value\n")
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
	g.parser.writei(fmt.Sprintf("NewNode(\"%s\", item, NewSpan(start, p.Location())),\n", n.Name))

	g.parser.unindent()
	g.parser.writei(")\n")

	g.parser.unindent()
	g.parser.write("\n}\n")
}

func (g *goCodeEmitter) visitSequenceNode(n *SequenceNode) {
	shouldConsumeSpaces := g.lexLevel == 0 && g.isUnderRuleLevel() && !n.IsSyntactic()
	g.parser.write("(func(p Backtrackable) (Value, error) {\n")
	g.parser.indent()

	g.parser.writei("var (\n")
	g.parser.indent()
	g.parser.writei("start = p.Location()\n")
	g.parser.writei("items []Value\n")

	if len(n.Items) > 0 {
		g.parser.writei("item  Value\n")
		g.parser.writei("err   error\n")
	}

	g.parser.unindent()
	g.parser.writei(")\n")

	for _, item := range n.Items {
		_, isLexNode := item.(*LexNode)
		if shouldConsumeSpaces && !isLexNode {
			if _, ok := g.grammarNode.DefsByName["Spacing"]; ok {
				g.parser.writei("item, err = p.(*Parser).ParseSpacing()\n")
			} else {
				g.parser.writei("item, err = p.(*Parser).parseSpacing()\n")
			}
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
	g.parser.write("(func(p Backtrackable) (Value, error) {\n")
	g.parser.indent()

	g.parser.writei("start := p.Location()\n")
	g.parser.writei("items, err := OneOrMore(p, func(p Backtrackable) (Value, error) {\n")
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
	g.parser.write("(func(p Backtrackable) (Value, error) {\n")
	g.parser.indent()

	g.parser.writei("start := p.Location()\n")
	g.parser.writei("items, err := ZeroOrMore(p, func(p Backtrackable) (Value, error) {\n")
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
	g.parser.write("Choice(p, []ParserFn[Value]{\n")
	g.parser.indent()

	g.writeExprFn(n.Expr)
	g.parser.write(",\n")

	g.parser.writei("func(p Backtrackable) (Value, error) {\n")
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
		g.parser.write("Choice(p, []ParserFn[Value]{\n")
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
	g.parser.write("And(p, func(p Backtrackable) (Value, error) {\n")
	g.parser.indent()

	g.parser.writei("p.EnterPredicate()\n")
	g.parser.writei("defer func() { p.LeavePredicate() }()\n")

	g.parser.writei("return ")
	g.visit(n.Expr)
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("})")
}

func (g *goCodeEmitter) visitNotNode(n *NotNode) {
	g.parser.write("Not(p, func(p Backtrackable) (Value, error) {\n")
	g.parser.indent()

	g.parser.writei("p.EnterPredicate()\n")
	g.parser.writei("defer func() { p.LeavePredicate() }()\n")

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

	g.parser.write("func(p Backtrackable) (Value, error) {\n")
	g.parser.indent()
	g.parser.writei("start := p.Location()\n")

	g.parser.writei("return Choice(p, []ParserFn[Value]{\n")
	g.parser.indent()

	// Write the expression as the first option
	g.writeExprFn(n.Expr)
	g.parser.write(",\n")

	// if the expression failed, throw an error
	g.parser.writei("func(p Backtrackable) (Value, error) {\n")
	g.parser.indent()

	g.parser.writei("if p.WithinPredicate() {\n")
	g.parser.indent()
	g.parser.writei("return nil, p.NewError")
	fmt.Fprintf(g.parser.buffer, "(\"%s\", \"%s\", NewSpan(start, p.Location()))\n", n.Label, n.Label)

	g.parser.unindent()
	g.parser.writei("}\n")

	g.parser.writeIndent()
	fmt.Fprintf(g.parser.buffer, "if fn, ok := p.(*Parser).recoveryTable[\"%s\"]; ok {\n", n.Label)
	g.parser.indent()
	g.parser.writei("return fn(p)\n")
	g.parser.unindent()
	g.parser.writei("}\n")

	g.parser.writei("return nil, p.Throw")
	g.parser.write(fmt.Sprintf(`("%s", NewSpan(start, p.Location()))`, n.Label))
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("},\n")

	g.parser.unindent()
	g.parser.writei("})\n")

	g.parser.unindent()
	g.parser.writei("}(p)\n")
}

func (g *goCodeEmitter) visitIdentifierNode(n *IdentifierNode) {
	s := "p.(*Parser).Parse%s()"
	if g.isAtRuleLevel() {
		s = "p.Parse%s()"
	}
	g.parser.write(fmt.Sprintf(s, n.Value))
}

var quoteSanitizer = strings.NewReplacer(`"`, `\"`)

func (g *goCodeEmitter) visitLiteralNode(n *LiteralNode) {
	s := `p.(*Parser).parseLiteral("%s")`
	if g.isAtRuleLevel() {
		s = "p.Parse%s()"
	}
	g.parser.write(fmt.Sprintf(s, quoteSanitizer.Replace(n.Value)))
}

func (g *goCodeEmitter) visitClassNode(n *ClassNode) {
	switch len(n.Items) {
	case 0:
	case 1:
		g.visit(n.Items[0])
	default:
		g.parser.write("Choice(p, []ParserFn[Value]{\n")
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
	s := "p.(*Parser).parseRange('%s', '%s')"
	if g.isAtRuleLevel() {
		s = "p.parseRange('%s', '%s')"
	}
	g.parser.write(fmt.Sprintf(s, n.Left, n.Right))
}

func (g *goCodeEmitter) visitAnyNode() {
	s := "p.(*Parser).parseAny()"
	if g.isAtRuleLevel() {
		s = "p.parseAny()"
	}
	g.parser.write(s)
}

// Utilities to write data into the output buffer

func (g *goCodeEmitter) writePrelude() {
	g.parser.write("package ")
	g.parser.write(g.options.PackageName)
	g.parser.write("\n\n")

	g.parser.write("import (\n")
	g.parser.indent()
	g.parser.writei(`"fmt"`)
	g.parser.write("\n")

	if !g.options.RemoveLib {
		g.parser.writei(`"strconv"`)
		g.parser.write("\n")
		g.parser.writei(`"strings"`)
		g.parser.write("\n")
	}

	g.parser.unindent()
	g.parser.write(")\n\n")

	if g.options.RemoveLib {
		return
	}

	s, err := cleanGoModule("parser.go")
	if err != nil {
		panic(err.Error())
	}
	g.parser.write(s)
}

func (g *goCodeEmitter) writeConstructor() {
	g.parser.writei("\nfunc NewParser() *Parser {\n")
	g.parser.indent()

	g.parser.writei("p := &Parser{\n")
	g.parser.indent()
	g.parser.writei("captureSpaces: true,\n")
	g.parser.writei("recoveryTable: map[string]ParserFn[Value]{},\n")
	g.parser.unindent()
	g.parser.writei("}\n")

	for label := range g.labels {
		if _, ok := g.grammarNode.DefsByName[label]; ok {
			g.parser.writei("p.recoveryTable[\"")
			g.parser.write(label)

			g.parser.write("\"] = func(p Backtrackable) (Value, error) {\n")
			g.parser.indent()

			g.parser.writei("start := p.Location()\n")
			g.parser.writei("item, err := p.(*Parser).Parse")
			g.parser.write(label)
			g.parser.write("()\n")
			g.writeIfErr()
			g.parser.writei("return NewError")
			fmt.Fprintf(g.parser.buffer, "(\"%s\", item, NewSpan(start, p.Location())), nil\n", label)

			g.parser.unindent()
			g.parser.writei("}\n")
		}
	}
	g.parser.writei("return p\n")

	g.parser.unindent()
	g.parser.writei("}\n")
}

func (g *goCodeEmitter) writeEmbeds() {
	if g.options.RemoveLib {
		return
	}

	value, err := cleanGoModule("value.go")
	if err != nil {
		panic(err.Error())
	}
	g.parser.write(value)

	errors, err := cleanGoModule("errors.go")
	if err != nil {
		panic(err.Error())
	}
	g.parser.write(errors)
}

func (g *goCodeEmitter) writeSeqOrNode() {
	g.parser.writei("return wrapSeq(items, NewSpan(start, p.Location())), nil\n")
}

func (g *goCodeEmitter) writeExprFn(expr AstNode) {
	g.parser.writei("func(p Backtrackable) (Value, error) {\n")
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

// transform

func cleanGoModule(fileName string) (string, error) {
	var (
		out  = &strings.Builder{}
		fset = token.NewFileSet()
	)

	data, err := content.ReadFile(fileName)
	if err != nil {
		return "", err
	}

	node, err := parser.ParseFile(fset, fileName, data, parser.AllErrors)
	if err != nil {
		return "", err
	}

	// Filter out the package and import statements
	for _, decl := range node.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok {
			if gd.Tok == token.PACKAGE || gd.Tok == token.IMPORT {
				continue
			}
		}
		if err := printer.Fprint(out, fset, decl); err != nil {
			return "", err
		}
		out.WriteString("\n")
	}
	return out.String(), nil
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
