package houserules

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
)

var Breathe = &analysis.Analyzer{
	Name: "breathe",
	Doc:  "reports missing blank lines around control flow, function literals, returns, branches and var blocks",
	Run:  runBreathe,
}

func runBreathe(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}

		ast.Inspect(file, func(current ast.Node) bool {
			switch node := current.(type) {
			case *ast.BlockStmt:
				checkSpacing(pass, node.List)
			case *ast.CaseClause:
				checkSpacing(pass, node.Body)
			case *ast.CommClause:
				checkSpacing(pass, node.Body)
			case *ast.FuncLit:
				checkFunctionLiteralBody(pass, node)
			}

			return true
		})
	}

	return nil, nil
}

func checkSpacing(pass *analysis.Pass, statements []ast.Stmt) {
	for index := 1; index < len(statements); index++ {
		previous := statements[index-1]
		next := statements[index]

		previousEnd := pass.Fset.Position(previous.End()).Line
		nextStart := pass.Fset.Position(next.Pos()).Line

		if isControlFlow(next) {
			checkMultipleIntroductionBoundary(pass, statements, index)
		}

		// A comment line between the statements counts as separation.
		if nextStart > previousEnd+1 {
			continue
		}

		if containsFunctionLiteral(previous) {
			pass.Reportf(next.Pos(), "missing blank line after function literal")

			continue
		}

		if containsFunctionLiteral(next) {
			pass.Reportf(next.Pos(), "missing blank line before function literal")

			continue
		}

		if isControlFlow(previous) && (!isIf(previous) || previousEnd > pass.Fset.Position(previous.Pos()).Line) {
			pass.Reportf(next.Pos(), "missing blank line after control-flow block")

			continue
		}

		if isGroupedVarBlock(previous) {
			pass.Reportf(next.Pos(), "missing blank line after var block")

			continue
		}

		if isGroupedVarBlock(next) {
			pass.Reportf(next.Pos(), "missing blank line before var block")

			continue
		}

		branchToken, isSeparatedBranch := separatedBranchToken(next)
		if isSeparatedBranch {
			pass.Reportf(next.Pos(), "missing blank line before %s", branchToken)

			continue
		}

		returnStatement, isReturn := unlabel(next).(*ast.ReturnStmt)
		if isReturn && !introducesReturn(previous, returnStatement) {
			pass.Reportf(next.Pos(), "missing blank line before return: only a statement feeding its result may sit directly above")

			continue
		}

		if isControlFlow(next) {
			checkControlFlowIntroduction(pass, statements, index)
		}
	}
}

func checkFunctionLiteralBody(pass *analysis.Pass, function *ast.FuncLit) {
	if len(function.Body.List) == 0 {
		return
	}

	openingLine := pass.Fset.Position(function.Body.Lbrace).Line
	closingLine := pass.Fset.Position(function.Body.Rbrace).Line
	firstLine := pass.Fset.Position(function.Body.List[0].Pos()).Line
	lastLine := pass.Fset.Position(function.Body.List[len(function.Body.List)-1].End()).Line

	if openingLine == firstLine || lastLine == closingLine {
		pass.Reportf(function.Pos(), "function literal body must start and end on separate lines")
	}
}

func checkControlFlowIntroduction(pass *analysis.Pass, statements []ast.Stmt, index int) {
	previous := statements[index-1]
	next := statements[index]

	if !introduces(previous, next) {
		pass.Reportf(next.Pos(), "missing blank line before control-flow block: only a statement feeding its condition may sit directly above")

		return
	}

	introductionStart := introductionGroupStart(pass, statements, index)
	if index-introductionStart > 1 {
		pass.Reportf(next.Pos(), "missing blank line before control-flow block: multiple statements feed its condition")

		return
	}

	if index < 2 {
		return
	}

	beforePrevious := statements[index-2]

	beforePreviousEnd := pass.Fset.Position(beforePrevious.End()).Line
	previousStart := pass.Fset.Position(previous.Pos()).Line

	if previousStart > beforePreviousEnd+1 {
		return
	}

	if introduces(beforePrevious, next) {
		pass.Reportf(next.Pos(), "missing blank line before control-flow block: multiple statements feed its condition")

		return
	}

	if isGroupedVarBlock(beforePrevious) {
		return
	}

	beforePreviousStart := pass.Fset.Position(beforePrevious.Pos()).Line
	if isControlFlow(beforePrevious) && beforePreviousEnd > beforePreviousStart {
		return
	}

	pass.Reportf(previous.Pos(), "missing blank line before statement feeding control-flow block")
}

