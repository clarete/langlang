package langlang

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Core Pipeline Tests

func TestQueryBasicPipeline(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- "hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	t.Run("ParsedGrammarQuery", func(t *testing.T) {
		grammar, err := Get(db, ParsedGrammarQuery, FilePath("test.peg"))
		require.NoError(t, err)
		require.NotNil(t, grammar)
		assert.Len(t, grammar.Definitions, 1)
		assert.Equal(t, "G", grammar.Definitions[0].Name)
	})

	t.Run("ResolvedImportsQuery", func(t *testing.T) {
		grammar, err := Get(db, ResolvedImportsQuery, FilePath("test.peg"))
		require.NoError(t, err)
		require.NotNil(t, grammar)
		// Now includes builtins (Spacing, Space, EOF, EOL) merged during import resolution
		assert.Greater(t, len(grammar.Definitions), 1)
		assert.Contains(t, grammar.DefsByName, "G")
		assert.Contains(t, grammar.DefsByName, "Spacing") // builtin
	})

	t.Run("TransformedGrammarQuery", func(t *testing.T) {
		grammar, err := Get(db, TransformedGrammarQuery, FilePath("test.peg"))
		require.NoError(t, err)
		require.NotNil(t, grammar)
		// Should have builtins (now added during import resolution)
		assert.Greater(t, len(grammar.Definitions), 1)
	})

	t.Run("CompiledProgramQuery", func(t *testing.T) {
		program, err := Get(db, CompiledProgramQuery, FilePath("test.peg"))
		require.NoError(t, err)
		require.NotNil(t, program)
		assert.NotEmpty(t, program.code)
	})

	t.Run("EncodedBytecodeQuery", func(t *testing.T) {
		bytecode, err := Get(db, EncodedBytecodeQuery, FilePath("test.peg"))
		require.NoError(t, err)
		require.NotNil(t, bytecode)
		assert.NotEmpty(t, bytecode.code)
	})
}

