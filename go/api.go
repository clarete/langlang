package langlang

// Matcher provides a dynamic PEG parsing interface created at runtime
// from a grammar definition.  Unlike ahead-of-time code generation,
// Matchers are built on-the-fly by compiling PEG grammars into
// bytecode and creating a virtual machine for running it.
type Matcher interface {
	// Match attempts to parse the input data against the
	// grammar's start rule, returning a syntax tree (Value), the
	// number of bytes consumed, and any error encountered in the
	// process.
	Match([]byte) (Value, int, error)
}

// MatcherFromBytes creates a Matcher from a PEG grammar definition
// provided as a byte slice.  The function parses the grammar,
// compiles it into assembly instructions, encodes it into bytecode,
// and returns a VirtualMachine ready to match input against the
// grammar rules.
//
// The cfg parameter controls various grammar transformations such as
// adding built-in rules, character set optimizations, whitespace
// handling, and capture generation.
//
// Returns an error if the grammar cannot be parsed, compiled, or if
// any transformation fails.
func MatcherFromBytes(input []byte, cfg *Config) (Matcher, error) {
	ast, err := GrammarFromBytes(input, cfg)
	if err != nil {
		return nil, err
	}
	asm, err := Compile(ast, cfg)
	if err != nil {
		return nil, err
	}
	code := Encode(asm)
	return NewVirtualMachine(code, nil, nil, true), nil
}

// MatcherFromFile creates a Matcher from a PEG grammar file at the
// specified path. The function loads the grammar file (resolving any
// imports), parses it, compiles it into assembly instructions,
// encodes it into bytecode, and returns a VirtualMachine ready to
// match input against the grammar rules.
//
// The cfg parameter controls various grammar transformations such as
// adding built-in rules, character set optimizations, whitespace
// handling, and capture generation.
//
// Returns an error if the file cannot be read, the grammar cannot be
// parsed or compiled, or if any transformation fails.
func MatcherFromFile(path string, cfg *Config) (Matcher, error) {
	ast, err := GrammarFromFile(path, cfg)
	if err != nil {
		return nil, err
	}
	asm, err := Compile(ast, cfg)
	if err != nil {
		return nil, err
	}
	code := Encode(asm)
	return NewVirtualMachine(code, nil, nil, true), nil
}

// GrammarFromBytes takes a `grammar` string definition alongside with
// an instance of a configuration object and returns the Grammar AST
// transformed according to the configured values.
func GrammarFromBytes(grammar []byte, cfg *Config) (AstNode, error) {
	ast, err := NewGrammarParser(grammar).Parse()
	if err != nil {
		return nil, err
	}
	return GrammarTransformations(ast, cfg)
}

// GrammarFromFile takes a grammar `path` string alongside with an
// instance of a configuration object and returns the Grammar AST
// transformed according to the configured values.
func GrammarFromFile(path string, cfg *Config) (AstNode, error) {
	importLoader := NewRelativeImportLoader()
	importResolver := NewImportResolver(importLoader)
	ast, err := importResolver.Resolve(path)
	if err != nil {
		return nil, err
	}
	return GrammarTransformations(ast, cfg)
}

// GrammarTransformations applies various transformations to the
// grammar ast node `expr` based on the values set in the
// configuration object `cfg`.
func GrammarTransformations(expr AstNode, cfg *Config) (AstNode, error) {
	var err error

	if cfg.GetBool("grammar.add_builtins") {
		expr, err = AddBuiltins(expr)
		if err != nil {
			return nil, err
		}
	}

	if cfg.GetBool("grammar.add_charsets") {
		expr, err = AddCharsets(expr)
		if err != nil {
			return nil, err
		}
	}

	if cfg.GetBool("grammar.handle_spaces") {
		expr, err = InjectWhitespaces(expr)
		if err != nil {
			return nil, err
		}
	}

	if cfg.GetBool("grammar.captures") {
		expr, err = AddCaptures(expr, cfg)
		if err != nil {
			return nil, err
		}
	}
	return expr, nil
}
