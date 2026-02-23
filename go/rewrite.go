package langlang

import "fmt"

// RewritePattern represents the left-hand side of a rewrite rule.
// Patterns match against trees and bind variables.
type RewritePattern interface {
	rewritePattern()
	fmt.Stringer
}

// PatWild matches any tree node without binding.
type PatWild struct{}

func (PatWild) rewritePattern() {}
func (PatWild) String() string  { return "_" }

// PatVar matches any tree node and binds it to a variable.
type PatVar struct {
	Name string
}

func (PatVar) rewritePattern()    {}
func (p PatVar) String() string   { return "?" + p.Name }

// PatStr matches a NodeType_String with specific text.
type PatStr struct {
	Text string
}

func (PatStr) rewritePattern()    {}
func (p PatStr) String() string   { return fmt.Sprintf("%q", p.Text) }

// PatNamed matches a NodeType_Node with a specific name,
// then matches the child against the body pattern.
type PatNamed struct {
	NodeName string
	Body     RewritePattern
}

func (PatNamed) rewritePattern()  {}
func (p PatNamed) String() string { return fmt.Sprintf("%s(%s)", p.NodeName, p.Body) }

// PatSeq matches a NodeType_Sequence with positional child patterns.
type PatSeq struct {
	Elems []RewritePattern
}

func (PatSeq) rewritePattern() {}
func (p PatSeq) String() string {
	s := "["
	for i, e := range p.Elems {
		if i > 0 {
			s += ", "
		}
		s += e.String()
	}
	return s + "]"
}

// RewriteConstruction represents the right-hand side of a rewrite rule.
// Constructions build new trees using variables bound during matching.
type RewriteConstruction interface {
	rewriteConstruction()
	fmt.Stringer
}

// ConVar inserts the tree bound to a variable.
type ConVar struct {
	Name string
}

func (ConVar) rewriteConstruction() {}
func (c ConVar) String() string     { return "?" + c.Name }

// ConStr builds a string node with literal text.
type ConStr struct {
	Text string
}

func (ConStr) rewriteConstruction() {}
func (c ConStr) String() string     { return fmt.Sprintf("%q", c.Text) }

// ConNamed builds a NodeType_Node wrapping a child.
type ConNamed struct {
	NodeName string
	Body     RewriteConstruction
}

func (ConNamed) rewriteConstruction() {}
func (c ConNamed) String() string {
	return fmt.Sprintf("%s(%s)", c.NodeName, c.Body)
}

// ConSeq builds a NodeType_Sequence from child constructions.
type ConSeq struct {
	Elems []RewriteConstruction
}

func (ConSeq) rewriteConstruction() {}
func (c ConSeq) String() string {
	s := "["
	for i, e := range c.Elems {
		if i > 0 {
			s += ", "
		}
		s += e.String()
	}
	return s + "]"
}

// ConCall applies another rewrite rule to a construction argument.
// e.g., expr(?e) calls the "expr" rule set on the subtree bound to ?e.
type ConCall struct {
	RuleName string
	Args     []RewriteConstruction
}

func (ConCall) rewriteConstruction() {}
func (c ConCall) String() string {
	s := c.RuleName + "("
	for i, a := range c.Args {
		if i > 0 {
			s += ", "
		}
		s += a.String()
	}
	return s + ")"
}

// ConEach maps a rewrite rule over each element of a sequence.
// e.g., each(expr, ?args) applies "expr" to each named-node child of ?args.
// Produces a sequence of the results.
type ConEach struct {
	RuleName string
	SeqArg   RewriteConstruction
}

func (ConEach) rewriteConstruction() {}
func (c ConEach) String() string {
	return fmt.Sprintf("each(%s, %s)", c.RuleName, c.SeqArg)
}

// ConLen returns the count of named-node children in a sequence as a string literal.
// e.g., len(?args) produces "2" if ?args has two elements.
type ConLen struct {
	SeqArg RewriteConstruction
}

func (ConLen) rewriteConstruction() {}
func (c ConLen) String() string {
	return fmt.Sprintf("len(%s)", c.SeqArg)
}

