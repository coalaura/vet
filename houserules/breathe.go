package houserules

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var Breathe = &analysis.Analyzer{
	Name: "breathe",
	Doc:  "reports spacing violations around declarations, error checks, control flow, function literals, returns, branches, var declarations and mutex operations",
	Run:  runBreathe,
}

func runBreathe(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}

		checkFunctionDeclarationSpacing(pass, file)

		ast.Inspect(file, func(current ast.Node) bool {
			switch node := current.(type) {
			case *ast.BlockStmt:
				checkSpacing(pass, file, node.List)
			case *ast.CaseClause:
				checkSpacing(pass, file, node.Body)
			case *ast.CommClause:
				checkSpacing(pass, file, node.Body)
			case *ast.FuncLit:
				checkFunctionLiteralBody(pass, node)
			case *ast.IfStmt:
				checkConditionFunctionLiterals(pass, node.Cond)
			case *ast.ForStmt:
				checkConditionFunctionLiterals(pass, node.Cond)
			}

			return true
		})
	}

	return nil, nil
}

func checkFunctionDeclarationSpacing(pass *analysis.Pass, file *ast.File) {
	for index := 1; index < len(file.Decls); index++ {
		previous, previousOK := file.Decls[index-1].(*ast.FuncDecl)
		current, currentOK := file.Decls[index].(*ast.FuncDecl)

		if !previousOK || !currentOK || hasSeparationLine(pass, file, previous.End(), current.Pos()) {
			continue
		}

		pass.Reportf(current.Pos(), "missing blank line between function declarations")
	}
}

func checkSpacing(pass *analysis.Pass, file *ast.File, statements []ast.Stmt) {
	for index := 1; index < len(statements); index++ {
		previous := statements[index-1]
		current := statements[index]

		if isSimpleErrorCheck(pass, previous, current) && hasBlankLine(pass, file, previous.End(), current.Pos()) {
			pass.Reportf(current.Pos(), "blank line before simple error check")
		}

		reason := statementSpacingReason(pass, statements, index)
		if reason != "" {
			pass.Reportf(current.Pos(), "missing blank line %s", reason)
		}

		checkIntroductionBoundary(pass, statements, index)
	}
}

func isSimpleErrorCheck(pass *analysis.Pass, previous, next ast.Stmt) bool {
	if !isErrorCheck(pass, previous, next) {
		return false
	}

	previousStart := pass.Fset.Position(previous.Pos()).Line
	previousEnd := pass.Fset.Position(previous.End()).Line

	return previousStart == previousEnd
}

func isErrorCheck(pass *analysis.Pass, previous, next ast.Stmt) bool {
	assignment, ok := unlabel(previous).(*ast.AssignStmt)
	if !ok {
		return false
	}

	ifStatement, ok := unlabel(next).(*ast.IfStmt)
	if !ok || ifStatement.Init != nil {
		return false
	}

	condition, ok := unparen(ifStatement.Cond).(*ast.BinaryExpr)
	if !ok || condition.Op != token.NEQ {
		return false
	}

	checked := condition.X
	if isNilExpression(checked) {
		checked = condition.Y
	} else if !isNilExpression(condition.Y) {
		return false
	}

	errorObject := types.Universe.Lookup("error")
	if errorObject == nil || !types.AssignableTo(pass.TypesInfo.TypeOf(checked), errorObject.Type()) {
		return false
	}

	for _, target := range assignment.Lhs {
		if sameExpression(pass, target, checked) {
			return true
		}
	}

	return false
}

func isNilExpression(expression ast.Expr) bool {
	identifier, ok := unparen(expression).(*ast.Ident)

	return ok && identifier.Name == "nil"
}

