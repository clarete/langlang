package langlang

import (
	"fmt"
	"sort"
)

type compiler struct {
	config *Config

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

	defRecursiveMap map[string]struct{}

	defSizeMap map[string]int

	grammarNode *GrammarNode

	dryRun bool

	// withinPredicate tracks if we're inside a predicate (&/!)
	// where captures are not needed, allowing us to emit
	// IPartialCommit instead of IPartialCommitCap
	withinPredicate bool
}

func Compile(expr AstNode, config *Config) (*Program, error) {
	var err error
	c := newCompiler(config)
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
		stringsMap:  c.stringsMap,
		code:        c.code,
	}, nil
}

func newCompiler(config *Config) *compiler {
	return &compiler{
		config:           config,
		identifiers:      map[int]int{},
		definitionLabels: map[int]ILabel{},
		defSizeMap:       map[string]int{},
		openAddrs:        map[int]int{},
		stringsMap:       map[string]int{},
		strings:          []string{},
		errorLabelIDs:    map[int]struct{}{},
		recovery:         map[int]recoveryEntry{},
	}
}

func (c *compiler) VisitGrammarNode(node *GrammarNode) error {
	c.emit(ICall{})
	c.emit(IHalt{})
	c.grammarNode = node
	c.defRecursiveMap = getIsRecursiveFromGrammar(node)
	c.collectErrorLabels(node)
	return WalkGrammarNode(c, node)
}

func (c *compiler) VisitImportNode(node *ImportNode) error {
	return fmt.Errorf("import isn't translatable")
}

func (c *compiler) VisitDefinitionNode(node *DefinitionNode) error {
	if inline, err := c.shouldInline(node); err != nil || inline && !c.config.GetBool("compiler.inline.emit.inlined") {
		return err
	}
	var (
		id = c.intern(node.Name)
		l0 = NewILabel()
	)

	c.identifiers[c.cursor] = id
	c.definitionLabels[id] = l0

	c.emit(l0)

	if err := node.Expr.Accept(c); err != nil {
		return err
	}

	c.emitReturn()

	return nil
}

func (c *compiler) VisitCaptureNode(node *CaptureNode) error {
	if !c.shouldCapture() {
		return node.Expr.Accept(c)
	}

	id := c.intern(node.Name)

	if sz, ok := c.capExprSize(node.Expr); ok {
		if err := node.Expr.Accept(c); err != nil {
			return err
		}

		if node.Name == "" {
			c.emit(ICapTerm{Offset: sz})
			return nil
		}

		c.emit(ICapNonTerm{ID: id, Offset: sz})
		return nil
	}

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
	return c.VisitZeroOrMoreNode(NewZeroOrMoreNode(node.Expr, node.Range()))
}

