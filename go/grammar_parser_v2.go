package langlang

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

//go:generate go run ./cmd/langlang -grammar ../grammars/langlang.peg -output-language go -output-path ./grammar_parser_bootstrap.go -go-remove-lib -go-package langlang -go-parser GrammarParserBootstrap --disable-capture-spaces

type GrammarParserV2 struct {
	input  []byte
	file   string
	fileID FileID
	tree   Tree
}

func NewGrammarParserV2(grammar []byte) *GrammarParserV2 {
	return &GrammarParserV2{input: grammar}
}

func (p *GrammarParserV2) SetGrammarFile(file string) {
	p.file = file
}

func (p *GrammarParserV2) SetGrammarFileID(fileID FileID) {
	p.fileID = fileID
}

func (p *GrammarParserV2) sloc(n NodeID) SourceLocation {
	return NewSourceLocation(p.fileID, p.tree.Span(n))
}

// Parse kicks off parsing the input string and generates an AST
// describing a grammar
func (p *GrammarParserV2) Parse() (AstNode, error) {
	parser := NewGrammarParserBootstrap()
	parser.SetInput(p.input)
	tree, err := parser.Parse()
	if err != nil {
		return nil, errors.New("Parse: " + err.Error())
	}
	p.tree = tree
	root, ok := tree.Root()
	if !ok {
		return nil, errors.New("Parse: no root node found")
	}
	return p.parseGrammar(root)
}

// Grammar <- Import* Definition* EOF
func (p *GrammarParserV2) parseGrammar(id NodeID) (*GrammarNode, error) {
	var (
		imports    []*ImportNode
		defs       []*DefinitionNode
		defsByName = map[string]*DefinitionNode{}
		items      []NodeID
	)

	// Get the expression child of this node
	childID, ok := p.tree.Child(id)
	if !ok {
		return nil, errors.New("parseGrammar: no child node found")
	}
	if p.tree.Type(childID) == NodeType_Sequence {
		items = p.tree.Children(childID)
	} else {
		items = []NodeID{childID}
	}

	for _, itemID := range items {
		if p.tree.Type(itemID) != NodeType_Node {
			continue
		}
		switch p.tree.Name(itemID) {
		case "Import":
			imports = append(imports, p.parseImport(itemID))
		case "Definition":
			def, err := p.parseDefinition(itemID)
			if err != nil {
				return nil, err
			}
			defs = append(defs, def)
			defsByName[def.Name] = def
		}
	}
	//sloc :=
	return NewGrammarNode(imports, defs, defsByName, p.sloc(id)), nil
}

// Import <- "@import" Identifier ("," Identifier)* "from" Literal
func (p *GrammarParserV2) parseImport(id NodeID) *ImportNode {
	var (
		child, _ = p.tree.Child(id)
		names    []*LiteralNode
		items    = p.tree.Children(child)
		idx      = 1
	)
	for _, itemID := range items[idx:] {
		itemType := p.tree.Type(itemID)
		switch itemType {
		case NodeType_String:
			idx++
			if p.tree.Text(itemID) == "from" {
				break
			}
			continue
		case NodeType_Node:
			childID, _ := p.tree.Child(itemID)
			if p.tree.Type(childID) == NodeType_String {
				idx++
				names = append(names, NewLiteralNode(p.tree.Text(childID), p.sloc(childID)))
			}
			continue
		}
		break
	}
	path, _ := unescape(p.tree.Text(items[idx]))
	path = path[1 : len(path)-1]
	return NewImportNode(NewLiteralNode(path, p.sloc(items[3])), names, p.sloc(id))
}

// Definition <- Identifier LEFTARROW Expression
func (p *GrammarParserV2) parseDefinition(id NodeID) (*DefinitionNode, error) {
	child, ok := p.tree.Child(id)
	if !ok {
		return nil, errors.New("parseDefinition: no child node found")
	}
	var (
		items = p.tree.Children(child)
		name  = p.tree.Text(items[0])
		expr  AstNode
		err   error
	)
	if len(items) == 3 {
		expr, err = p.parseExpression(items[2])
		if err != nil {
			return nil, err
		}
	} else {
		expr = NewSequenceNode([]AstNode{}, p.sloc(id))
	}
	return NewDefinitionNode(name, expr, p.sloc(id)), nil
}

