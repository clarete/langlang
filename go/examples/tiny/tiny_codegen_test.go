package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	langlang "github.com/clarete/langlang/go"
	"github.com/stretchr/testify/require"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func repoRoot() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "..")
}

// loadRewriteFile compiles langlang_rewrite.peg at runtime, uses it to
// parse tiny_rewrites.peg, then converts the parse tree to AST types.
func loadRewriteFile(t *testing.T) *langlang.RewriteFile {
	t.Helper()
	dir := testdataDir()

	grammarPath := filepath.Join(repoRoot(), "grammars", "langlang_rewrite.peg")
	matcher, err := langlang.MatcherFromFilePath(grammarPath)
	if err != nil {
		t.Fatalf("compiling langlang_rewrite.peg: %v", err)
	}

	rulesContent, err := os.ReadFile(filepath.Join(dir, "tiny_rewrites.peg"))
	if err != nil {
		t.Fatalf("reading tiny_rewrites.peg: %v", err)
	}

	tree, _, err := matcher.Match(rulesContent)
	if err != nil {
		t.Fatalf("parsing tiny_rewrites.peg: %v", err)
	}

	root, ok := tree.Root()
	if !ok {
		t.Fatal("empty parse tree")
	}

	rf, err := langlang.ParseRewriteFile(tree, root)
	if err != nil {
		t.Logf("Parse tree:\n%s", tree.Pretty(root))
		t.Fatalf("converting parse tree: %v", err)
	}

	return rf
}

// parseTinyExpr compiles tiny.peg at runtime and parses the given
// expression source, returning a langlang.Tree (not the generated parser.Tree).
func parseTinyExpr(t *testing.T, src string) (langlang.Tree, langlang.NodeID) {
	t.Helper()
	pegPath := filepath.Join(testdataDir(), "tiny.peg")
	matcher, err := langlang.MatcherFromFilePath(pegPath)
	require.NoError(t, err, "compiling tiny.peg")

	tree, _, err := matcher.Match([]byte(src))
	require.NoError(t, err, "parsing %q", src)

	root, ok := tree.Root()
	require.True(t, ok, "empty parse tree for %q", src)
	return tree, root
}

// ===================================================================
// Tree-walking interpreter for rewrite rules
// ===================================================================

type treeInterpreter struct {
	b        *langlang.TreeBuilder
	ruleSets map[string]*langlang.RewriteRuleSet
}

func newInterpreter(b *langlang.TreeBuilder, ruleSets []*langlang.RewriteRuleSet) *treeInterpreter {
	m := make(map[string]*langlang.RewriteRuleSet, len(ruleSets))
	for _, rs := range ruleSets {
		m[rs.Name] = rs
	}
	return &treeInterpreter{b: b, ruleSets: m}
}

func (interp *treeInterpreter) tree() langlang.Tree { return interp.b.Tree() }

// childrenOf returns the direct children of a node, unwrapping a
// single-child sequence inside a named node.
func (interp *treeInterpreter) childrenOf(id langlang.NodeID) []langlang.NodeID {
	t := interp.tree()
	switch t.Type(id) {
	case langlang.NodeType_Sequence:
		return t.Children(id)
	case langlang.NodeType_Node:
		c, ok := t.Child(id)
		if !ok {
			return nil
		}
		if t.Type(c) == langlang.NodeType_Sequence {
			return t.Children(c)
		}
		return []langlang.NodeID{c}
	default:
		return nil
	}
}

// ---------------------------------------------------------------
// Core interpreter
// ---------------------------------------------------------------

func (interp *treeInterpreter) applyRuleSet(rsName string, id langlang.NodeID) (langlang.NodeID, error) {
	rs, ok := interp.ruleSets[rsName]
	if !ok {
		return 0, fmt.Errorf("unknown rule set: %s", rsName)
	}

	for _, rule := range rs.Rules {
		bindings := map[string]langlang.NodeID{}
		if interp.matchPattern(rule.Pattern, id, bindings) {
			result, err := interp.buildConstruction(rule.Constr, bindings)
			if err != nil {
				return 0, fmt.Errorf("building %s: %w", rule.Name, err)
			}
			return result, nil
		}
	}

	return 0, fmt.Errorf("no rule matched in %s for node %s", rsName, interp.tree().Name(id))
}