func (c *compiler) VisitZeroOrMoreNode(node *ZeroOrMoreNode) error {
	if csn, ok := node.Expr.(*CharsetNode); ok {
		c.emit(ISpan{cs: csn.cs})
		return nil
	}

	l0 := NewILabel()
	l1 := NewILabel()
	l2 := NewILabel()

	switch c.config.GetInt("compiler.optimize") {
	case 0:
		c.emit(l0)
	}

	c.emit(IChoice{Label: l2})
	c.emit(l1)

	if err := node.Expr.Accept(c); err != nil {
		return err
	}

	switch c.config.GetInt("compiler.optimize") {
	case 0:
		c.emitCommit(l0)
	default:
		if c.shouldCapture() {
			c.emit(ICapPartialCommit{Label: l1})
		} else {
			c.emit(IPartialCommit{Label: l1})
		}
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

	c.emitCommit(lb)
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

	c.emitCommit(l2)
	c.emit(l1)

	if err := node.Right.Accept(c); err != nil {
		return err
	}

	c.emit(l2)

	return nil
}

func (c *compiler) VisitAndNode(node *AndNode) error {
	switch c.config.GetInt("compiler.optimize") {
	case 0:
		return c.VisitNotNode(NewNotNode(NewNotNode(node.Expr, node.Range()), node.Range()))

	default:
		l1 := NewILabel()
		l2 := NewILabel()

		c.emit(IChoicePred{Label: l1})

		old := c.withinPredicate
		c.withinPredicate = true
		err := node.Expr.Accept(c)
		c.withinPredicate = old
		if err != nil {
			return err
		}

		c.emitBackCommit(l2)
		c.emit(l1)
		c.emit(IFail{})
		c.emit(l2)
	}
	return nil
}

func (c *compiler) VisitNotNode(node *NotNode) error {
	l1 := NewILabel()

	c.emit(IChoicePred{Label: l1})

	old := c.withinPredicate
	c.withinPredicate = true
	err := node.Expr.Accept(c)
	c.withinPredicate = old
	if err != nil {
		return err
	}

	switch c.config.GetInt("compiler.optimize") {
	case 0:
		l2 := NewILabel()
		c.emitCommit(l2)
		c.emit(l2)
		c.emit(IFail{})
	default:
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
	id := c.intern(node.Label)

	c.errorLabelIDs[id] = struct{}{}
	c.emit(IChoice{Label: l1})

	if err := node.Expr.Accept(c); err != nil {
		return nil
	}

	c.emitCommit(l2)
	c.emit(l1)
	c.emit(IThrow{ErrorLabel: id})
	c.emit(l2)
	return nil
}

func (c *compiler) VisitIdentifierNode(node *IdentifierNode) error {
	id := c.intern(node.Value)
	def := c.grammarNode.DefsByName[node.Value]
	inline, err := c.shouldInline(def)
	if err != nil {
		return err
	}
	if inline {
		return def.Expr.Accept(c)
	}

	// TODO[bounded left recursion]
	precedence := 0

	// if the definition indexed by `ID` has already been seen,
	// we're just going to write its address here.  Otherwise, the
	// combination (cursor, id) are saved within the `openAddrs`
	// map so the backpatching can fix this later.
	if label, ok := c.definitionLabels[id]; ok {
		c.emit(ICall{Label: label, Precedence: precedence})
	} else {
		c.saveOpenAddr(id)
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
		span := NewRange(node.Items[i].Range().Start, accum.Range().End)
		accum = NewChoiceNode(node.Items[i], accum, span)
	}

	return c.VisitChoiceNode(accum.(*ChoiceNode))
}

func (c *compiler) VisitRangeNode(node *RangeNode) error {
	c.emit(IRange{Lo: node.Left, Hi: node.Right})
	return nil
}

func (c *compiler) VisitCharsetNode(n *CharsetNode) error {
	c.emit(ISet{cs: n.cs})
	return nil
}

func (c *compiler) VisitLiteralNode(node *LiteralNode) error {
	for _, r := range node.Value {
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
		return fmt.Errorf("production `%s` does not exist", c.strings[id])
	}

	// Patch main call
	def := c.grammarNode.FirstDefinition()
	id := c.intern(def.Name)
	label := c.definitionLabels[id]
	c.emitAt(0, ICall{Label: label})

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

func (c *compiler) shouldCapture() bool {
	return c.config.GetBool("grammar.captures") && !c.withinPredicate
}

func (c *compiler) emitCommit(label ILabel) {
	if c.shouldCapture() {
		c.emit(ICapCommit{Label: label})
	} else {
		c.emit(ICommit{Label: label})
	}
}

func (c *compiler) emitBackCommit(label ILabel) {
	if c.shouldCapture() {
		c.emit(ICapBackCommit{Label: label})
	} else {
		c.emit(IBackCommit{Label: label})
	}
}

func (c *compiler) emitReturn() {
	if c.shouldCapture() {
		c.emit(ICapReturn{})
	} else {
		c.emit(IReturn{})
	}
}

func (c *compiler) saveOpenAddr(addr int) {
	c.openAddrs[c.cursor] = addr
}

func (c *compiler) emit(i Instruction) {
	c.code = append(c.code, i)
	c.cursor++
}

func (c *compiler) emitAt(cursor int, i Instruction) {
	c.code[cursor] = i
}

func (c *compiler) intern(s string) int {
	strID := len(c.strings)
	if savedString, ok := c.stringsMap[s]; ok {
		return savedString
	}

	c.strings = append(c.strings, s)
	c.stringsMap[s] = strID
	return strID
}

func (c *compiler) collectErrorLabels(node *GrammarNode) {
	// Scan the grammar AST to find all error labels (^label syntax)
	// so we can avoid inlining recovery definitions
	Inspect(node, func(n AstNode) bool {
		if labeled, ok := n.(*LabeledNode); ok {
			id := c.intern(labeled.Label)
			c.errorLabelIDs[id] = struct{}{}
		}
		return true
	})
}

func (c *compiler) shouldInline(def *DefinitionNode) (bool, error) {
	id := c.intern(def.Name)
	if c.dryRun || !c.config.GetBool("compiler.inline.enabled") {
		return false, nil
	}
	if c.grammarNode.FirstDefinition().Name == def.Name {
		return false, nil
	}
	// Don't inline recovery/error label definitions
	if _, isRecovery := c.errorLabelIDs[id]; isRecovery {
		return false, nil
	}
	if _, isRecursive := c.defRecursiveMap[def.Name]; isRecursive {
		return false, nil
	}
	size, err := c.getDefSize(def)
	if err != nil {
		return false, err
	}
	if size > c.config.GetInt("compiler.inline.max_size") {
		return false, nil
	}
	return true, nil
}

func (c *compiler) getDefSize(def *DefinitionNode) (int, error) {
	if s, ok := c.defSizeMap[def.Name]; ok {
		return s, nil
	}

	tmpc := newCompiler(c.config)
	tmpc.dryRun = true
	tmpc.grammarNode = c.grammarNode

	if err := def.Accept(tmpc); err != nil {
		return 0, err
	}

	size := tmpc.cursor

	c.defSizeMap[def.Name] = size

	return size, nil
}

func (c *compiler) capExprSize(node AstNode) (int, bool) {
	switch n := node.(type) {
	case *CharsetNode:
		return 1, true

	case *LiteralNode:
		return len(n.Value), true

	case *ClassNode:
		val := -1
		for _, item := range n.Items {
			if is, ok := c.capExprSize(item); ok {
				if val >= 0 && val != is {
					return 0, false
				}
				val = is
			} else {
				return 0, false
			}
		}
		if val > 0 {
			return val, true
		}
		return 0, false

	case *ChoiceNode:
		left, ok := c.capExprSize(n.Left)
		if !ok {
			return 0, false
		}
		right, ok := c.capExprSize(n.Right)
		if !ok {
			return 0, false
		}
		if left == right {
			return left, true
		}
		return 0, false

	case *SequenceNode:
		var total int
		for _, item := range n.Items {
			if is, ok := c.capExprSize(item); ok {
				total += is
			} else {
				return 0, false
			}
		}
		if total > 0 {
			return total, true
		}
		return 0, false

	case *LexNode:
		return c.capExprSize(n.Expr)

	default:
		return 0, false
	}
}

type callGraph map[string]map[string]struct{}

func getIsRecursiveFromGrammar(g *GrammarNode) map[string]struct{} {
	cg := make(callGraph, len(g.Definitions))
	for _, d := range g.Definitions {
		if _, ok := cg[d.Name]; !ok {
			cg[d.Name] = make(map[string]struct{})
		}
		for _, id := range findIdentifiers(d.Expr) {
			cg[d.Name][id] = struct{}{}
		}
	}
	return cg.getIsRecursive()
}

func findIdentifiers(node AstNode) []string {
	var ids []string
	Inspect(node, func(n AstNode) bool {
		if id, ok := n.(*IdentifierNode); ok {
			ids = append(ids, id.Value)
		}
		return true
	})
	return ids
}

func (g callGraph) getIsRecursive() map[string]struct{} {
	var (
		stack   = []string{}
		onStack = map[string]bool{}
		visited = map[string]bool{}
		recurse = map[string]struct{}{}
		pos     = map[string]int{}
		dfs     func(string)
	)
	dfs = func(v string) {
		visited[v] = true
		pos[v] = len(stack)
		stack = append(stack, v)
		onStack[v] = true

		// Sort edges for deterministic traversal
		edges := make([]string, 0, len(g[v]))
		for w := range g[v] {
			edges = append(edges, w)
		}
		sort.Strings(edges)

		for _, w := range edges {
			if !visited[w] {
				dfs(w)
				continue
			}
			if onStack[w] {
				// back edge v -> w: mark the cycle path w..v
				for i := pos[w]; i < len(stack); i++ {
					recurse[stack[i]] = struct{}{}
				}
			}
		}
		stack = stack[:len(stack)-1]
		onStack[v] = false
	}

	// ensure all vertices (even isolated ones) are visited Sort
	// vertices for deterministic traversal order (Go map
	// iteration is randomized)
	vertices := make([]string, 0, len(g))
	for v := range g {
		vertices = append(vertices, v)
	}
	sort.Strings(vertices)

	for _, v := range vertices {
		if !visited[v] {
			dfs(v)
		}
	}
	return recurse
}
