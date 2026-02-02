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

func TestQueryMatcher(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- "hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	matcher, err := QueryMatcher(db, "test.peg")
	require.NoError(t, err)
	require.NotNil(t, matcher)

	tree, n, err := matcher.Match([]byte("hello"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.NotNil(t, tree)
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

func TestDeepDependencyChainCaching(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`G <- "hello"`))

	cfg := NewConfig()
	db := NewDatabase(cfg, loader)

	// Query ShouldInlineQuery which has deep dependencies
	_, err := Get(db, ShouldInlineQuery, DefKey{File: "test.peg", Name: "G"})
	require.NoError(t, err)

	stats := db.Stats()
	// Should have many queries cached due to dependency chain
	assert.Greater(t, stats.CachedCount, 5)
	assert.Greater(t, stats.DepsCount, 0)
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

func TestErrorLabelsQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- "a"^ErrA
B <- "b"^ErrB
C <- A^ErrA B
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	labels, err := Get(db, ErrorLabelsQuery, FilePath("test.peg"))
	require.NoError(t, err)

	assert.Contains(t, labels, "ErrA")
	assert.Contains(t, labels, "ErrB")
	assert.Len(t, labels, 2)
}

func TestDefinitionDepsQuery(t *testing.T) {
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

	deps, err := Get(db, DefinitionDepsQuery, DefKey{File: "test.peg", Name: "A"})
	require.NoError(t, err)

	// A depends on B, C, and transitively on D
	assert.Contains(t, deps, "B")
	assert.Contains(t, deps, "C")
	assert.Contains(t, deps, "D")
}

func TestShouldInlineQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
Entry <- Small Large Recursive Recovery
Small <- "a"
Large <- "a" "b" "c" "d" "e" "f" "g" "h" "i" "j" "k" "l" "m" "n" "o"
Recursive <- "x" Recursive / "y"
Recovery <- "recovery"
Other <- Small^Recovery
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	cfg.SetBool("compiler.inline.enabled", true)
	cfg.SetInt("compiler.inline.max_size", 10)
	db := NewDatabase(cfg, loader)

	t.Run("Small rule should be inlined", func(t *testing.T) {
		shouldInline, err := Get(db, ShouldInlineQuery, DefKey{File: "test.peg", Name: "Small"})
		require.NoError(t, err)
		assert.True(t, shouldInline)
	})

	t.Run("Large rule should not be inlined", func(t *testing.T) {
		shouldInline, err := Get(db, ShouldInlineQuery, DefKey{File: "test.peg", Name: "Large"})
		require.NoError(t, err)
		assert.False(t, shouldInline)
	})

	t.Run("Recursive rule should not be inlined", func(t *testing.T) {
		shouldInline, err := Get(db, ShouldInlineQuery, DefKey{File: "test.peg", Name: "Recursive"})
		require.NoError(t, err)
		assert.False(t, shouldInline)
	})

	t.Run("Recovery rule should not be inlined", func(t *testing.T) {
		shouldInline, err := Get(db, ShouldInlineQuery, DefKey{File: "test.peg", Name: "Recovery"})
		require.NoError(t, err)
		assert.False(t, shouldInline)
	})

	t.Run("Entry point should not be inlined", func(t *testing.T) {
		shouldInline, err := Get(db, ShouldInlineQuery, DefKey{File: "test.peg", Name: "Entry"})
		require.NoError(t, err)
		assert.False(t, shouldInline)
	})
}

func TestIsSyntacticQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
Syntactic <- "hello" [a-z]+
NonSyntactic <- Syntactic Other
Other <- "other"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	t.Run("Syntactic rule", func(t *testing.T) {
		isSyntactic, err := Get(db, IsSyntacticQuery, DefKey{File: "test.peg", Name: "Syntactic"})
		require.NoError(t, err)
		assert.True(t, isSyntactic)
	})

	t.Run("Non-syntactic rule (has identifier)", func(t *testing.T) {
		isSyntactic, err := Get(db, IsSyntacticQuery, DefKey{File: "test.peg", Name: "NonSyntactic"})
		require.NoError(t, err)
		assert.False(t, isSyntactic)
	})
}

func TestCapExprSizeQuery(t *testing.T) {
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

	t.Run("Fixed size literal", func(t *testing.T) {
		result, err := Get(db, CapExprSizeQuery, DefKey{File: "test.peg", Name: "Fixed"})
		require.NoError(t, err)
		assert.True(t, result.IsFixed)
		assert.Equal(t, 2, result.Size)
	})

	t.Run("Variable size choice", func(t *testing.T) {
		result, err := Get(db, CapExprSizeQuery, DefKey{File: "test.peg", Name: "Variable"})
		require.NoError(t, err)
		assert.False(t, result.IsFixed)
	})

	t.Run("Fixed size choice", func(t *testing.T) {
		result, err := Get(db, CapExprSizeQuery, DefKey{File: "test.peg", Name: "FixedChoice"})
		require.NoError(t, err)
		assert.True(t, result.IsFixed)
		assert.Equal(t, 2, result.Size)
	})

	t.Run("Fixed size range (1 char)", func(t *testing.T) {
		result, err := Get(db, CapExprSizeQuery, DefKey{File: "test.peg", Name: "FixedRange"})
		require.NoError(t, err)
		// Character ranges always match exactly 1 character
		assert.True(t, result.IsFixed)
		assert.Equal(t, 1, result.Size)
	})
}

func TestStringTableQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- "a"^ErrA
B <- "b"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	st, err := Get(db, StringTableQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// Should have rule names and error labels
	assert.Contains(t, st.StringsMap, "A")
	assert.Contains(t, st.StringsMap, "B")
	assert.Contains(t, st.StringsMap, "ErrA")

	// First string should be empty sentinel
	assert.Equal(t, "", st.Strings[0])
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
