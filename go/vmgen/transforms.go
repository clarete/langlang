package main

import (
	"go/ast"
	"go/token"
)

// Common transforms applied to all flavors

// removeInitializationsFromFunction removes the `reset()` and `bindInput()`
// calls from the function body. They should be called once by the caller,
// not on every step/advance.
func removeInitializationsFromFunction(funcDecl *ast.FuncDecl) {
	var newStmts []ast.Stmt
	for _, stmt := range funcDecl.Body.List {
		if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
			if callExpr, ok := exprStmt.X.(*ast.CallExpr); ok {
				if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
					if selectorExpr.Sel.Name == "reset" || selectorExpr.Sel.Name == "bindInput" {
						continue
					}
				}
			}
		}
		newStmts = append(newStmts, stmt)
	}
	funcDecl.Body.List = newStmts
}

// removeVarDeclarationsFromFunction removes the `pc` and `cursor` variable
// declarations from the function body, as they become parameters.
func removeVarDeclarationsFromFunction(funcDecl *ast.FuncDecl) {
	var newStmts []ast.Stmt
	for _, stmt := range funcDecl.Body.List {
		declStmt, ok := stmt.(*ast.DeclStmt)
		if !ok {
			newStmts = append(newStmts, stmt)
			continue
		}
		varDecl, ok := declStmt.Decl.(*ast.GenDecl)
		if !ok || varDecl.Tok != token.VAR {
			newStmts = append(newStmts, stmt)
			continue
		}
		var newSpecs []ast.Spec
		for _, spec := range varDecl.Specs {
			varSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				newSpecs = append(newSpecs, spec)
				continue
			}
			var (
				filteredNames  []*ast.Ident
				filteredValues []ast.Expr
			)
			for i, name := range varSpec.Names {
				if name.Name != "pc" && name.Name != "cursor" {
					filteredNames = append(filteredNames, name)
					filteredValues = append(filteredValues, varSpec.Values[i])
				}
			}
			if len(filteredNames) == 0 {
				continue
			}
			varSpec.Names = filteredNames
			varSpec.Values = filteredValues
			newSpecs = append(newSpecs, varSpec)
		}
		varDecl.Specs = newSpecs
		newStmts = append(newStmts, stmt)
	}
	funcDecl.Body.List = newStmts
}

// removeRuleAddressCheckFromFunction removes the `if ruleAddress > 0` check
// from the function body.
func removeRuleAddressCheckFromFunction(funcDecl *ast.FuncDecl) {
	var newStmts []ast.Stmt
	for _, stmt := range funcDecl.Body.List {
		if _, ok := stmt.(*ast.IfStmt); !ok {
			newStmts = append(newStmts, stmt)
		}
	}
	funcDecl.Body.List = newStmts
}

// Flavor-driven transforms

// adjustFunctionSignature updates the function name, parameters, and return types
// according to the flavor configuration.
func adjustFunctionSignature(funcDecl *ast.FuncDecl, flavor VMFlavor) {
	funcDecl.Name.Name = flavor.FuncName
	funcDecl.Type.Params.List = ToASTFields(flavor.Params)
	funcDecl.Type.Results.List = ToASTFields(flavor.Returns)
}

// addHooksToFunction injects the pre-opcode and post-opcode hooks into the
// main VM loop, if the flavor provides them.
func addHooksToFunction(funcDecl *ast.FuncDecl, flavor VMFlavor) {
	if flavor.PreOpcodeHook == nil && flavor.PostOpcodeHook == nil {
		return
	}

	vmForLoop := getVmForLoop(funcDecl)
	newBodyList := vmForLoop.Body.List

	if flavor.PreOpcodeHook != nil {
		newBodyList = append([]ast.Stmt{flavor.PreOpcodeHook()}, newBodyList...)
	}
	if flavor.PostOpcodeHook != nil {
		newBodyList = append(newBodyList, flavor.PostOpcodeHook())
	}

	vmForLoop.Body.List = newBodyList
}

// adjustReturnStatements updates the return statements throughout the function
// according to the flavor's return builders.
func adjustReturnStatements(funcDecl *ast.FuncDecl, flavor VMFlavor) {
	// opHalt return
	if flavor.MkSuccessReturn != nil {
		adjustOpcodeReturn(funcDecl, "opHalt", flavor.MkSuccessReturn)
	}

	// opThrow return (uses MkThrowReturn if provided, otherwise MkFailReturn)
	throwReturn := flavor.MkThrowReturn
	if throwReturn == nil {
		throwReturn = flavor.MkFailReturn
	}
	if throwReturn != nil {
		adjustOpcodeReturn(funcDecl, "opThrow", throwReturn)
	}

	// Final return after fail label
	if flavor.MkFailReturn != nil {
		adjustFinalReturn(funcDecl, flavor.MkFailReturn)
	}
}

