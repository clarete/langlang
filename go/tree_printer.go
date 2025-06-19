package langlang

import (
	"strings"
)

type FormatFunc[T any] func(input string, token T) string

type treePrinter[T any] struct {
	padStr *[]string
	output *strings.Builder
	format FormatFunc[T]
}

func newTreePrinter[T any](format FormatFunc[T]) *treePrinter[T] {
	return &treePrinter[T]{
		padStr: &[]string{},
		output: &strings.Builder{},
		format: format,
	}
}

func (tp *treePrinter[T]) indent(s string) {
	*tp.padStr = append(*tp.padStr, s)
}

func (tp *treePrinter[T]) unindent() {
	index := len(*tp.padStr) - 1
	*tp.padStr = (*tp.padStr)[:index]
}

func (tp *treePrinter[T]) padding() {
	for _, item := range *tp.padStr {
		tp.write(item)
	}
}

func (tp *treePrinter[T]) writel(s string) {
	tp.write(s)
	tp.output.WriteRune('\n')
}

func (tp *treePrinter[T]) pwritel(s string) {
	tp.pwrite(s)
	tp.output.WriteRune('\n')
}

func (tp *treePrinter[T]) write(s string) {
	tp.output.WriteString(s)
}

func (tp *treePrinter[T]) pwrite(s string) {
	tp.padding()
	tp.write(s)
}

var literalSanitizer = strings.NewReplacer(
	`"`, `\"`,
	// `'`, `\'`,
	`\`, `\\`,
	string('\n'), `\n`,
	string('\r'), `\r`,
	string('\t'), `\t`,
)

func escapeLiteral(s string) string {
	return literalSanitizer.Replace(s)
}
