package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run ../../cmd/langlang -language go -grammar ./import_gr_expr.peg -output ./import.go

func TestImport(t *testing.T) {
	for _, test := range []struct {
		Name  string
		Match string
	}{
		{
			Name:  "First Level",
			Match: "true",
		},
		{
			Name:  "Second Level: Single Quote",
			Match: "'foo bar'",
		},
		{
			Name:  "Second Level: Number",
			Match: "0.55",
		},
		{
			Name:  "Third Level",
			Match: "3 + 2",
		},
		{
			Name:  "Override",
			Match: "0xC0FFEE",
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput(test.Match)
			v, err := p.ParseExpr()
			require.NoError(t, err)
			assert.Equal(t, test.Match, v.Text())
		})
	}

	for _, test := range []struct {
		Name  string
		Input string
		Error string
	}{
		{
			Name:  "Dont accept overriden",
			Input: "3+#xC0FFEE",
			Error: "TermRightOperand @ 2",
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput(test.Input)
			_, err := p.ParseExpr()
			require.Error(t, err)
			assert.Equal(t, test.Error, err.Error())
		})
	}

	for _, test := range []struct {
		Name     string
		Input    string
		Expected string
	}{
		{
			Name:     "label dependency",
			Input:    "0xG + 2",
			Expected: "0xerror[LabelHex: G] + 2",
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewParser()
			p.SetInput(test.Input)
			v, err := p.ParseExpr()
			require.NoError(t, err)
			assert.Equal(t, test.Expected, v.Text())
		})
	}
}

func BenchmarkParser(b *testing.B) {
	type scenario struct {
		Name  string
		Query string
	}
	for _, scenario := range []scenario{
		{
			Name:  "Single Digit",
			Query: "1",
		},
		{
			Name:  "Term",
			Query: "41 + 22",
		},
		{
			Name:  "Multi",
			Query: "33 * 44",
		},
	} {
		b.Run(scenario.Name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				p := NewParser()
				p.SetInput(scenario.Query)
				p.ParseExpr()
			}
		})
	}
}
