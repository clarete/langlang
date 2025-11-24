package diagram

import (
	"fmt"
	"github.com/clarete/langlang/go"
)

func DiagramString(n langlang.AstNode) string {
	return fromAstNode(n).String()
}

func DiagramToLayout(n langlang.AstNode) (layout, error) {
	d := fromAstNode(n)
	if err := computeVerticalMetrics(d); err != nil {
		return nil, err
	}
	fmt.Println(d)

	l, err := lineWrap(d)
	if err != nil {
		return nil, err
	}
	fmt.Println(l)
	return l, nil
}

func DiagramToACII(n langlang.AstNode) (string, error) {
	l, err := DiagramToLayout(n)
	if err != nil {
		return "", err
	}
	return layoutToASCII(l, 200, 8), nil
}
