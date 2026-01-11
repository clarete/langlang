package langlang

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