// Expression <- Sequence ("/" Sequence)*
func (p *GrammarParserV2) parseExpression(id NodeID) (AstNode, error) {
	childID, ok := p.tree.Child(id)
	if !ok {
		return nil, errors.New("parseExpression: no child node found")
	}
	childType := p.tree.Type(childID)
	switch childType {
	case NodeType_Sequence:
		return p.parseChoice(childID)
	case NodeType_Node:
		return p.parseSequence(childID)
	default:
		return nil, fmt.Errorf("unknown node type for parseExpression: %v", childType)
	}
}

func (p *GrammarParserV2) parseChoice(seqID NodeID) (AstNode, error) {
	items := p.tree.Children(seqID)
	head, err := p.parseSequence(items[0])
	if err != nil {
		return nil, err
	}
	tail := make([]AstNode, 0, len(items))
	for i := 1; i < len(items); i++ {
		itemID := items[i]
		if p.tree.Text(itemID) == "/" {
			continue
		}
		seq, err := p.parseSequence(itemID)
		if err != nil {
			return nil, err
		}
		tail = append(tail, seq)
	}

	allItems := append([]AstNode{head}, tail...)

	accum := allItems[len(allItems)-1]

	for i := len(allItems) - 2; i >= 0; i-- {
		strsl := allItems[i].SourceLocation()
		start := strsl.Span.Start
		end := accum.SourceLocation().Span.End
		newsl := NewSourceLocation(strsl.FileID, NewSpan(start, end))
		accum = NewChoiceNode(allItems[i], accum, newsl)
	}

	return accum, nil
}

func (p *GrammarParserV2) parseSequence(id NodeID) (AstNode, error) {
	var (
		err   error
		items []AstNode
	)
	childID, ok := p.tree.Child(id)
	if !ok {
		return nil, errors.New("parseSequence: no child node found")
	}
	childType := p.tree.Type(childID)

	switch childType {
	case NodeType_Sequence:
		children := p.tree.Children(childID)
		items = make([]AstNode, len(children))
		for i, expID := range children {
			if items[i], err = p.parsePrefix(expID); err != nil {
				return nil, err
			}
		}
	case NodeType_Node:
		prefix, err := p.parsePrefix(childID)
		if err != nil {
			return nil, err
		}
		items = []AstNode{prefix}
	default:
		return nil, fmt.Errorf("unknown node type for parseSequence: %v", childType)
	}
	return NewSequenceNode(items, p.sloc(id)), nil
}

func (p *GrammarParserV2) parsePrefix(id NodeID) (AstNode, error) {
	childID, ok := p.tree.Child(id)
	if !ok {
		return nil, errors.New("parsePrefix: no child node found")
	}
	childType := p.tree.Type(childID)

	switch childType {
	case NodeType_Sequence:
		items := p.tree.Children(childID)
		labeled, err := p.parseLabeled(items[1])
		if err != nil {
			return nil, err
		}
		switch p.tree.Text(items[0]) {
		case "!":
			return NewNotNode(labeled, p.sloc(childID)), nil
		case "&":
			return NewAndNode(labeled, p.sloc(childID)), nil
		case "#":
			return NewLexNode(labeled, p.sloc(childID)), nil
		}
	case NodeType_Node:
		return p.parseLabeled(childID)
	default:
		return nil, fmt.Errorf("unknown node type for parsePrefix: %v", childType)
	}
	panic("unreachable")
}

func (p *GrammarParserV2) parseLabeled(id NodeID) (AstNode, error) {
	childID, ok := p.tree.Child(id)
	if !ok {
		return nil, errors.New("parseLabeled: no child node found")
	}
	childType := p.tree.Type(childID)

	switch childType {
	case NodeType_Sequence:
		items := p.tree.Children(childID)
		suffix, err := p.parseSuffix(items[0])
		if err != nil {
			return nil, err
		}
		return NewLabeledNode(p.tree.Text(items[2]), suffix, p.sloc(childID)), nil
	case NodeType_Node:
		return p.parseSuffix(childID)
	default:
		return nil, fmt.Errorf("unknown node type for parseLabeled: %v", childType)
	}
}

