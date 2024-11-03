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
	labelsMap   map[string]struct{}
	labels      []string
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
		options:   opt,
		parser:    newOutputWriter(),
		labelsMap: map[string]struct{}{},
		labels:    []string{},
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
	g.parser.writel("() (Value, error) {")
	g.parser.indent()

	g.parser.writeil(fmt.Sprintf(`p.PushTraceSpan(TracerSpan{Name: "%s"})`, n.Name))
	g.parser.writeil("defer p.PopTraceSpan()")
	g.parser.writeil("if p.printTraceback {")
	g.parser.indent()
	g.parser.writeil("fmt.Printf(\"%s; %s\\n\", p.Location(), p.PrintStackTrace())")
	g.parser.unindent()
	g.parser.writeil("}")

	g.parser.writeil("var (")
	g.parser.indent()
	g.parser.writeil("start = p.Location()")
	g.parser.writeil("item  Value")
	g.parser.writeil("err   error")
	g.parser.unindent()
	g.parser.writeil(")")

	g.parser.writei("item, err = ")
	g.visit(n.Expr)
	g.parser.write("\n")
	g.writeIfErr()
	g.parser.writeil("if item == nil {")
	g.parser.indent()
	g.parser.writeil("return nil, nil")
	g.parser.unindent()
	g.parser.writeil("}")

	g.parser.writeil("return p.RunAction(")
	g.parser.indent()
	g.parser.writeil(fmt.Sprintf(`"%s",`, n.Name))
	g.parser.writeil(fmt.Sprintf(`NewNode("%s", item, NewSpan(start, p.Location())),`, n.Name))
	g.parser.unindent()
	g.parser.writeil(")")

	g.parser.unindent()
	g.parser.writel("\n}")
}