func adjustOpcodeReturn(funcDecl *ast.FuncDecl, opName string, mkReturn func() *ast.ReturnStmt) {
	caseClause := getCaseClause(getMainSwitchStatement(funcDecl), opName)
	if caseClause == nil {
		return
	}
	// Remove the last statement (original return) and add the new one
	bodyWithoutReturn := caseClause.Body[:len(caseClause.Body)-1]
	caseClause.Body = append(bodyWithoutReturn, mkReturn())
}

func adjustFinalReturn(funcDecl *ast.FuncDecl, mkReturn func() *ast.ReturnStmt) {
	body := funcDecl.Body.List
	bodyWithoutReturn := body[:len(body)-1]
	funcDecl.Body.List = append(bodyWithoutReturn, mkReturn())
}

// transformEOFToSuccess transforms `if cursor >= ilen { goto fail }`
// patterns into success returns.  Used for prefix/oracle matching
// where running out of input is success (need more input) rather than
// failure.
func transformEOFToSuccess(funcDecl *ast.FuncDecl, mkEOFReturn func() *ast.ReturnStmt) {
	if mkEOFReturn == nil {
		return
	}

	mainSwitch := getMainSwitchStatement(funcDecl)

	// Walk through each case clause and transform EOF checks
	for _, stmt := range mainSwitch.Body.List {
		caseClause, ok := stmt.(*ast.CaseClause)
		if !ok {
			continue
		}
		transformEOFInStatements(caseClause.Body, mkEOFReturn)
	}
}

func transformEOFInStatements(stmts []ast.Stmt, mkEOFReturn func() *ast.ReturnStmt) {
	for i, stmt := range stmts {
		ifStmt, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}

		// Check if this is `if cursor >= ilen { goto fail }`
		if isEOFCheck(ifStmt) {
			// Replace `goto fail` with success return
			ifStmt.Body.List = []ast.Stmt{mkEOFReturn()}
			stmts[i] = ifStmt
		}

		// Recurse into nested blocks
		if ifStmt.Body != nil {
			transformEOFInStatements(ifStmt.Body.List, mkEOFReturn)
		}
		if ifStmt.Else != nil {
			if elseBlock, ok := ifStmt.Else.(*ast.BlockStmt); ok {
				transformEOFInStatements(elseBlock.List, mkEOFReturn)
			}
		}
	}
}

// transformSpanForOracle modifies the opSpan case to track whether any
// characters matched and return success (staying at span PC) if EOF is
// hit during matching. This allows NextChars to include the span's charset.
//
// Before:
//
//	case opSpan:
//	    sid := decodeU16(code, pc+1)
//	    set := sets[sid]
//	    for cursor < ilen {
//	        c := data[cursor]
//	        if set.hasByte(c) { cursor++; continue }
//	        break
//	    }
//	    pc += opSetSizeInBytes
//
// After:
//
//	case opSpan:
//	    sid := decodeU16(code, pc+1)
//	    set := sets[sid]
//	    matchedAny := false
//	    for cursor < ilen {
//	        c := data[cursor]
//	        if set.hasByte(c) { cursor++; matchedAny = true; continue }
//	        break
//	    }
//	    if cursor >= ilen && matchedAny {
//	        return pc, cursor, stack, true
//	    }
//	    pc += opSetSizeInBytes
func transformSpanForOracle(funcDecl *ast.FuncDecl, mkEOFReturn func() *ast.ReturnStmt) {
	if mkEOFReturn == nil {
		return
	}

	mainSwitch := getMainSwitchStatement(funcDecl)
	spanCase := getCaseClause(mainSwitch, "opSpan")
	if spanCase == nil {
		return
	}

	// Find the for loop in the span case
	var forStmt *ast.ForStmt
	var forIdx int
	for i, stmt := range spanCase.Body {
		if f, ok := stmt.(*ast.ForStmt); ok {
			forStmt = f
			forIdx = i
			break
		}
	}
	if forStmt == nil {
		return
	}

	// 1. Insert `matchedAny := false` before the for loop
	matchedAnyDecl := &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("matchedAny")},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{ast.NewIdent("false")},
	}

	// 2. Add `matchedAny = true` after `cursor++` inside the if block
	// Find the if statement inside the for loop body
	for _, stmt := range forStmt.Body.List {
		ifStmt, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}
		// Look for the if block that has cursor++ and continue
		for i, bodyStmt := range ifStmt.Body.List {
			if incStmt, ok := bodyStmt.(*ast.IncDecStmt); ok {
				if ident, ok := incStmt.X.(*ast.Ident); ok && ident.Name == "cursor" {
					// Insert matchedAny = true after cursor++
					setMatchedAny := &ast.AssignStmt{
						Lhs: []ast.Expr{ast.NewIdent("matchedAny")},
						Tok: token.ASSIGN,
						Rhs: []ast.Expr{ast.NewIdent("true")},
					}
					newBody := make([]ast.Stmt, 0, len(ifStmt.Body.List)+1)
					newBody = append(newBody, ifStmt.Body.List[:i+1]...)
					newBody = append(newBody, setMatchedAny)
					newBody = append(newBody, ifStmt.Body.List[i+1:]...)
					ifStmt.Body.List = newBody
					break
				}
			}
		}
	}

	// 3. Insert EOF check after the for loop:
	//    if cursor >= ilen && matchedAny { return success }
	eofCheck := &ast.IfStmt{
		Cond: &ast.BinaryExpr{
			X: &ast.BinaryExpr{
				X:  ast.NewIdent("cursor"),
				Op: token.GEQ,
				Y:  ast.NewIdent("ilen"),
			},
			Op: token.LAND,
			Y:  ast.NewIdent("matchedAny"),
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{mkEOFReturn()},
		},
	}

	// Rebuild the case body: [before for] + matchedAnyDecl + for + eofCheck + [after for]
	newBody := make([]ast.Stmt, 0, len(spanCase.Body)+2)
	newBody = append(newBody, spanCase.Body[:forIdx]...)
	newBody = append(newBody, matchedAnyDecl)
	newBody = append(newBody, forStmt)
	newBody = append(newBody, eofCheck)
	newBody = append(newBody, spanCase.Body[forIdx+1:]...)
	spanCase.Body = newBody
}

