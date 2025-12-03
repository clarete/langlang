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
