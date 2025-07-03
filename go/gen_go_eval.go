package langlang

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"text/template"
)

//go:embed vm.go vm_stack.go vm_input.go vm_charset.go tree_printer.go errors.go value.go
var goEvalContent embed.FS

func GenGoEval(asm *Program, opt GenGoOptions) (string, error) {
	g := newGoEvalEmitter(opt)
	g.writePrelude()
	g.writeParserProgram(Encode(asm))
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
		g.parser.writeil(`"encoding/binary"`)
		g.parser.writeil(`"encoding/hex"`)
		g.parser.writeil(`"fmt"`)
		g.parser.writeil(`"io"`)
		g.parser.writeil(`"math/bits"`)
		g.parser.writeil(`"strconv"`)
		g.parser.writeil(`"strings"`)
		g.parser.writeil(`"unicode/utf8"`)
		g.parser.unindent()
		g.parser.writel(")\n")
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
	g.parser.writeil("input         string")
	g.parser.writeil("captureSpaces bool")
	g.parser.writeil("showFails     bool")
	g.parser.writeil("suppress      map[int]struct{}")
	g.parser.writeil("errLabels     map[string]string")
	g.parser.unindent()
	g.parser.writel("}")
}

func (g *goEvalEmitter) writeParserConstructor() {
	g.parser.writel(fmt.Sprintf("func New%s() *%s {", g.options.ParserName, g.options.ParserName))
	g.parser.indent()
	g.parser.writeil(fmt.Sprintf(`suppress := map[int]struct{}{bytecodeFor%s.smap["Spacing"]: struct{}{}}`, g.options.ParserName))
	g.parser.writeil(fmt.Sprintf("return &%s{captureSpaces: true, suppress: suppress}", g.options.ParserName))
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
		g.parser.write(fmt.Sprintf("func (p *%s) Parse%s() (Value, error) { ", g.options.ParserName, name))
		g.parser.write(fmt.Sprintf("return p.parseFn(%d)", addrmap[addr]))
		g.parser.writel(" }")
	}
	g.parser.writel(fmt.Sprintf("func (p *%s) Parse() (Value, error)                 { return p.parseFn(5) }", g.options.ParserName))
	g.parser.writel(fmt.Sprintf("func (p *%s) SetInput(input string)                 { p.input = input }", g.options.ParserName))
	g.parser.writel(fmt.Sprintf("func (p *%s) SetLabelMessages(el map[string]string) { p.errLabels = el }", g.options.ParserName))
	g.parser.writel(fmt.Sprintf("func (p *%s) SetCaptureSpaces(v bool)               { p.captureSpaces = v }", g.options.ParserName))
	g.parser.writel(fmt.Sprintf("func (p *%s) SetShowFails(v bool)                   { p.showFails = v }", g.options.ParserName))
	g.parser.writel(fmt.Sprintf("func (p *%s) parseFn(addr uint16) (Value, error)    {", g.options.ParserName))
	g.parser.indent()
	g.parser.writeil(fmt.Sprintf("writeU16(bytecodeFor%s.code[1:], addr)", g.options.ParserName))
	g.parser.writeil("var suppress map[int]struct{}")
	g.parser.writeil("if !p.captureSpaces {")
	g.parser.indent()
	g.parser.writeil("suppress = p.suppress")
	g.parser.unindent()
	g.parser.writeil("}")
	g.parser.writeil(fmt.Sprintf("vm := newVirtualMachine(bytecodeFor%s, p.errLabels, suppress, p.showFails)", g.options.ParserName))
	g.parser.writeil("input := NewMemInput(p.input)")
	g.parser.writeil("val, _, err := vm.Match(&input)")
	g.parser.writeil("return val, err")
	g.parser.unindent()
	g.parser.writel("}")
}

func (g *goEvalEmitter) writeDeps() {
	if g.options.RemoveLib {
		return
	}
	for _, file := range []string{
		"value.go", "tree_printer.go", "errors.go",
		"vm_stack.go", "vm_charset.go", "vm_input.go",
		"vm.go",
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
	return output.String(), nil
}
