package langlang

import "sort"

// TransformedGrammarQuery applies all grammar transformations
// (builtins, charsets, whitespace, captures) to a resolved grammar.
//
// This orchestrates the transformation pipeline:
//
//   - ResolvedImportsQuery
//   - WithCharsetsQuery
//   - WithWhitespaceQuery
//   - WithCapturesQuery
var TransformedGrammarQuery = &Query[FilePath, *GrammarNode]{
	Name:    "TransformedGrammar",
	Compute: computeTransformedGrammar,
}

func computeTransformedGrammar(db *Database, key FilePath) (*GrammarNode, error) {
	// The final transformation is WithCapturesQuery, which chains through
	// all previous transformations via their dependencies.
	return Get(db, WithCapturesQuery, key)
}

// WithCharsetsQuery optimizes character classes into efficient charsets.
// Depends on ResolvedImportsQuery.
var WithCharsetsQuery = &Query[FilePath, *GrammarNode]{
	Name:    "WithCharsets",
	Compute: computeWithCharsets,
}

func computeWithCharsets(db *Database, key FilePath) (*GrammarNode, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}
	if !db.Config().GetBool("grammar.add_charsets") {
		return grammar, nil
	}
	return AddCharsets(grammar)
}

// WithWhitespaceQuery injects automatic whitespace handling.
// Depends on WithCharsetsQuery.
var WithWhitespaceQuery = &Query[FilePath, *GrammarNode]{
	Name:    "WithWhitespace",
	Compute: computeWithWhitespace,
}

func computeWithWhitespace(db *Database, key FilePath) (*GrammarNode, error) {
	grammar, err := Get(db, WithCharsetsQuery, key)
	if err != nil {
		return nil, err
	}
	if !db.Config().GetBool("grammar.handle_spaces") {
		return grammar, nil
	}
	result, err := InjectWhitespaces(grammar)
	if err != nil {
		return nil, err
	}
	return result.(*GrammarNode), nil
}

// WithCapturesQuery adds capture nodes for AST construction.
// Depends on WithWhitespaceQuery. This is the final transformation.
var WithCapturesQuery = &Query[FilePath, *GrammarNode]{
	Name:    "WithCaptures",
	Compute: computeWithCaptures,
}

func computeWithCaptures(db *Database, key FilePath) (*GrammarNode, error) {
	grammar, err := Get(db, WithWhitespaceQuery, key)
	if err != nil {
		return nil, err
	}
	if !db.Config().GetBool("grammar.captures") {
		return grammar, nil
	}
	// AddCaptures returns (*GrammarNode, error) directly
	return AddCaptures(grammar, db.Config())
}

// RecursiveSetQuery computes the set of all recursive definitions in
// a grammar file.
var RecursiveSetQuery = &Query[FilePath, map[string]struct{}]{
	Name:    "RecursiveSet",
	Compute: computeRecursiveSet,
}

func computeRecursiveSet(db *Database, key FilePath) (map[string]struct{}, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}
	return getIsRecursiveFromGrammar(grammar), nil
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

// IsRecursiveQuery determines if a definition is recursive (directly
// or indirectly calls itself).
var IsRecursiveQuery = &Query[DefKey, bool]{
	Name:    "IsRecursive",
	Compute: computeIsRecursive,
}

func computeIsRecursive(db *Database, key DefKey) (bool, error) {
	recursiveSet, err := Get(db, RecursiveSetQuery, FilePath(key.File))
	if err != nil {
		return false, err
	}
	_, isRecursive := recursiveSet[key.Name]
	return isRecursive, nil
}

// Left Recursion Analysis Queries

// LeftRecursiveSetQuery computes the set of all left-recursive
// definitions in a grammar file, including indirect left recursion.
// A rule is left-recursive if it (directly or through other rules)
// can call itself as the first element of some alternative.
var LeftRecursiveSetQuery = &Query[FilePath, map[string]struct{}]{
	Name:    "LeftRecursiveSet",
	Compute: computeLeftRecursiveSet,
}

func computeLeftRecursiveSet(db *Database, key FilePath) (map[string]struct{}, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}
	nullableSet, err := Get(db, NullableRulesQuery, key)
	if err != nil {
		return nil, err
	}
	return getLeftRecursiveFromGrammar(grammar, nullableSet), nil
}

