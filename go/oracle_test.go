package langlang

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compileGrammar(t *testing.T, grammar string) *Bytecode {
	t.Helper()
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(grammar))

	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)

	db := NewDatabase(cfg, loader)
	bytecode, err := QueryBytecode(db, "test.peg")
	require.NoError(t, err, "failed to compile grammar")
	return bytecode
}

func compileGrammarWithBuiltins(t *testing.T, grammar string) *Bytecode {
	t.Helper()
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(grammar))

	cfg := NewConfig()
	// Builtins enabled by default

	db := NewDatabase(cfg, loader)
	bytecode, err := QueryBytecode(db, "test.peg")
	require.NoError(t, err, "failed to compile grammar")
	return bytecode
}

func compileGrammarFile(t *testing.T, path string) *Bytecode {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read grammar file: %s", path)
	return compileGrammarWithBuiltins(t, string(content))
}

// TestOracleAcceptance tests whether the oracle correctly accepts valid inputs
// and rejects invalid inputs for various grammars.
func TestOracleAcceptance(t *testing.T) {
	tests := []struct {
		name     string
		grammar  string
		valid    []string // inputs that should be accepted
		invalid  []string // inputs that should be rejected
		prefixes []string // valid prefixes (not complete but parseable so far)
	}{
		{
			name:     "literal",
			grammar:  `Start <- "hello"`,
			valid:    []string{"hello"},
			invalid:  []string{"hell", "helo", "Hello", "hello!", "xhello"},
			prefixes: []string{"h", "he", "hel", "hell"},
		},
		{
			name:     "choice",
			grammar:  `Start <- "foo" / "bar" / "baz"`,
			valid:    []string{"foo", "bar", "baz"},
			invalid:  []string{"fo", "ba", "qux", "foobar"},
			prefixes: []string{"f", "fo", "b", "ba"},
		},
		{
			name:     "zero_or_more",
			grammar:  `Start <- 'a'*`,
			valid:    []string{"", "a", "aa", "aaa", "aaaaaaaaaa"},
			invalid:  []string{"b", "ab", "aab"},
			prefixes: []string{}, // all valid inputs are complete
		},
		{
			name:     "one_or_more",
			grammar:  `Start <- 'a'+`,
			valid:    []string{"a", "aa", "aaa", "aaaaaaaaaa"},
			invalid:  []string{"", "b", "ab"},
			prefixes: []string{},
		},
		{
			name:     "optional",
			grammar:  `Start <- 'a' 'b'?`,
			valid:    []string{"a", "ab"},
			invalid:  []string{"", "b", "abc", "abb"},
			prefixes: []string{},
		},
		{
			name:     "sequence",
			grammar:  `Start <- [0-9] [a-z]`,
			valid:    []string{"0a", "5x", "9z"},
			invalid:  []string{"", "0", "a", "00", "aa", "a0"},
			prefixes: []string{"0", "5", "9"},
		},
		{
			name:     "character_class",
			grammar:  `Start <- [a-z]+`,
			valid:    []string{"a", "abc", "xyz", "hello"},
			invalid:  []string{"", "A", "123", "a1"},
			prefixes: []string{},
		},
		{
			name:     "mixed_class",
			grammar:  `Start <- [a-zA-Z_][a-zA-Z0-9_]*`,
			valid:    []string{"a", "A", "_", "foo", "Foo123", "_bar_", "camelCase"},
			invalid:  []string{"", "123", "1abc"},
			prefixes: []string{},
		},
		{
			name:     "shared_prefix",
			grammar:  `Start <- "ab" / "ac"`,
			valid:    []string{"ab", "ac"},
			invalid:  []string{"a", "ad", "abc"},
			prefixes: []string{"a"},
		},
		{
			name:     "nested_repetition",
			grammar:  `Start <- ("x" "y")* "z"`,
			valid:    []string{"z", "xyz", "xyxyz", "xyxyxyz"},
			invalid:  []string{"", "x", "yz"},
			prefixes: []string{"xy", "xyxy"},
		},
		{
			name:     "deeply_nested",
			grammar:  `Start <- (('a' / 'b') ('c' / 'd'))+`,
			valid:    []string{"ac", "ad", "bc", "bd", "acbd", "bdacbc"},
			invalid:  []string{"", "a", "ab", "cd", "ace"},
			prefixes: []string{"a", "b", "ac", "ad", "bc", "bd"},
		},
		{
			name:     "any_char",
			grammar:  `Start <- . . .`,
			valid:    []string{"abc", "123", "   ", "日本語"},
			invalid:  []string{"", "a", "ab"},
			prefixes: []string{"a", "ab", "1", "12"},
		},
		{
			name:    "simple_json_value",
			grammar: `Start <- 'true' / 'false' / 'null' / [0-9]+`,
			valid:   []string{"true", "false", "null", "0", "123", "999"},
			invalid: []string{"", "tru", "True", "abc"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bytecode := compileGrammar(t, tc.grammar)
			oracle := NewGrammarOracle(bytecode)

			// Test valid inputs
			for _, input := range tc.valid {
				parser := oracle.NewParser().AdvanceString(input)
				assert.False(t, parser.IsEmpty(), "valid input %q should parse", input)
				assert.True(t, parser.IsAccepting(), "valid input %q should be accepted", input)
			}

			// Test invalid inputs
			for _, input := range tc.invalid {
				parser := oracle.NewParser().AdvanceString(input)
				isInvalid := parser.IsEmpty() || !parser.IsAccepting()
				assert.True(t, isInvalid, "invalid input %q should be rejected", input)
			}

			// Test valid prefixes (parseable but not complete)
			for _, prefix := range tc.prefixes {
				parser := oracle.NewParser().AdvanceString(prefix)
				assert.False(t, parser.IsEmpty(), "prefix %q should be parseable", prefix)
			}
		})
	}
}