// isEOFCheck returns true if the if statement is `if cursor >= ilen { goto fail }`
func isEOFCheck(ifStmt *ast.IfStmt) bool {
	// Check condition: cursor >= ilen
	binExpr, ok := ifStmt.Cond.(*ast.BinaryExpr)
	if !ok || binExpr.Op != token.GEQ {
		return false
	}

	xIdent, ok := binExpr.X.(*ast.Ident)
	if !ok || xIdent.Name != "cursor" {
		return false
	}

	yIdent, ok := binExpr.Y.(*ast.Ident)
	if !ok || yIdent.Name != "ilen" {
		return false
	}

	// Check body: single `goto fail` statement
	if len(ifStmt.Body.List) != 1 {
		return false
	}

	branchStmt, ok := ifStmt.Body.List[0].(*ast.BranchStmt)
	if !ok || branchStmt.Tok != token.GOTO {
		return false
	}

	return branchStmt.Label.Name == "fail"
}

// AST navigation helpers

func getMatchRuleFunction(node *ast.File) *ast.FuncDecl {
	for _, decl := range node.Decls {
		if funcDecl, ok := decl.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == "MatchRule" {
				return funcDecl
			}
		}
	}
	panic("MatchRule function not found")
}

func getVmForLoop(funcDecl *ast.FuncDecl) *ast.ForStmt {
	labeledStmt := getLabeledStmt(funcDecl, "code")
	if forStmt, ok := labeledStmt.Stmt.(*ast.ForStmt); ok {
		return forStmt
	}
	panic("for statement not found in labeled statement")
}

func getLabeledStmt(funcDecl *ast.FuncDecl, name string) *ast.LabeledStmt {
	for _, stmt := range funcDecl.Body.List {
		if labeled, ok := stmt.(*ast.LabeledStmt); ok {
			if labeled.Label.Name == name {
				return labeled
			}
		}
	}
	panic("labeled statement not found for " + name)
}

func getMainSwitchStatement(funcDecl *ast.FuncDecl) *ast.SwitchStmt {
	vmForLoop := getVmForLoop(funcDecl)
	for _, stmt := range vmForLoop.Body.List {
		if switchStmt, ok := stmt.(*ast.SwitchStmt); ok {
			return switchStmt
		}
	}
	panic("main switch statement not found")
}

func getCaseClause(switchStmt *ast.SwitchStmt, name string) *ast.CaseClause {
	for _, stmt := range switchStmt.Body.List {
		if caseStmt, ok := stmt.(*ast.CaseClause); ok && len(caseStmt.List) > 0 {
			if ident, ok := caseStmt.List[0].(*ast.Ident); ok && ident.Name == name {
				return caseStmt
			}
		}
	}
	return nil // Not found (don't panic, some ops may not exist)
}

func mkPackage(funcDecl *ast.FuncDecl) *ast.File {
	return &ast.File{
		Name:  ast.NewIdent("langlang"),
		Decls: []ast.Decl{funcDecl},
	}
}