// getLeftRecursiveFromGrammar builds a "left-call graph" where an
// edge from A to B means A can start with a call to B in at least one
// alternative (considering nullable prefixes). Then finds all cycles
// in this graph - any rule in a cycle is left-recursive.
func getLeftRecursiveFromGrammar(g *GrammarNode, nullableSet map[string]struct{}) map[string]struct{} {
	lcg := make(callGraph, len(g.Definitions))
	for _, d := range g.Definitions {
		if _, ok := lcg[d.Name]; !ok {
			lcg[d.Name] = make(map[string]struct{})
		}
		// For each alternative, find ALL potential left-calls
		// (including those after nullable prefixes)
		for _, alt := range flattenChoices(d.Expr) {
			for _, call := range getLeftCalls(alt, g, nullableSet) {
				lcg[d.Name][call] = struct{}{}
			}
		}
	}
	// Reuse the same cycle detection algorithm used for RecursiveSetQuery
	return lcg.getIsRecursive()
}

// getLeftCalls returns all rule names that could be "first" in an
// expression, considering nullable prefixes. For example, in "B? A",
// both B and A are potential left-calls because B? can match empty.
func getLeftCalls(node AstNode, g *GrammarNode, nullableSet map[string]struct{}) []string {
	var calls []string
	collectLeftCalls(node, g, nullableSet, &calls)
	return calls
}

// collectLeftCalls recursively collects all potential left-calls.
func collectLeftCalls(node AstNode, g *GrammarNode, nullableSet map[string]struct{}, calls *[]string) {
	switch n := node.(type) {
	case *SequenceNode:
		// For sequences, collect calls from each item until we hit
		// a non-nullable item
		for _, item := range n.Items {
			collectLeftCalls(item, g, nullableSet, calls)
			if !isNullableWithSet(item, nullableSet) {
				break
			}
		}
	case *IdentifierNode:
		*calls = append(*calls, n.Value)
	case *PrecedenceNode:
		if id, ok := n.Expr.(*IdentifierNode); ok {
			*calls = append(*calls, id.Value)
		}
	case *LexNode:
		collectLeftCalls(n.Expr, g, nullableSet, calls)
	case *CaptureNode:
		collectLeftCalls(n.Expr, g, nullableSet, calls)
	case *LabeledNode:
		collectLeftCalls(n.Expr, g, nullableSet, calls)
	case *ZeroOrMoreNode:
		collectLeftCalls(n.Expr, g, nullableSet, calls)
	case *OneOrMoreNode:
		collectLeftCalls(n.Expr, g, nullableSet, calls)
	case *OptionalNode:
		collectLeftCalls(n.Expr, g, nullableSet, calls)
	case *AndNode:
		// Lookahead doesn't consume input but we should still
		// check what's after it
	case *NotNode:
		// Not doesn't consume input either
	}
}

// NullableRulesQuery computes the set of all nullable rule names in a
// grammar using a fixed-point iteration.  A rule is nullable if its
// body can match the empty string, considering transitive rule
// references of arbitrary depth.
var NullableRulesQuery = &Query[FilePath, map[string]struct{}]{
	Name:    "NullableRules",
	Compute: computeNullableRules,
}

func computeNullableRules(db *Database, key FilePath) (map[string]struct{}, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}
	return computeNullableSet(grammar), nil
}

