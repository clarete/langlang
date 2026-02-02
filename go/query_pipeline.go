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
	return getLeftRecursiveFromGrammar(grammar), nil
}

// getLeftRecursiveFromGrammar builds a "left-call graph" where an
// edge from A to B means A can start with a call to B in at least one
// alternative (considering nullable prefixes). Then finds all cycles
// in this graph - any rule in a cycle is left-recursive.
func getLeftRecursiveFromGrammar(g *GrammarNode) map[string]struct{} {
	lcg := make(callGraph, len(g.Definitions))
	for _, d := range g.Definitions {
		if _, ok := lcg[d.Name]; !ok {
			lcg[d.Name] = make(map[string]struct{})
		}
		// For each alternative, find ALL potential left-calls
		// (including those after nullable prefixes)
		for _, alt := range flattenChoices(d.Expr) {
			for _, call := range getLeftCalls(alt, g) {
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
func getLeftCalls(node AstNode, g *GrammarNode) []string {
	var calls []string
	collectLeftCalls(node, g, &calls)
	return calls
}

// collectLeftCalls recursively collects all potential left-calls.
func collectLeftCalls(node AstNode, g *GrammarNode, calls *[]string) {
	switch n := node.(type) {
	case *SequenceNode:
		// For sequences, collect calls from each item until we hit
		// a non-nullable item
		for _, item := range n.Items {
			collectLeftCalls(item, g, calls)
			if !isNullable(item, g) {
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
		collectLeftCalls(n.Expr, g, calls)
	case *CaptureNode:
		collectLeftCalls(n.Expr, g, calls)
	case *LabeledNode:
		collectLeftCalls(n.Expr, g, calls)
	case *ZeroOrMoreNode:
		collectLeftCalls(n.Expr, g, calls)
	case *OneOrMoreNode:
		collectLeftCalls(n.Expr, g, calls)
	case *OptionalNode:
		collectLeftCalls(n.Expr, g, calls)
	case *AndNode:
		// Lookahead doesn't consume input but we should still
		// check what's after it
	case *NotNode:
		// Not doesn't consume input either
	}
}

// isNullable returns true if the expression can match the empty string.
func isNullable(node AstNode, g *GrammarNode) bool {
	switch n := node.(type) {
	case *ZeroOrMoreNode:
		return true // * always nullable
	case *OptionalNode:
		return true // ? always nullable
	case *AndNode:
		return true // & doesn't consume input
	case *NotNode:
		return true // ! doesn't consume input
	case *SequenceNode:
		// Sequence is nullable if ALL items are nullable
		for _, item := range n.Items {
			if !isNullable(item, g) {
				return false
			}
		}
		return true
	case *ChoiceNode:
		// Choice is nullable if ANY alternative is nullable
		return isNullable(n.Left, g) || isNullable(n.Right, g)
	case *LexNode:
		return isNullable(n.Expr, g)
	case *CaptureNode:
		return isNullable(n.Expr, g)
	case *LabeledNode:
		return isNullable(n.Expr, g)
	case *IdentifierNode:
		// Rule reference - check if the referenced rule is nullable
		// To avoid infinite recursion, we use a simple heuristic:
		// assume rules are NOT nullable unless trivially so
		if def, ok := g.DefsByName[n.Value]; ok {
			return isNullableSimple(def.Expr)
		}
		return false
	case *OneOrMoreNode:
		return false // + requires at least one match
	default:
		// Terminals (Literal, Char, Range, Any, etc.) are not nullable
		return false
	}
}

// isNullableSimple checks nullability without following rule references
// to avoid infinite recursion. Used as a simple heuristic.
func isNullableSimple(node AstNode) bool {
	switch n := node.(type) {
	case *ZeroOrMoreNode:
		return true
	case *OptionalNode:
		return true
	case *AndNode:
		return true
	case *NotNode:
		return true
	case *SequenceNode:
		for _, item := range n.Items {
			if !isNullableSimple(item) {
				return false
			}
		}
		return true
	case *ChoiceNode:
		return isNullableSimple(n.Left) || isNullableSimple(n.Right)
	case *LexNode:
		return isNullableSimple(n.Expr)
	case *CaptureNode:
		return isNullableSimple(n.Expr)
	case *LabeledNode:
		return isNullableSimple(n.Expr)
	default:
		return false
	}
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
