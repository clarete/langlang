package main

import (
	"syscall/js"

	langlang "github.com/clarete/langlang/go"
)

func compileAndMatch(this js.Value, args []js.Value) interface{} {
	grammar := args[0].String()
	input := args[1].String()

	val, err := CompileAndMatch(grammar, input)
	if err != nil {
		return err.Error()
	}

	return val.String([]byte(input))
}

func formatValuePlain(input string, _ langlang.FormatToken) string {
	return input
}

func formatValue(input string, node langlang.Value) string {
	p := NewJsonPrinter([]byte(input), formatValuePlain)
	node.Accept(p)

	return p.output.String()
}

func compileJson(this js.Value, args []js.Value) interface{} {
	grammar := args[0].String()
	input := args[1].String()

	val, err := CompileAndMatch(grammar, input)
	if err != nil {
		return err.Error()
	}

	return formatValue(input, val)
}