func (p *GrammarParserV2) parseSuffix(id NodeID) (AstNode, error) {
	childID, ok := p.tree.Child(id)
	if !ok {
		return nil, errors.New("parseSuffix: no child node found")
	}
	childType := p.tree.Type(childID)

	switch childType {
	case NodeType_Sequence:
		items := p.tree.Children(childID)
		primary, err := p.parsePrimary(items[0])
		if err != nil {
			return nil, err
		}

		switch p.tree.Text(items[1]) {
		case "?":
			return NewOptionalNode(primary, p.sloc(childID)), nil
		case "*":
			return NewZeroOrMoreNode(primary, p.sloc(childID)), nil
		case "+":
			return NewOneOrMoreNode(primary, p.sloc(childID)), nil
		}
	case NodeType_Node:
		return p.parsePrimary(childID)
	default:
		return nil, fmt.Errorf("unknown node type for parseSuffix: %v", childType)
	}
	panic("unreachable")
}

func (p *GrammarParserV2) parsePrimary(id NodeID) (AstNode, error) {
	childID, ok := p.tree.Child(id)
	if !ok {
		return nil, errors.New("parsePrimary: no child node found")
	}
	childType := p.tree.Type(childID)

	switch childType {
	case NodeType_Sequence:
		items := p.tree.Children(childID)
		return p.parseExpression(items[1])
	case NodeType_Node:
		switch p.tree.Name(childID) {
		case "Identifier":
			return p.parseIdentifier(childID)
		case "Literal":
			return p.parseLiteral(childID)
		case "Class":
			return p.parseClass(childID)
		case "Any":
			return NewAnyNode(p.sloc(childID)), nil
		}
	default:
		return nil, fmt.Errorf("unknown node type for parsePrimary: %v", childType)
	}
	panic("unreachable")
}

func (p *GrammarParserV2) parseLiteral(id NodeID) (*LiteralNode, error) {
	var (
		err  error
		text = p.tree.Text(id)
		sloc = p.sloc(id)
	)

	sloc.Span.Start.Cursor++
	sloc.Span.Start.Column++
	sloc.Span.End.Cursor--
	sloc.Span.End.Column--

	text, err = unescape(text[1 : len(text)-1])
	if err != nil {
		return nil, err
	}
	return NewLiteralNode(text, sloc), nil
}

func (p *GrammarParserV2) parseClass(id NodeID) (*ClassNode, error) {
	childID, ok := p.tree.Child(id)
	if !ok {
		return nil, errors.New("parseClass: no child node found")
	}
	all := p.tree.Children(childID)
	items := all[1 : len(all)-1]
	output := make([]AstNode, len(items))
	var err error

	for i, itemID := range items {
		output[i], err = p.parseSpan(itemID)
		if err != nil {
			return nil, err
		}
	}
	return NewClassNode(output, p.sloc(id)), nil
}

func (p *GrammarParserV2) parseSpan(id NodeID) (AstNode, error) {
	childID, ok := p.tree.Child(id)
	if !ok {
		return nil, errors.New("parseRange: no child node found")
	}
	childType := p.tree.Type(childID)

	switch childType {
	case NodeType_Sequence:
		items := p.tree.Children(childID)
		left, err := p.parseChar(items[0])
		if err != nil {
			return nil, err
		}
		right, err := p.parseChar(items[2])
		if err != nil {
			return nil, err
		}
		return NewRangeNode(r(left), r(right), p.sloc(childID)), nil
	case NodeType_Node:
		s, err := p.parseChar(childID)
		if err != nil {
			return nil, err
		}
		return NewLiteralNode(s, p.sloc(childID)), nil
	default:
		panic(fmt.Sprintf("NO ENTIENDO: %v", childType))
	}
}

// Identifier <- [a-zA-Z_][a-zA-Z0-9_]*
func (p *GrammarParserV2) parseIdentifier(id NodeID) (*IdentifierNode, error) {
	childID, ok := p.tree.Child(id)
	if !ok {
		return nil, errors.New("parseIdentifier: no child node found")
	}
	idText := p.tree.Text(childID)
	return NewIdentifierNode(idText, p.sloc(id)), nil
}

func (p *GrammarParserV2) parseChar(id NodeID) (string, error) {
	return unescape(p.tree.Text(id))
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

func r(s string) rune {
	for _, c := range s {
		return c
	}
	return 0
}