func TestQueryWithImports(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("number.peg", []byte(`Number <- [0-9]+`))
	loader.Add("main.peg", []byte(`
@import Number from "./number.peg"
Main <- Number "+" Number
`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	grammar, err := Get(db, ResolvedImportsQuery, FilePath("main.peg"))
	require.NoError(t, err)
	require.NotNil(t, grammar)

	// Should have both Main and Number definitions
	assert.Contains(t, grammar.DefsByName, "Main")
	assert.Contains(t, grammar.DefsByName, "Number")
}

func TestMatcherFromString(t *testing.T) {
	grammar := `G <- "hello"`
	matcher, err := MatcherFromString(grammar)
	require.NoError(t, err)
	require.NotNil(t, matcher)

	tree, n, err := matcher.Match([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.NotNil(t, tree)
}

func TestMatcherFromBytes(t *testing.T) {
	grammar := []byte(`G <- "hello"`)
	matcher, err := MatcherFromBytes(grammar)
	require.NoError(t, err)
	require.NotNil(t, matcher)

	tree, n, err := matcher.Match([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.NotNil(t, tree)

	// With explicit config
	cfg := NewConfig()
	matcher2, err := MatcherFromBytesWithConfig(grammar, cfg)
	require.NoError(t, err)
	require.NotNil(t, matcher2)
	tree2, n2, err := matcher2.Match([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n2)
	assert.NotNil(t, tree2)
}

func TestQueryResolver(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- "hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)
	resolver := NewQueryResolver(db)

	t.Run("Resolve", func(t *testing.T) {
		grammar, err := resolver.Resolve("test.peg")
		require.NoError(t, err)
		require.NotNil(t, grammar)
	})

	t.Run("MatcherFor", func(t *testing.T) {
		matcher, err := resolver.MatcherFor("test.peg")
		require.NoError(t, err)
		require.NotNil(t, matcher)

		tree, n, err := matcher.Match([]byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.NotNil(t, tree)
	})

	t.Run("Stats", func(t *testing.T) {
		stats := resolver.Stats()
		assert.Greater(t, stats.CachedCount, 0)
	})
}

// Caching and Invalidation Tests

func TestQueryCaching(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- "hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	// First query
	grammar1, err := Get(db, ParsedGrammarQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// Second query should return cached result
	grammar2, err := Get(db, ParsedGrammarQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// Should be the same pointer (cached)
	assert.Same(t, grammar1, grammar2)

	stats := db.Stats()
	assert.Equal(t, 1, stats.CachedCount)
}

func TestQueryInvalidation(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- "hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	// First query
	_, err := Get(db, ParsedGrammarQuery, FilePath("test.peg"))
	require.NoError(t, err)

	stats1 := db.Stats()
	assert.Equal(t, 1, stats1.CachedCount)

	// Invalidate
	db.InvalidateFile("test.peg")

	stats2 := db.Stats()
	assert.Equal(t, 0, stats2.CachedCount)
}

func TestQueryDependencyTracking(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- "hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	// Query the full pipeline to establish dependencies
	_, err := Get(db, EncodedBytecodeQuery, FilePath("test.peg"))
	require.NoError(t, err)

	stats := db.Stats()
	// Should have cached multiple queries in the pipeline
	assert.Greater(t, stats.CachedCount, 1)
	// Should have recorded dependencies
	assert.Greater(t, stats.DepsCount, 0)
}

func TestTransformationPipelineCaching(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- "hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	// Query the full pipeline
	_, err := Get(db, TransformedGrammarQuery, FilePath("test.peg"))
	require.NoError(t, err)

	stats := db.Stats()
	// Should have cached each step of the pipeline
	// ResolvedImports, WithBuiltins, WithCharsets, WithWhitespace, WithCaptures, Transformed
	assert.GreaterOrEqual(t, stats.CachedCount, 5)
}

func TestCascadingInvalidation(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- "hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	// Build up cache
	_, err := Get(db, TransformedGrammarQuery, FilePath("test.peg"))
	require.NoError(t, err)

	stats1 := db.Stats()
	assert.Greater(t, stats1.CachedCount, 0)

	// Invalidate at the parsed level - should cascade
	Invalidate(db, ParsedGrammarQuery, FilePath("test.peg"))

	stats2 := db.Stats()
	// All dependent queries should be invalidated
	assert.Less(t, stats2.CachedCount, stats1.CachedCount)
}

// Builtins Tests

func TestWithBuiltinsQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- "hello"`))

	t.Run("With builtins enabled", func(t *testing.T) {
		cfg := NewConfig()
		cfg.SetBool("grammar.add_builtins", true)
		db := NewDatabase(cfg, loader)

		grammar, err := Get(db, ResolvedImportsQuery, FilePath("test.peg"))
		require.NoError(t, err)

		// Should have builtins added
		assert.Contains(t, grammar.DefsByName, "Spacing")
		assert.Contains(t, grammar.DefsByName, "EOF")
	})

	t.Run("With builtins disabled", func(t *testing.T) {
		cfg := NewConfig()
		cfg.SetBool("grammar.add_builtins", false)
		db := NewDatabase(cfg, loader)

		grammar, err := Get(db, ResolvedImportsQuery, FilePath("test.peg"))
		require.NoError(t, err)

		// Should NOT have builtins
		assert.NotContains(t, grammar.DefsByName, "Spacing")
		assert.NotContains(t, grammar.DefsByName, "EOF")
	})
}

func TestBuiltinsHaveProperFileIDs(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- Spacing "hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	grammar, err := Get(db, ResolvedImportsQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// Check that source files include both the user file and builtins
	assert.Contains(t, grammar.SourceFiles, "test.peg")
	assert.Contains(t, grammar.SourceFiles, BuiltinsPath)

	// Check that builtin definitions have a different FileID than user definitions
	userDef := grammar.DefsByName["G"]
	builtinDef := grammar.DefsByName["Spacing"]

	require.NotNil(t, userDef)
	require.NotNil(t, builtinDef)

	// User definition should have FileID for test.peg
	// Builtin definition should have FileID for langlang:builtins.peg
	assert.NotEqual(t, userDef.SourceLocation().FileID, builtinDef.SourceLocation().FileID,
		"User and builtin definitions should have different FileIDs")

	// Verify the FileIDs map to correct paths
	userFileID := userDef.SourceLocation().FileID
	builtinFileID := builtinDef.SourceLocation().FileID

	assert.Equal(t, "test.peg", grammar.SourceFiles[userFileID])
	assert.Equal(t, BuiltinsPath, grammar.SourceFiles[builtinFileID])
}

func TestBuiltinsCanBeOverridden(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// User defines their own Spacing rule
	loader.Add("test.peg", []byte(`
G <- Spacing "hello"
Spacing <- [ \t]*
`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	grammar, err := Get(db, ResolvedImportsQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// User's Spacing should take precedence
	spacingDef := grammar.DefsByName["Spacing"]
	require.NotNil(t, spacingDef)

	// Should have user's FileID, not builtin's
	userFileID := grammar.DefsByName["G"].SourceLocation().FileID
	assert.Equal(t, userFileID, spacingDef.SourceLocation().FileID,
		"User-defined Spacing should have same FileID as user grammar")
}

// Transformation Tests

func TestWithCharsetsQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- [a-z]`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	cfg.SetBool("grammar.add_charsets", true)
	db := NewDatabase(cfg, loader)

	grammar, err := Get(db, WithCharsetsQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// The class should be converted to a charset
	def := grammar.DefsByName["G"]
	_, isCharset := def.Expr.(*CharsetNode)
	assert.True(t, isCharset, "Expected CharsetNode, got %T", def.Expr)
}

func TestWithWhitespaceQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Use a non-syntactic rule (with identifier) to ensure spacing is injected
	loader.Add("test.peg", []byte(`
G <- A B
A <- "a"
B <- "b"
Spacing <- ' '*
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	cfg.SetBool("grammar.add_charsets", true)
	cfg.SetBool("grammar.handle_spaces", true)
	db := NewDatabase(cfg, loader)

	grammar, err := Get(db, WithWhitespaceQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// G should have Spacing calls injected (because it references other rules)
	def := grammar.DefsByName["G"]
	hasSpacingCall := false
	Inspect(def.Expr, func(n AstNode) bool {
		if id, ok := n.(*IdentifierNode); ok && id.Value == "Spacing" {
			hasSpacingCall = true
		}
		return true
	})
	assert.True(t, hasSpacingCall, "Expected Spacing calls to be injected")
}

func TestWithWhitespaceQueryNoDoubleSpacing(t *testing.T) {
	// ZeroOrMore and OneOrMore with an inner sequence of
	// non-terminals should not inject a Spacing call before the
	// first item of the inner sequence, because the repetition
	// wrapper already adds one.
	//
	// Correct:   (Spacing (A Spacing B))*
	// Incorrect: (Spacing (Spacing A Spacing B))*
	for _, tt := range []struct {
		name        string
		grammar     string
		expectedAST string
	}{
		{
			name: "ZeroOrMore with sequence",
			grammar: `
G <- (A B)*
A <- "a"
B <- "b"
Spacing <- ' '*
`,
			expectedAST: `ZeroOrMore (2:6..2:12)
└── Sequence (2:6..2:12)
    ├── Identifier[Spacing] (1)
    └── Sequence (2:7..2:10)
        ├── Identifier[A] (2:7..2:8)
        ├── Identifier[Spacing] (1)
        └── Identifier[B] (2:9..2:10)`,
		},
		{
			name: "OneOrMore with sequence",
			grammar: `
G <- (A B)+
A <- "a"
B <- "b"
Spacing <- ' '*
`,
			expectedAST: `OneOrMore (2:6..2:12)
└── Sequence (2:6..2:12)
    ├── Identifier[Spacing] (1)
    └── Sequence (2:7..2:10)
        ├── Identifier[A] (2:7..2:8)
        ├── Identifier[Spacing] (1)
        └── Identifier[B] (2:9..2:10)`,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewInMemoryImportLoader()
			loader.Add("test.peg", []byte(tt.grammar))

			cfg := NewConfig()
			cfg.SetBool("grammar.add_builtins", false)
			cfg.SetBool("grammar.handle_spaces", true)
			db := NewDatabase(cfg, loader)

			grammar, err := Get(db, WithWhitespaceQuery, FilePath("test.peg"))
			require.NoError(t, err)

			def := grammar.DefsByName["G"]
			assert.Equal(t, tt.expectedAST, def.Expr.PrettyString())
		})
	}
}

func TestWithCapturesQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- "hello"`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	cfg.SetBool("grammar.add_charsets", true)
	cfg.SetBool("grammar.handle_spaces", false)
	cfg.SetBool("grammar.captures", true)
	db := NewDatabase(cfg, loader)

	grammar, err := Get(db, WithCapturesQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// G should have capture nodes
	def := grammar.DefsByName["G"]
	_, isCapture := def.Expr.(*CaptureNode)
	assert.True(t, isCapture, "Expected CaptureNode wrapper")
}

// Recursion Detection Tests

func TestIsRecursiveQuery(t *testing.T) {
	tests := []struct {
		name     string
		grammar  string
		expected map[string]bool
	}{
		{"direct recursion",
			`E <- E '+' 'n' / 'n'`,
			map[string]bool{"E": true}},
		{"right recursion",
			`E <- 'n' '+' E / 'n'`,
			map[string]bool{"E": true}},
		{"no recursion",
			`D <- '0' / '1'`,
			map[string]bool{"D": false}},
		{"mutual recursion",
			`A <- B 'a'
			 B <- A 'b' / 'c'
			 C <- 'd'`,
			map[string]bool{"A": true, "B": true, "C": false}},
		{"mixed direct and non-recursive",
			`E <- E '+' E / D
			 D <- '0' / '1'`,
			map[string]bool{"E": true, "D": false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewInMemoryImportLoader()
			loader.Add("test.peg", []byte(tt.grammar))
			cfg := NewConfig()
			cfg.SetBool("grammar.add_builtins", false)
			db := NewDatabase(cfg, loader)
			for name, expected := range tt.expected {
				isRecursive, err := Get(db, IsRecursiveQuery, DefKey{File: "test.peg", Name: name})
				require.NoError(t, err)
				assert.Equal(t, expected, isRecursive, "definition %s", name)
			}
		})
	}
}

// Analysis Query Tests

func TestRuleGraphLabelTargets(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- "a"^ErrA
B <- "b"^ErrB
C <- A^ErrA B
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	rg, err := Get(db, RuleGraphQuery, FilePath("test.peg"))
	require.NoError(t, err)
	labels := rg.LabelTargets()

	assert.Contains(t, labels, "ErrA")
	assert.Contains(t, labels, "ErrB")
	assert.Len(t, labels, 2)
}

func TestRuleGraphClosure(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- B C
B <- D "b"
C <- "c"
D <- "d"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	rg, err := Get(db, RuleGraphQuery, FilePath("test.peg"))
	require.NoError(t, err)
	deps := rg.Closure("A")

	// A depends on B, C, and transitively on D
	assert.Contains(t, deps, "B")
	assert.Contains(t, deps, "C")
	assert.Contains(t, deps, "D")
}

func TestCompilerCapExprSize(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Use multi-char literals that won't be converted to charsets
	// Single char literals like "a" become CharsetNode after transformation
	loader.Add("test.peg", []byte(`
Fixed <- "ab"
Variable <- "a" / "abc"
FixedChoice <- "ab" / "cd"
FixedRange <- [a-z]
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	// Disable charset transformation to keep literals as-is for this test
	cfg.SetBool("grammar.add_charsets", false)
	cfg.SetBool("grammar.handle_spaces", false)
	cfg.SetBool("grammar.captures", false)
	db := NewDatabase(cfg, loader)

	grammar, err := Get(db, TransformedGrammarQuery, FilePath("test.peg"))
	require.NoError(t, err)

	c := newCompilerWithDB(db, "test.peg")
	size := func(name string) (int, bool) {
		return c.capExprSize(grammar.DefsByName[name].Expr)
	}

	t.Run("Fixed size literal", func(t *testing.T) {
		sz, fixed := size("Fixed")
		assert.True(t, fixed)
		assert.Equal(t, 2, sz)
	})

	t.Run("Variable size choice", func(t *testing.T) {
		_, fixed := size("Variable")
		assert.False(t, fixed)
	})

	t.Run("Fixed size choice", func(t *testing.T) {
		sz, fixed := size("FixedChoice")
		assert.True(t, fixed)
		assert.Equal(t, 2, sz)
	})

	t.Run("Fixed size range (1 char)", func(t *testing.T) {
		_, fixed := size("FixedRange")
		assert.False(t, fixed)
	})
}

func TestIsLeftRecursiveQuery(t *testing.T) {
	tests := []struct {
		name     string
		grammar  string
		expected map[string]bool
	}{
		{"simple left recursion",
			`E <- E '+' 'n' / 'n'`,
			map[string]bool{"E": true}},
		{"non left recursive",
			`D <- '0' / '1'`,
			map[string]bool{"D": false}},
		{"multiple lr alternatives",
			`E <- E '+' E / E '*' E / 'n'`,
			map[string]bool{"E": true}},
		{"lr with non lr dep e",
			`E <- E '+' E / D
    			 D <- '0' / '1'`,
			map[string]bool{"E": true, "D": false}},
		{"lr with non lr dep d",
			`E <- E '+' E / D
			 D <- '0' / '1'`,
			map[string]bool{"E": true, "D": false}},
		{"right recursive not lr",
			`E <- 'n' '+' E / 'n'`,
			map[string]bool{"E": false}},
		{"indirect left recursion",
			`A <- B 'x'
			 B <- A 'y' / 'z'`,
			map[string]bool{"A": true, "B": true}},
		{"indirect left recursion chain",
			`A <- B 'x'
			 B <- C 'y'
			 C <- A 'z' / 'w'`,
			map[string]bool{"A": true, "B": true, "C": true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewInMemoryImportLoader()
			loader.Add("test.peg", []byte(tt.grammar))
			cfg := NewConfig()
			cfg.SetBool("grammar.add_builtins", false)
			db := NewDatabase(cfg, loader)
			for name, expected := range tt.expected {
				isLR, err := Get(db, IsLeftRecursiveQuery, DefKey{File: "test.peg", Name: name})
				require.NoError(t, err)
				assert.Equal(t, expected, isLR)
			}
		})
	}
}

// Nullability Tests

func TestNullableRulesQuery(t *testing.T) {
	tests := []struct {
		name     string
		grammar  string
		nullable []string // rule names expected to be nullable
		solid    []string // rule names expected to NOT be nullable
	}{
		{"optional body",
			`A <- 'x'?`,
			[]string{"A"}, []string{}},
		{"zero or more",
			`A <- 'x'*`,
			[]string{"A"}, []string{}},
		{"literal",
			`A <- 'x'`,
			[]string{}, []string{"A"}},
		{"one or more non-nullable",
			`A <- 'x'+`,
			[]string{}, []string{"A"}},
		{"one or more nullable",
			`A <- 'x'?
			 B <- A+`,
			[]string{"A", "B"}, []string{}},
		{"transitive chain",
			`A <- B
			 B <- C
			 C <- 'x'?`,
			[]string{"A", "B", "C"}, []string{}},
		{"transitive chain blocked",
			`A <- B
			 B <- C
			 C <- 'x'`,
			[]string{}, []string{"A", "B", "C"}},
		{"sequence all nullable",
			`A <- 'x'? 'y'?`,
			[]string{"A"}, []string{}},
		{"sequence one solid",
			`A <- 'x' 'y'?`,
			[]string{}, []string{"A"}},
		{"choice one nullable",
			`A <- 'x' / 'y'?`,
			[]string{"A"}, []string{}},
		{"and predicate",
			`A <- &'x'`,
			[]string{"A"}, []string{}},
		{"not predicate",
			`A <- !'x'`,
			[]string{"A"}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewInMemoryImportLoader()
			loader.Add("test.peg", []byte(tt.grammar))
			cfg := NewConfig()
			cfg.SetBool("grammar.add_builtins", false)
			db := NewDatabase(cfg, loader)

			nullableSet, err := Get(db, NullableRulesQuery, FilePath("test.peg"))
			require.NoError(t, err)

			for _, name := range tt.nullable {
				_, ok := nullableSet[name]
				assert.True(t, ok, "expected %s to be nullable", name)
			}
			for _, name := range tt.solid {
				_, ok := nullableSet[name]
				assert.False(t, ok, "expected %s to NOT be nullable", name)
			}
		})
	}
}

// Infinite Loop Detection Tests

func TestInfiniteLoopRisksQuery(t *testing.T) {
	tests := []struct {
		name      string
		grammar   string
		wantCount int
		wantOps   []string // expected operators in order
		wantVia   []bool   // true = has ViaRule, false = structural
		wantDef   []bool   // true = Definitive (error), false = input-dependent (warning)
	}{
		// Definitive: body always succeeds, loop can never exit
		{"optional star",
			`A <- ('x'?)*`,
			1, []string{"*"}, []bool{false}, []bool{true}},
		{"optional plus",
			`A <- ('x'?)+`,
			1, []string{"+"}, []bool{false}, []bool{true}},
		{"star star",
			`A <- ('x'*)*`,
			1, []string{"*"}, []bool{false}, []bool{true}},
		{"all nullable sequence star",
			`A <- ('x'? 'y'?)*`,
			1, []string{"*"}, []bool{false}, []bool{true}},
		{"any nullable choice star definitive",
			`A <- ('x'? / 'y'?)*`,
			1, []string{"*"}, []bool{false}, []bool{true}},
		{"nullable rule star definitive",
			`A <- B*
			 B <- 'x'?`,
			1, []string{"*"}, []bool{true}, []bool{true}},
		{"deep chain star definitive",
			`A <- B*
			 B <- C
			 C <- 'x'?`,
			1, []string{"*"}, []bool{true}, []bool{true}},
		{"nullable ref sequence definitive",
			`A <- (B C)*
			 B <- 'x'?
			 C <- 'y'?`,
			1, []string{"*"}, []bool{true}, []bool{true}},

		// Input-dependent: body can fail, loop might exit
		{"and star",
			`A <- (&'x')*`,
			1, []string{"*"}, []bool{false}, []bool{false}},
		{"not star",
			`A <- (!'x')*`,
			1, []string{"*"}, []bool{false}, []bool{false}},
		{"nullable choice left always succeeds",
			`A <- ('x'? / 'y')*`,
			1, []string{"*"}, []bool{false}, []bool{true}},
		{"not guard plus nullable ref",
			`A <- (!'x' B)*
			 B <- 'y'?`,
			1, []string{"*"}, []bool{true}, []bool{false}},

		// Safe patterns: should produce zero risks
		{"terminal star",
			`A <- 'x'*`,
			0, nil, nil, nil},
		{"class star",
			`A <- [a-z]*`,
			0, nil, nil, nil},
		{"dot star",
			`A <- .*`,
			0, nil, nil, nil},
		{"one or more non-nullable star",
			`A <- ('x'+)*`,
			0, nil, nil, nil},
		{"solid prefix in sequence",
			`A <- ('x' 'y'?)*`,
			0, nil, nil, nil},
		{"non-nullable rule star",
			`A <- B*
			 B <- 'x'`,
			0, nil, nil, nil},

		// Multiple risks in one grammar
		{"definitive and input-dependent mix",
			`A <- ('x'?)* (!'y')+`,
			2, []string{"*", "+"}, []bool{false, false}, []bool{true, false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewInMemoryImportLoader()
			loader.Add("test.peg", []byte(tt.grammar))
			cfg := NewConfig()
			cfg.SetBool("grammar.add_builtins", false)
			db := NewDatabase(cfg, loader)

			risks, err := Get(db, InfiniteLoopRisksQuery, FilePath("test.peg"))
			require.NoError(t, err)
			require.Len(t, risks, tt.wantCount, "unexpected number of risks")

			for i, risk := range risks {
				if i < len(tt.wantOps) {
					assert.Equal(t, tt.wantOps[i], risk.Operator, "risk %d operator", i)
				}
				if i < len(tt.wantVia) {
					if tt.wantVia[i] {
						assert.NotEmpty(t, risk.ViaRule, "risk %d expected ViaRule to be set", i)
					} else {
						assert.Empty(t, risk.ViaRule, "risk %d expected structural (no ViaRule)", i)
					}
				}
				if i < len(tt.wantDef) {
					assert.Equal(t, tt.wantDef[i], risk.Definitive, "risk %d Definitive", i)
				}
			}
		})
	}
}

func TestInfiniteLoopDiagnostics(t *testing.T) {
	tests := []struct {
		name         string
		grammar      string
		wantCount    int
		wantSeverity DiagnosticSeverity
		wantSubstr   string // expected substring in the diagnostic message
	}{
		// Definitive infinite loops produce errors
		{"optional star",
			`A <- ('x'?)*`,
			1, DiagnosticError, "Infinite loop"},
		{"star star",
			`A <- ('x'*)*`,
			1, DiagnosticError, "Infinite loop"},
		{"optional plus",
			`A <- ('x'?)+`,
			1, DiagnosticError, "Infinite loop"},
		{"nullable rule ref star",
			`A <- B*
			 B <- 'x'?`,
			1, DiagnosticError, "Infinite loop"},
		{"deep nullable chain star",
			`A <- B*
			 B <- C
			 C <- D
			 D <- 'x'?`,
			1, DiagnosticError, "Infinite loop"},
		{"all nullable sequence star",
			`A <- ('x'? 'y'? 'z'?)*`,
			1, DiagnosticError, "Infinite loop"},
		{"nullable choice left always succeeds",
			`A <- ('x'? / 'y')*`,
			1, DiagnosticError, "Infinite loop"},
		{"nullable refs in sequence",
			`A <- (B C)*
			 B <- 'x'?
			 C <- 'y'?`,
			1, DiagnosticError, "Infinite loop"},

		// Input-dependent infinite loops produce warnings
		{"not predicate star",
			`A <- (!'x')*`,
			1, DiagnosticWarning, "Possible infinite loop"},
		{"and predicate star",
			`A <- (&'x')*`,
			1, DiagnosticWarning, "Possible infinite loop"},
		{"not guard with nullable ref",
			`A <- (!'x' B)*
			 B <- 'y'?`,
			1, DiagnosticWarning, "Possible infinite loop"},
		{"not guard with deep nullable chain",
			`A <- (!'x' B)*
			 B <- C
			 C <- 'y'?`,
			1, DiagnosticWarning, "Possible infinite loop"},
		{"predicate sequence star",
			`A <- (&'x' !'y')*`,
			1, DiagnosticWarning, "Possible infinite loop"},

		// Safe patterns: no infinite-loop diagnostics
		{"terminal star",
			`A <- 'x'*`,
			0, 0, ""},
		{"class star",
			`A <- [a-z]*`,
			0, 0, ""},
		{"solid prefix in sequence",
			`A <- ('x' 'y'?)*`,
			0, 0, ""},
		{"non-nullable rule star",
			`A <- B*
			 B <- 'x'`,
			0, 0, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := NewInMemoryImportLoader()
			loader.Add("test.peg", []byte(tt.grammar))
			cfg := NewConfig()
			cfg.SetBool("grammar.add_builtins", false)
			db := NewDatabase(cfg, loader)

			diags, err := Get(db, DiagnosticsQuery, FilePath("test.peg"))
			require.NoError(t, err)

			var loopDiags []Diagnostic
			for _, d := range diags {
				if d.Code == "infinite-loop" {
					loopDiags = append(loopDiags, d)
				}
			}
			require.Len(t, loopDiags, tt.wantCount, "unexpected number of infinite-loop diagnostics")
			if tt.wantCount > 0 {
				assert.Equal(t, tt.wantSeverity, loopDiags[0].Severity, "severity")
				assert.Contains(t, loopDiags[0].Message, tt.wantSubstr, "message")
			}
		})
	}
}
