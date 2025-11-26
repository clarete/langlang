package main

import (
	"fmt"
	"strconv"

	langlang "github.com/clarete/langlang/go"
)

type JsonPrinter struct {
	input []byte
	*treePrinter[langlang.FormatToken]
}

func NewJsonPrinter(input []byte, format langlang.FormatFunc[langlang.FormatToken]) *JsonPrinter {
	return &JsonPrinter{input: input, treePrinter: newTreePrinter(format)}
}

func (v *JsonPrinter) VisitString(n *langlang.String) error {
	escaped := strconv.Quote(n.String(v.input))
	v.pwrite("{")
	v.write("\"type\": \"string\",")
	v.write("\"value\": ")
	v.write(v.format(escaped, langlang.FormatToken_Literal))
	v.write("}")

	return nil
}

func (v *JsonPrinter) VisitSequence(n *langlang.Sequence) error {
	v.pwrite("{")
	v.pwrite("\"type\": \"sequence\",")
	v.pwrite("\"count\": ")
	v.write(v.format(fmt.Sprintf("%d", len(n.Items)), langlang.FormatToken_Literal))
	v.pwrite(",")
	v.pwrite("\"items\": [")

	for i, item := range n.Items {
		switch {
		case i == len(n.Items)-1:
			v.pwrite("")
			item.Accept(v)
			v.pwrite("")
			v.pwrite("")
		default:
			v.pwrite("")
			item.Accept(v)
			v.pwrite("")
			v.pwrite(",")
		}
	}
	v.pwrite("]")
	v.pwrite("}")
	return nil
}

func (v *JsonPrinter) VisitNode(n *langlang.Node) error {
	v.pwrite("{")
	v.pwrite("\"type\": \"node\",")
	v.pwrite("\"name\": \"")
	v.write(v.format(n.Name, langlang.FormatToken_Literal))
	v.write("\",")
	v.pwrite("\"expr\": ")
	n.Expr.Accept(v)
	v.pwrite("")
	v.pwrite("}")

	return nil
}

func (v *JsonPrinter) VisitError(n *langlang.Error) error {
	v.pwrite("{")
	v.pwrite("\"type\": \"error\"")

	if n.Expr != nil {
		v.write(",")
		v.pwrite("\"expr\": ")
		v.pwrite("{")
		n.Expr.Accept(v)
		v.pwrite("\n")
		v.pwrite("}")
	}

	if n.Label != "" {
		v.write(",")
		v.pwrite("\"label\": \"")
		v.write(v.format(n.Label, langlang.FormatToken_Literal))
		v.write("\"")
	}

	if n.Message != "" {
		v.write(",")
		v.pwrite("\"message\": \"")
		v.write(v.format(n.Message, langlang.FormatToken_Literal))
		v.write("\"")
	}

	v.pwrite("}")
	return nil
}