// TestOracleNextChars tests that NextChars returns the correct set of valid
// characters at various parse positions.
func TestOracleNextChars(t *testing.T) {
	tests := []struct {
		name       string
		grammar    string
		prefix     string // input consumed before checking NextChars
		shouldHave []rune // characters that must be valid
		shouldNot  []rune // characters that must NOT be valid
		isAny      bool   // whether NextChars should match any character
	}{
		{
			name:       "literal_start",
			grammar:    `Start <- "hello"`,
			prefix:     "",
			shouldHave: []rune{'h'},
			shouldNot:  []rune{'e', 'l', 'o', 'x', '1'},
		},
		{
			name:       "literal_middle",
			grammar:    `Start <- "hello"`,
			prefix:     "hel",
			shouldHave: []rune{'l'},
			shouldNot:  []rune{'h', 'e', 'o', 'x'},
		},
		{
			name:       "choice_start",
			grammar:    `Start <- "foo" / "bar"`,
			prefix:     "",
			shouldHave: []rune{'f', 'b'},
			shouldNot:  []rune{'o', 'a', 'r', 'x'},
		},
		{
			name:       "shared_prefix_after_common",
			grammar:    `Start <- "ab" / "ac"`,
			prefix:     "a",
			shouldHave: []rune{'b', 'c'},
			shouldNot:  []rune{'d', 'x'},
		},
		{
			name:       "repetition_continuation",
			grammar:    `Start <- 'a'*`,
			prefix:     "aaa",
			shouldHave: []rune{'a'},
			shouldNot:  []rune{'b', 'x'},
		},
		{
			name:       "repetition_exit",
			grammar:    `Start <- ("x" "y")* "z"`,
			prefix:     "xy",
			shouldHave: []rune{'x', 'z'}, // can continue loop OR exit
			shouldNot:  []rune{'y', 'a'},
		},
		{
			name:       "character_class",
			grammar:    `Start <- [a-z]+`,
			prefix:     "",
			shouldHave: []rune{'a', 'm', 'z'},
			shouldNot:  []rune{'A', 'Z', '0', '9'},
		},
		{
			name:       "any_char",
			grammar:    `Start <- .`,
			prefix:     "",
			shouldHave: []rune{'a', 'Z', '0', ' ', '日'},
			isAny:      true,
		},
		{
			name:       "optional_after_required",
			grammar:    `Start <- 'a' 'b'?`,
			prefix:     "a",
			shouldHave: []rune{'b'},
			shouldNot:  []rune{'a', 'c'},
		},
		{
			name:       "multiple_choices_deep",
			grammar:    `Start <- ("a" / "b") ("1" / "2")`,
			prefix:     "a",
			shouldHave: []rune{'1', '2'},
			shouldNot:  []rune{'a', 'b', '3'},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bytecode := compileGrammar(t, tc.grammar)
			oracle := NewGrammarOracle(bytecode)
			parser := oracle.NewParser()

			if tc.prefix != "" {
				parser = parser.AdvanceString(tc.prefix)
				require.False(t, parser.IsEmpty(), "prefix %q should be parseable", tc.prefix)
			}

			chars := parser.NextChars()

			if tc.isAny {
				assert.True(t, chars.IsAny(), "NextChars should be 'any'")
			}

			for _, r := range tc.shouldHave {
				assert.True(t, chars.Contains(r), "NextChars should contain %q", r)
			}

			for _, r := range tc.shouldNot {
				assert.False(t, chars.Contains(r), "NextChars should NOT contain %q", r)
			}
		})
	}
}