func hasBlankLine(pass *analysis.Pass, file *ast.File, previousEnd, currentStart token.Pos) bool {
	previousLine := pass.Fset.Position(previousEnd).Line
	currentLine := pass.Fset.Position(currentStart).Line

	for line := previousLine + 1; line < currentLine; line++ {
		coveredByComment := false

		for _, comment := range file.Comments {
			commentStart := pass.Fset.Position(comment.Pos()).Line
			commentEnd := pass.Fset.Position(comment.End()).Line

			if commentStart <= line && line <= commentEnd {
				coveredByComment = true

				break
			}
		}

		if !coveredByComment {
			return true
		}
	}

	return false
}

func hasSeparationLine(pass *analysis.Pass, file *ast.File, previousEnd, currentStart token.Pos) bool {
	previousLine := pass.Fset.Position(previousEnd).Line
	currentLine := pass.Fset.Position(currentStart).Line

	if currentLine > previousLine+1 {
		return true
	}

	for _, comment := range file.Comments {
		commentLine := pass.Fset.Position(comment.Pos()).Line
		if previousLine < commentLine && commentLine < currentLine {
			return true
		}
	}

	return false
}

// statementSpacingReason selects one diagnostic for the boundary. Feeder
// lookbacks consult the same decision so they cannot report it a second time.
func statementSpacingReason(pass *analysis.Pass, statements []ast.Stmt, index int) string {
	previous := statements[index-1]
	next := statements[index]

	previousEnd := pass.Fset.Position(previous.End()).Line
	nextStart := pass.Fset.Position(next.Pos()).Line

	// A comment line between the statements counts as separation.
	if nextStart > previousEnd+1 || canGroupVarStatements(pass, previous, next) {
		return ""
	}

	if containsFunctionLiteral(previous) {
		return "after function literal"
	}

	if containsFunctionLiteral(next) {
		return "before function literal"
	}

	if isControlFlow(previous) && (!isIf(previous) || previousEnd > pass.Fset.Position(previous.Pos()).Line) {
		return "after control-flow block"
	}

	if isVarDeclaration(previous) {
		return "after var declaration"
	}

	if isVarDeclaration(next) {
		return "before var declaration"
	}

	mutexBoundary := mutexSpacingBoundary(pass, statements, index)
	if mutexBoundary != "" {
		return mutexBoundary
	}

	branchToken, _ := separatedBranchToken(next)

	switch branchToken {
	case token.BREAK:
		return "before break"
	case token.CONTINUE:
		return "before continue"
	}

	returnStatement, isReturn := unlabel(next).(*ast.ReturnStmt)
	if isReturn && !introducesReturn(pass, previous, returnStatement) {
		return "before return: only a statement feeding its result may sit directly above"
	}

	if isControlFlow(next) {
		if !introduces(pass, previous, next) {
			return "before control-flow block: only a statement feeding its condition may sit directly above"
		}

		if isErrorCheck(pass, previous, next) && previousEnd > pass.Fset.Position(previous.Pos()).Line {
			return "before error check with multiline assignment"
		}

		introductionStart := introductionGroupStart(pass, statements, index)
		if index-introductionStart > 1 {
			return "before control-flow block: multiple statements feed its condition"
		}
	}

	return ""
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

func checkConditionFunctionLiterals(pass *analysis.Pass, condition ast.Expr) {
	if condition == nil {
		return
	}

	ast.Inspect(condition, func(current ast.Node) bool {
		function, ok := current.(*ast.FuncLit)
		if !ok {
			return true
		}

		pass.Reportf(function.Pos(), "function literal in condition must be assigned before use")

		return false
	})
}

func checkIntroductionBoundary(pass *analysis.Pass, statements []ast.Stmt, index int) {
	if !isIf(statements[index]) {
		return
	}

	introductionStart := introductionGroupStart(pass, statements, index)
	if introductionStart == index || introductionStart == 0 {
		return
	}

	introductionCount := index - introductionStart
	previousEnd := pass.Fset.Position(statements[index-1].End()).Line
	nextStart := pass.Fset.Position(statements[index].Pos()).Line

	// An isolated feeder separated from its if need not start a new section.
	if introductionCount == 1 && nextStart > previousEnd+1 {
		return
	}

	beforeIntroduction := statements[introductionStart-1]
	firstIntroduction := statements[introductionStart]

	beforeIntroductionEnd := pass.Fset.Position(beforeIntroduction.End()).Line
	firstIntroductionStart := pass.Fset.Position(firstIntroduction.Pos()).Line

	if firstIntroductionStart > beforeIntroductionEnd+1 {
		return
	}

	// Ordinary boundary rules take precedence over feeder-specific spacing.
	if statementSpacingReason(pass, statements, introductionStart) != "" {
		return
	}

	// Keep declarations groupable and a single lock attached to protected work.
	if canGroupVarStatements(pass, beforeIntroduction, firstIntroduction) || mutexOperation(pass, beforeIntroduction) != mutexNone {
		return
	}

	if introductionCount == 1 {
		pass.Reportf(firstIntroduction.Pos(), "missing blank line before statement feeding control-flow block")

		return
	}

	pass.Reportf(firstIntroduction.Pos(), "missing blank line before statements feeding control-flow block")
}

func introductionGroupStart(pass *analysis.Pass, statements []ast.Stmt, index int) int {
	next := statements[index]
	start := index - 1

	if !introduces(pass, statements[start], next) {
		return index
	}

	for start > 0 {
		candidate := statements[start-1]
		if !introduces(pass, candidate, next) {
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

// introduces reports whether previous updates a value that appears in the
// header of the control-flow statement next.
func introduces(pass *analysis.Pass, previous, next ast.Stmt) bool {
	if !isIf(next) {
		return false
	}

	switch statement := unlabel(previous).(type) {
	case *ast.AssignStmt:
		for _, target := range statement.Lhs {
			if headerUsesExpression(pass, next, target) {
				return true
			}
		}
	case *ast.IncDecStmt:
		return headerUsesExpression(pass, next, statement.X)
	}

	return false
}

func introducesReturn(pass *analysis.Pass, previous ast.Stmt, returnStatement *ast.ReturnStmt) bool {
	assignment, ok := unlabel(previous).(*ast.AssignStmt)
	if !ok {
		return false
	}

	for _, result := range returnStatement.Results {
		for _, target := range assignment.Lhs {
			if nodeUsesExpression(pass, result, target, false) {
				return true
			}
		}
	}

	return false
}

func headerUsesExpression(pass *analysis.Pass, statement ast.Stmt, target ast.Expr) bool {
	header, ok := unlabel(statement).(*ast.IfStmt)
	if !ok {
		return false
	}

	initializer, isAssignment := header.Init.(*ast.AssignStmt)
	if isAssignment {
		// Initializer targets are writes, not reads of the preceding value.
		for _, value := range initializer.Rhs {
			if nodeUsesExpression(pass, value, target, true) {
				return true
			}
		}

		for _, destination := range initializer.Lhs {
			if assignmentTargetUsesExpression(pass, destination, target) {
				return true
			}
		}

		for _, destination := range initializer.Lhs {
			// Replacing the target or any part of its address invalidates the
			// previous result, e.g. changing the index in values[index].
			if nodeUsesExpression(pass, target, destination, false) {
				return false
			}
		}
	} else if nodeUsesExpression(pass, header.Init, target, true) {
		return true
	}

	return nodeUsesExpression(pass, header.Cond, target, true)
}

func containsFunctionLiteral(statement ast.Stmt) bool {
	statement = unlabel(statement)
	if isControlFlow(statement) {
		return false
	}

	found := false
	root := ast.Node(statement)

	ast.Inspect(root, func(current ast.Node) bool {
		if found {
			return false
		}

		if current != root {
			_, isNestedStatement := current.(ast.Stmt)
			if isNestedStatement {
				return false
			}
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
