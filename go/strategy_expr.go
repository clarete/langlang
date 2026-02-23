package langlang

import (
	"fmt"
	"strings"
	"unicode"
)

// ParseStrategyExpr parses a strategy expression string into a Strategy.
// Supports: "rule_name" (StratLift of that rule set), "innermost(rule_name)".
// The rewrite file rf is used to resolve rule set names.
func ParseStrategyExpr(s string, rf *RewriteFile) (Strategy, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty strategy expression")
	}
	if strings.HasPrefix(s, "innermost(") {
		s = s[len("innermost("):]
		s = strings.TrimSpace(s)
		if len(s) == 0 || s[len(s)-1] != ')' {
			return nil, fmt.Errorf("innermost(...) must end with ')'")
		}
		inner := strings.TrimSpace(s[:len(s)-1])
		rs := RuleSetByName(rf, inner)
		if rs == nil {
			return nil, fmt.Errorf("unknown rule set %q", inner)
		}
		return StratInnermost{Inner: StratLift{RuleSet: rs}}, nil
	}
	// Plain rule set name
	rs := RuleSetByName(rf, s)
	if rs == nil {
		return nil, fmt.Errorf("unknown rule set %q", s)
	}
	return StratLift{RuleSet: rs}, nil
}

// ParseStrategyExprToEntry parses either a rule set name or a strategy expression.
// Returns the entry kind and the compiled bytecode. If the string is a rule set
// name, compiles with CompileRewriteFile(rf, name). If it's "innermost(name)",
// compiles with CompileRewriteFileWithStrategy.
func ParseStrategyExprToEntry(s string, rf *RewriteFile) (bytecode *Bytecode, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty entry expression")
	}
	if strings.HasPrefix(s, "innermost(") {
		strat, err := ParseStrategyExpr(s, rf)
		if err != nil {
			return nil, err
		}
		return CompileRewriteFileWithStrategy(rf, strat)
	}
	// Rule set name
	if !isIdent(s) {
		return nil, fmt.Errorf("invalid rule set name %q", s)
	}
	bytecode, _, errCompile := CompileRewriteFile(rf, s)
	if errCompile != nil {
		return nil, errCompile
	}
	return bytecode, nil
}

func isIdent(s string) bool {
	for i, r := range s {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return len(s) > 0
}