// ConFoldl left-folds an alternating sequence [term, op, term, op, ...]
// into nested constructor nodes. Applies a rule to each term.
// e.g., foldl(Binary, reshape_expr, ?elems) folds [a,"+",b,"-",c] into
// Binary("-", Binary("+", reshape(a), reshape(b)), reshape(c)).
// If the sequence has a single element, just applies the rule.
type ConFoldl struct {
	CtorName string
	RuleName string
	SeqArg   RewriteConstruction
}

func (ConFoldl) rewriteConstruction() {}
func (c ConFoldl) String() string {
	return fmt.Sprintf("foldl(%s, %s, %s)", c.CtorName, c.RuleName, c.SeqArg)
}

// RewriteRule pairs a pattern (LHS) with a construction (RHS).
type RewriteRule struct {
	Name    string
	Pattern RewritePattern
	Constr  RewriteConstruction
	sl      SourceLocation
}

func (r *RewriteRule) String() string {
	return fmt.Sprintf("%s <~ %s -> %s", r.Name, r.Pattern, r.Constr)
}

// RewriteRuleSet is an ordered-choice set of rewrite rules sharing a name.
type RewriteRuleSet struct {
	Name  string
	Rules []*RewriteRule
	sl    SourceLocation
}

// CompileRewriteRule compiles a single rewrite rule into bytecode
// instructions. The instructions match a tree (the pattern) and then
// construct a new tree (the construction).
func CompileRewriteRule(rule *RewriteRule, c *compiler) error {
	// Compile pattern (LHS): emit tree-matching instructions
	varTable := map[string]int{}
	nextVar := 0
	if err := compilePattern(rule.Pattern, c, varTable, &nextVar); err != nil {
		return fmt.Errorf("compiling pattern for %s: %w", rule.Name, err)
	}

	// Compile construction (RHS): emit tree-building instructions
	if err := compileConstruction(rule.Constr, c, varTable); err != nil {
		return fmt.Errorf("compiling construction for %s: %w", rule.Name, err)
	}

	return nil
}

func compilePattern(pat RewritePattern, c *compiler, vars map[string]int, nextVar *int) error {
	switch p := pat.(type) {
	case PatWild:
		c.emit(IMatchAnyNode{})

	case PatVar:
		c.emit(IMatchAnyNode{})
		id, exists := vars[p.Name]
		if exists {
			c.emit(ICheckBind{VarID: id})
		} else {
			id = *nextVar
			vars[p.Name] = id
			*nextVar++
			c.emit(IBind{VarID: id})
		}

	case PatStr:
		strID := c.intern(p.Text)
		c.emit(IMatchString{StrID: strID})

	case PatNamed:
		nameID := c.intern(p.NodeName)
		c.emit(IMatchNode{NameID: nameID})
		c.emit(IEnterChild{})
		if err := compilePattern(p.Body, c, vars, nextVar); err != nil {
			return err
		}
		c.emit(IPopCursor{})

	case PatSeq:
		c.emit(IMatchSeq{})
		if len(p.Elems) > 0 {
			c.emit(ICheckSeqLen{ExpectedLen: len(p.Elems)})
		}
		for i, elem := range p.Elems {
			c.emit(IEnterIndex{Index: i})
			if err := compilePattern(elem, c, vars, nextVar); err != nil {
				return err
			}
			c.emit(IPopCursor{})
		}

	default:
		return fmt.Errorf("unsupported pattern type: %T", pat)
	}
	return nil
}

