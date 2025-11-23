package diagram

import (
	"github.com/clarete/langlang/go"
)

func DiagramString(n langlang.AstNode) string {
	return fromAstNode(n).String()
}
