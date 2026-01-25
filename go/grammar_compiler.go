package langlang

import (
	"fmt"
	"sort"
)

type compiler struct {
	config *Config

	// db is an optional query database for query-based compilation.
	// When set, the compiler will use queries for recursive checks
	// and definition size calculations.
	db *Database

	// filePath is the path of the grammar file being compiled,
	// used as a key for queries when db is set.
	filePath string

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

	dryRun bool

	// withinPredicate tracks if we're inside a predicate (&/!)
	// where captures are not needed, allowing us to emit
	// IPartialCommit instead of IPartialCommitCap
	withinPredicate bool

	// srcStack tracks source locations as we recurse into the
	// AST, allowing instructions to reference their originating
	// grammar location
	srcStack []SourceLocation
}

func newCompiler(config *Config) *compiler {
	return &compiler{
		config:           config,
		identifiers:      map[int]int{},
		definitionLabels: map[int]ILabel{},
		openAddrs:        map[int]int{},
		stringsMap:       map[string]int{},
		strings:          []string{""}, // Reserve index 0 for "no name" sentinel
		errorLabelIDs:    map[int]struct{}{},
		recovery:         map[int]recoveryEntry{},
	}
}

// newCompilerWithDB creates a compiler that uses the query database
// for recursive checks and definition size calculations.
func newCompilerWithDB(db *Database, filePath string) *compiler {
	c := newCompiler(db.Config())
	c.db = db
	c.filePath = filePath
	return c
}

// CompileWithDB compiles an AST using the query database for caching.
func CompileWithDB(db *Database, filePath string, expr AstNode) (*Program, error) {
	var err error
	c := newCompilerWithDB(db, filePath)
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
		sourceFiles: db.AllFilePaths(),
	}, nil
}

func (c *compiler) VisitGrammarNode(node *GrammarNode) error {
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
	c.emit(ICall{sl: c.currentSrc()})
	c.emit(IHalt{sl: c.currentSrc()})
	c.grammarNode = node
	c.collectErrorLabels(node)
	return WalkGrammarNode(c, node)
}

func (c *compiler) VisitImportNode(node *ImportNode) error {
	return fmt.Errorf("import isn't translatable")
}

func (c *compiler) VisitErrorNode(node *ErrorNode) error {
	return fmt.Errorf("error isn't translatable")
}

