package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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
// Helpers for VM-based rewrite tests
// ===================================================================

// childrenOf returns the direct children of a node, unwrapping a
// single-child sequence inside a named node.
func childrenOf(t langlang.Tree, id langlang.NodeID) []langlang.NodeID {
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

func linearize(t langlang.Tree, id langlang.NodeID) []string {
	if t.Type(id) == langlang.NodeType_Sequence {
		var result []string
		for _, c := range t.Children(id) {
			result = append(result, linearize(t, c)...)
		}
		return result
	}
	if t.Type(id) != langlang.NodeType_Node {
		return nil
	}
	name := t.Name(id)
	cs := childrenOf(t, id)
	switch name {
	case "Seq":
		var result []string
		for _, c := range cs {
			result = append(result, linearize(t, c)...)
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
	bytecode, _, err := langlang.CompileRewriteFile(rf, "reshape_expr")
	require.NoError(t, err)

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
			imported := importTree(b, parseTree, root)

			result, err := langlang.RewriteWithBytecode(bytecode, b.Tree(), imported)
			require.NoError(t, err, "reshape %q", tc.src)

			tr := b.Tree()
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
	bytecode, _, err := langlang.CompileRewriteFile(rf, "reshape_expr")
	require.NoError(t, err)
	parseTree, root := parseTinyExpr(t, "2+3+4")
	b := langlang.NewTreeBuilder()
	imported := importTree(b, parseTree, root)

	result, err := langlang.RewriteWithBytecode(bytecode, b.Tree(), imported)
	require.NoError(t, err)

	tr := b.Tree()
	require.Equal(t, "Binary", tr.Name(result))
	cs := childrenOf(tr, result)
	require.Equal(t, "+", tr.Text(cs[0]))
	require.Equal(t, "Binary", tr.Name(cs[1]))
	leftCs := childrenOf(tr, cs[1])
	require.Equal(t, "+", tr.Text(leftCs[0]))
	require.Equal(t, "NumLit", tr.Name(leftCs[1]))
	require.Equal(t, "NumLit", tr.Name(leftCs[2]))
	require.Equal(t, "NumLit", tr.Name(cs[2]))
	t.Logf("2+3+4 -> %s", tr.Pretty(result))
}

// TestReshapePrecedence verifies 2+3*4 respects precedence: 2+(3*4).
func TestReshapePrecedence(t *testing.T) {
	rf := loadRewriteFile(t)
	bytecode, _, err := langlang.CompileRewriteFile(rf, "reshape_expr")
	require.NoError(t, err)
	parseTree, root := parseTinyExpr(t, "2+3*4")
	b := langlang.NewTreeBuilder()
	imported := importTree(b, parseTree, root)

	result, err := langlang.RewriteWithBytecode(bytecode, b.Tree(), imported)
	require.NoError(t, err)

	tr := b.Tree()
	t.Logf("2+3*4 -> %s", tr.Pretty(result))
	require.Equal(t, "Binary", tr.Name(result))
	cs := childrenOf(tr, result)
	require.Equal(t, 3, len(cs), "Binary has 3 children")
	// Binary(Left, Op, Right): find operator by value (order may vary by grammar)
	opIdx := -1
	for i, c := range cs {
		if tr.Text(c) == "+" {
			opIdx = i
			break
		}
	}
	require.GreaterOrEqual(t, opIdx, 0, "operator + not found in Binary children")
	require.Equal(t, "Binary", tr.Name(cs[2]), "right child is Binary(3*4)")
	rightCs := childrenOf(tr, cs[2])
	require.Equal(t, 3, len(rightCs))
	opIdxInner := -1
	for i, c := range rightCs {
		if tr.Text(c) == "*" {
			opIdxInner = i
			break
		}
	}
	require.GreaterOrEqual(t, opIdxInner, 0, "operator * not found in inner Binary")
	t.Logf("2+3*4 -> %s", tr.Pretty(result))
}

// TestReshapeCall verifies Call nodes are reshaped correctly.
func TestReshapeCall(t *testing.T) {
	rf := loadRewriteFile(t)
	bytecode, _, err := langlang.CompileRewriteFile(rf, "reshape_expr")
	require.NoError(t, err)

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

			result, err := langlang.RewriteWithBytecode(bytecode, b.Tree(), imported)
			require.NoError(t, err)

			tr := b.Tree()
			require.Equal(t, "Call", tr.Name(result))
			cs := childrenOf(tr, result)
			require.Equal(t, tc.callee, tr.Text(cs[0]))
			argKids := childrenOf(tr, cs[1])
			require.Equal(t, tc.numArgs, len(argKids),
				"expected %d args, got %d", tc.numArgs, len(argKids))
			t.Logf("reshape(%q) -> %s", tc.src, tr.Pretty(result))
		})
	}
}

