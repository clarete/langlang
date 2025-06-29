package langlang

import "fmt"

type CompilerConfig struct {
	Optimize int
}

type compiler struct {
	config CompilerConfig

	// cursor is the index of the last instruction written the
	// `code` vector
	cursor int

	// code is a vector where the compiler writes down the
	// instructions
	code []Instruction

	// definitionLabels is a map from set of production string ids
	// to the set of labels
	definitionLabels map[int]ILabel

	// openAddrs is a map from call site addresses to production
	// names that keeps calls that need to be patched because they
	// occurred syntaticaly before the definition of the
	// production
	openAddrs map[int]int

	// identifiers is a map from the address of the first
	// instruction in a production to the id of the name of the
	// production
	identifiers map[int]int

	// strings is a table of all the strings in a grammar, both
	// from literals, as well as from production names
	strings []string

	// stringsMap keeps a record of what position in the strings
	// table a string instance points to
	stringsMap map[string]int

	// errorLabelIDs is a set of all label IDs
	errorLabelIDs map[int]struct{}

	recovery map[int]recoveryEntry

	grammarNode *GrammarNode
}

func Compile(expr AstNode, config CompilerConfig) (*Program, error) {
	var err error
	c := &compiler{
		config:           config,
		identifiers:      map[int]int{},
		definitionLabels: map[int]ILabel{},
		openAddrs:        map[int]int{},
		stringsMap:       map[string]int{},
		strings:          []string{},
		errorLabelIDs:    map[int]struct{}{},
		recovery:         map[int]recoveryEntry{},
	}
	if err = expr.Accept(c); err != nil {
		return nil, err
	}
	if err = c.backpatchCallSites(); err != nil {
		return nil, err
	}
	if err = c.mapRecoveryExprs(); err != nil {
		return nil, err
	}
	return &Program{
		identifiers: c.identifiers,
		recovery:    c.recovery,
		strings:     c.strings,
		code:        c.code,
	}, nil
}

func (c *compiler) VisitGrammarNode(node *GrammarNode) error {
	c.emit(ICall{})
	c.emit(IHalt{})
	c.grammarNode = node
	return WalkGrammarNode(c, node)
}

func (c *compiler) VisitImportNode(node *ImportNode) error {
	return fmt.Errorf("Import isn't translatable")
}

func (c *compiler) VisitDefinitionNode(node *DefinitionNode) error {
	var (
		id = c.pushString(node.Name)
		l0 = NewILabel()
	)

	c.identifiers[c.cursor] = id
	c.definitionLabels[id] = l0

	c.emit(l0)

	if err := node.Expr.Accept(c); err != nil {
		return err
	}

	c.emit(IReturn{})

	return nil
}

func (c *compiler) VisitCaptureNode(node *CaptureNode) error {
	id := c.pushString(node.Name)

	c.emit(ICapBegin{ID: id})

	if err := node.Expr.Accept(c); err != nil {
		return err
	}

	c.emit(ICapEnd{})
	return nil
}

func (c *compiler) VisitSequenceNode(node *SequenceNode) error {
	return WalkSequenceNode(c, node)
}

func (c *compiler) VisitOneOrMoreNode(node *OneOrMoreNode) error {
	if err := node.Expr.Accept(c); err != nil {
		return err
	}
	return c.VisitZeroOrMoreNode(NewZeroOrMoreNode(node.Expr, node.Span()))
}

func (c *compiler) VisitZeroOrMoreNode(node *ZeroOrMoreNode) error {
	l0 := NewILabel()
	l1 := NewILabel()
	l2 := NewILabel()

	switch c.config.Optimize {
	case 0:
		c.emit(l0)
	}

	c.emit(IChoice{Label: l2})
	c.emit(l1)

	if err := node.Expr.Accept(c); err != nil {
		return err
	}

	switch c.config.Optimize {
	case 0:
		c.emit(ICommit{Label: l0})
	case 1:
		c.emit(IPartialCommit{Label: l1})
	}

	c.emit(l2)

	return nil
}

func (c *compiler) VisitOptionalNode(node *OptionalNode) error {
	lb := NewILabel()

	c.emit(IChoice{Label: lb})

	if err := node.Expr.Accept(c); err != nil {
		return err
	}

	c.emit(ICommit{Label: lb})
	c.emit(lb)
	return nil
}

func (c *compiler) VisitChoiceNode(node *ChoiceNode) error {
	l1 := NewILabel()
	l2 := NewILabel()

	c.emit(IChoice{Label: l1})

	if err := node.Left.Accept(c); err != nil {
		return err
	}

	c.emit(ICommit{Label: l2})
	c.emit(l1)

	if err := node.Right.Accept(c); err != nil {
		return err
	}

	c.emit(l2)

	return nil
}

