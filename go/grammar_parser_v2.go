package langlang

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

//go:generate go run ./cmd/langlang -grammar ../grammars/langlang.peg -output-language go -output-path ./grammar_parser_bootstrap.go -go-remove-lib -go-package langlang -go-parser GrammarParserBootstrap

type GrammarParserV2 struct {
	file  string
	input string
}

func NewGrammarParserV2(grammar string) *GrammarParserV2 {
	return &GrammarParserV2{input: grammar}
}

func (p *GrammarParserV2) SetGrammarFile(file string) {
	p.file = file
}

// Parse kicks off parsing the input string and generates an AST
// describing a grammar
func (p *GrammarParserV2) Parse() (AstNode, error) {
	parser := NewGrammarParserBootstrap()
	parser.SetInput(p.input)
	parser.SetCaptureSpaces(false)
	val, err := parser.ParseGrammar()
	if err != nil {
		return nil, err
	}
	return parseGrammar(val)
}

// Grammar <- Import* Definition* EOF
func parseGrammar(v Value) (*GrammarNode, error) {
	var (
		nodeValue  = v.(*Node)
		span       = v.Span()
		imports    []*ImportNode
		defs       []*DefinitionNode
		defsByName = map[string]*DefinitionNode{}
		items      []Value
	)
	switch nn := nodeValue.Expr.(type) {
	case *Sequence:
		items = nn.Items
	default:
		items = []Value{nn}
	}
	for _, expr := range items {
		item, ok := expr.(*Node)
		if !ok {
			continue
		}
		switch item.Name {
		case "Import":
			imports = append(imports, parseImport(item))
		case "Definition":
			def, err := parseDefinition(item)
			if err != nil {
				return nil, err
			}
			defs = append(defs, def)
			defsByName[def.Name] = def
		}
	}
	return NewGrammarNode(imports, defs, defsByName, span), nil
}

// Import <- "@import" Identifier ("," Identifier)* "from" Literal
func parseImport(node *Node) *ImportNode {
	var (
		names []*LiteralNode
		items = node.Expr.(*Sequence).Items
		idx   = 1
	)
	for _, item := range items[idx:] {
		switch it := item.(type) {
		case *String:
			idx++
			if it.Value == "from" {
				break
			}
			continue
		case *Node:
			if s, ok := it.Expr.(*String); ok {
				idx++
				names = append(names, NewLiteralNode(s.Value, s.Span()))
			}
			continue
		}
		break
	}
	path, _ := unescape(items[idx].Text())
	path = path[1 : len(path)-1]
	return NewImportNode(NewLiteralNode(path, items[3].Span()), names, node.Span())
}

// Definition <- Identifier LEFTARROW Expression
func parseDefinition(node *Node) (*DefinitionNode, error) {
	var (
		items = node.Expr.(*Sequence).Items
		name  = items[0].(*Node).Expr.(*String).Value
		expr  AstNode
		err   error
	)
	if len(items) == 3 {
		expr, err = parseExpression(items[2])
		if err != nil {
			return nil, err
		}
	} else {
		expr = NewSequenceNode([]AstNode{}, node.Span())
	}
	return NewDefinitionNode(name, expr, node.Span()), nil
}

// Expression <- Sequence ("/" Sequence)*
func parseExpression(v Value) (AstNode, error) {
	switch e := v.(*Node).Expr.(type) {
	case *Sequence:
		return parseChoice(e)
	case *Node:
		return parseSequence(e)
	default:
		return nil, fmt.Errorf("unknown node type for parseExpression: %s", e)
	}
}