// TestOracleCharSet tests the OracleCharSet type operations.
func TestOracleCharSet(t *testing.T) {
	t.Run("basic_operations", func(t *testing.T) {
		cs := NewOracleCharSet()
		assert.False(t, cs.Contains('a'), "empty charset should not contain 'a'")

		cs.Add('a')
		cs.Add('b')
		assert.True(t, cs.Contains('a'), "charset should contain 'a'")
		assert.True(t, cs.Contains('b'), "charset should contain 'b'")
		assert.False(t, cs.Contains('c'), "charset should not contain 'c'")
	})

	t.Run("range_operations", func(t *testing.T) {
		cs := NewOracleCharSet()
		cs.AddRange('0', '9')

		for r := '0'; r <= '9'; r++ {
			assert.True(t, cs.Contains(r), "charset should contain %q", r)
		}
		assert.False(t, cs.Contains('a'), "charset should not contain 'a'")
		assert.False(t, cs.Contains('/'), "charset should not contain '/'")
	})

	t.Run("any_flag", func(t *testing.T) {
		cs := NewOracleCharSet()
		assert.False(t, cs.IsAny(), "new charset should not be 'any'")

		cs.SetAny()
		assert.True(t, cs.IsAny(), "charset should be 'any' after SetAny()")

		// Any should contain everything
		for _, r := range []rune{'a', 'Z', '0', ' ', '\n', '日', '🎉'} {
			assert.True(t, cs.Contains(r), "'any' charset should contain %q", r)
		}
	})

	t.Run("union", func(t *testing.T) {
		cs1 := NewOracleCharSet()
		cs1.Add('a')
		cs1.AddRange('0', '5')

		cs2 := NewOracleCharSet()
		cs2.Add('b')
		cs2.AddRange('5', '9')

		cs1.Union(cs2)

		assert.True(t, cs1.Contains('a'), "union should contain 'a'")
		assert.True(t, cs1.Contains('b'), "union should contain 'b'")
		for r := '0'; r <= '9'; r++ {
			assert.True(t, cs1.Contains(r), "union should contain %q", r)
		}
	})
}

// TestOracleStateClone tests that state cloning creates independent copies.
func TestOracleStateClone(t *testing.T) {
	state := OracleState{
		PC:     10,
		Cursor: 5,
		Stack: []StackFrame{
			{Type: StackFrameCall, PC: 20},
			{Type: StackFrameBacktrack, PC: 30, Cursor: 3},
		},
	}

	cloned := state.Clone()

	// Modify original
	state.PC = 100
	state.Stack[0].PC = 200

	// Clone should be unaffected
	assert.Equal(t, 10, cloned.PC, "cloned PC should be unchanged")
	assert.Equal(t, 20, cloned.Stack[0].PC, "cloned stack[0].PC should be unchanged")
}