func (c *compiler) VisitAndNode(node *AndNode) error {
	switch c.config.Optimize {
	case 0:
		return c.VisitNotNode(NewNotNode(NewNotNode(node.Expr, node.Span()), node.Span()))

	case 1:
		l1 := NewILabel()
		l2 := NewILabel()

		c.emit(IChoicePred{Label: l1})

		if err := node.Expr.Accept(c); err != nil {
			return nil
		}

		c.emit(IBackCommit{Label: l2})
		c.emit(l1)
		c.emit(IFail{})
		c.emit(l2)
	}
	return nil
}

func (c *compiler) VisitNotNode(node *NotNode) error {
	l1 := NewILabel()

	c.emit(IChoicePred{Label: l1})

	if err := node.Expr.Accept(c); err != nil {
		return nil
	}

	switch c.config.Optimize {
	case 0:
		l2 := NewILabel()
		c.emit(ICommit{Label: l2})
		c.emit(l2)
		c.emit(IFail{})
	case 1:
		c.emit(IFailTwice{})
	}

	c.emit(l1)
	return nil
}

func (c *compiler) VisitLexNode(node *LexNode) error {
	return node.Expr.Accept(c)
}

func (c *compiler) VisitLabeledNode(node *LabeledNode) error {
	l1 := NewILabel()
	l2 := NewILabel()
	id := c.pushString(node.Label)

	c.errorLabelIDs[id] = struct{}{}
	c.emit(IChoice{Label: l1})

	if err := node.Expr.Accept(c); err != nil {
		return nil
	}

	c.emit(ICommit{Label: l2})
	c.emit(l1)
	c.emit(IThrow{ErrorLabel: id})
	c.emit(l2)
	return nil
}

func (c *compiler) VisitIdentifierNode(node *IdentifierNode) error {
	id := c.pushString(node.Value)

	// TODO[bounded left recursion]
	precedence := 0

	// if the definition indexed by `ID` has already been seen,
	// we're just going to write its address here.  Otherwise, the
	// combination (cursor, id) are saved within the `openAddrs`
	// map so the backpatching can fix this later.
	if label, ok := c.definitionLabels[id]; ok {
		c.emit(ICall{Label: label, Precedence: precedence})
	} else {
		c.openAddrs[c.cursor] = id
		c.emit(ICall{Precedence: precedence})
	}

	return nil
}

func (c *compiler) VisitClassNode(node *ClassNode) error {
	switch len(node.Items) {
	case 0:
		return nil
	case 1:
		return node.Items[0].Accept(c)
	}

	accum := node.Items[len(node.Items)-1]

	for i := len(node.Items) - 2; i >= 0; i-- {
		span := NewSpan(node.Items[i].Span().Start, accum.Span().End)
		accum = NewChoiceNode(node.Items[i], accum, span)
	}

	return c.VisitChoiceNode(accum.(*ChoiceNode))
}

func (c *compiler) VisitRangeNode(node *RangeNode) error {
	c.emit(ISpan{Lo: node.Left, Hi: node.Right})
	return nil
}

func (c *compiler) VisitLiteralNode(node *LiteralNode) error {
	for _, r := range []rune(node.Value) {
		c.emit(IChar{Char: r})
	}
	return nil
}

func (c *compiler) VisitAnyNode(node *AnyNode) error {
	c.emit(IAny{})
	return nil
}

func (c *compiler) backpatchCallSites() error {
	// addr is where in the bytecode we need to backpatch, and ID
	// is what function should be called from that backpatch site.
	for callAddr, id := range c.openAddrs {
		// we're now looking up the function by its ID, and if
		// found we're just going to need to adjust for
		// forward and backward jumping.
		if label, ok := c.definitionLabels[id]; ok {
			// TODO: precedence
			c.code[callAddr] = ICall{Label: label}
			continue
		}
		return fmt.Errorf("Production `%s` does not exist", c.strings[id])
	}

	// patch up call to main
	def := c.grammarNode.FirstDefinition()
	defID := c.pushString(def.Name)
	label := c.definitionLabels[defID]
	c.code[0] = ICall{Label: label}

	return nil
}

func (c *compiler) mapRecoveryExprs() error {
	for id := range c.errorLabelIDs {
		if label, ok := c.definitionLabels[id]; ok {
			// TODO[bounded left recursion]: precedence
			c.recovery[id] = recoveryEntry{label: label}
		}
	}
	return nil
}

func (c *compiler) emit(i Instruction) {
	c.code = append(c.code, i)
	c.cursor++
}

func (c *compiler) pushString(s string) int {
	strID := len(c.strings)
	if savedString, ok := c.stringsMap[s]; ok {
		return savedString
	}

	c.strings = append(c.strings, s)
	c.stringsMap[s] = strID
	return strID
}
