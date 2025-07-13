package langlang

import (
	"fmt"
)

//go:generate go run ./cmd/langlang -grammar ../grammars/wirth.peg -output-language goeval -output-path ./grammar_parser_wirth_bootstrap.go -go-remove-lib -go-package langlang -go-parser GrammarParserWirthBootstrap

type WirthGrammarParser struct {
	file  string
	input string
}

func NewWirthGrammarParser(grammar string) *WirthGrammarParser {
	return &WirthGrammarParser{input: grammar}
}

func (p *WirthGrammarParser) SetGrammarFile(file string) {
	p.file = file
}

// Parse kicks off parsing the input string and generates an AST
// describing a grammar
func (p *WirthGrammarParser) Parse() (AstNode, error) {
	parser := NewGrammarParserWirthBootstrap()
	parser.SetInput(p.input)
	parser.SetCaptureSpaces(false)
	val, err := parser.ParseSyntax()
	if err != nil {
		return nil, err
	}
	grammar, err := parseWirthSyntax(val)
	if err != nil {
		return nil, err
	}
	return grammar, nil
}

// Syntax <- Production* EOF
func parseWirthSyntax(v Value) (*GrammarNode, error) {
	var (
		span       = v.Span()
		nodeValue  = v.(*Node)
		defsByName = map[string]*DefinitionNode{}
		defs       []*DefinitionNode
		items      []Value
	)
	switch nn := nodeValue.Expr.(type) {
	case *Sequence:
		items = nn.Items
	default:
		items = []Value{nn}
	}
	for _, expr := range items {
		def, err := parseWirthProduction(expr)
		if err != nil {
			return nil, err
		}
		defs = append(defs, def)
		defsByName[def.Name] = def
	}
	return NewGrammarNode(nil, defs, defsByName, span), nil
}

func parseWirthProduction(v Value) (*DefinitionNode, error) {
	items := v.(*Node).Expr.(*Sequence).Items
	switch len(items) {
	case 4:
		name := parseWirthIdentifierStr(items[0])
		expr, err := parseWirthExpression(items[2])
		if err != nil {
			return nil, err
		}
		return NewDefinitionNode(name, expr, v.Span()), nil
	}
	return nil, fmt.Errorf("Not Implemented")
}

func parseWirthIdentifierStr(v Value) string {
	return v.(*Node).Expr.(*String).Value
}

func parseWirthExpression(v Value) (AstNode, error) {
	switch e := v.(*Node).Expr.(type) {
	case *Sequence:
		return parseWirthOr(e)
	case *Node:
		return parseWirthTerm(e)
	default:
		return nil, fmt.Errorf("unknown node type for parseWirthExpression: %s", e)
	}
}

func parseWirthOr(v Value) (AstNode, error) {
	s := v.(*Sequence)
	head, err := parseWirthTerm(s.Items[0])
	if err != nil {
		return nil, err
	}
	tail := make([]AstNode, 0, len(s.Items))
	for i := 1; i < len(s.Items); i++ {
		item := s.Items[i]
		if item.Text() == "|" {
			continue
		}
		seq, err := parseWirthTerm(item)
		if err != nil {
			return nil, err
		}
		tail = append(tail, seq)
	}

	items := append([]AstNode{head}, tail...)

	accum := items[len(items)-1]

	for i := len(items) - 2; i >= 0; i-- {
		span := NewSpan(items[i].Span().Start, accum.Span().End)
		accum = NewChoiceNode(items[i], accum, span)
	}
	return accum, nil
}

func parseWirthTerm(v Value) (AstNode, error) {
	switch e := v.(*Node).Expr.(type) {
	case *Sequence:
		return parseWirthSequence(e)
	case *Node:
		return parseWirthFactor(e)
	default:
		return nil, fmt.Errorf("unknown node type for parseWirthTerm: %s", e)
	}
	return nil, fmt.Errorf("Not Implemented")
}

func parseWirthSequence(v Value) (AstNode, error) {
	var (
		err   error
		items []AstNode
	)
	switch e := v.(type) {
	case *Sequence:
		items = make([]AstNode, len(e.Items))
		for i, exp := range e.Items {
			items[i], err = parseWirthFactor(exp.(*Node))
			if err != nil {
				return nil, err
			}
		}
		return NewSequenceNode(items, e.Span()), nil
	case *Node:
		prefix, err := parseWirthFactor(e)
		if err != nil {
			return nil, err
		}
		items = []AstNode{prefix}
	default:
		return nil, fmt.Errorf("unknown node type for parseSequence: %s", e)
	}
	return NewSequenceNode(items, v.Span()), nil
}

func parseWirthFactor(v Value) (AstNode, error) {
	var (
		subNode = v.(*Node).Expr.(*Node)
		subName = subNode.Name
	)
	switch subName {
	case "Identifier":
		return NewIdentifierNode(parseWirthIdentifierStr(subNode), v.Span()), nil
	case "Range":
		return parseWirthRange(subNode)
	case "Repetition":
		return parseWirthRepetition(subNode)
	case "Option":
		return parseWirthOption(subNode)
	case "Group":
		return parseWirthGroup(subNode)
	}
	fmt.Printf("%s:\n%s\n", subName, subNode.Expr.HighlightPrettyString())
	return nil, fmt.Errorf("Not Implemented")
}

func parseWirthRange(v Value) (AstNode, error) {
	switch val := v.(*Node).Expr.(type) {
	case *Node:
		text := val.Expr.(*String).Value
		text = text[1 : len(text)-1]
		text, _ = unescape(text)
		return NewLiteralNode(text, v.Span()), nil
	case *Sequence:
		left, err := parseChar(val.Items[0].(*Node))
		if err != nil {
			return nil, err
		}
		right, err := parseChar(val.Items[2].(*Node))
		if err != nil {
			return nil, err
		}
		lf := r(left[1 : len(left)-1])
		rf := r(right[1 : len(right)-1])
		rangeNode := NewRangeNode(lf, rf, v.Span())
		// fmt.Printf("DE RANGE: %s\n", rangeNode.HighlightPrettyString())
		return NewClassNode([]AstNode{rangeNode}, v.Span()), nil
	default:
		fmt.Printf("foo: %s\n", val.HighlightPrettyString())
	}
	return nil, fmt.Errorf("Not Implemented")
}

func parseWirthGroup(v Value) (AstNode, error) {
	var (
		seq = v.(*Node).Expr.(*Sequence)
		exp = seq.Items[1]
	)
	return parseWirthExpression(exp)
}

func parseWirthOption(v Value) (AstNode, error) {
	var (
		seq = v.(*Node).Expr.(*Sequence)
		exp = seq.Items[1]
	)
	sub, err := parseWirthExpression(exp)
	if err != nil {
		return nil, err
	}
	return NewOptionalNode(sub, v.Span()), nil
}

func parseWirthRepetition(v Value) (AstNode, error) {
	var (
		seq = v.(*Node).Expr.(*Sequence)
		exp = seq.Items[1]
	)
	sub, err := parseWirthExpression(exp)
	if err != nil {
		return nil, err
	}
	return NewZeroOrMoreNode(sub, v.Span()), nil
}