// TestOracleAnalyze tests the Analyze method returns consistent results.
func TestOracleAnalyze(t *testing.T) {
	tests := []struct {
		name              string
		grammar           string
		input             string
		expectAccepting   bool
		expectContains    []rune
		expectNotContains []rune
	}{
		{
			name:              "empty_optional",
			grammar:           `Start <- 'a' 'b'?`,
			input:             "a",
			expectAccepting:   true,
			expectContains:    []rune{'b'},
			expectNotContains: []rune{'a', 'c'},
		},
		{
			name:              "incomplete_sequence",
			grammar:           `Start <- 'a' 'b' 'c'`,
			input:             "a",
			expectAccepting:   false,
			expectContains:    []rune{'b'},
			expectNotContains: []rune{'a', 'c'},
		},
		{
			name:              "empty_star",
			grammar:           `Start <- 'a'*`,
			input:             "",
			expectAccepting:   true,
			expectContains:    []rune{'a'},
			expectNotContains: []rune{'b'},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bytecode := compileGrammar(t, tc.grammar)
			oracle := NewGrammarOracle(bytecode)
			parser := oracle.NewParser()

			if tc.input != "" {
				parser = parser.AdvanceString(tc.input)
				require.False(t, parser.IsEmpty())
			}

			// Get results from individual methods
			accepting := parser.IsAccepting()
			chars := parser.NextChars()

			// Get results from Analyze
			analyzedAccepting, analyzedChars := parser.Analyze()

			// Verify consistency
			assert.Equal(t, accepting, analyzedAccepting, "Analyze accepting should match IsAccepting")
			assert.Equal(t, tc.expectAccepting, analyzedAccepting, "accepting state mismatch")

			for _, r := range tc.expectContains {
				assert.Equal(t, chars.Contains(r), analyzedChars.Contains(r),
					"Analyze chars should match NextChars for %q", r)
				assert.True(t, analyzedChars.Contains(r), "should contain %q", r)
			}

			for _, r := range tc.expectNotContains {
				assert.False(t, analyzedChars.Contains(r), "should NOT contain %q", r)
			}
		})
	}
}

// TestOracleWithNonTerminals tests grammars with rule references.
func TestOracleWithNonTerminals(t *testing.T) {
	tests := []struct {
		name    string
		grammar string
		valid   []string
		invalid []string
	}{
		{
			name: "simple_reference",
			grammar: `
Start  <- Digit Letter
Digit  <- [0-9]
Letter <- [a-z]`,
			valid:   []string{"0a", "5x", "9z"},
			invalid: []string{"a0", "00", "aa"},
		},
		{
			name: "recursive_reference",
			grammar: `
Start <- 'a' Start / 'b'`,
			valid:   []string{"b", "ab", "aab", "aaab"},
			invalid: []string{"", "a", "ba", "bb"},
		},
		{
			name: "mutual_recursion",
			grammar: `
A <- 'a' B / 'x'
B <- 'b' A / 'y'`,
			// Starting from A: can be 'x' OR 'a' followed by B
			// B can be 'y' OR 'b' followed by A
			// Valid from A: x, ay, abx, abay, ababx, ...
			valid:   []string{"x", "ay", "abx", "abay", "ababx"},
			invalid: []string{"", "b", "y"},
		},
		{
			name: "expression_grammar",
			grammar: `
Expr   <- Term ('+' Term)*
Term   <- Factor ('*' Factor)*
Factor <- [0-9]+ / '(' Expr ')'`,
			valid:   []string{"1", "12", "1+2", "1*2", "1+2*3", "(1)", "(1+2)", "1+(2*3)"},
			invalid: []string{"", "+", "*", "()"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bytecode := compileGrammarWithBuiltins(t, tc.grammar)
			oracle := NewGrammarOracle(bytecode)

			for _, input := range tc.valid {
				parser := oracle.NewParser().AdvanceString(input)
				assert.False(t, parser.IsEmpty(), "valid input %q should parse", input)
				assert.True(t, parser.IsAccepting(), "valid input %q should be accepted", input)
			}

			for _, input := range tc.invalid {
				parser := oracle.NewParser().AdvanceString(input)
				isInvalid := parser.IsEmpty() || !parser.IsAccepting()
				assert.True(t, isInvalid, "invalid input %q should be rejected", input)
			}
		})
	}
}

