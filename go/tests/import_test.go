package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:generate go run ../cmd -language go -grammar ./import_gr_expr.peg -go-prefix Import -output ./import.go

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
			p := NewImportParser()
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
			Error: "term_right_operand @ 2",
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			p := NewImportParser()
			p.SetInput(test.Input)
			_, err := p.ParseExpr()
			require.Error(t, err)
			assert.Equal(t, test.Error, err.Error())
		})
	}
}
