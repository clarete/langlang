package diagram

import (
	"github.com/clarete/langlang/go"
)

func fromAstNode(node langlang.AstNode) diagram {
	vi := &ast2diagram{}
	node.Accept(vi)
	return vi.pop()
}

type ast2diagram struct {
	stack []diagram
}

func (vi *ast2diagram) push(i diagram) {
	vi.stack = append(vi.stack, i)
}

func (vi *ast2diagram) pop() diagram {
	idx := len(vi.stack) - 1
	top := vi.stack[idx]
	vi.stack = vi.stack[:idx]
	return top
}

func (vi *ast2diagram) VisitGrammarNode(node *langlang.GrammarNode) error {
	var items []diagram
	for _, item := range node.Definitions {
		item.Accept(vi)
		items = append(items, vi.pop())
	}
	vi.push(newseq(items))
	return nil
}

func (vi *ast2diagram) VisitDefinitionNode(node *langlang.DefinitionNode) error {
	return node.Expr.Accept(vi)
}

func (vi *ast2diagram) VisitLiteralNode(node *langlang.LiteralNode) error {
	vi.push(newterm(node.Value))
	return nil
}

func (vi *ast2diagram) VisitCharsetNode(node *langlang.CharsetNode) error {
	vi.push(newterm(node.String()))
	return nil
}

func (vi *ast2diagram) VisitAnyNode(node *langlang.AnyNode) error {
	vi.push(newterm("ANY"))
	return nil
}

func (vi *ast2diagram) VisitIdentifierNode(node *langlang.IdentifierNode) error {
	vi.push(newnonterm(node.Value))
	return nil
}

func (vi *ast2diagram) VisitSequenceNode(node *langlang.SequenceNode) error {
	var items []diagram
	for _, item := range node.Items {
		item.Accept(vi)
		items = append(items, vi.pop())
	}
	vi.push(newseq(items))
	return nil
}

func (vi *ast2diagram) VisitOneOrMoreNode(node *langlang.OneOrMoreNode) error {
	node.Expr.Accept(vi)
	vi.push(newstack(pol_minus, vi.pop(), newempty()))
	return nil
}

func (vi *ast2diagram) VisitZeroOrMoreNode(node *langlang.ZeroOrMoreNode) error {
	node.Expr.Accept(vi)
	vi.push(newstack(pol_minus, vi.pop(), newempty()))
	return nil
}

func (vi *ast2diagram) VisitOptionalNode(node *langlang.OptionalNode) error {
	node.Expr.Accept(vi)
	vi.push(newstack(pol_plus, vi.pop(), newempty()))
	return nil
}

func (vi *ast2diagram) VisitChoiceNode(node *langlang.ChoiceNode) error {
	node.Left.Accept(vi)
	top := vi.pop()
	node.Right.Accept(vi)
	bottom := vi.pop()
	vi.push(newstack(pol_plus, top, bottom))
	return nil
}

func (vi *ast2diagram) VisitImportNode(node *langlang.ImportNode) error   { return nil }
func (vi *ast2diagram) VisitCaptureNode(node *langlang.CaptureNode) error { return nil }
func (vi *ast2diagram) VisitAndNode(node *langlang.AndNode) error         { return nil }
func (vi *ast2diagram) VisitNotNode(node *langlang.NotNode) error         { return nil }
func (vi *ast2diagram) VisitLexNode(node *langlang.LexNode) error         { return nil }
func (vi *ast2diagram) VisitLabeledNode(node *langlang.LabeledNode) error { return nil }
func (vi *ast2diagram) VisitClassNode(node *langlang.ClassNode) error     { return nil }
func (vi *ast2diagram) VisitRangeNode(node *langlang.RangeNode) error     { return nil }
