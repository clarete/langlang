package extract

import "fmt"

// Validate cross-checks struct fields against grammar rules. It reclassifies
// fields using grammar knowledge and returns errors for mismatches.
// The returned []StructInfo has NameID populated on each field.
func Validate(structs []StructInfo, rules map[string]RuleInfo) ([]StructInfo, []error) {
	var errs []error
	result := make([]StructInfo, len(structs))
	copy(result, structs)

	for i := range result {
		for j := range result[i].Fields {
			f := &result[i].Fields[j]

			rule, ok := rules[f.LLTag]
			if !ok {
				errs = append(errs, fmt.Errorf(
					"%s.%s: unknown rule %q", result[i].Name, f.GoName, f.LLTag))
				continue
			}

			f.NameID = rule.NameID

			if err := validateFieldAgainstRule(f, &rule, &result[i]); err != nil {
				errs = append(errs, fmt.Errorf(
					"%s.%s: %w", result[i].Name, f.GoName, err))
			}
		}
	}

	return result, errs
}

func validateFieldAgainstRule(f *FieldInfo, rule *RuleInfo, parent *StructInfo) error {
	switch f.Kind {
	case FieldText:
		switch rule.Kind {
		case RuleLeaf, RuleAlias, RuleSequence:
			// Sequence is allowed for text fields — the entire rule text
			// is extracted as a string (e.g., String <- '"' #(Char* '"'))
			return nil
		default:
			return fmt.Errorf(
				"string field mapped to %s rule %q (expected leaf, alias, or sequence)",
				ruleKindString(rule.Kind), f.LLTag)
		}

	case FieldNamedRule:
		switch rule.Kind {
		case RuleSequence, RuleChoice, RuleAlias, RuleRepeat:
			return nil
		case RuleLeaf:
			return fmt.Errorf(
				"struct field mapped to leaf rule %q", f.LLTag)
		}

	case FieldOptional:
		return nil

	case FieldSlice:
		return nil

	case FieldChoice:
		if rule.Kind != RuleChoice {
			return fmt.Errorf(
				"choice struct mapped to non-choice rule %q", f.LLTag)
		}
	}

	return nil
}

func ruleKindString(k RuleKind) string {
	switch k {
	case RuleLeaf:
		return "leaf"
	case RuleSequence:
		return "sequence"
	case RuleChoice:
		return "choice"
	case RuleRepeat:
		return "repeat"
	case RuleOptional:
		return "optional"
	case RuleAlias:
		return "alias"
	default:
		return "unknown"
	}
}