func parseChoice(s *Sequence) (AstNode, error) {
	head, err := parseSequence(s.Items[0])
	if err != nil {
		return nil, err
	}
	tail := make([]AstNode, 0, len(s.Items))
	for i := 1; i < len(s.Items); i++ {
		item := s.Items[i]
		if item.Text() == "/" {
			continue
		}
		seq, err := parseSequence(item)
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

func parseSequence(v Value) (AstNode, error) {
	var (
		err   error
		items []AstNode
	)
	switch e := v.(*Node).Expr.(type) {
	case *Sequence:
		items = make([]AstNode, len(e.Items))
		for i, exp := range e.Items {
			items[i], err = parsePrefix(exp.(*Node))
			if err != nil {
				return nil, err
			}
		}
		return NewSequenceNode(items, e.Span()), nil
	case *Node:
		prefix, err := parsePrefix(e)
		if err != nil {
			return nil, err
		}
		items = []AstNode{prefix}
	default:
		return nil, fmt.Errorf("unknown node type for parseSequence: %s", e)
	}
	return NewSequenceNode(items, v.Span()), nil
}

func parsePrefix(v *Node) (AstNode, error) {
	switch e := v.Expr.(type) {
	case *Sequence:
		labeled, err := parseLabeled(e.Items[1].(*Node))
		if err != nil {
			return nil, err
		}
		switch e.Items[0].Text() {
		case "!":
			return NewNotNode(labeled, e.Span()), nil
		case "&":
			return NewAndNode(labeled, e.Span()), nil
		case "#":
			return NewLexNode(labeled, e.Span()), nil
		}
	case *Node:
		return parseLabeled(e)
	default:
		return nil, fmt.Errorf("unknown node type for parsePrefix: %s", e)
	}
	panic("unreachable")
}

func parseLabeled(v *Node) (AstNode, error) {
	switch e := v.Expr.(type) {
	case *Sequence:
		suffix, err := parseSuffix(e.Items[0].(*Node))
		if err != nil {
			return nil, err
		}
		return NewLabeledNode(e.Items[2].Text(), suffix, e.Span()), nil
	case *Node:
		return parseSuffix(e)
	default:
		return nil, fmt.Errorf("unknown node type for parseLabeled: %s", e)
	}
}

func parseSuffix(v *Node) (AstNode, error) {
	switch e := v.Expr.(type) {
	case *Sequence:
		primary, err := parsePrimary(e.Items[0].(*Node))
		if err != nil {
			return nil, err
		}

		switch e.Items[1].Text() {
		case "?":
			return NewOptionalNode(primary, e.Span()), nil
		case "*":
			return NewZeroOrMoreNode(primary, e.Span()), nil
		case "+":
			return NewOneOrMoreNode(primary, e.Span()), nil
		}
	case *Node:
		return parsePrimary(e)
	default:
		return nil, fmt.Errorf("unknown node type for parseSuffix: %s", e)
	}
	panic("unreachable")
}

func parsePrimary(v *Node) (AstNode, error) {
	switch e := v.Expr.(type) {
	case *Sequence:
		return parseExpression(e.Items[1])
	case *Node:
		switch e.Name {
		case "Identifier":
			return parseIdentifier(e)
		case "Literal":
			return parseLiteral(e)
		case "Class":
			return parseClass(e)
		case "Any":
			return NewAnyNode(e.Span()), nil
		}
	default:
		return nil, fmt.Errorf("unknown node type for parsePrimary: %s", e)
	}
	panic("unreachable")
}

func parseLiteral(v *Node) (*LiteralNode, error) {
	var (
		all       = v.Expr.(*Sequence).Items
		noquote   = all[1 : len(all)-1]
		newseq    = NewSequence(noquote, v.Span())
		text, err = unescape(newseq.Text())
	)
	if err != nil {
		return nil, err
	}
	return NewLiteralNode(text, v.Span()), nil
}

// Unescape takes a string and unescapes it
func unescape(value string) (string, error) {
	n := len(value)
	if n < 1 {
		return value, nil
	}
	// If there is nothing to escape, then return.
	if !strings.ContainsRune(value, '\\') {
		return value, nil
	}
	// Inspired by `strconv/quote.go`
	var runeTmp [utf8.UTFMax]byte
	buf := make([]byte, 0, 3*n/2)
	for len(value) > 0 {
		c, encode, rest, err := unescapeChar(value)
		if err != nil {
			return "", err
		}
		value = rest
		if c < utf8.RuneSelf || !encode {
			buf = append(buf, byte(c))
		} else {
			n := utf8.EncodeRune(runeTmp[:], c)
			buf = append(buf, runeTmp[:n]...)
		}
	}
	return string(buf), nil
}

func unescapeChar(s string) (value rune, encode bool, tail string, err error) {
	switch c := s[0]; {
	case c >= utf8.RuneSelf:
		r, size := utf8.DecodeRuneInString(s)
		return r, true, s[size:], nil
	case c != '\\':
		return rune(s[0]), false, s[1:], nil
	}
	if len(s) <= 1 {
		err = errors.New("unable to unescape string, found '\\' as last character")
		return
	}
	control := s[1]
	s = s[2:]

	switch control {
	case '-':
		value = '-'
	case 'n':
		value = '\n'
	case 'r':
		value = '\r'
	case 't':
		value = '\t'
	case '[':
		value = '['
	case ']':
		value = ']'
	case '\\':
		value = '\\'
	case '\'':
		value = '\''
	case '"':
		value = '"'
	case 'u':
		n := 0
		encode = true
		n = 4
		var v rune
		if len(s) < n {
			err = errors.New("unable to unescape string")
			return
		}
		for j := 0; j < n; j++ {
			x, ok := unhex(s[j])
			if !ok {
				err = errors.New("unable to unescape string")
				return
			}
			v = v<<4 | x
		}
		s = s[n:]
		value = v

	default:
		err = fmt.Errorf("unknown unescape sequence: %c", control)
	}

	tail = s
	return
}

func unhex(b byte) (rune, bool) {
	c := rune(b)
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10, true
	}
	return 0, false
}

func parseClass(v *Node) (*ClassNode, error) {
	var (
		all    = v.Expr.(*Sequence).Items
		items  = all[1 : len(all)-1]
		output = make([]AstNode, len(items))
		err    error
	)
	for i, item := range items {
		// unpack `Range` node as well
		output[i], err = parseRange(item.(*Node))
		if err != nil {
			return nil, err
		}
	}
	return NewClassNode(output, v.Span()), nil
}

func parseRange(v *Node) (AstNode, error) {
	switch e := v.Expr.(type) {
	case *Sequence:
		left, err := parseChar(e.Items[0].(*Node))
		if err != nil {
			return nil, err
		}
		right, err := parseChar(e.Items[2].(*Node))
		if err != nil {
			return nil, err
		}
		return NewRangeNode(r(left), r(right), e.Span()), nil
	case *Node:
		s, err := parseChar(e)
		if err != nil {
			return nil, err
		}
		return NewLiteralNode(s, e.Span()), nil
	default:
		panic(fmt.Sprintf("NO ENTIENDO: %s", e))
	}
}

// Identifier <- [a-zA-Z_][a-zA-Z0-9_]*
func parseIdentifier(n *Node) (*IdentifierNode, error) {
	return NewIdentifierNode(n.Expr.(*String).Value, n.Span()), nil
}

func parseChar(n *Node) (string, error) {
	switch e := n.Expr.(type) {
	case *Node:
		return unescape(e.Expr.Text())
	case *String:
		return unescape(e.Value)
	default:
		panic(fmt.Sprintf("NO ENTIENDO: %s", e))
	}
}

func parseEscape(n Value) (string, error) {
	switch e := n.(type) {
	case *String:
		return unescape(e.Value)
	default:
		panic(fmt.Sprintf("NO ENTIENDO: %s", e))
	}
}

func r(s string) rune {
	for _, c := range s {
		return c
	}
	return 0
}