// computeNullableSet computes the set of all nullable rule names
// using a fixed-point iteration.  On each pass it checks every
// definition body against the current set; whenever a new rule
// becomes nullable, another pass is triggered.  Converges in at most
// N iterations (N = number of rules here) because nullability is
// monotone, as it only moves from false to true.
func computeNullableSet(g *GrammarNode) map[string]struct{} {
	nullableSet := make(map[string]struct{})
	for {
		changed := false
		for _, def := range g.Definitions {
			if _, already := nullableSet[def.Name]; already {
				continue
			}
			if isNullableWithSet(def.Expr, nullableSet) {
				nullableSet[def.Name] = struct{}{}
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return nullableSet
}

// isNullableWithSet returns true if the expression can match the
// empty string.  Rule references are resolved by looking up the
// pre-computed nullableSet instead of recursing into rule bodies,
// which avoids infinite recursion and handles chains of any depth.
func isNullableWithSet(node AstNode, nullableSet map[string]struct{}) bool {
	switch n := node.(type) {
	case *AndNode:
		return true
	case *NotNode:
		return true
	case *OptionalNode:
		return true
	case *ZeroOrMoreNode:
		return true
	case *OneOrMoreNode:
		// e+ is nullable iff e is nullable (at least one
		// match is required, but that match can be empty)
		return isNullableWithSet(n.Expr, nullableSet)
	case *SequenceNode:
		// Sequence is nullable if ALL items are nullable
		for _, item := range n.Items {
			if !isNullableWithSet(item, nullableSet) {
				return false
			}
		}
		return true
	case *ChoiceNode:
		// Choice is nullable if ANY alternative is nullable
		return isNullableWithSet(n.Left, nullableSet) || isNullableWithSet(n.Right, nullableSet)
	case *LexNode:
		return isNullableWithSet(n.Expr, nullableSet)
	case *CaptureNode:
		return isNullableWithSet(n.Expr, nullableSet)
	case *LabeledNode:
		return isNullableWithSet(n.Expr, nullableSet)
	case *IdentifierNode:
		_, ok := nullableSet[n.Value]
		return ok
	default:
		return false
	}
}

// AlwaysSucceedsRulesQuery computes the set of all rule names whose
// body always succeeds regardless of input.
//
// A rule always succeeds if its body is composed entirely of
// expressions that cannot fail: e?, e*, etc.
//
// This is used to distinguish between:
//
//   - definitive infinite loops: body always succeeds creates loops
//     that can never exit
//
//   - input-dependent infinite loops: body can fail, which might
//     cause loops to exit.
var AlwaysSucceedsRulesQuery = &Query[FilePath, map[string]struct{}]{
	Name:    "AlwaysSucceedsRules",
	Compute: computeAlwaysSucceedsRules,
}

func computeAlwaysSucceedsRules(db *Database, key FilePath) (map[string]struct{}, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}
	return computeAlwaysSucceedsSet(grammar), nil
}

// computeAlwaysSucceedsSet computes the set of all rule names whose
// body always succeeds using a fixed-point iteration, analogous to
// computeNullableSet.
func computeAlwaysSucceedsSet(g *GrammarNode) map[string]struct{} {
	asSet := make(map[string]struct{})
	for {
		changed := false
		for _, def := range g.Definitions {
			if _, already := asSet[def.Name]; already {
				continue
			}
			if alwaysSucceeds(def.Expr, asSet) {
				asSet[def.Name] = struct{}{}
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return asSet
}

// alwaysSucceeds returns true if the expression can never fail,
// regardless of input.
func alwaysSucceeds(node AstNode, asSet map[string]struct{}) bool {
	switch n := node.(type) {
	case *AndNode:
		return false
	case *NotNode:
		return false
	case *OptionalNode:
		return true
	case *ZeroOrMoreNode:
		return true
	case *OneOrMoreNode:
		// e+ succeeds iff the first attempt succeeds
		return alwaysSucceeds(n.Expr, asSet)
	case *SequenceNode:
		for _, item := range n.Items {
			if !alwaysSucceeds(item, asSet) {
				return false
			}
		}
		return true
	case *ChoiceNode:
		return alwaysSucceeds(n.Left, asSet) || alwaysSucceeds(n.Right, asSet)
	case *LexNode:
		return alwaysSucceeds(n.Expr, asSet)
	case *CaptureNode:
		return alwaysSucceeds(n.Expr, asSet)
	case *LabeledNode:
		return alwaysSucceeds(n.Expr, asSet)
	case *IdentifierNode:
		_, ok := asSet[n.Value]
		return ok
	default:
		return false
	}
}

// Infinite Loop Detection

// InfiniteLoopRisk describes a repetition operator whose body can
// match the empty string, which may cause an infinite loop at
// runtime.
//
// Definitive means the body always succeeds, so the loop can never
// exit, so this is an unconditional infinite loop.  Non-definitive
// means the body can fail on some inputs, so the loop might exit,
// which is a potential infinite loop on certain inputs.
type InfiniteLoopRisk struct {
	DefName    string         // which definition contains the loop
	Location   SourceLocation // location of the * or + node
	Operator   string         // "*" or "+"
	BodyExpr   string         // String() of the nullable body for the message
	ViaRule    string         // if non-empty, the rule reference that makes the body nullable
	Definitive bool           // true if the body always succeeds (loop can never exit)
}

// InfiniteLoopRisksQuery detects repetition operators (* and +) whose
// body is nullable (can match the empty string).  Definitive findings
// (body always succeeds) are errors; non-definitive findings (body
// can fail) are warnings.
var InfiniteLoopRisksQuery = &Query[FilePath, []InfiniteLoopRisk]{
	Name:    "InfiniteLoopRisks",
	Compute: computeInfiniteLoopRisks,
}

func computeInfiniteLoopRisks(db *Database, key FilePath) ([]InfiniteLoopRisk, error) {
	grammar, err := Get(db, ResolvedImportsQuery, key)
	if err != nil {
		return nil, err
	}
	nullableSet, err := Get(db, NullableRulesQuery, key)
	if err != nil {
		return nil, err
	}
	asSet, err := Get(db, AlwaysSucceedsRulesQuery, key)
	if err != nil {
		return nil, err
	}
	var risks []InfiniteLoopRisk
	for _, def := range grammar.Definitions {
		Inspect(def.Expr, func(n AstNode) bool {
			var body AstNode
			var op string
			switch x := n.(type) {
			case *ZeroOrMoreNode:
				body, op = x.Expr, "*"
			case *OneOrMoreNode:
				body, op = x.Expr, "+"
			default:
				return true
			}
			if !isNullableWithSet(body, nullableSet) {
				return true
			}
			risk := InfiniteLoopRisk{
				DefName:    def.Name,
				Location:   n.SourceLocation(),
				Operator:   op,
				BodyExpr:   body.String(),
				Definitive: alwaysSucceeds(body, asSet),
			}
			// If the body is not nullable with an empty
			// set, then some rule reference is involved:
			// find the first nullable identifier to
			// populate ViaRule.
			emptySet := map[string]struct{}{}
			if !isNullableWithSet(body, emptySet) {
				risk.ViaRule = firstNullableRef(body, nullableSet)
			}
			risks = append(risks, risk)
			return true
		})
	}
	return risks, nil
}

// firstNullableRef finds the first IdentifierNode within node whose
// name is in nullableSet.  Used to populate the ViaRule field in
// diagnostics for cross-rule nullable bodies.
func firstNullableRef(node AstNode, nullableSet map[string]struct{}) string {
	var found string
	Inspect(node, func(n AstNode) bool {
		if found != "" {
			return false
		}
		if id, ok := n.(*IdentifierNode); ok {
			if _, isNullable := nullableSet[id.Value]; isNullable {
				found = id.Value
				return false
			}
		}
		return true
	})
	return found
}

var IsLeftRecursiveQuery = &Query[DefKey, bool]{
	Name:    "IsLeftRecursive",
	Compute: computeIsLeftRecursive,
}

func computeIsLeftRecursive(db *Database, key DefKey) (bool, error) {
	leftRecursiveSet, err := Get(db, LeftRecursiveSetQuery, FilePath(key.File))
	if err != nil {
		return false, err
	}
	_, isLeftRecursive := leftRecursiveSet[key.Name]
	return isLeftRecursive, nil
}

func flattenChoices(node AstNode) []AstNode {
	var (
		alternatives []AstNode
		flatten      func(n AstNode)
	)
	flatten = func(n AstNode) {
		switch x := n.(type) {
		case *ChoiceNode:
			flatten(x.Left)
			flatten(x.Right)
		case *CaptureNode:
			flatten(x.Expr)
		case *LexNode:
			flatten(x.Expr)
		default:
			alternatives = append(alternatives, n)
		}
	}
	flatten(node)
	return alternatives
}

// getFirstExpr returns the first "terminal" or identifier in an
// expression, skipping through sequences and other structural nodes.
func getFirstExpr(node AstNode) AstNode {
	switch n := node.(type) {
	case *SequenceNode:
		if len(n.Items) > 0 {
			return getFirstExpr(n.Items[0])
		}
		return nil
	case *LexNode:
		return getFirstExpr(n.Expr)
	case *CaptureNode:
		return getFirstExpr(n.Expr)
	case *LabeledNode:
		return getFirstExpr(n.Expr)
	default:
		return node
	}
}

// DefSizeQuery computes the compiled size (in instructions) of a
// definition, used for inlining decisions.
var DefSizeQuery = &Query[DefKey, int]{
	Name:    "DefSize",
	Compute: computeDefSize,
}

func computeDefSize(db *Database, key DefKey) (int, error) {
	// Get the transformed grammar
	grammar, err := Get(db, TransformedGrammarQuery, FilePath(key.File))
	if err != nil {
		return 0, err
	}

	def, ok := grammar.DefsByName[key.Name]
	if !ok {
		return 0, nil
	}

	// Create a temporary compiler in dry-run mode to count
	// instructions
	file := db.FileIDToPath(def.SourceLocation().FileID)
	tmpc := newCompilerWithDB(db, file)
	tmpc.dryRun = true
	tmpc.grammarNode = grammar

	if err := def.Accept(tmpc); err != nil {
		return 0, err
	}

	return tmpc.cursor, nil
}

// Compilation Queries

// CompiledProgramQuery compiles a grammar file to a Program (IR).
var CompiledProgramQuery = &Query[FilePath, *Program]{
	Name:    "CompiledProgram",
	Compute: computeCompiledProgram,
}

func computeCompiledProgram(db *Database, key FilePath) (*Program, error) {
	// Get the transformed grammar
	grammar, err := Get(db, TransformedGrammarQuery, key)
	if err != nil {
		return nil, err
	}

	// Compile using the query-aware compiler
	return CompileWithDB(db, string(key), grammar)
}

// EncodedBytecodeQuery encodes a compiled program to bytecode.
var EncodedBytecodeQuery = &Query[FilePath, *Bytecode]{
	Name:    "EncodedBytecode",
	Compute: computeEncodedBytecode,
}

func computeEncodedBytecode(db *Database, key FilePath) (*Bytecode, error) {
	// Get the compiled program
	program, err := Get(db, CompiledProgramQuery, key)
	if err != nil {
		return nil, err
	}

	// Encode using the existing Encode function
	return Encode(program, db.Config()), nil
}