func compileConstruction(con RewriteConstruction, c *compiler, vars map[string]int) error {
	switch co := con.(type) {
	case ConVar:
		id, exists := vars[co.Name]
		if !exists {
			return fmt.Errorf("unbound variable ?%s in construction", co.Name)
		}
		c.emit(IBuildRef{VarID: id})

	case ConStr:
		strID := c.intern(co.Text)
		c.emit(IBuildStr{StrID: strID})

	case ConNamed:
		if err := compileConstruction(co.Body, c, vars); err != nil {
			return err
		}
		nameID := c.intern(co.NodeName)
		c.emit(IBuildNode{NameID: nameID, FieldCount: 1})

	case ConSeq:
		for _, elem := range co.Elems {
			if err := compileConstruction(elem, c, vars); err != nil {
				return err
			}
		}
		c.emit(IBuildSeq{Count: len(co.Elems)})

	case ConCall:
		if len(co.Args) != 1 {
			return fmt.Errorf("ConCall %s: expected 1 arg, got %d", co.RuleName, len(co.Args))
		}
		if c.ruleSetLabels == nil {
			return fmt.Errorf("ConCall %s: ruleSetLabels not set (compile with CompileRewriteFile)", co.RuleName)
		}
		label, ok := c.ruleSetLabels[co.RuleName]
		if !ok {
			return fmt.Errorf("ConCall: unknown rule set %q", co.RuleName)
		}
		if err := compileConstruction(co.Args[0], c, vars); err != nil {
			return err
		}
		c.emit(ISetCursorFromBuild{})
		c.emit(ICall{Label: label})
		c.emit(IPopCursor{})

	case ConEach:
		if c.ruleSetLabels == nil {
			return fmt.Errorf("ConEach %s: ruleSetLabels not set (compile with CompileRewriteFile)", co.RuleName)
		}
		label, ok := c.ruleSetLabels[co.RuleName]
		if !ok {
			return fmt.Errorf("ConEach: unknown rule set %q", co.RuleName)
		}
		if err := compileConstruction(co.SeqArg, c, vars); err != nil {
			return err
		}
		c.emit(IBuildEach{Label: label})

	case ConLen:
		if err := compileConstruction(co.SeqArg, c, vars); err != nil {
			return err
		}
		c.emit(IBuildLen{})

	case ConFoldl:
		if c.ruleSetLabels == nil {
			return fmt.Errorf("ConFoldl %s: ruleSetLabels not set (compile with CompileRewriteFile)", co.RuleName)
		}
		label, ok := c.ruleSetLabels[co.RuleName]
		if !ok {
			return fmt.Errorf("ConFoldl: unknown rule set %q", co.RuleName)
		}
		if err := compileConstruction(co.SeqArg, c, vars); err != nil {
			return err
		}
		ctorNameID := c.intern(co.CtorName)
		c.emit(IBuildFoldl{CtorNameID: ctorNameID, Label: label})

	default:
		return fmt.Errorf("unsupported construction type: %T", con)
	}
	return nil
}

// CompileRewriteRuleSet compiles an ordered-choice set of rewrite rules.
// Each rule is tried in order; if its pattern fails, the next is tried.
func CompileRewriteRuleSet(ruleSet *RewriteRuleSet, c *compiler) error {
	endLabel := NewILabel()

	for i, rule := range ruleSet.Rules {
		var failLabel ILabel
		isLast := i == len(ruleSet.Rules)-1

		if !isLast {
			failLabel = NewILabel()
			c.emit(IChoice{Label: failLabel})
		}

		if err := CompileRewriteRule(rule, c); err != nil {
			return err
		}

		if !isLast {
			c.emit(ICommit{Label: endLabel})
			c.emit(failLabel)
		}
	}

	c.emit(endLabel)
	return nil
}

// Vars collects all variable names mentioned in a pattern.
func PatternVars(p RewritePattern) []string {
	var result []string
	seen := map[string]bool{}
	collectPatVars(p, &result, seen)
	return result
}

func collectPatVars(p RewritePattern, result *[]string, seen map[string]bool) {
	switch pp := p.(type) {
	case PatVar:
		if !seen[pp.Name] {
			seen[pp.Name] = true
			*result = append(*result, pp.Name)
		}
	case PatNamed:
		collectPatVars(pp.Body, result, seen)
	case PatSeq:
		for _, e := range pp.Elems {
			collectPatVars(e, result, seen)
		}
	}
}

// ConstructionVars collects all variable names mentioned in a construction.
func ConstructionVars(c RewriteConstruction) []string {
	var result []string
	seen := map[string]bool{}
	collectConVars(c, &result, seen)
	return result
}