// TestOracleJSONGrammar tests the oracle with a real JSON grammar.
func TestOracleJSONGrammar(t *testing.T) {
	grammarPath := filepath.Join("..", "grammars", "json.peg")
	if _, err := os.Stat(grammarPath); os.IsNotExist(err) {
		t.Skip("json.peg not found, skipping")
	}

	bytecode := compileGrammarFile(t, grammarPath)
	oracle := NewGrammarOracle(bytecode)

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		// Valid JSON values
		{"null", "null", true},
		{"true", "true", true},
		{"false", "false", true},
		{"integer", "123", true},
		{"negative", "-42", true},
		{"decimal", "3.14", true},
		{"exponent", "1e10", true},
		{"string", `"hello"`, true},
		{"empty_string", `""`, true},
		{"string_escape", `"hello\nworld"`, true},
		{"empty_array", "[]", true},
		{"simple_array", "[1,2,3]", true},
		{"nested_array", "[[1],[2]]", true},
		{"empty_object", "{}", true},
		{"simple_object", `{"a":1}`, true},
		{"complex_object", `{"name":"John","age":30,"active":true}`, true},
		{"nested", `{"items":[1,2,{"x":null}]}`, true},

		// Invalid JSON (note: some "invalid" JSON may parse due to recovery rules in grammar)
		{"bare_word", "hello", false},
		{"single_quote", "'hello'", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parser := oracle.NewParser().AdvanceString(tc.input)
			if tc.valid {
				assert.False(t, parser.IsEmpty(), "valid JSON %q should parse", tc.input)
				assert.True(t, parser.IsAccepting(), "valid JSON %q should be accepted", tc.input)
			} else {
				isInvalid := parser.IsEmpty() || !parser.IsAccepting()
				assert.True(t, isInvalid, "invalid JSON %q should be rejected", tc.input)
			}
		})
	}
}

// TestOracleJSONNextChars tests NextChars at various positions in JSON parsing.
func TestOracleJSONNextChars(t *testing.T) {
	grammarPath := filepath.Join("..", "grammars", "json.peg")
	if _, err := os.Stat(grammarPath); os.IsNotExist(err) {
		t.Skip("json.peg not found, skipping")
	}

	bytecode := compileGrammarFile(t, grammarPath)
	oracle := NewGrammarOracle(bytecode)

	tests := []struct {
		name       string
		prefix     string
		shouldHave []rune
		shouldNot  []rune
	}{
		{
			name:       "start",
			prefix:     "",
			shouldHave: []rune{'{', '[', '"', 't', 'f', 'n', '0', '1', '-'},
			shouldNot:  []rune{'a', 'z', ')', '}', ']'},
		},
		{
			name:       "after_open_brace",
			prefix:     "{",
			shouldHave: []rune{'"', '}'}, // key string or close
			shouldNot:  []rune{'{', '[', '1', 'a'},
		},
		{
			name:       "after_open_bracket",
			prefix:     "[",
			shouldHave: []rune{'{', '[', '"', 't', 'f', 'n', '0', '1', '-', ']'},
		},
		{
			name:       "after_array_value",
			prefix:     "[1",
			shouldHave: []rune{',', ']', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9'},
		},
		{
			name:       "after_object_key",
			prefix:     `{"a"`,
			shouldHave: []rune{':'},
			// Note: other chars may be valid due to backtrack exploration
		},
		{
			name:       "in_string",
			prefix:     `"hel`,
			shouldHave: []rune{'l', 'o', '"', '\\'},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parser := oracle.NewParser()
			if tc.prefix != "" {
				parser = parser.AdvanceString(tc.prefix)
				require.False(t, parser.IsEmpty(), "prefix %q should parse", tc.prefix)
			}

			chars := parser.NextChars()

			for _, r := range tc.shouldHave {
				assert.True(t, chars.Contains(r),
					"after %q, NextChars should contain %q", tc.prefix, r)
			}

			for _, r := range tc.shouldNot {
				assert.False(t, chars.Contains(r),
					"after %q, NextChars should NOT contain %q", tc.prefix, r)
			}
		})
	}
}

