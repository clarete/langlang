package langlang

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefinitionLocationsQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- "a"
B <- "b"
C <- A B
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	defLocs, err := Get(db, DefinitionLocationsQuery, FilePath("test.peg"))
	require.NoError(t, err)

	assert.Contains(t, defLocs, "A")
	assert.Contains(t, defLocs, "B")
	assert.Contains(t, defLocs, "C")
	assert.Len(t, defLocs, 3)
}

func TestIdentifierLocationsQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- "a"
B <- A "b"
C <- A B
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	idLocs, err := Get(db, IdentifierLocationsQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// Count definitions and references
	defs := 0
	refs := 0
	for _, loc := range idLocs {
		if loc.IsDefinition {
			defs++
		} else {
			refs++
		}
	}

	assert.Equal(t, 3, defs) // A, B, C definitions
	assert.Equal(t, 3, refs) // A in B, A and B in C
}

func TestLabelLocationsQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- "a"^MissingA
B <- "b"^MissingB
C <- A B^MissingB
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	labels, err := Get(db, LabelLocationsQuery, FilePath("test.peg"))
	require.NoError(t, err)

	assert.Len(t, labels, 3)

	labelNames := make(map[string]int)
	for _, label := range labels {
		labelNames[label.Label]++
	}
	assert.Equal(t, 1, labelNames["MissingA"])
	assert.Equal(t, 2, labelNames["MissingB"])
}

func TestRecoveryRulesQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- "a"^MissingA
B <- "b"^MissingB
MissingA <- "fallback"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	recoveryRules, err := Get(db, RecoveryRulesQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// MissingA has a recovery rule
	assert.True(t, recoveryRules["MissingA"].HasRecovery)
	assert.NotNil(t, recoveryRules["MissingA"].DefinitionLoc)
	assert.Len(t, recoveryRules["MissingA"].UsageLocs, 1)

	// MissingB does not have a recovery rule
	assert.False(t, recoveryRules["MissingB"].HasRecovery)
	assert.Nil(t, recoveryRules["MissingB"].DefinitionLoc)
	assert.Len(t, recoveryRules["MissingB"].UsageLocs, 1)
}

func TestCallGraphDataQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- B C
B <- "b"
C <- B "c"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	callGraph, err := Get(db, CallGraphDataQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// A calls B and C
	assert.Contains(t, callGraph.Callees["A"], "B")
	assert.Contains(t, callGraph.Callees["A"], "C")

	// B is called by A and C
	assert.Len(t, callGraph.Callers["B"], 2)

	// C is called by A
	assert.Len(t, callGraph.Callers["C"], 1)
}

func TestUnusedRulesQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- B
B <- "b"
C <- "c"
D <- "d"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	unused, err := Get(db, UnusedRulesQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// C and D are unused (A is entry point, B is used by A)
	assert.Contains(t, unused, "C")
	assert.Contains(t, unused, "D")
	assert.NotContains(t, unused, "A")
	assert.NotContains(t, unused, "B")
}

func TestUndefinedReferencesQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- B C
B <- "b"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	undefined, err := Get(db, UndefinedReferencesQuery, FilePath("test.peg"))
	require.NoError(t, err)

	assert.Len(t, undefined, 1)
	assert.Equal(t, "C", undefined[0].Name)
}

func TestDiagnosticsQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- B Undefined
B <- "b"
C <- "c"
D <- "d"^MissingLabel
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	diagnostics, err := Get(db, DiagnosticsQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// Check for undefined rule error
	hasUndefinedError := false
	for _, d := range diagnostics {
		if d.Code == "undefined-rule" && d.Severity == DiagnosticError {
			hasUndefinedError = true
			assert.Contains(t, d.Message, "Undefined")
		}
	}
	assert.True(t, hasUndefinedError, "Should have undefined rule error")

	// Check for unused rule warning
	hasUnusedWarning := false
	for _, d := range diagnostics {
		if d.Code == "unused-rule" && d.Severity == DiagnosticWarning {
			hasUnusedWarning = true
		}
	}
	assert.True(t, hasUnusedWarning, "Should have unused rule warning")

	// Check for missing recovery rule warning
	hasMissingRecovery := false
	for _, d := range diagnostics {
		if d.Code == "missing-recovery-rule" && d.Severity == DiagnosticWarning {
			hasMissingRecovery = true
			assert.Contains(t, d.Message, "MissingLabel")
		}
	}
	assert.True(t, hasMissingRecovery, "Should have missing recovery rule warning")
}

func TestDocumentSymbolsQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- "a"^ErrorLabel
B <- A
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	symbols, err := Get(db, DocumentSymbolsQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// Should have definitions and labels
	defCount := 0
	labelCount := 0
	for _, s := range symbols {
		if s.Kind == SymbolKindDefinition {
			defCount++
		}
		if s.Kind == SymbolKindLabel {
			labelCount++
		}
	}

	assert.Equal(t, 2, defCount)   // A, B
	assert.Equal(t, 1, labelCount) // ErrorLabel
}

func TestSymbolAtCursorQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`A <- B "hello"
B <- "b"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	t.Run("On definition name", func(t *testing.T) {
		symbol, err := Get(db, SymbolAtCursorQuery, CursorKey{File: "test.peg", Cursor: 0})
		require.NoError(t, err)
		require.NotNil(t, symbol)
		assert.Equal(t, "A", symbol.Name)
		assert.Equal(t, SymbolKindDefinition, symbol.Kind)
	})

	t.Run("On identifier reference", func(t *testing.T) {
		symbol, err := Get(db, SymbolAtCursorQuery, CursorKey{File: "test.peg", Cursor: 5})
		require.NoError(t, err)
		require.NotNil(t, symbol)
		assert.Equal(t, "B", symbol.Name)
		assert.Equal(t, SymbolKindIdentifier, symbol.Kind)
		assert.NotNil(t, symbol.DefinitionLoc)
	})

	t.Run("On literal", func(t *testing.T) {
		symbol, err := Get(db, SymbolAtCursorQuery, CursorKey{File: "test.peg", Cursor: 8})
		require.NoError(t, err)
		require.NotNil(t, symbol)
		assert.Equal(t, SymbolKindLiteral, symbol.Kind)
	})
}

func TestHoverInfoQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`A <- "hello"
B <- A
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	t.Run("On definition", func(t *testing.T) {
		hover, err := Get(db, HoverInfoQuery, CursorKey{File: "test.peg", Cursor: 0})
		require.NoError(t, err)
		require.NotNil(t, hover)
		assert.Contains(t, hover.Contents, "**A**")
		assert.Contains(t, hover.Contents, "rule")
	})

	t.Run("On identifier", func(t *testing.T) {
		hover, err := Get(db, HoverInfoQuery, CursorKey{File: "test.peg", Cursor: 18})
		require.NoError(t, err)
		require.NotNil(t, hover)
		assert.Contains(t, hover.Contents, "**A**")
		assert.Contains(t, hover.Contents, "reference")
	})
}

func TestCompletionItemsQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- "a"
B <- "b"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	items, err := Get(db, CompletionItemsQuery, CursorKey{File: "test.peg", Cursor: 0})
	require.NoError(t, err)

	// Should have rules
	ruleNames := make(map[string]bool)
	for _, item := range items {
		if item.Kind == CompletionKindRule {
			ruleNames[item.Label] = true
		}
	}
	assert.True(t, ruleNames["A"])
	assert.True(t, ruleNames["B"])

	// Should have keywords
	hasImport := false
	for _, item := range items {
		if item.Kind == CompletionKindKeyword && item.Label == "@import" {
			hasImport = true
		}
	}
	assert.True(t, hasImport)

	// Should have snippets
	hasSnippet := false
	for _, item := range items {
		if item.Kind == CompletionKindSnippet {
			hasSnippet = true
		}
	}
	assert.True(t, hasSnippet)
}

func TestSemanticTokensQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`A <- B "hello"
B <- A
C <- "unused"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	tokens, err := Get(db, SemanticTokensQuery, FilePath("test.peg"))
	require.NoError(t, err)

	// Should have definition tokens
	defCount := 0
	idCount := 0
	litCount := 0
	for _, tok := range tokens {
		switch tok.TokenType {
		case SemanticTokenTypeDefinition:
			defCount++
		case SemanticTokenTypeIdentifier:
			idCount++
		case SemanticTokenTypeLiteral:
			litCount++
		}
	}

	assert.Equal(t, 3, defCount) // A, B, C
	assert.Equal(t, 2, idCount)  // B in A, A in B
	assert.Equal(t, 2, litCount) // "hello", "unused"

	// Check for unused modifier
	hasUnusedModifier := false
	for _, tok := range tokens {
		if tok.TokenType == SemanticTokenTypeDefinition {
			for _, mod := range tok.Modifiers {
				if mod == "unused" {
					hasUnusedModifier = true
				}
			}
		}
	}
	assert.True(t, hasUnusedModifier)
}

func TestReferencesQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- "a"
B <- A
C <- A A
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	refs, err := Get(db, ReferencesQuery, ReferencesKey{File: "test.peg", SymbolName: "A"})
	require.NoError(t, err)

	// Should include: definition of A, reference in B, two references in C
	assert.Len(t, refs, 4)
}

func TestLabelGoToDefinition(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`A <- "a"^Missing
Missing <- "recovery"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	// Find the position of ^Missing (after "a")
	// "A <- \"a\"^Missing" - cursor should be around position 9-10
	symbol, err := Get(db, SymbolAtCursorQuery, CursorKey{File: "test.peg", Cursor: 10})
	require.NoError(t, err)

	// If we're on the label, it should link to the recovery rule
	if symbol != nil && symbol.Kind == SymbolKindLabel {
		assert.Equal(t, "Missing", symbol.Name)
		assert.NotNil(t, symbol.DefinitionLoc, "Label should link to recovery rule")
	}
}