func collectConVars(c RewriteConstruction, result *[]string, seen map[string]bool) {
	switch cc := c.(type) {
	case ConVar:
		if !seen[cc.Name] {
			seen[cc.Name] = true
			*result = append(*result, cc.Name)
		}
	case ConNamed:
		collectConVars(cc.Body, result, seen)
	case ConSeq:
		for _, e := range cc.Elems {
			collectConVars(e, result, seen)
		}
	case ConCall:
		for _, a := range cc.Args {
			collectConVars(a, result, seen)
		}
	case ConEach:
		collectConVars(cc.SeqArg, result, seen)
	case ConLen:
		collectConVars(cc.SeqArg, result, seen)
	case ConFoldl:
		collectConVars(cc.SeqArg, result, seen)
	}
}

// CheckWellFormed verifies that all variables used in the construction
// are bound by the pattern (static well-formedness from the Alloy model).
func CheckWellFormed(rule *RewriteRule) error {
	patVars := map[string]bool{}
	for _, v := range PatternVars(rule.Pattern) {
		patVars[v] = true
	}
	for _, v := range ConstructionVars(rule.Constr) {
		if !patVars[v] {
			return fmt.Errorf("variable ?%s used in construction but not bound in pattern", v)
		}
	}
	return nil
}

// CompileRewriteFile compiles all rule sets in rf into a single bytecode blob.
// entryRuleName is the rule set whose code is placed at address 0 (the rewrite entry point).
// Returns the bytecode and a map from rule set name to bytecode address for use with RewriteAt.
func CompileRewriteFile(rf *RewriteFile, entryRuleName string) (*Bytecode, map[string]int, error) {
	// Order rule sets: entry first, then the rest
	var ordered []*RewriteRuleSet
	var entryIndex int
	for i, rs := range rf.RuleSets {
		if rs.Name == entryRuleName {
			entryIndex = i
			break
		}
	}
	if entryIndex >= len(rf.RuleSets) {
		return nil, nil, fmt.Errorf("entry rule set %q not found in rewrite file", entryRuleName)
	}
	ordered = append(ordered, rf.RuleSets[entryIndex])
	for i, rs := range rf.RuleSets {
		if i != entryIndex {
			ordered = append(ordered, rs)
		}
	}

	c := newCompiler(NewConfig())
	c.ruleSetLabels = make(map[string]ILabel)
	for _, rs := range ordered {
		c.ruleSetLabels[rs.Name] = NewILabel()
	}

	haltLabel := NewILabel()
	for i, rs := range ordered {
		c.emit(c.ruleSetLabels[rs.Name])
		if err := CompileRewriteRuleSet(rs, c); err != nil {
			return nil, nil, fmt.Errorf("compiling rule set %s: %w", rs.Name, err)
		}
		c.emit(IReturn{})
		if i == 0 {
			// Entry rule set: IReturn pops to this halt so Rewrite can start at 0 without a frame
			c.emit(haltLabel)
			c.emit(IHalt{})
		}
	}

	// Ensure "Spacing" is in the string table so bytecode can skip Spacing nodes in each/foldl
	spacingID := c.intern("Spacing")
	p := &Program{
		code:               c.code,
		strings:            c.strings,
		stringsMap:         c.stringsMap,
		SpacingNameID:      int32(spacingID),
		RewriteHaltLabel:   haltLabel,
	}
	cfg := NewConfig()
	bytecode := Encode(p, cfg)

	// Build label -> address from the encoded code (labels were resolved during Encode)
	labels := make(map[ILabel]int)
	addr := 0
	for _, inst := range c.code {
		switch ii := inst.(type) {
		case ILabel:
			labels[ii] = addr
		default:
			addr += inst.SizeInBytes()
		}
	}
	ruleAddrs := make(map[string]int, len(ordered))
	for _, rs := range ordered {
		ruleAddrs[rs.Name] = labels[c.ruleSetLabels[rs.Name]]
	}

	return bytecode, ruleAddrs, nil
}