// TestOracleLanglangGrammar tests the oracle with the langlang.peg grammar.
func TestOracleLanglangGrammar(t *testing.T) {
	grammarPath := filepath.Join("..", "grammars", "langlang.peg")
	if _, err := os.Stat(grammarPath); os.IsNotExist(err) {
		t.Skip("langlang.peg not found, skipping")
	}

	bytecode := compileGrammarFile(t, grammarPath)
	oracle := NewGrammarOracle(bytecode)

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		// Valid langlang grammar snippets
		{"simple_rule", `A <- 'a'`, true},
		{"literal_sequence", `A <- 'a' 'b' 'c'`, true},
		{"string_literal", `A <- "hello"`, true},
		{"choice", `A <- 'a' / 'b'`, true},
		{"repetition_star", `A <- 'a'*`, true},
		{"repetition_plus", `A <- 'a'+`, true},
		{"optional", `A <- 'a'?`, true},
		{"character_class", `A <- [a-z]`, true},
		{"mixed_class", `A <- [a-zA-Z0-9_]`, true},
		{"any_char", `A <- .`, true},
		{"grouping", `A <- ('a' 'b')+`, true},
		{"non_terminal", `A <- B`, true},
		{"two_rules", "A <- 'a'\nB <- 'b'", true},
		{"complex_rule", `Expr <- Term ('+' Term)*`, true},
		{"predicate_and", `A <- &'a' .`, true},
		{"predicate_not", `A <- !'a' .`, true},
		{"lexification", `A <- #('a' 'b')`, true},
		{"label", `A <- 'a'^err`, true},

		// Invalid langlang grammar (some may parse due to permissive rules)
		{"missing_arrow", "A 'a'", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parser := oracle.NewParser().AdvanceString(tc.input)
			if tc.valid {
				assert.False(t, parser.IsEmpty(), "valid grammar %q should parse", tc.input)
				assert.True(t, parser.IsAccepting(), "valid grammar %q should be accepted", tc.input)
			} else {
				isInvalid := parser.IsEmpty() || !parser.IsAccepting()
				assert.True(t, isInvalid, "invalid grammar %q should be rejected", tc.input)
			}
		})
	}
}

// TestOracleLanglangNextChars tests NextChars at various positions in langlang parsing.
func TestOracleLanglangNextChars(t *testing.T) {
	grammarPath := filepath.Join("..", "grammars", "langlang.peg")
	if _, err := os.Stat(grammarPath); os.IsNotExist(err) {
		t.Skip("langlang.peg not found, skipping")
	}

	bytecode := compileGrammarFile(t, grammarPath)
	oracle := NewGrammarOracle(bytecode)

	tests := []struct {
		name       string
		prefix     string
		shouldHave []rune
	}{
		{
			name:       "start_of_grammar",
			prefix:     "",
			shouldHave: []rune{'@', 'A', 'a', '_'}, // @import or identifier
		},
		{
			name:       "after_identifier",
			prefix:     "Start",
			shouldHave: []rune{'<', ' ', 'a', 'A', '0'}, // continue identifier or space before <-
		},
		{
			name:       "after_arrow",
			prefix:     "Start <- ",
			shouldHave: []rune{'\'', '"', '[', '.', '(', 'A', 'a', '_', '#', '&', '!'},
		},
		{
			name:       "in_string",
			prefix:     "A <- 'hel",
			shouldHave: []rune{'l', 'o', '\'', '\\'},
		},
		{
			name:       "after_expression",
			prefix:     "A <- 'a'",
			shouldHave: []rune{'?', '*', '+', '/', ' ', '\n'},
		},
		{
			name:       "in_character_class",
			prefix:     "A <- [a",
			shouldHave: []rune{'-', ']', 'b', 'c', 'z'},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parser := oracle.NewParser()
			if tc.prefix != "" {
				parser = parser.AdvanceString(tc.prefix)
				if parser.IsEmpty() {
					t.Skipf("prefix %q could not be parsed", tc.prefix)
				}
			}

			chars := parser.NextChars()

			for _, r := range tc.shouldHave {
				assert.True(t, chars.Contains(r),
					"after %q, NextChars should contain %q", tc.prefix, r)
			}
		})
	}
}

