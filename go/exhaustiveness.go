package langlang

import "fmt"

// TypeDecl describes an algebraic data type with named constructors.
// This corresponds to @type declarations in the surface syntax.
type TypeDecl struct {
	Name         string
	Constructors []*Constructor
}

// Constructor describes a single variant of an algebraic type.
type Constructor struct {
	Name   string
	Fields []*Field
}

// Field describes a named field of a constructor.
type Field struct {
	Name     string
	TypeName string // name of another TypeDecl, or "string" for terminals
}

// UncoveredPattern represents a constructor shape not matched by any pattern.
type UncoveredPattern struct {
	Constructor string
	Fields      []string // human-readable field descriptions
}

func (u UncoveredPattern) String() string {
	if len(u.Fields) == 0 {
		return u.Constructor + "()"
	}
	s := u.Constructor + "("
	for i, f := range u.Fields {
		if i > 0 {
			s += ", "
		}
		s += f
	}
	return s + ")"
}

// CheckExhaustiveness analyzes a set of rewrite rule patterns against
// a type declaration and returns uncovered constructors (if any).
// An empty result means the patterns are exhaustive.
//
// The algorithm follows Maranget (2007) "Warnings for pattern matching":
//   1. Start with the universe of all constructor shapes from the @type
//   2. For each pattern in the rule, subtract the shapes it covers
//   3. Report remaining shapes as uncovered
func CheckExhaustiveness(typ *TypeDecl, patterns []RewritePattern) []UncoveredPattern {
	covered := make(map[string]bool)

	for _, pat := range patterns {
		collectCoveredConstructors(pat, covered)
	}

	// A wildcard or variable at the top level covers all constructors
	if covered["*"] {
		return nil
	}

	var uncovered []UncoveredPattern
	for _, ctor := range typ.Constructors {
		if !covered[ctor.Name] {
			fields := make([]string, len(ctor.Fields))
			for i, f := range ctor.Fields {
				fields[i] = fmt.Sprintf("%s: %s", f.Name, f.TypeName)
			}
			uncovered = append(uncovered, UncoveredPattern{
				Constructor: ctor.Name,
				Fields:      fields,
			})
		}
	}

	return uncovered
}

// collectCoveredConstructors walks a pattern and records which
// constructor names it covers. A wildcard or variable at the top level
// covers ALL constructors (the caller should handle this).
func collectCoveredConstructors(pat RewritePattern, covered map[string]bool) {
	switch p := pat.(type) {
	case PatWild:
		// Wildcard covers everything -- mark a sentinel
		covered["*"] = true
	case PatVar:
		// Variable covers everything
		covered["*"] = true
	case PatNamed:
		covered[p.NodeName] = true
	case PatStr:
		// String literal covers a specific string terminal
		covered[p.Text] = true
	case PatSeq:
		// Sequence patterns cover the sequence shape; for exhaustiveness
		// of individual elements, recursive checking is needed.
		// At the top level, a sequence pattern doesn't cover named constructors.
	}
}

// FormatExhaustivenessWarning produces a human-readable warning message.
func FormatExhaustivenessWarning(ruleName string, typeName string, uncovered []UncoveredPattern) string {
	if len(uncovered) == 0 {
		return ""
	}
	s := fmt.Sprintf("warning: rewrite rule '%s' is not exhaustive over %s\n", ruleName, typeName)
	s += "  uncovered constructors:\n"
	for _, u := range uncovered {
		s += fmt.Sprintf("    - %s\n", u)
	}
	return s
}

// ExhaustivenessKey identifies an exhaustiveness check request.
type ExhaustivenessKey struct {
	RuleName string
	TypeName string
}

// ExhaustivenessResult holds the outcome of an exhaustiveness check.
type ExhaustivenessResult struct {
	Uncovered []UncoveredPattern
}

// ExhaustivenessQuery can be registered in the query pipeline.
var ExhaustivenessQuery = &Query[ExhaustivenessKey, ExhaustivenessResult]{
	Name: "Exhaustiveness",
	Compute: func(db *Database, key ExhaustivenessKey) (ExhaustivenessResult, error) {
		// This would resolve the type and rule from the database.
		// For now, return empty (no uncovered patterns) as a placeholder.
		// The actual implementation will be wired up when @type declarations
		// and <~ rules are parsed and stored in the database.
		return ExhaustivenessResult{}, nil
	},
}
