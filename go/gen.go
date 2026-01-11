package langlang

import (
	"bytes"
	"embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

//go:embed api.go vm.go vm_stack.go vm_charset.go tree.go tree_printer.go errors.go pos.go
var goEvalContent embed.FS

type GenGoOptions struct {
	PackageName string
	ParserName  string
	RemoveLib   bool
}

type tmplRenderOpts struct {
	PackageName string
}

func GenGoEval(asm *Program, cfg *Config, opt GenGoOptions) (string, error) {
	g := newGoEvalEmitter(opt)
	g.writePrelude()
	g.writeInterfaces()
	g.writeParserProgram(Encode(asm, cfg))
	g.writeParserStruct()
	g.writeParserConstructor()
	g.writeParserMethods(asm)
	g.writeDeps()
	return g.output()
}

type goEvalEmitter struct {
	options GenGoOptions
	parser  *outputWriter
}

func newGoEvalEmitter(opt GenGoOptions) *goEvalEmitter {
	return &goEvalEmitter{
		options: opt,
		parser:  newOutputWriter("\t"),
	}
}

func (g *goEvalEmitter) writePrelude() {
	g.parser.write("package ")
	g.parser.write(g.options.PackageName)
	g.parser.writel("\n")

	if !g.options.RemoveLib {
		g.parser.write("import (\n")
		g.parser.indent()
		g.parser.writeil(`"encoding/hex"`)
		g.parser.writeil(`"fmt"`)
		g.parser.writeil(`"math/bits"`)
		g.parser.writeil(`"sort"`)
		g.parser.writeil(`"strconv"`)
		g.parser.writeil(`"strings"`)
		g.parser.writeil(`"unicode/utf8"`)
		g.parser.unindent()
		g.parser.writel(")\n")
	}
}

func (g *goEvalEmitter) writeInterfaces() {
	if !g.options.RemoveLib {
		g.parser.write(readAPI(goEvalContent, "api.go"))
	}
}

func (g *goEvalEmitter) writeParserProgram(bt *Bytecode) {
	g.parser.writel(fmt.Sprintf("var bytecodeFor%s = &Bytecode{", g.options.ParserName))
	g.parser.indent()

	// The Bytecode

	g.parser.writeil("code: []byte{")
	g.parser.indent()
	g.parser.writei("")
	for _, byte := range bt.code {
		g.parser.write(fmt.Sprintf("%d, ", byte))
	}
	g.parser.writel("")
	g.parser.unindent()
	g.parser.writeil("},")

	// Strings Table

	g.parser.writeil("strs: []string{")
	g.parser.indent()
	g.parser.writei("")
	for _, s := range bt.strs {
		g.parser.write(fmt.Sprintf(`"%s", `, escapeLiteral(s)))
	}
	g.parser.writel("")
	g.parser.unindent()
	g.parser.writeil("},")

	// Recovery Expressions Map

	g.parser.writeil("rxps: map[int]int{")
	g.parser.indent()
	g.parser.writei("")

	xps := make([]int, 0, len(bt.rxps))
	for xp := range bt.rxps {
		xps = append(xps, xp)
	}
	sort.Ints(xps)
	for _, k := range xps {
		v := bt.rxps[k]
		g.parser.write(fmt.Sprintf("%d: %d, ", k, v))
	}
	g.parser.writel("")
	g.parser.unindent()
	g.parser.writeil("},")

	// Recovery Expressions Bitset

	g.parser.writeil("rxbs: bitset512{")
	g.parser.indent()
	g.parser.writei("")
	for _, k := range bt.rxbs {
		g.parser.write(fmt.Sprintf("%d,", k))
	}
	g.parser.writel("\n},\n")
	g.parser.unindent()

	// strings map

	g.parser.writeil("smap: map[string]int{")
	g.parser.indent()
	g.parser.writei("")

	smap := make([]string, 0, len(bt.smap))
	for xp := range bt.smap {
		smap = append(smap, xp)
	}
	sort.Strings(smap)
	for _, k := range smap {
		v := bt.smap[k]
		g.parser.write(fmt.Sprintf(`"%s": %d, `, k, v))
	}
	g.parser.writel("")
	g.parser.unindent()
	g.parser.writeil("},")

	g.parser.writeil("sets: []charset{")
	g.parser.indent()
	for _, set := range bt.sets {
		g.parser.writei("{bits: [32]byte{")

		for i := 0; i < len(set.bits); i++ {
			g.parser.write(fmt.Sprintf("%d,", set.bits[i]))
		}

		g.parser.writel("}},")
	}
	g.parser.unindent()
	g.parser.writeil("},")

	g.parser.writeil("sexp: [][]expected{")
	g.parser.indent()
	for _, item := range bt.sexp {
		g.parser.writei("{")
		for _, sub := range item {
			if sub.b == 0 {
				g.parser.write(fmt.Sprintf("expected{a: %s},", strconv.QuoteRune(sub.a)))
			} else {
				g.parser.write(fmt.Sprintf("expected{a: %s, b: %s},", strconv.QuoteRune(sub.a), strconv.QuoteRune(sub.b)))
			}
		}
		g.parser.writel("},")
	}
	g.parser.unindent()
	g.parser.writeil("},")

	g.parser.unindent()
	g.parser.writel("}")
}

func (g *goEvalEmitter) writeParserStruct() {
	g.parser.writel(fmt.Sprintf("type %s struct{", g.options.ParserName))
	g.parser.indent()
	g.parser.writeil("input []byte")
	g.parser.writeil("vm    *virtualMachine")
	g.parser.unindent()
	g.parser.writel("}")
}

func (g *goEvalEmitter) writeParserConstructor() {
	g.parser.writel(fmt.Sprintf("func New%s() *%s {", g.options.ParserName, g.options.ParserName))
	g.parser.indent()

	g.parser.writei("vm := NewVirtualMachine(")
	g.parser.write(fmt.Sprintf("bytecodeFor%s,", g.options.ParserName))
	g.parser.writel(")")
	g.parser.writeil(fmt.Sprintf("return &%s{vm: vm}", g.options.ParserName))

	g.parser.unindent()
	g.parser.writel("}")
}

func (g *goEvalEmitter) writeParserMethods(asm *Program) {
	var (
		cursor  = 0
		addrmap = make(map[int]int, len(asm.identifiers))
	)
	for i, instruction := range asm.code {
		switch instruction.(type) {
		case ILabel:
			addrmap[i] = cursor
		default:
			cursor += instruction.SizeInBytes()
		}
	}
	addrs := make([]int, 0, len(asm.identifiers))
	for addr := range asm.identifiers {
		addrs = append(addrs, addr)
	}
	sort.Ints(addrs)
	for _, addr := range addrs {
		strID := asm.identifiers[addr]
		name := asm.strings[strID]
		g.parser.write(fmt.Sprintf("func (p *%s) Parse%s() (Tree, error) { ", g.options.ParserName, name))
		g.parser.write(fmt.Sprintf("return p.parseFn(%d)", addrmap[addr]))
		g.parser.writel(" }")
	}

	g.parser.writel(fmt.Sprintf("func (p *%s) Parse() (Tree, error)                  { return p.parseFn(5) }", g.options.ParserName))
	g.parser.writel(fmt.Sprintf("func (p *%s) SetInput(input []byte)                 { p.input = input }", g.options.ParserName))
	g.parser.writel(fmt.Sprintf("func (p *%s) GetInput() []byte                      { return p.input }", g.options.ParserName))
	g.parser.writel(fmt.Sprintf("func (p *%s) SetLabelMessages(el map[int]int)       { p.vm.SetLabelMessages(el) }", g.options.ParserName))
	g.parser.writel(fmt.Sprintf("func (p *%s) SetShowFails(v bool)                   { p.vm.SetShowFails(v) }", g.options.ParserName))
	g.parser.writel("")

	if !g.options.RemoveLib {
		g.parser.writel(fmt.Sprintf(
			"func LabelMessagesFor%s(labels map[string]string) map[int]int { return bytecodeFor%s.CompileErrorLabels(labels) }",
			g.options.ParserName,
			g.options.ParserName,
		))
	}

	// The entrypoint for parsing
	g.parser.writel(fmt.Sprintf("func (p *%s) parseFn(addr int) (Tree, error) {", g.options.ParserName))
	g.parser.indent()
	g.parser.writeil("val, _, err := p.vm.MatchRule(p.input, addr)")
	g.parser.writeil("return val, err")
	g.parser.unindent()
	g.parser.writel("}")
}

func (g *goEvalEmitter) writeDeps() {
	if g.options.RemoveLib {
		return
	}
	for _, file := range []string{
		"pos.go", "tree.go", "tree_printer.go", "errors.go",
		"vm_stack.go", "vm_charset.go", "vm.go",
	} {
		s, err := cleanGoModule(goEvalContent, file)
		if err != nil {
			panic(err.Error())
		}
		g.parser.write(s)
	}
}

func (g *goEvalEmitter) output() (string, error) {
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
	formatted, err := format.Source(output.Bytes())
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

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

func readAPI(fs embed.FS, fileName string) string {
	var (
		out  = &strings.Builder{}
		fset = token.NewFileSet()
	)
	data, err := fs.ReadFile(fileName)
	if err != nil {
		panic(err.Error())
	}
	node, err := parser.ParseFile(fset, fileName, data, parser.AllErrors)
	if err != nil {
		panic(err.Error())
	}
	ast.Inspect(node, func(n ast.Node) bool {
		genDecl, ok := n.(*ast.GenDecl)
		if !ok {
			return true
		}
		if genDecl.Tok != token.TYPE && genDecl.Tok != token.CONST {
			return true
		}
		if err := printer.Fprint(out, fset, n); err != nil {
			panic(err.Error())
		}
		out.WriteString("\n")
		return true
	})
	return out.String()
}

type outputWriter struct {
	buffer      *strings.Builder
	indentLevel int
	space       string
}

func newOutputWriter(space string) *outputWriter {
	return &outputWriter{
		buffer: &strings.Builder{},
		space:  space,
	}
}

func (o *outputWriter) indent() {
	o.indentLevel++
}

func (o *outputWriter) unindent() {
	o.indentLevel--
}

func (o *outputWriter) writeIndent() {
	for i := 0; i < o.indentLevel; i++ {
		o.buffer.WriteString(o.space)
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
