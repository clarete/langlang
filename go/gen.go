package langlang

import "strings"

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
