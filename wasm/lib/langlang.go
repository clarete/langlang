package main

import (
	langlang "github.com/clarete/langlang/go"
)

func CompileAndMatch(grammar string, input string) (langlang.Value, error) {
	cfg := langlang.NewConfig()

	cfg.SetBool("grammar.add_builtins", true)
	cfg.SetBool("grammar.add_charsets", true)
	cfg.SetBool("grammar.captures", true)
	cfg.SetBool("grammar.capture_spaces", false)
	cfg.SetBool("grammar.disable_spaces", false)
	cfg.SetInt("compiler.optimize", 1)

	matcher, _ := langlang.MatcherFromBytes([]byte(grammar), cfg)
	val, _, err := matcher.Match([]byte(input))

	return val, err

}