// CompileRewriteFileWithStrategy compiles all rule sets and places the strategy
// code at address 0, so that Rewrite runs the strategy (e.g. innermost(fold)).
// Use this when the entry point is a strategy rather than a single rule set.
func CompileRewriteFileWithStrategy(rf *RewriteFile, strat Strategy) (*Bytecode, error) {
	c := newCompiler(NewConfig())
	c.ruleSetLabels = make(map[string]ILabel)
	for _, rs := range rf.RuleSets {
		c.ruleSetLabels[rs.Name] = NewILabel()
	}
	// Emit strategy at 0, then all rule sets
	entryLabel := NewILabel()
	c.emit(entryLabel)
	if err := CompileStrategy(strat, c); err != nil {
		return nil, fmt.Errorf("compiling strategy: %w", err)
	}
	c.emit(IHalt{})
	for _, rs := range rf.RuleSets {
		c.emit(c.ruleSetLabels[rs.Name])
		if err := CompileRewriteRuleSet(rs, c); err != nil {
			return nil, fmt.Errorf("compiling rule set %s: %w", rs.Name, err)
		}
		c.emit(IReturn{})
	}
	spacingID := c.intern("Spacing")
	p := &Program{
		code:          c.code,
		strings:       c.strings,
		stringsMap:    c.stringsMap,
		SpacingNameID: int32(spacingID),
	}
	return Encode(p, NewConfig()), nil
}

// RuleSetByName returns the rule set with the given name in rf, or nil.
func RuleSetByName(rf *RewriteFile, name string) *RewriteRuleSet {
	for _, rs := range rf.RuleSets {
		if rs.Name == name {
			return rs
		}
	}
	return nil
}

// --- Strategy types ---

// Strategy represents a tree traversal/transformation strategy.
type Strategy interface {
	strategy()
	fmt.Stringer
}

// StratId is the identity strategy: always succeeds, returns the input.
type StratId struct{}

func (StratId) strategy()    {}
func (StratId) String() string { return "id" }

// StratFail always fails.
type StratFail struct{}

func (StratFail) strategy()    {}
func (StratFail) String() string { return "fail" }

// StratLift applies a rewrite rule set at the current node.
type StratLift struct {
	RuleSet *RewriteRuleSet
}

func (StratLift) strategy()      {}
func (s StratLift) String() string { return s.RuleSet.Name }

// StratSeq applies s1 then s2 on s1's result.
type StratSeq struct {
	First, Second Strategy
}

func (StratSeq) strategy()      {}
func (s StratSeq) String() string { return fmt.Sprintf("(%s ; %s)", s.First, s.Second) }

// StratChoice tries s1; if it fails, tries s2.
type StratChoice struct {
	Left, Right Strategy
}

func (StratChoice) strategy()      {}
func (s StratChoice) String() string { return fmt.Sprintf("(%s <+ %s)", s.Left, s.Right) }

// StratNot succeeds (with the input) if s fails, fails if s succeeds.
type StratNot struct {
	Inner Strategy
}

func (StratNot) strategy()      {}
func (s StratNot) String() string { return fmt.Sprintf("not(%s)", s.Inner) }

// StratTry applies s; if it fails, succeeds with the input.
// Equivalent to choice(s, id).
type StratTry struct {
	Inner Strategy
}

func (StratTry) strategy()      {}
func (s StratTry) String() string { return fmt.Sprintf("try(%s)", s.Inner) }

// StratRepeat applies s until it fails.
type StratRepeat struct {
	Inner Strategy
}

func (StratRepeat) strategy()      {}
func (s StratRepeat) String() string { return fmt.Sprintf("repeat(%s)", s.Inner) }

// StratTopDown applies s at each node, root to leaves.
type StratTopDown struct {
	Inner Strategy
}

func (StratTopDown) strategy()      {}
func (s StratTopDown) String() string { return fmt.Sprintf("topdown(%s)", s.Inner) }

// StratBottomUp applies s at each node, leaves to root.
type StratBottomUp struct {
	Inner Strategy
}