func (c *compiler) VisitDefinitionNode(node *DefinitionNode) error {
	if inline, err := c.shouldInline(node); err != nil || inline && !c.config.GetBool("compiler.inline.emit.inlined") {
		return err
	}
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()

	var (
		id = c.intern(node.Name)
		l0 = NewILabelWithSourceLocation(c.currentSrc())
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

	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
	id := c.intern(node.Name)

	if sz, ok := c.capExprSize(node.Expr); ok {
		if err := node.Expr.Accept(c); err != nil {
			return err
		}

		if node.Name == "" {
			c.emit(ICapTerm{Offset: sz, sl: c.currentSrc()})
			return nil
		}

		c.emit(ICapNonTerm{ID: id, Offset: sz, sl: c.currentSrc()})
		return nil
	}
	if c.shouldUseCapOffset(node) {
		if node.Name == "" {
			c.emit(ICapTermBeginOffset{sl: c.currentSrc()})
		} else {
			c.emit(ICapNonTermBeginOffset{ID: id, sl: c.currentSrc()})
		}
		if err := node.Expr.Accept(c); err != nil {
			return err
		}
		c.emit(ICapEndOffset{sl: c.currentSrc()})
		return nil
	}

	c.emit(ICapBegin{ID: id, sl: c.currentSrc()})

	if err := node.Expr.Accept(c); err != nil {
		return err
	}

	c.emit(ICapEnd{sl: c.currentSrc()})
	return nil
}

func (c *compiler) VisitSequenceNode(node *SequenceNode) error {
	return WalkSequenceNode(c, node)
}

func (c *compiler) VisitOneOrMoreNode(node *OneOrMoreNode) error {
	if err := node.Expr.Accept(c); err != nil {
		return err
	}
	return c.VisitZeroOrMoreNode(NewZeroOrMoreNode(node.Expr, node.SourceLocation()))
}

func (c *compiler) VisitZeroOrMoreNode(node *ZeroOrMoreNode) error {
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
	if csn, ok := node.Expr.(*CharsetNode); ok {
		c.emit(ISpan{cs: csn.cs, sl: c.currentSrc()})
		return nil
	}

	l0 := NewILabelWithSourceLocation(c.currentSrc())
	l1 := NewILabelWithSourceLocation(c.currentSrc())
	l2 := NewILabelWithSourceLocation(c.currentSrc())

	switch c.config.GetInt("compiler.optimize") {
	case 0:
		c.emit(l0)
	}

	c.emit(IChoice{Label: l2, sl: c.currentSrc()})
	c.emit(l1)

	if err := node.Expr.Accept(c); err != nil {
		return err
	}

	switch c.config.GetInt("compiler.optimize") {
	case 0:
		c.emitCommit(l0)
	default:
		if c.shouldCapture() {
			c.emit(ICapPartialCommit{Label: l1, sl: c.currentSrc()})
		} else {
			c.emit(IPartialCommit{Label: l1, sl: c.currentSrc()})
		}
	}

	c.emit(l2)

	return nil
}

func (c *compiler) VisitOptionalNode(node *OptionalNode) error {
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
	lb := NewILabelWithSourceLocation(c.currentSrc())

	c.emit(IChoice{Label: lb, sl: c.currentSrc()})

	if err := node.Expr.Accept(c); err != nil {
		return err
	}

	c.emitCommit(lb)
	c.emit(lb)
	return nil
}

func (c *compiler) VisitChoiceNode(node *ChoiceNode) error {
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
	l1 := NewILabelWithSourceLocation(c.currentSrc())
	l2 := NewILabelWithSourceLocation(c.currentSrc())

	c.emit(IChoice{Label: l1, sl: c.currentSrc()})

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
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
	switch c.config.GetInt("compiler.optimize") {
	case 0:
		return c.VisitNotNode(NewNotNode(NewNotNode(node.Expr, node.SourceLocation()), node.SourceLocation()))

	default:
		l1 := NewILabelWithSourceLocation(c.currentSrc())
		l2 := NewILabelWithSourceLocation(c.currentSrc())

		c.emit(IChoicePred{Label: l1, sl: c.currentSrc()})

		old := c.withinPredicate
		c.withinPredicate = true
		err := node.Expr.Accept(c)
		c.withinPredicate = old
		if err != nil {
			return err
		}

		c.emitBackCommit(l2)
		c.emit(l1)
		c.emit(IFail{sl: c.currentSrc()})
		c.emit(l2)
	}
	return nil
}

func (c *compiler) VisitNotNode(node *NotNode) error {
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
	l1 := NewILabelWithSourceLocation(c.currentSrc())

	c.emit(IChoicePred{Label: l1, sl: c.currentSrc()})

	old := c.withinPredicate
	c.withinPredicate = true
	err := node.Expr.Accept(c)
	c.withinPredicate = old
	if err != nil {
		return err
	}

	switch c.config.GetInt("compiler.optimize") {
	case 0:
		l2 := NewILabelWithSourceLocation(c.currentSrc())
		c.emitCommit(l2)
		c.emit(l2)
		c.emit(IFail{sl: c.currentSrc()})
	default:
		c.emit(IFailTwice{sl: c.currentSrc()})
	}

	c.emit(l1)
	return nil
}

func (c *compiler) VisitLexNode(node *LexNode) error {
	return node.Expr.Accept(c)
}

func (c *compiler) VisitLabeledNode(node *LabeledNode) error {
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
	l1 := NewILabelWithSourceLocation(c.currentSrc())
	l2 := NewILabelWithSourceLocation(c.currentSrc())
	id := c.intern(node.Label)

	c.errorLabelIDs[id] = struct{}{}
	c.emit(IChoice{Label: l1, sl: c.currentSrc()})

	if err := node.Expr.Accept(c); err != nil {
		return nil
	}

	c.emitCommit(l2)
	c.emit(l1)
	c.emit(IThrow{ErrorLabel: id, sl: c.currentSrc()})
	c.emit(l2)
	return nil
}

func (c *compiler) VisitIdentifierNode(node *IdentifierNode) error {
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
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
		c.emit(ICall{Label: label, Precedence: precedence, sl: c.currentSrc()})
	} else {
		c.saveOpenAddr(id)
		c.emit(ICall{Precedence: precedence, sl: c.currentSrc()})
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
		strsl := node.Items[i].SourceLocation()
		start := strsl.Span.Start
		end := accum.SourceLocation().Span.End
		newsl := NewSourceLocation(strsl.FileID, NewSpan(start, end))
		accum = NewChoiceNode(node.Items[i], accum, newsl)
	}

	return c.VisitChoiceNode(accum.(*ChoiceNode))
}

