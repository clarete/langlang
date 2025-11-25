package diagram

import (
	"fmt"

	langlang "github.com/clarete/langlang/go"
)

func DiagramString(n langlang.AstNode) string {
	return fromAstNode(n).String()
}

func DiagramToLayout(n langlang.AstNode, maxWidth, maxHeight int) (layout, error) {
	d := fromAstNode(n)
	if err := computeVerticalMetrics(d); err != nil {
		return nil, err
	}
	fmt.Println(d)

	// Scale maxWidth from character units to abstract units
	// In the ASCII renderer, we divide by 8 to convert abstract to chars
	// So multiply by 8 here to convert chars to abstract
	abstractMaxWidth := float64(maxWidth) * 8.0

	l, err := lineWrap(d, abstractMaxWidth)
	if err != nil {
		return nil, err
	}
	fmt.Println(l)
	return l, nil
}

func DiagramToACII(n langlang.AstNode, maxWidth, maxHeight int) (string, error) {
	l, err := DiagramToLayout(n, maxWidth, maxHeight)
	if err != nil {
		return "", err
	}
	return layoutToASCII(l, maxWidth, maxHeight), nil
}
