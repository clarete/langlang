package extract

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// Analyze parses a Go source file and returns StructInfo for each struct
// that has at least one field with an ll:"..." tag. Fields without ll tags
// are excluded. Classification is Go-type-only at this stage; grammar-aware
// reclassification happens in Validate.
func Analyze(filename string) ([]StructInfo, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, parser.AllErrors)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}

	// Collect all struct names that have ll: tags (for Inner resolution)
	taggedStructs := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		for _, field := range st.Fields.List {
			if field.Tag != nil {
				if _, ok := extractLLTag(field.Tag.Value); ok {
					taggedStructs[ts.Name.Name] = true
					break
				}
			}
		}
		return true
	})

	var structs []StructInfo
	ast.Inspect(file, func(n ast.Node) bool {
		ts, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			return true
		}
		if !taggedStructs[ts.Name.Name] {
			return true
		}

		si := StructInfo{Name: ts.Name.Name}
		for _, field := range st.Fields.List {
			if field.Tag == nil {
				continue
			}
			tag, ok := extractLLTag(field.Tag.Value)
			if !ok {
				continue
			}
			if len(field.Names) == 0 {
				continue
			}

			fi := FieldInfo{
				GoName: field.Names[0].Name,
				LLTag:  tag,
				GoType: typeString(field.Type),
			}
			fi.Kind, fi.ElemType = classifyGoType(field.Type, taggedStructs)
			si.Fields = append(si.Fields, fi)
		}

		if len(si.Fields) > 0 {
			structs = append(structs, si)
		}
		return true
	})

	return structs, nil
}

// classifyGoType classifies a field by its Go type expression.
func classifyGoType(expr ast.Expr, tagged map[string]bool) (FieldKind, string) {
	switch t := expr.(type) {
	case *ast.Ident:
		if t.Name == "string" {
			return FieldText, ""
		}
		if tagged[t.Name] {
			return FieldNamedRule, ""
		}
		return FieldText, ""

	case *ast.StarExpr:
		return FieldOptional, ""

	case *ast.ArrayType:
		elemType := typeString(t.Elt)
		return FieldSlice, elemType

	default:
		return FieldText, ""
	}
}

// typeString returns a string representation of a Go type expression.
func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.ArrayType:
		return "[]" + typeString(t.Elt)
	case *ast.SelectorExpr:
		return typeString(t.X) + "." + t.Sel.Name
	default:
		return fmt.Sprintf("%T", expr)
	}
}

// isChoiceStruct checks if a struct has all-pointer ll:-tagged fields,
// indicating it represents an ordered choice.
func isChoiceStruct(si *StructInfo) bool {
	if len(si.Fields) == 0 {
		return false
	}
	for _, f := range si.Fields {
		if !strings.HasPrefix(f.GoType, "*") {
			return false
		}
	}
	return true
}