func (StratBottomUp) strategy()      {}
func (s StratBottomUp) String() string { return fmt.Sprintf("bottomup(%s)", s.Inner) }

// StratInnermost applies s bottom-up repeatedly until no more changes.
// Equivalent to repeat(bottomup(try(s))).
type StratInnermost struct {
	Inner Strategy
}

func (StratInnermost) strategy()      {}
func (s StratInnermost) String() string { return fmt.Sprintf("innermost(%s)", s.Inner) }

// --- Strategy compilation ---
// Strategies compile to bytecode using the existing backtracking machinery.
// The key insight: a strategy that "fails" uses the PEG fail/backtrack path.

// CompileStrategy emits bytecode for a strategy. The strategy operates
// on the tree cursor; on success, the build stack has the result.
func CompileStrategy(strat Strategy, c *compiler) error {
	switch s := strat.(type) {
	case StratId:
		// Push the current cursor as-is onto build stack
		c.emit(IBuildCopy{})

	case StratFail:
		c.emit(IFail{})

	case StratLift:
		return CompileRewriteRuleSet(s.RuleSet, c)

	case StratSeq:
		// Apply first; if it succeeds, apply second to the result.
		// The first strategy leaves a result on the build stack.
		// For now, sequential composition means: run first (modifies
		// build stack), then run second.
		if err := CompileStrategy(s.First, c); err != nil {
			return err
		}
		return CompileStrategy(s.Second, c)

	case StratChoice:
		failLabel := NewILabel()
		endLabel := NewILabel()
		c.emit(IChoice{Label: failLabel})
		if err := CompileStrategy(s.Left, c); err != nil {
			return err
		}
		c.emit(ICommit{Label: endLabel})
		c.emit(failLabel)
		if err := CompileStrategy(s.Right, c); err != nil {
			return err
		}
		c.emit(endLabel)

	case StratNot:
		// not(s): succeed (with input) if s fails; fail if s succeeds.
		// Compiled as: choice_pred L_fail; s; fail_twice; L_fail: build_copy
		failLabel := NewILabel()
		c.emit(IChoicePred{Label: failLabel})
		if err := CompileStrategy(s.Inner, c); err != nil {
			return err
		}
		c.emit(IFailTwice{})
		c.emit(failLabel)
		c.emit(IBuildCopy{})

	case StratTry:
		// try(s) = choice(s, id)
		return CompileStrategy(StratChoice{Left: s.Inner, Right: StratId{}}, c)

	case StratRepeat:
		// repeat(s): apply s until it fails.
		// Compiled as a loop: L_start: choice L_end; s; commit L_start; L_end:
		startLabel := NewILabel()
		endLabel := NewILabel()
		c.emit(startLabel)
		c.emit(IChoice{Label: endLabel})
		if err := CompileStrategy(s.Inner, c); err != nil {
			return err
		}
		c.emit(ICommit{Label: startLabel})
		c.emit(endLabel)

	case StratTopDown:
		// topdown(s) = seq(s, all(topdown(s)))
		// Since we can't compile infinite recursion, we emit a
		// call-based recursive pattern:
		//   L_td: s; for_each_child L_td
		tdLabel := NewILabel()
		c.emit(tdLabel)
		if err := CompileStrategy(s.Inner, c); err != nil {
			return err
		}
		c.emit(IForEachChild{Label: tdLabel})
		c.emit(IReturn{})

	case StratBottomUp:
		// bottomup(s) = seq(all(bottomup(s)), s)
		//   L_bu: for_each_child L_bu; s
		buLabel := NewILabel()
		c.emit(buLabel)
		c.emit(IForEachChild{Label: buLabel})
		if err := CompileStrategy(s.Inner, c); err != nil {
			return err
		}
		c.emit(IReturn{})

	case StratInnermost:
		// innermost(s) = repeat(bottomup(try(s)))
		return CompileStrategy(
			StratRepeat{Inner: StratBottomUp{Inner: StratTry{Inner: s.Inner}}},
			c,
		)

	default:
		return fmt.Errorf("unsupported strategy type: %T", strat)
	}
	return nil
}
