package langlang

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed vm.go vm_stack.go tree_printer.go errors.go value.go
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

	g.parser.write("import (\n")
	g.parser.indent()
	g.parser.writeil(`"encoding/binary"`)
	g.parser.writeil(`"fmt"`)
	g.parser.writeil(`"strconv"`)
	g.parser.writeil(`"strings"`)

	if !g.options.RemoveLib {
		g.parser.writeil(`"io"`)
	}

	g.parser.unindent()
	g.parser.writel(")\n")
}

func (g *goEvalEmitter) writeParserProgram(bt *Bytecode) {
	g.parser.writel("var parserProgram = &Bytecode{")
	g.parser.indent()

	g.parser.writeil("code: []byte{")
	g.parser.indent()
	g.parser.writei("")
	for _, byte := range bt.code {
		g.parser.write(fmt.Sprintf("%d, ", byte))
	}
	g.parser.writel("")
	g.parser.unindent()
	g.parser.writeil("},")

	g.parser.writeil("strs: []string{")
	g.parser.indent()
	g.parser.writei("")
	for _, s := range bt.strs {
		g.parser.write(fmt.Sprintf(`"%s", `, escapeLiteral(s)))
	}
	g.parser.writel("")
	g.parser.unindent()
	g.parser.writeil("},")

	g.parser.writeil("rxps: map[int]int{")
	g.parser.indent()
	for k, v := range bt.rxps {
		g.parser.write(fmt.Sprintf("%d: %d,", k, v))
	}
	g.parser.unindent()
	g.parser.writeil("},")

	g.parser.unindent()
	g.parser.writel("}")
}

func (g *goEvalEmitter) writeParserStruct() {
	g.parser.writel("type Parser struct{")
	g.parser.indent()
	g.parser.writeil("input     string")
	g.parser.writeil("errLabels map[string]string")
	g.parser.unindent()
	g.parser.writel("}")
}

func (g *goEvalEmitter) writeParserConstructor() {
	g.parser.writel("func NewParser() *Parser { return &Parser{} }")
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
	for addr, strID := range asm.identifiers {
		name := asm.strings[strID]
		g.parser.write(fmt.Sprintf("func (p *Parser) Parse%s() (Value, error) { ", name))
		g.parser.write(fmt.Sprintf("return p.parseFn(%d)", addrmap[addr]))
		g.parser.writel(" }")
	}
	g.parser.writel("func (p *Parser) Parser() (Value, error) { return p.parseFn(5) }")
	g.parser.writel("func (p *Parser) SetInput(input string) { p.input = input }")
	g.parser.writel("func (p *Parser) SetLabelMessages(el map[string]string) { p.errLabels = el }")
	g.parser.writel("func (p *Parser) parseFn(addr uint16) (Value, error) {")
	g.parser.indent()
	g.parser.writeil("writeU16(parserProgram.code[1:], addr)")
	g.parser.writeil("vm := newVirtualMachine(parserProgram, p.errLabels)")
	g.parser.writeil("val, _, err := vm.Match(strings.NewReader(p.input))")
	g.parser.writeil("return val, err")
	g.parser.unindent()
	g.parser.writel("}")
}

func (g *goEvalEmitter) writeDeps() {
	if g.options.RemoveLib {
		return
	}
	for _, file := range []string{
		"value.go", "tree_printer.go", "errors.go", "vm_stack.go", "vm.go",
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