func (c *compiler) VisitRangeNode(node *RangeNode) error {
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
	c.emit(IRange{Lo: node.Left, Hi: node.Right, sl: c.currentSrc()})
	return nil
}

func (c *compiler) VisitCharsetNode(n *CharsetNode) error {
	c.pushSrc(n.SourceLocation())
	defer c.popSrc()
	c.emit(ISet{cs: n.cs, sl: c.currentSrc()})
	return nil
}

func (c *compiler) VisitLiteralNode(node *LiteralNode) error {
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
	for _, r := range node.Value {
		c.emit(IChar{Char: r, sl: c.currentSrc()})
	}
	return nil
}

func (c *compiler) VisitAnyNode(node *AnyNode) error {
	c.pushSrc(node.SourceLocation())
	defer c.popSrc()
	c.emit(IAny{sl: c.currentSrc()})
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
			// Preserve the source location from the original instruction
			origCall := c.code[callAddr].(ICall)
			c.code[callAddr] = ICall{Label: label, Precedence: origCall.Precedence, sl: origCall.sl}
			continue
		}
		return fmt.Errorf("production `%s` does not exist", c.strings[id])
	}

	// Patch main call - use grammar node's source location
	def := c.grammarNode.FirstDefinition()
	id := c.intern(def.Name)
	label := c.definitionLabels[id]
	origCall := c.code[0].(ICall)
	c.emitAt(0, ICall{Label: label, sl: origCall.sl})

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
		c.emit(ICapCommit{Label: label, sl: c.currentSrc()})
	} else {
		c.emit(ICommit{Label: label, sl: c.currentSrc()})
	}
}

func (c *compiler) emitBackCommit(label ILabel) {
	if c.shouldCapture() {
		c.emit(ICapBackCommit{Label: label, sl: c.currentSrc()})
	} else {
		c.emit(IBackCommit{Label: label, sl: c.currentSrc()})
	}
}

func (c *compiler) emitReturn() {
	if c.shouldCapture() {
		c.emit(ICapReturn{sl: c.currentSrc()})
	} else {
		c.emit(IReturn{sl: c.currentSrc()})
	}
}

func (c *compiler) saveOpenAddr(addr int) {
	c.openAddrs[c.cursor] = addr
}

func (c *compiler) emit(i Instruction) {
	c.code = append(c.code, i)
	c.cursor++
}

func (c *compiler) pushSrc(src SourceLocation) {
	c.srcStack = append(c.srcStack, src)
}

func (c *compiler) popSrc() {
	if len(c.srcStack) > 0 {
		c.srcStack = c.srcStack[:len(c.srcStack)-1]
	}
}

func (c *compiler) currentSrc() SourceLocation {
	if len(c.srcStack) == 0 {
		return SourceLocation{}
	}
	return c.srcStack[len(c.srcStack)-1]
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
	if def == nil {
		return false, nil
	}
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

	// Check if definition is recursive
	isRecursive, err := c.isDefRecursive(def.Name)
	if err != nil {
		return false, err
	}
	if isRecursive {
		return false, nil
	}

	// Get definition size
	size, err := c.getDefSize(def)
	if err != nil {
		return false, err
	}
	if size > c.config.GetInt("compiler.inline.max_size") {
		return false, nil
	}
	return true, nil
}

// isDefRecursive checks if a definition is recursive using the query system.
func (c *compiler) isDefRecursive(name string) (bool, error) {
	return Get(c.db, IsRecursiveQuery, DefKey{File: c.filePath, Name: name})
}

// getDefSize returns the compiled size of a definition using the query system.
func (c *compiler) getDefSize(def *DefinitionNode) (int, error) {
	return Get(c.db, DefSizeQuery, DefKey{File: c.filePath, Name: def.Name})
}

func (c *compiler) shouldUseCapOffset(cap *CaptureNode) bool {
	// Don't optimize if not syntactic
	if !isSyntactic(cap.Expr, true) {
		return false
	}

	// Don't optimize recovery expressions - they need special Error node wrapping
	if cap.Name != "" {
		id := c.intern(cap.Name)
		if _, isRecovery := c.errorLabelIDs[id]; isRecovery {
			return false
		}
	}

	// Don't optimize if contains nested captures, labels, or identifiers
	found := false
	Inspect(cap.Expr, func(n AstNode) bool {
		switch n.(type) {
		case *CaptureNode, *LabeledNode, *IdentifierNode:
			found = true
		}
		return true
	})
	return !found
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