func checkMultipleIntroductionBoundary(pass *analysis.Pass, statements []ast.Stmt, index int) {
	introductionStart := introductionGroupStart(pass, statements, index)
	if index-introductionStart < 2 || introductionStart == 0 {
		return
	}

	beforeIntroduction := statements[introductionStart-1]
	firstIntroduction := statements[introductionStart]

	beforeIntroductionEnd := pass.Fset.Position(beforeIntroduction.End()).Line
	firstIntroductionStart := pass.Fset.Position(firstIntroduction.Pos()).Line

	if firstIntroductionStart > beforeIntroductionEnd+1 {
		return
	}

	if isGroupedVarBlock(beforeIntroduction) {
		return
	}

	beforeIntroductionStart := pass.Fset.Position(beforeIntroduction.Pos()).Line
	if isControlFlow(beforeIntroduction) && beforeIntroductionEnd > beforeIntroductionStart {
		return
	}

	pass.Reportf(firstIntroduction.Pos(), "missing blank line before statements feeding control-flow block")
}

func introductionGroupStart(pass *analysis.Pass, statements []ast.Stmt, index int) int {
	next := statements[index]
	start := index - 1

	if !introduces(statements[start], next) {
		return index
	}

	for start > 0 {
		candidate := statements[start-1]
		if !introduces(candidate, next) {
			break
		}

		candidateEnd := pass.Fset.Position(candidate.End()).Line
		currentStart := pass.Fset.Position(statements[start].Pos()).Line

		if currentStart > candidateEnd+1 {
			break
		}

		start--
	}

	return start
}

// introduces reports whether previous is an assignment whose results
// appear in the header of the control-flow statement next.
func introduces(previous, next ast.Stmt) bool {
	if !isIf(next) {
		return false
	}

	assignment, ok := unlabel(previous).(*ast.AssignStmt)
	if !ok {
		return false
	}

	header := headerIdents(next)

	for _, target := range assignment.Lhs {
		identifier, ok := target.(*ast.Ident)
		if ok && header[identifier.Name] {
			return true
		}
	}

	return false
}

func introducesReturn(previous ast.Stmt, returnStatement *ast.ReturnStmt) bool {
	assignment, ok := unlabel(previous).(*ast.AssignStmt)
	if !ok {
		return false
	}

	resultNames := make(map[string]bool)

	for _, result := range returnStatement.Results {
		collectIdents(resultNames, result)
	}

	for _, target := range assignment.Lhs {
		identifier, ok := target.(*ast.Ident)
		if ok && resultNames[identifier.Name] {
			return true
		}
	}

	return false
}

func headerIdents(statement ast.Stmt) map[string]bool {
	names := make(map[string]bool)

	switch header := unlabel(statement).(type) {
	case *ast.IfStmt:
		collectIdents(names, header.Init)
		collectIdents(names, header.Cond)
	case *ast.ForStmt:
		collectIdents(names, header.Init)
		collectIdents(names, header.Cond)
		collectIdents(names, header.Post)
	case *ast.RangeStmt:
		collectIdents(names, header.X)
	case *ast.SwitchStmt:
		collectIdents(names, header.Init)
		collectIdents(names, header.Tag)
	case *ast.TypeSwitchStmt:
		collectIdents(names, header.Init)
		collectIdents(names, header.Assign)
	}

	return names
}

func collectIdents(names map[string]bool, node ast.Node) {
	if node == nil {
		return
	}

	ast.Inspect(node, func(current ast.Node) bool {
		identifier, ok := current.(*ast.Ident)
		if ok {
			names[identifier.Name] = true
		}

		return true
	})
}

func containsFunctionLiteral(node ast.Node) bool {
	found := false

	ast.Inspect(node, func(current ast.Node) bool {
		if found {
			return false
		}

		_, found = current.(*ast.FuncLit)

		return !found
	})

	return found
}

func isControlFlow(statement ast.Stmt) bool {
	switch unlabel(statement).(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return true
	}

	return false
}

func isIf(statement ast.Stmt) bool {
	_, ok := unlabel(statement).(*ast.IfStmt)

	return ok
}

func isGroupedVarBlock(statement ast.Stmt) bool {
	declaration, ok := varDeclaration(statement)

	return ok && declaration.Tok == token.VAR && declaration.Lparen.IsValid()
}

func separatedBranchToken(statement ast.Stmt) (token.Token, bool) {
	branch, ok := unlabel(statement).(*ast.BranchStmt)
	if !ok {
		return token.ILLEGAL, false
	}

	return branch.Tok, branch.Tok == token.BREAK || branch.Tok == token.CONTINUE
}

func unlabel(statement ast.Stmt) ast.Stmt {
	labeled, ok := statement.(*ast.LabeledStmt)
	if ok {
		return labeled.Stmt
	}

	return statement
}
