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
	lexLevel    int
	labelsMap   map[string]struct{}
	labels      []string
	grammarNode *GrammarNode
}

type GenGoOptions struct {
	PackageName string
	ParserName  string
	RemoveLib   bool
}

func GenGo(node AstNode, opt GenGoOptions) (string, error) {
	g := newGoCodeEmitter(opt)
	g.writePrelude()

	if err := node.Accept(g); err != nil {
		return "", err
	}

	g.writeConstructor()
	g.writeEmbeds()
	return g.output()
}

type tmplRenderOpts struct {
	PackageName string
}

//go:embed parser.go tree_printer.go value.go errors.go
var content embed.FS

func newGoCodeEmitter(opt GenGoOptions) *goCodeEmitter {
	return &goCodeEmitter{
		options:   opt,
		parser:    newOutputWriter("\t"),
		labelsMap: map[string]struct{}{},
		labels:    []string{},
	}
}

func (g *goCodeEmitter) VisitImportNode(n *ImportNode) error {
	return fmt.Errorf("unreachable")
}

func (g *goCodeEmitter) VisitGrammarNode(n *GrammarNode) error {
	g.grammarNode = n
	return WalkGrammarNode(g, n)
}

func (g *goCodeEmitter) VisitDefinitionNode(n *DefinitionNode) error {
	g.parser.write("\nfunc (p *Parser) Parse")
	g.parser.write(n.Name)
	g.parser.writel("() (val Value, err error) {")
	g.parser.indent()

	g.parser.writeil("var (")
	g.parser.indent()
	g.parser.writeil("start = p.Location()")
	g.parser.writeil(fmt.Sprintf(`key   = "%s_" + strconv.Itoa(start.Cursor)`, n.Name))
	g.parser.writeil("item  Value")
	g.parser.unindent()
	g.parser.writeil(")")

	g.parser.writeil("cached, ok := p.mtable[key]")
	g.parser.writeil("if ok {")
	g.parser.indent()
	g.parser.writeil("p.Backtrack(cached.end)")
	g.parser.writeil("return cached.val, cached.err")
	g.parser.unindent()
	g.parser.writeil("}")

	g.parser.writeil("defer func() {")
	g.parser.indent()
	g.parser.writeil("p.mtable[key] = mentry{end: p.Location(), val: val, err: err}")
	g.parser.unindent()
	g.parser.writeil("}()")

	g.parser.writei("item, err = ")

	if err := n.Expr.Accept(g); err != nil {
		return err
	}

	g.parser.write("\n")
	g.writeIfErr()
	g.parser.writeil("if item == nil {")
	g.parser.indent()
	g.parser.writeil("return nil, nil")
	g.parser.unindent()
	g.parser.writeil("}")

	g.parser.writeil("val, err = p.RunAction(")
	g.parser.indent()
	g.parser.writeil(fmt.Sprintf(`"%s",`, n.Name))
	g.parser.writeil(fmt.Sprintf(`NewNode("%s", item, NewSpan(start, p.Location())),`, n.Name))
	g.parser.unindent()
	g.parser.writeil(")")
	g.parser.writeil("return val, err")

	g.parser.unindent()
	g.parser.writel("\n}")

	return nil
}

func (g *goCodeEmitter) VisitCaptureNode(n *CaptureNode) error {
	return n.Expr.Accept(g)
}

func (g *goCodeEmitter) VisitSequenceNode(n *SequenceNode) error {
	switch len(n.Items) {
	case 0:
		g.parser.write("nil, nil")
		return nil
	case 1:
		return n.Items[0].Accept(g)
	}

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
		g.parser.writei("item, err = ")

		if err := item.Accept(g); err != nil {
			return err
		}

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

	return nil
}

