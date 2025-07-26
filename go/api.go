package langlang

// GrammarFromString takes a `grammar` string definition alongside
// with an instance of a configuration object and returns the Grammar
// AST transformed according to the configured values.
func GrammarFromString(grammar string, cfg *Config) (AstNode, error) {
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

	// If captures are enabled [grammar.captures=true], but
	// capturing *spaces* is disabled,
	// [grammar.capture_spaces=false], we apply `AddCaptures`
	// *before* `InjectWhitespaces`.
	if cfg.GetBool("grammar.captures") && !cfg.GetBool("grammar.capture_spaces") {
		expr, err = AddCaptures(expr)
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

	// if captures and capturing spaces are all enabled
	// [grammar.captures=true], [grammar.capture_spaces=true], we
	// then call `AddCaptures` *after* `InjectWhitespaces`.
	if cfg.GetBool("grammar.capture_spaces") && cfg.GetBool("grammar.capture_spaces") {
		expr, err = AddCaptures(expr)
		if err != nil {
			return nil, err
		}
	}
	return expr, nil
}