// TestConstantFolding tests the fold pass on manually-built ASTs.
func TestConstantFolding(t *testing.T) {
	rf := loadRewriteFile(t)
	foldRS := langlang.RuleSetByName(rf, "fold")
	require.NotNil(t, foldRS)
	bytecode, err := langlang.CompileRewriteFileWithStrategy(rf, langlang.StratInnermost{Inner: langlang.StratLift{RuleSet: foldRS}})
	require.NoError(t, err)

	b := langlang.NewTreeBuilder()
	varX := b.Node("Var", b.Str("x"))
	num0 := b.Node("NumLit", b.Str("0"))
	num1 := b.Node("NumLit", b.Str("1"))
	add0X := b.Node("Binary", num0, b.Str("+"), varX)   // Left, Op, Right
	mul1 := b.Node("Binary", add0X, b.Str("*"), num1)

	result, err := langlang.RewriteWithBytecode(bytecode, b.Tree(), mul1)
	require.NoError(t, err)

	tr := b.Tree()
	require.Equal(t, "Var", tr.Name(result),
		"expected Var after folding (0+x)*1, got %s", tr.Name(result))
	t.Logf("(0 + x) * 1 -> %s", tr.Name(result))
}

// TestCodegen tests: parse source -> reshape -> fold -> codegen -> verify instructions.
func TestCodegen(t *testing.T) {
	rf := loadRewriteFile(t)
	bytecodeReshape, _, err := langlang.CompileRewriteFile(rf, "reshape_expr")
	require.NoError(t, err)
	foldRS := langlang.RuleSetByName(rf, "fold")
	require.NotNil(t, foldRS)
	bytecodeFold, err := langlang.CompileRewriteFileWithStrategy(rf, langlang.StratInnermost{Inner: langlang.StratLift{RuleSet: foldRS}})
	require.NoError(t, err)
	bytecodeExpr, _, err := langlang.CompileRewriteFile(rf, "expr")
	require.NoError(t, err)

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

			reshaped, err := langlang.RewriteWithBytecode(bytecodeReshape, b.Tree(), imported)
			require.NoError(t, err, "reshape")

			folded, err := langlang.RewriteWithBytecode(bytecodeFold, b.Tree(), reshaped)
			require.NoError(t, err, "fold")

			ir, err := langlang.RewriteWithBytecode(bytecodeExpr, b.Tree(), folded)
			require.NoError(t, err, "codegen")

			tr := b.Tree()
			got := linearize(tr, ir)
			require.Equal(t, tc.want, got,
				"codegen(%q): expected %v, got %v", tc.src, tc.want, got)
			t.Logf("codegen(%q) -> %s", tc.src, strings.Join(got, "; "))
		})
	}
}

// TestCodegenWithCall tests codegen on expressions containing function calls.
func TestCodegenWithCall(t *testing.T) {
	rf := loadRewriteFile(t)
	bytecodeReshape, _, err := langlang.CompileRewriteFile(rf, "reshape_expr")
	require.NoError(t, err)
	bytecodeExpr, _, err := langlang.CompileRewriteFile(rf, "expr")
	require.NoError(t, err)
	parseTree, root := parseTinyExpr(t, "f(1,x)")
	b := langlang.NewTreeBuilder()
	imported := importTree(b, parseTree, root)

	reshaped, err := langlang.RewriteWithBytecode(bytecodeReshape, b.Tree(), imported)
	require.NoError(t, err, "reshape")

	ir, err := langlang.RewriteWithBytecode(bytecodeExpr, b.Tree(), reshaped)
	require.NoError(t, err, "codegen")

	tr := b.Tree()
	got := linearize(tr, ir)
	expected := []string{"push 1", "load x", "call f 2"}
	require.Equal(t, expected, got)
	t.Logf("codegen(f(1,x)) -> %s", strings.Join(got, "; "))
}

// TestFoldThenCodegen tests constant folding flowing into codegen.
func TestFoldThenCodegen(t *testing.T) {
	rf := loadRewriteFile(t)
	bytecodeReshape, _, err := langlang.CompileRewriteFile(rf, "reshape_expr")
	require.NoError(t, err)
	foldRS := langlang.RuleSetByName(rf, "fold")
	require.NotNil(t, foldRS)
	bytecodeFold, err := langlang.CompileRewriteFileWithStrategy(rf, langlang.StratInnermost{Inner: langlang.StratLift{RuleSet: foldRS}})
	require.NoError(t, err)
	bytecodeExpr, _, err := langlang.CompileRewriteFile(rf, "expr")
	require.NoError(t, err)

	parseTree, root := parseTinyExpr(t, "0+x")
	b := langlang.NewTreeBuilder()
	imported := importTree(b, parseTree, root)

	reshaped, err := langlang.RewriteWithBytecode(bytecodeReshape, b.Tree(), imported)
	require.NoError(t, err)

	folded, err := langlang.RewriteWithBytecode(bytecodeFold, b.Tree(), reshaped)
	require.NoError(t, err)

	ir, err := langlang.RewriteWithBytecode(bytecodeExpr, b.Tree(), folded)
	require.NoError(t, err)

	tr := b.Tree()
	got := linearize(tr, ir)
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