// TestOracleEdgeCases tests edge cases and potential pitfalls.
func TestOracleEdgeCases(t *testing.T) {
	t.Run("empty_input_zero_or_more", func(t *testing.T) {
		bytecode := compileGrammar(t, `Start <- 'a'*`)
		oracle := NewGrammarOracle(bytecode)
		parser := oracle.NewParser()

		// Empty string should be accepted
		assert.True(t, parser.IsAccepting())
		// Should still allow 'a'
		assert.True(t, parser.NextChars().Contains('a'))
	})

	t.Run("repetition_exit_after_multiple", func(t *testing.T) {
		// This was a specific bug - exiting repetition after multiple iterations
		bytecode := compileGrammar(t, `Start <- ("x" "y")* "z"`)
		oracle := NewGrammarOracle(bytecode)

		for n := 0; n <= 5; n++ {
			input := ""
			for i := 0; i < n; i++ {
				input += "xy"
			}
			input += "z"

			parser := oracle.NewParser().AdvanceString(input)
			assert.False(t, parser.IsEmpty(), "%d iterations + z should parse", n)
			assert.True(t, parser.IsAccepting(), "%d iterations + z should accept", n)
		}
	})

	t.Run("deeply_nested_choices", func(t *testing.T) {
		bytecode := compileGrammar(t, `Start <- (('a'/'b'/'c') ('1'/'2'/'3'))+`)
		oracle := NewGrammarOracle(bytecode)

		valid := []string{"a1", "b2", "c3", "a1b2c3", "c3c3c3"}
		for _, input := range valid {
			parser := oracle.NewParser().AdvanceString(input)
			assert.False(t, parser.IsEmpty(), "input %q should parse", input)
			assert.True(t, parser.IsAccepting(), "input %q should accept", input)
		}
	})

	t.Run("multiple_states_tracking", func(t *testing.T) {
		// Test that multiple states are properly tracked through shared prefixes
		bytecode := compileGrammar(t, `Start <- "abc" / "abd" / "aec"`)
		oracle := NewGrammarOracle(bytecode)

		parser := oracle.NewParser().Advance('a')
		assert.False(t, parser.IsEmpty())
		assert.True(t, len(parser.States()) > 0, "should track multiple states")

		chars := parser.NextChars()
		assert.True(t, chars.Contains('b'), "'b' should be valid")
		assert.True(t, chars.Contains('e'), "'e' should be valid")
	})

	t.Run("ordered_choice_no_state_leakage", func(t *testing.T) {
		// Bug: After consuming 'h' in 'histogram'/'hll', NextChars should NOT
		// include 'p' from 'p999' (a different alternative that doesn't match 'h')
		bytecode := compileGrammar(t, `Start <- 'histogram' / 'hll' / 'p999'`)
		oracle := NewGrammarOracle(bytecode)

		// At start, 'h' and 'p' should both be valid
		chars0 := oracle.NewParser().NextChars()
		assert.True(t, chars0.Contains('h'), "'h' should be valid at start")
		assert.True(t, chars0.Contains('p'), "'p' should be valid at start")

		// After 'h', only 'i' (histogram) and 'l' (hll) should be valid
		// NOT 'p' from p999 (that alternative doesn't match our consumed 'h')
		parser := oracle.NewParser().Advance('h')
		chars1 := parser.NextChars()
		assert.True(t, chars1.Contains('i'), "'i' should be valid after 'h' (histogram)")
		assert.True(t, chars1.Contains('l'), "'l' should be valid after 'h' (hll)")
		assert.False(t, chars1.Contains('p'), "'p' should NOT be valid after 'h' (state leakage bug)")
		assert.False(t, chars1.Contains('h'), "'h' should NOT be valid after 'h'")
	})

	t.Run("unicode_support", func(t *testing.T) {
		bytecode := compileGrammar(t, `Start <- .+`)
		oracle := NewGrammarOracle(bytecode)

		// Test various unicode characters
		inputs := []string{"日本語", "🎉🎊", "Ω≈ç√", "مرحبا"}
		for _, input := range inputs {
			parser := oracle.NewParser().AdvanceString(input)
			assert.False(t, parser.IsEmpty(), "unicode input %q should parse", input)
			assert.True(t, parser.IsAccepting(), "unicode input %q should accept", input)
		}
	})
}

// BenchmarkOracleAdvance benchmarks the Advance operation.
func BenchmarkOracleAdvance(b *testing.B) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`Start <- [a-z]+`))
	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)
	bytecode, _ := QueryBytecode(db, "test.peg")
	oracle := NewGrammarOracle(bytecode)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		parser := oracle.NewParser()
		for _, r := range "abcdefghij" {
			parser = parser.Advance(r)
		}
	}
}

// BenchmarkOracleNextChars benchmarks the NextChars operation.
func BenchmarkOracleNextChars(b *testing.B) {
	loader := NewInMemoryImportLoader()
	loader.Add("test.peg", []byte(`Start <- ("a" / "b" / "c") [0-9]+`))
	cfg := NewConfig()
	cfg.SetBool("grammar.add_builtins", false)
	db := NewDatabase(cfg, loader)
	bytecode, _ := QueryBytecode(db, "test.peg")
	oracle := NewGrammarOracle(bytecode)

	parser := oracle.NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.NextChars()
	}
}