func (interp *treeInterpreter) matchPattern(pat langlang.RewritePattern, id langlang.NodeID, bindings map[string]langlang.NodeID) bool {
	t := interp.tree()
	switch p := pat.(type) {
	case langlang.PatWild:
		return true
	case langlang.PatVar:
		bindings[p.Name] = id
		return true
	case langlang.PatStr:
		return t.Type(id) == langlang.NodeType_String && t.Text(id) == p.Text
	case langlang.PatNamed:
		if t.Type(id) != langlang.NodeType_Node || t.Name(id) != p.NodeName {
			return false
		}
		child, ok := t.Child(id)
		if !ok {
			return false
		}
		return interp.matchPattern(p.Body, child, bindings)
	case langlang.PatSeq:
		children := interp.childrenOf(id)
		if len(children) != len(p.Elems) {
			return false
		}
		for i, elem := range p.Elems {
			if !interp.matchPattern(elem, children[i], bindings) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (interp *treeInterpreter) buildConstruction(con langlang.RewriteConstruction, bindings map[string]langlang.NodeID) (langlang.NodeID, error) {
	switch c := con.(type) {
	case langlang.ConVar:
		id, ok := bindings[c.Name]
		if !ok {
			return 0, fmt.Errorf("unbound variable ?%s", c.Name)
		}
		return id, nil
	case langlang.ConStr:
		return interp.b.Str(c.Text), nil
	case langlang.ConNamed:
		child, err := interp.buildConstruction(c.Body, bindings)
		if err != nil {
			return 0, err
		}
		return interp.b.Named(c.NodeName, child), nil
	case langlang.ConSeq:
		children := make([]langlang.NodeID, len(c.Elems))
		for i, elem := range c.Elems {
			child, err := interp.buildConstruction(elem, bindings)
			if err != nil {
				return 0, err
			}
			children[i] = child
		}
		return interp.b.Seq(children...), nil
	case langlang.ConCall:
		if len(c.Args) != 1 {
			return 0, fmt.Errorf("ConCall %s: expected 1 arg, got %d", c.RuleName, len(c.Args))
		}
		argNode, err := interp.buildConstruction(c.Args[0], bindings)
		if err != nil {
			return 0, err
		}
		return interp.applyRuleSet(c.RuleName, argNode)
	case langlang.ConEach:
		return interp.buildEach(c, bindings)
	case langlang.ConLen:
		return interp.buildLen(c, bindings)
	case langlang.ConFoldl:
		return interp.buildFoldl(c, bindings)
	default:
		return 0, fmt.Errorf("unsupported construction type: %T", con)
	}
}

func (interp *treeInterpreter) isSkippable(id langlang.NodeID) bool {
	t := interp.tree()
	if t.Type(id) == langlang.NodeType_String {
		return true
	}
	if t.Type(id) == langlang.NodeType_Node && t.Name(id) == "Spacing" {
		return true
	}
	return false
}

func (interp *treeInterpreter) buildEach(c langlang.ConEach, bindings map[string]langlang.NodeID) (langlang.NodeID, error) {
	seqID, err := interp.buildConstruction(c.SeqArg, bindings)
	if err != nil {
		return 0, err
	}
	t := interp.tree()

	var items []langlang.NodeID
	if t.Type(seqID) == langlang.NodeType_Sequence {
		for _, kid := range t.Children(seqID) {
			if interp.isSkippable(kid) {
				continue
			}
			result, err := interp.applyRuleSet(c.RuleName, kid)
			if err != nil {
				return 0, fmt.Errorf("each(%s): %w", c.RuleName, err)
			}
			items = append(items, result)
		}
	} else if t.Type(seqID) == langlang.NodeType_Node {
		result, err := interp.applyRuleSet(c.RuleName, seqID)
		if err != nil {
			return 0, fmt.Errorf("each(%s): %w", c.RuleName, err)
		}
		items = append(items, result)
	}
	return interp.b.Seq(items...), nil
}

func (interp *treeInterpreter) buildLen(c langlang.ConLen, bindings map[string]langlang.NodeID) (langlang.NodeID, error) {
	seqID, err := interp.buildConstruction(c.SeqArg, bindings)
	if err != nil {
		return 0, err
	}
	t := interp.tree()

	count := 0
	if t.Type(seqID) == langlang.NodeType_Sequence {
		for _, kid := range t.Children(seqID) {
			if !interp.isSkippable(kid) {
				count++
			}
		}
	} else if t.Type(seqID) == langlang.NodeType_Node {
		count = 1
	}
	return interp.b.Str(strconv.Itoa(count)), nil
}

func (interp *treeInterpreter) buildFoldl(c langlang.ConFoldl, bindings map[string]langlang.NodeID) (langlang.NodeID, error) {
	seqID, err := interp.buildConstruction(c.SeqArg, bindings)
	if err != nil {
		return 0, err
	}
	t := interp.tree()

	if t.Type(seqID) != langlang.NodeType_Sequence {
		return interp.applyRuleSet(c.RuleName, seqID)
	}

	kids := t.Children(seqID)
	if len(kids) == 0 {
		return 0, fmt.Errorf("foldl(%s): empty sequence", c.CtorName)
	}
	if len(kids) == 1 {
		return interp.applyRuleSet(c.RuleName, kids[0])
	}
	if len(kids) < 3 || len(kids)%2 == 0 {
		return 0, fmt.Errorf("foldl(%s): bad sequence length %d", c.CtorName, len(kids))
	}

	acc, err := interp.applyRuleSet(c.RuleName, kids[0])
	if err != nil {
		return 0, err
	}
	for i := 1; i < len(kids); i += 2 {
		op := t.Text(kids[i])
		right, err := interp.applyRuleSet(c.RuleName, kids[i+1])
		if err != nil {
			return 0, err
		}
		acc = interp.b.Node(c.CtorName, interp.b.Str(op), acc, right)
	}
	return acc, nil
}

func (interp *treeInterpreter) applyInnermost(rsName string, id langlang.NodeID) langlang.NodeID {
	for {
		next := interp.applyBottomUpOnce(rsName, id)
		if next == id {
			return id
		}
		id = next
	}
}

func (interp *treeInterpreter) applyBottomUpOnce(rsName string, id langlang.NodeID) langlang.NodeID {
	t := interp.tree()

	if t.Type(id) == langlang.NodeType_Node {
		child, ok := t.Child(id)
		if ok {
			newChild := interp.applyBottomUpOnce(rsName, child)
			if newChild != child {
				id = interp.b.Named(t.Name(id), newChild)
			}
		}
	} else if t.Type(id) == langlang.NodeType_Sequence {
		kids := t.Children(id)
		newKids := make([]langlang.NodeID, len(kids))
		changed := false
		for i, k := range kids {
			newKids[i] = interp.applyBottomUpOnce(rsName, k)
			if newKids[i] != k {
				changed = true
			}
		}
		if changed {
			id = interp.b.Seq(newKids...)
		}
	}

	rs := interp.ruleSets[rsName]
	if rs == nil {
		return id
	}
	for _, rule := range rs.Rules {
		bindings := map[string]langlang.NodeID{}
		if interp.matchPattern(rule.Pattern, id, bindings) {
			result, err := interp.buildConstruction(rule.Constr, bindings)
			if err == nil {
				return result
			}
			break
		}
	}
	return id
}

func (interp *treeInterpreter) linearize(id langlang.NodeID) []string {
	t := interp.tree()
	if t.Type(id) == langlang.NodeType_Sequence {
		var result []string
		for _, c := range t.Children(id) {
			result = append(result, interp.linearize(c)...)
		}
		return result
	}
	if t.Type(id) != langlang.NodeType_Node {
		return nil
	}
	name := t.Name(id)
	cs := interp.childrenOf(id)

	switch name {
	case "Seq":
		var result []string
		for _, c := range cs {
			result = append(result, interp.linearize(c)...)
		}
		return result
	case "Push":
		return []string{"push " + t.Text(cs[0])}
	case "Load":
		return []string{"load " + t.Text(cs[0])}
	case "Store":
		return []string{"store " + t.Text(cs[0])}
	case "BinOp":
		return []string{"binop " + t.Text(cs[0])}
	case "AsmCall":
		return []string{"call " + t.Text(cs[0]) + " " + t.Text(cs[1])}
	case "Nop":
		return nil
	default:
		return []string{"<" + name + ">"}
	}
}

// ===================================================================
// Tests
// ===================================================================

func TestParseRewriteFile(t *testing.T) {
	rf := loadRewriteFile(t)

	typeNames := make([]string, len(rf.Types))
	for i, td := range rf.Types {
		typeNames[i] = td.Name
	}
	t.Logf("Types: %v", typeNames)

	for _, want := range []string{"Expr", "Asm"} {
		found := false
		for _, got := range typeNames {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing type: %s", want)
		}
	}

	ruleNames := make([]string, len(rf.RuleSets))
	for i, rs := range rf.RuleSets {
		ruleNames[i] = rs.Name
	}
	t.Logf("Rules: %v", ruleNames)

	for _, want := range []string{"reshape_expr", "fold", "expr"} {
		found := false
		for _, got := range ruleNames {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing rule: %s", want)
		}
	}
}

// TestParseAndReshape verifies the full reshape pass: PEG parse tree -> structured AST.
func TestParseAndReshape(t *testing.T) {
	rf := loadRewriteFile(t)

	tests := []struct {
		src  string
		want string // top-level AST node name
	}{
		{"42", "NumLit"},
		{"x", "Var"},
		{"2+3", "Binary"},
		{"(2+3)", "Binary"},
		{"2+3*4", "Binary"},
	}

	for _, tc := range tests {
		t.Run(tc.src, func(t *testing.T) {
			parseTree, root := parseTinyExpr(t, tc.src)

			b := langlang.NewTreeBuilder()
			// Import the parse tree into the builder's arena.
			imported := importTree(b, parseTree, root)

			interp := newInterpreter(b, rf.RuleSets)
			result, err := interp.applyRuleSet("reshape_expr", imported)
			require.NoError(t, err, "reshape %q", tc.src)

			tr := interp.tree()
			require.Equal(t, langlang.NodeType_Node, tr.Type(result))
			require.Equal(t, tc.want, tr.Name(result),
				"reshape(%q): expected top node %s, got %s", tc.src, tc.want, tr.Name(result))
			t.Logf("reshape(%q) -> %s", tc.src, tr.Pretty(result))
		})
	}
}

// TestReshapeBinaryAssociativity verifies 2+3+4 is left-associated: (2+3)+4.
func TestReshapeBinaryAssociativity(t *testing.T) {
	rf := loadRewriteFile(t)
	parseTree, root := parseTinyExpr(t, "2+3+4")

	b := langlang.NewTreeBuilder()
	imported := importTree(b, parseTree, root)
	interp := newInterpreter(b, rf.RuleSets)

	result, err := interp.applyRuleSet("reshape_expr", imported)
	require.NoError(t, err)

	// result should be Binary("+", Binary("+", NumLit("2"), NumLit("3")), NumLit("4"))
	tr := interp.tree()
	require.Equal(t, "Binary", tr.Name(result))
	cs := interp.childrenOf(result)
	require.Equal(t, "+", tr.Text(cs[0]))

	// Left child should also be Binary
	require.Equal(t, "Binary", tr.Name(cs[1]))
	leftCs := interp.childrenOf(cs[1])
	require.Equal(t, "+", tr.Text(leftCs[0]))
	require.Equal(t, "NumLit", tr.Name(leftCs[1]))
	require.Equal(t, "NumLit", tr.Name(leftCs[2]))

	// Right child should be NumLit("4")
	require.Equal(t, "NumLit", tr.Name(cs[2]))
	t.Logf("2+3+4 -> %s", tr.Pretty(result))
}

// TestReshapePrecedence verifies 2+3*4 respects precedence: 2+(3*4).
func TestReshapePrecedence(t *testing.T) {
	rf := loadRewriteFile(t)
	parseTree, root := parseTinyExpr(t, "2+3*4")

	b := langlang.NewTreeBuilder()
	imported := importTree(b, parseTree, root)
	interp := newInterpreter(b, rf.RuleSets)

	result, err := interp.applyRuleSet("reshape_expr", imported)
	require.NoError(t, err)

	// Binary("+", NumLit("2"), Binary("*", NumLit("3"), NumLit("4")))
	tr := interp.tree()
	require.Equal(t, "Binary", tr.Name(result))
	cs := interp.childrenOf(result)
	require.Equal(t, "+", tr.Text(cs[0]))
	require.Equal(t, "NumLit", tr.Name(cs[1]))
	require.Equal(t, "Binary", tr.Name(cs[2]))

	rightCs := interp.childrenOf(cs[2])
	require.Equal(t, "*", tr.Text(rightCs[0]))
	t.Logf("2+3*4 -> %s", tr.Pretty(result))
}

// TestReshapeCall verifies Call nodes are reshaped correctly.
func TestReshapeCall(t *testing.T) {
	rf := loadRewriteFile(t)

	tests := []struct {
		src      string
		callee   string
		numArgs  int
	}{
		{"f()", "f", 0},
		{"g(1)", "g", 1},
		{"h(1,2)", "h", 2},
	}

	for _, tc := range tests {
		t.Run(tc.src, func(t *testing.T) {
			parseTree, root := parseTinyExpr(t, tc.src)
			b := langlang.NewTreeBuilder()
			imported := importTree(b, parseTree, root)
			interp := newInterpreter(b, rf.RuleSets)

			result, err := interp.applyRuleSet("reshape_expr", imported)
			require.NoError(t, err)

			tr := interp.tree()
			require.Equal(t, "Call", tr.Name(result))
			cs := interp.childrenOf(result)
			require.Equal(t, tc.callee, tr.Text(cs[0]))
			argKids := interp.childrenOf(cs[1])
			require.Equal(t, tc.numArgs, len(argKids),
				"expected %d args, got %d", tc.numArgs, len(argKids))
			t.Logf("reshape(%q) -> %s", tc.src, tr.Pretty(result))
		})
	}
}

// TestConstantFolding tests the fold pass on manually-built ASTs.
func TestConstantFolding(t *testing.T) {
	rf := loadRewriteFile(t)

	b := langlang.NewTreeBuilder()
	interp := newInterpreter(b, rf.RuleSets)

	varX := b.Node("Var", b.Str("x"))
	num0 := b.Node("NumLit", b.Str("0"))
	num1 := b.Node("NumLit", b.Str("1"))
	add0X := b.Node("Binary", b.Str("+"), num0, varX)
	mul1 := b.Node("Binary", b.Str("*"), add0X, num1)

	result := interp.applyInnermost("fold", mul1)

	tr := b.Tree()
	require.Equal(t, "Var", tr.Name(result),
		"expected Var after folding (0+x)*1, got %s", tr.Name(result))
	t.Logf("(0 + x) * 1 -> %s", tr.Name(result))
}

// TestCodegen tests: parse source -> reshape -> codegen -> verify instructions.
func TestCodegen(t *testing.T) {
	rf := loadRewriteFile(t)

	tests := []struct {
		src  string
		want []string
	}{
		{"42", []string{"push 42"}},
		{"x", []string{"load x"}},
		{"2+3", []string{"push 2", "push 3", "binop +"}},
		{"2+3*4", []string{"push 2", "push 3", "push 4", "binop *", "binop +"}},
	}

	for _, tc := range tests {
		t.Run(tc.src, func(t *testing.T) {
			parseTree, root := parseTinyExpr(t, tc.src)

			b := langlang.NewTreeBuilder()
			imported := importTree(b, parseTree, root)
			interp := newInterpreter(b, rf.RuleSets)

			reshaped, err := interp.applyRuleSet("reshape_expr", imported)
			require.NoError(t, err, "reshape")

			folded := interp.applyInnermost("fold", reshaped)

			ir, err := interp.applyRuleSet("expr", folded)
			require.NoError(t, err, "codegen")

			got := interp.linearize(ir)
			require.Equal(t, tc.want, got,
				"codegen(%q): expected %v, got %v", tc.src, tc.want, got)
			t.Logf("codegen(%q) -> %s", tc.src, strings.Join(got, "; "))
		})
	}
}

// TestCodegenWithCall tests codegen on expressions containing function calls.
func TestCodegenWithCall(t *testing.T) {
	rf := loadRewriteFile(t)
	parseTree, root := parseTinyExpr(t, "f(1,x)")

	b := langlang.NewTreeBuilder()
	imported := importTree(b, parseTree, root)
	interp := newInterpreter(b, rf.RuleSets)

	reshaped, err := interp.applyRuleSet("reshape_expr", imported)
	require.NoError(t, err, "reshape")

	ir, err := interp.applyRuleSet("expr", reshaped)
	require.NoError(t, err, "codegen")

	got := interp.linearize(ir)
	expected := []string{"push 1", "load x", "call f 2"}
	require.Equal(t, expected, got)
	t.Logf("codegen(f(1,x)) -> %s", strings.Join(got, "; "))
}

// TestFoldThenCodegen tests constant folding flowing into codegen.
func TestFoldThenCodegen(t *testing.T) {
	rf := loadRewriteFile(t)

	// 0+x should fold to x, then codegen to "load x"
	parseTree, root := parseTinyExpr(t, "0+x")
	b := langlang.NewTreeBuilder()
	imported := importTree(b, parseTree, root)
	interp := newInterpreter(b, rf.RuleSets)

	reshaped, err := interp.applyRuleSet("reshape_expr", imported)
	require.NoError(t, err)

	folded := interp.applyInnermost("fold", reshaped)

	ir, err := interp.applyRuleSet("expr", folded)
	require.NoError(t, err)

	got := interp.linearize(ir)
	require.Equal(t, []string{"load x"}, got,
		"0+x should fold to x then codegen to 'load x'")
	t.Logf("codegen(fold(reshape(0+x))) -> %s", strings.Join(got, "; "))
}

// ===================================================================
// Helpers
// ===================================================================

// importTree deep-copies a parse tree (from the runtime parser's arena)
// into a TreeBuilder's arena, so that the rewrite interpreter can
// work with a single unified tree.
func importTree(b *langlang.TreeBuilder, src langlang.Tree, id langlang.NodeID) langlang.NodeID {
	switch src.Type(id) {
	case langlang.NodeType_String:
		return b.Str(src.Text(id))
	case langlang.NodeType_Node:
		child, ok := src.Child(id)
		if !ok {
			// Node with no children — create a named node with an empty string child.
			return b.Named(src.Name(id), b.Str(""))
		}
		return b.Named(src.Name(id), importTree(b, src, child))
	case langlang.NodeType_Sequence:
		kids := src.Children(id)
		imported := make([]langlang.NodeID, len(kids))
		for i, k := range kids {
			imported[i] = importTree(b, src, k)
		}
		return b.Seq(imported...)
	default:
		return b.Str(fmt.Sprintf("<unknown-%d>", src.Type(id)))
	}
}