func (g *goCodeEmitter) VisitOneOrMoreNode(n *OneOrMoreNode) error {
	g.parser.writel("(func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writeil("start := p.Location()")
	g.parser.writeil("items, err := OneOrMore(p, func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writei("return ")

	if err := n.Expr.Accept(g); err != nil {
		return err
	}

	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writeil("})")
	g.writeIfErr()
	g.writeSeqOrNode()

	g.parser.unindent()
	g.parser.writei("}(p))")

	return nil
}

func (g *goCodeEmitter) VisitZeroOrMoreNode(n *ZeroOrMoreNode) error {
	g.parser.writel("(func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writeil("start := p.Location()")
	g.parser.writeil("items, err := ZeroOrMore(p, func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writei("return ")
	if err := n.Expr.Accept(g); err != nil {
		return err
	}
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writeil("})")
	g.writeIfErr()
	g.writeSeqOrNode()

	g.parser.unindent()
	g.parser.writei("}(p))")

	return nil
}

func (g *goCodeEmitter) VisitOptionalNode(n *OptionalNode) error {
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

	return nil
}

func (g *goCodeEmitter) VisitChoiceNode(n *ChoiceNode) error {
	g.parser.writel("Choice(p, []ParserFn[Value]{")
	g.parser.indent()

	g.writeExprFn(n.Left)
	g.parser.writel(",")

	g.writeExprFn(n.Right)
	g.parser.writel(",")

	g.parser.unindent()
	g.parser.writei("})")

	return nil
}

func (g *goCodeEmitter) VisitAndNode(n *AndNode) error {
	g.parser.writel("And(p, func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writeil("p.EnterPredicate()")
	g.parser.writeil("defer func() { p.LeavePredicate() }()")

	g.parser.writei("return ")
	if err := n.Expr.Accept(g); err != nil {
		return err
	}
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("})")

	return nil
}

func (g *goCodeEmitter) VisitNotNode(n *NotNode) error {
	g.parser.writel("Not(p, func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writeil("p.EnterPredicate()")
	g.parser.writeil("defer func() { p.LeavePredicate() }()")

	g.parser.writei("return ")
	if err := n.Expr.Accept(g); err != nil {
		return err
	}
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("})")

	return nil
}

func (g *goCodeEmitter) VisitLexNode(n *LexNode) error {
	g.lexLevel++
	if err := n.Expr.Accept(g); err != nil {
		return err
	}
	g.parser.write("\n")
	g.lexLevel--
	return nil
}

func (g *goCodeEmitter) VisitLabeledNode(n *LabeledNode) error {
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

	return nil
}

func (g *goCodeEmitter) VisitIdentifierNode(n *IdentifierNode) error {
	// If the user hasn't customized the spacing rule, we're going
	// to call the hand-crafted version of parseSpacing instead of
	// the one read from `builtins.peg`.
	//
	// This does help with performance, but the main reason this
	// is here is just so we can continue to support the
	// `SetCaptureSpaces()` method.  Which should be deprecated.
	if n.Value == "Spacing" && isBuiltinSpacing(g.grammarNode) {
		s := "p.(*Parser).parseSpacing()"
		if g.isAtRuleLevel() {
			s = "p.parseSpacing()"
		}
		g.parser.write(s)
		return nil
	}

	s := "p.(*Parser).Parse%s()"
	if g.isAtRuleLevel() {
		s = "p.Parse%s()"
	}
	g.parser.write(fmt.Sprintf(s, n.Value))
	return nil
}

func (g *goCodeEmitter) VisitLiteralNode(n *LiteralNode) error {
	s := `p.(*Parser).parseLiteral("%s")`
	if g.isAtRuleLevel() {
		s = `p.parseLiteral("%s")`
	}
	g.parser.write(fmt.Sprintf(s, escapeLiteral(n.Value)))

	return nil
}

func (g *goCodeEmitter) VisitClassNode(n *ClassNode) error {
	switch len(n.Items) {
	case 0:
	case 1:
		if err := n.Items[0].Accept(g); err != nil {
			return err
		}
	default:
		g.parser.writel("Choice(p, []ParserFn[Value]{")
		g.parser.indent()

		for _, expr := range n.Items {
			if err := g.writeExprFn(expr); err != nil {
				return err
			}
			g.parser.writel(",")
		}

		g.parser.unindent()
		g.parser.writei("})")
	}
	return nil
}

func (g *goCodeEmitter) VisitRangeNode(n *RangeNode) error {
	s := "p.(*Parser).parseRange('%c', '%c')"
	if g.isAtRuleLevel() {
		s = "p.parseRange('%c', '%c')"
	}
	g.parser.write(fmt.Sprintf(s, n.Left, n.Right))
	return nil
}

func (g *goCodeEmitter) VisitCharsetNode(*CharsetNode) error {
	return fmt.Errorf("Not Implemented")
}

func (g *goCodeEmitter) VisitAnyNode(_ *AnyNode) error {
	s := "p.(*Parser).parseAny()"
	if g.isAtRuleLevel() {
		s = "p.parseAny()"
	}
	g.parser.write(s)
	return nil
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

	s, err := cleanGoModule(content, "parser.go")
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
	g.parser.writeil("mtable:        map[string]mentry{},")
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

	treePrinter, err := cleanGoModule(content, "tree_printer.go")
	if err != nil {
		panic(err.Error())
	}
	g.parser.write(treePrinter)

	value, err := cleanGoModule(content, "value.go")
	if err != nil {
		panic(err.Error())
	}
	g.parser.write(value)

	errors, err := cleanGoModule(content, "errors.go")
	if err != nil {
		panic(err.Error())
	}
	g.parser.write(errors)
}

func (g *goCodeEmitter) writeSeqOrNode() {
	g.parser.writeil("return wrapSeq(items, NewSpan(start, p.Location())), nil")
}

func (g *goCodeEmitter) writeExprFn(expr AstNode) error {
	g.parser.writeil("func(p Backtrackable) (Value, error) {")
	g.parser.indent()

	g.parser.writei("return ")
	if err := expr.Accept(g); err != nil {
		return err
	}
	g.parser.write("\n")

	g.parser.unindent()
	g.parser.writei("}")
	return nil
}

func (g *goCodeEmitter) writeIfErr() {
	g.parser.writeil("if err != nil {")
	g.parser.indent()
	g.parser.writeil("return nil, err")
	g.parser.unindent()
	g.parser.writeil("}")
}

// transform

func cleanGoModule(fs embed.FS, fileName string) (string, error) {
	var (
		out  = &strings.Builder{}
		fset = token.NewFileSet()
	)

	data, err := fs.ReadFile(fileName)
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

// isBuiltinSpacing will return true if the grammar in `g` has the
// same spacing rules as the builtin grammar.  Notice that this relies
// on the `builtinsGrammar` global variable.
func isBuiltinSpacing(g *GrammarNode) bool {
	var (
		spacing = g.DefsByName["Spacing"].Equal(builtinsGrammar.DefsByName["Spacing"])
		space   = g.DefsByName["Space"].Equal(builtinsGrammar.DefsByName["Space"])
		eol     = g.DefsByName["EOL"].Equal(builtinsGrammar.DefsByName["EOL"])
	)
	return spacing && space && eol
}