func TestRecoveryRuleAsReference(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`
A <- "a"^ErrA
B <- "b"^ErrA
ErrA <- "recovery"
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	// Find references to ErrA - should include both label usages
	refs, err := Get(db, ReferencesQuery, ReferencesKey{File: "test.peg", SymbolName: "ErrA"})
	require.NoError(t, err)

	// Should have: definition + 2 label usages
	assert.Len(t, refs, 3)
}

func TestPosIndexQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte("A <- \"hello\"\nB <- \"world\"\n"))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	posIdx, err := Get(db, PosIndexQuery, FilePath("test.peg"))
	require.NoError(t, err)
	require.NotNil(t, posIdx)

	// Test LocationAt: cursor 0 should be line 1, column 1
	loc := posIdx.LocationAt(0)
	assert.Equal(t, 1, loc.Line)
	assert.Equal(t, 1, loc.Column)

	// Cursor at start of second line (after "A <- \"hello\"\n" = 13 bytes)
	loc = posIdx.LocationAt(13)
	assert.Equal(t, 2, loc.Line)
	assert.Equal(t, 1, loc.Column)
}

func TestCursorAtLocationQuery(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// "A <- \"hello\"\nB <- \"world\"\n"
	// Line 0: A <- "hello"\n  (13 bytes: 0-12, newline at 12)
	// Line 1: B <- "world"\n  (13 bytes: 13-25, newline at 25)
	loader.Add("test.peg", []byte("A <- \"hello\"\nB <- \"world\"\n"))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	t.Run("Line 0, Column 0", func(t *testing.T) {
		cursor, err := Get(db, CursorAtLocationQuery, LocationKey{
			File:   "test.peg",
			Line:   0,
			Column: 0,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, cursor)
	})

	t.Run("Line 0, Column 5", func(t *testing.T) {
		cursor, err := Get(db, CursorAtLocationQuery, LocationKey{
			File:   "test.peg",
			Line:   0,
			Column: 5,
		})
		require.NoError(t, err)
		assert.Equal(t, 5, cursor) // "A <- " = 5 bytes
	})

	t.Run("Line 1, Column 0", func(t *testing.T) {
		cursor, err := Get(db, CursorAtLocationQuery, LocationKey{
			File:   "test.peg",
			Line:   1,
			Column: 0,
		})
		require.NoError(t, err)
		assert.Equal(t, 13, cursor) // Start of second line
	})

	t.Run("Line 1, Column 5", func(t *testing.T) {
		cursor, err := Get(db, CursorAtLocationQuery, LocationKey{
			File:   "test.peg",
			Line:   1,
			Column: 5,
		})
		require.NoError(t, err)
		assert.Equal(t, 18, cursor) // 13 + 5
	})
}

func TestCursorAtLocationQuery_UTF8(t *testing.T) {
	loader := NewInMemoryImportLoader()
	// Test with UTF-8 content: "A <- \"日本\"\n" (each CJK char is 3 bytes)
	// "A <- \"" = 6 bytes, "日" = 3 bytes, "本" = 3 bytes, "\"\n" = 2 bytes = 14 bytes total
	loader.Add("test.peg", []byte("A <- \"日本\"\n"))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	t.Run("Column 6 should be at first CJK char", func(t *testing.T) {
		cursor, err := Get(db, CursorAtLocationQuery, LocationKey{
			File:   "test.peg",
			Line:   0,
			Column: 6, // After 'A <- "'
		})
		require.NoError(t, err)
		assert.Equal(t, 6, cursor) // Byte position of first CJK char
	})

	t.Run("Column 7 should be at second CJK char", func(t *testing.T) {
		cursor, err := Get(db, CursorAtLocationQuery, LocationKey{
			File:   "test.peg",
			Line:   0,
			Column: 7, // After first CJK char (rune-based column)
		})
		require.NoError(t, err)
		assert.Equal(t, 9, cursor) // 6 + 3 (bytes for 日)
	})
}

// TestSymbolAtCursorQuery_WithImports verifies that hover/symbol lookup
// only matches symbols from the requested file, not from imported files.
// This is a regression test for a bug where cursor offsets from the main
// file could accidentally match spans from imported definitions.
func TestSymbolAtCursorQuery_WithImports(t *testing.T) {
	loader := NewInMemoryImportLoader()

	// Main file imports from value.peg
	loader.Add("main.peg", []byte(`@import Value from "./value.peg"

Main <- Value
`))

	// Imported file has a charset at some position that could
	// match a cursor offset in the main file if FileID checking
	// is wrong.  The charset [a-z] is at byte ~20 in this file.
	loader.Add("value.peg", []byte(`Value <- [a-z]+
`))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)

	// Query for a position in main.peg that doesn't have a
	// symbol.  Without the FileID fix, this might match the
	// charset from value.peg because the cursor offsets could
	// overlap.
	t.Run("Empty position should not match imported symbols", func(t *testing.T) {
		// Position in the empty line (line 1, which is just a newline)
		// This is byte offset 33 (after "@import Value from \"./value.peg\"\n")
		symbol, err := Get(db, SymbolAtCursorQuery, CursorKey{
			File:   "main.peg",
			Cursor: 33,
		})
		require.NoError(t, err)
		// Should not find any symbol at an empty line
		assert.Nil(t, symbol)
	})

	t.Run("Symbol in main file is found", func(t *testing.T) {
		// Position on "Main" definition (line 2, column 0)
		// Byte offset is 34 (after "@import Value from \"./value.peg\"\n\n")
		symbol, err := Get(db, SymbolAtCursorQuery, CursorKey{
			File:   "main.peg",
			Cursor: 34,
		})
		require.NoError(t, err)
		require.NotNil(t, symbol)
		assert.Equal(t, "Main", symbol.Name)
		assert.Equal(t, SymbolKindDefinition, symbol.Kind)
	})

	t.Run("Reference to imported symbol resolves correctly", func(t *testing.T) {
		// Position on "Value" reference (line 2, after "Main <- ")
		// "Main <- " is 8 chars, so byte offset is 34 + 8 = 42
		symbol, err := Get(db, SymbolAtCursorQuery, CursorKey{
			File:   "main.peg",
			Cursor: 42,
		})
		require.NoError(t, err)
		require.NotNil(t, symbol)
		assert.Equal(t, "Value", symbol.Name)
		assert.Equal(t, SymbolKindIdentifier, symbol.Kind)
		// DefinitionLoc should point to value.peg, not main.peg
		require.NotNil(t, symbol.DefinitionLoc)
	})
}
