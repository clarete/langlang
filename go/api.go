package langlang

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