func (g *goCodeEmitter) visitSequenceNode(n *SequenceNode) {
	shouldConsumeSpaces := g.lexLevel == 0 && g.isUnderRuleLevel() && !n.IsSyntactic()
	g.parser.writel("(func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writeil("var (")
	g.parser.indent()
	g.parser.writeil("start = p.Location()")
	g.parser.writeil("items []Value")

	if len(n.Items) > 0 {
		g.parser.writeil("item  Value")
		g.parser.writeil("err   error")
	}

	g.parser.unindent()
	g.parser.writeil(")")

	for _, item := range n.Items {
		_, isLexNode := item.(*LexNode)
		if shouldConsumeSpaces && !isLexNode {
			if _, ok := g.grammarNode.DefsByName["Spacing"]; ok {
				g.parser.writeil("item, err = p.(*Parser).ParseSpacing()")
			} else {
				g.parser.writeil("item, err = p.(*Parser).parseSpacing()")
			}
			g.writeIfErr()
			g.parser.writeil("if item != nil {")
			g.parser.indent()
			g.parser.writeil("items = append(items, item)")
			g.parser.unindent()
			g.parser.writeil("}")
		}
		g.parser.writei("item, err = ")
		g.visit(item)
		g.parser.write("\n")
		g.writeIfErr()

		g.parser.writeil("if item != nil {")
		g.parser.indent()
		g.parser.writeil("items = append(items, item)")
		g.parser.unindent()
		g.parser.writeil("}")
	}

	g.writeSeqOrNode()

	g.parser.unindent()
	g.parser.writei("}(p))")
}

func (g *goCodeEmitter) visitOneOrMoreNode(n *OneOrMoreNode) {
	g.parser.writel("(func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writeil("start := p.Location()")
	g.parser.writeil("items, err := OneOrMore(p, func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writei("return ")
	g.visit(n.Expr)
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writeil("})")
	g.writeIfErr()
	g.writeSeqOrNode()

	g.parser.unindent()
	g.parser.writei("}(p))")
}

func (g *goCodeEmitter) visitZeroOrMoreNode(n *ZeroOrMoreNode) {
	g.parser.writel("(func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writeil("start := p.Location()")
	g.parser.writeil("items, err := ZeroOrMore(p, func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writei("return ")
	g.visit(n.Expr)
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writeil("})")
	g.writeIfErr()
	g.writeSeqOrNode()

	g.parser.unindent()
	g.parser.writei("}(p))")
}

func (g *goCodeEmitter) visitOptionalNode(n *OptionalNode) {
	g.parser.writel("Choice(p, []ParserFn[Value]{")
	g.parser.indent()

	g.writeExprFn(n.Expr)
	g.parser.writel(",")

	g.parser.writeil("func(p Backtrackable) (Value, error) {")
	g.parser.indent()
	g.parser.writeil("return nil, nil")
	g.parser.unindent()
	g.parser.writeil("},")

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
		g.parser.writel("Choice(p, []ParserFn[Value]{")
		g.parser.indent()

		for _, expr := range n.Items {
			g.writeExprFn(expr)
			g.parser.writel(",")
		}

		g.parser.unindent()
		g.parser.writei("})")
	}
}

func (g *goCodeEmitter) visitAndNode(n *AndNode) {
	g.parser.writel("And(p, func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writeil("p.EnterPredicate()")
	g.parser.writeil("defer func() { p.LeavePredicate() }()")

	g.parser.writei("return ")
	g.visit(n.Expr)
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("})")
}

func (g *goCodeEmitter) visitNotNode(n *NotNode) {
	g.parser.writel("Not(p, func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writeil("p.EnterPredicate()")
	g.parser.writeil("defer func() { p.LeavePredicate() }()")

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
	// keep both the set of labels as well as an ordered list.
	// The set prevents duplicates in the ordered list.
	// Duplicates come from using the same label more than once in
	// the grammar, which is totally valid.
	if _, ok := g.labelsMap[n.Label]; !ok {
		g.labelsMap[n.Label] = struct{}{}
		g.labels = append(g.labels, n.Label)
	}

	g.parser.writel("func(p Backtrackable) (Value, error) {")
	g.parser.indent()
	g.parser.writeil("start := p.Location()")

	g.parser.writeil("return Choice(p, []ParserFn[Value]{")
	g.parser.indent()

	// Write the expression as the first option
	g.writeExprFn(n.Expr)
	g.parser.writel(",")

	// if the expression failed, throw an error
	g.parser.writeil("func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writeil("if p.WithinPredicate() {")
	g.parser.indent()

	g.parser.writeil(fmt.Sprintf(`msg, ok := p.(*Parser).labelMsgs["%s"]`, n.Label))
	g.parser.writeil("if !ok {")
	g.parser.indent()
	g.parser.writeil(fmt.Sprintf(`msg = "%s"`, n.Label))
	g.parser.unindent()
	g.parser.writeil("}")

	g.parser.writeil(fmt.Sprintf(`return nil, p.NewError("%s", msg, NewSpan(start, p.Location()))`, n.Label))
	g.parser.unindent()
	g.parser.writeil("}")

	g.parser.writeil(fmt.Sprintf(`if fn, ok := p.(*Parser).recoveryTable["%s"]; ok {`, n.Label))
	g.parser.indent()
	g.parser.writeil("return fn(p)")
	g.parser.unindent()
	g.parser.writeil("}")

	g.parser.writeil(fmt.Sprintf(`return nil, p.Throw("%s", NewSpan(start, p.Location()))`, n.Label))
	g.parser.unindent()
	g.parser.writeil("},")

	g.parser.unindent()
	g.parser.writeil("})")

	g.parser.unindent()
	g.parser.writeil("}(p)")
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
		g.parser.writel("Choice(p, []ParserFn[Value]{")
		g.parser.indent()

		for _, expr := range n.Items {
			g.writeExprFn(expr)
			g.parser.writel(",")
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
	g.parser.writel("\n")

	g.parser.write("import (\n")
	g.parser.indent()
	g.parser.writeil(`"fmt"`)

	if !g.options.RemoveLib {
		g.parser.writeil(`"strconv"`)
		g.parser.writeil(`"strings"`)
	}

	g.parser.unindent()
	g.parser.writel(")\n")

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
	g.parser.writeil("\nfunc NewParser() *Parser {")
	g.parser.indent()
	g.parser.writeil("p := &Parser{")
	g.parser.indent()
	g.parser.writeil("captureSpaces: true,")
	g.parser.writeil("recoveryTable: map[string]ParserFn[Value]{},")
	g.parser.unindent()
	g.parser.writeil("}")
	for _, label := range g.labels {
		if _, ok := g.grammarNode.DefsByName[label]; ok {
			g.parser.writei(`p.recoveryTable["`)
			g.parser.write(label)
			g.parser.writel(`"] = func(p Backtrackable) (Value, error) {`)
			g.parser.indent()
			g.parser.writeil("start := p.Location()")
			g.parser.writei("item, err := p.(*Parser).Parse")
			g.parser.write(label)
			g.parser.writel("()")
			g.writeIfErr()
			g.parser.writeil(fmt.Sprintf(`msg, ok := p.(*Parser).labelMsgs["%s"]`, label))
			g.parser.writeil("if !ok {")
			g.parser.indent()
			g.parser.writeil(fmt.Sprintf(`msg = "%s"`, label))
			g.parser.unindent()
			g.parser.writeil("}")
			g.parser.writeil(fmt.Sprintf(
				`return NewError("%s", msg, item, NewSpan(start, p.Location())), nil`,
				label,
			))
			g.parser.unindent()
			g.parser.writeil("}")
		}
	}
	g.parser.writeil("return p")
	g.parser.unindent()
	g.parser.writeil("}")
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
	g.parser.writeil("return wrapSeq(items, NewSpan(start, p.Location())), nil")
}

func (g *goCodeEmitter) writeExprFn(expr AstNode) {
	g.parser.writeil("func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writei("return ")
	g.visit(expr)
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("}")
}

func (g *goCodeEmitter) writeIfErr() {
	g.parser.writeil("if err != nil {")
	g.parser.indent()
	g.parser.writeil("return nil, err")
	g.parser.unindent()
	g.parser.writeil("}")
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

func (o *outputWriter) writeil(s string) {
	o.writeIndent()
	o.write(s)
	o.write("\n")
}

func (o *outputWriter) writel(s string) {
	o.write(s)
	o.buffer.WriteString("\n")
}

func (o *outputWriter) write(s string) {
	o.buffer.WriteString(s)
}
