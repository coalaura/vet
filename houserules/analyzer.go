package houserules

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "houserules",
	Doc:  "reports specific go style violations",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}

		checkFileVarDeclarations(pass, file.Decls)

		declaredStructs := make(map[*ast.StructType]bool)

		ast.Inspect(file, func(current ast.Node) bool {
			switch node := current.(type) {
			case *ast.BlockStmt:
				checkStatementVarDeclarations(pass, node.List)
			case *ast.CaseClause:
				checkStatementVarDeclarations(pass, node.Body)
			case *ast.CommClause:
				checkStatementVarDeclarations(pass, node.Body)
			case *ast.TypeSpec:
				structType, ok := node.Type.(*ast.StructType)
				if ok {
					// A direct TypeSpec names this struct. Nested struct
					// types remain anonymous and are still reported.
					declaredStructs[structType] = true
				}
			case *ast.IfStmt:
				if node.Init != nil && !isCommaOK(node.Init) {
					pass.Reportf(node.Init.Pos(), "initializer in if statement: assign the value first, then check it")
				}
			case *ast.SwitchStmt:
				if node.Init != nil {
					pass.Reportf(node.Init.Pos(), "initializer in switch statement: assign the value first, then switch on it")
				}
			case *ast.TypeSwitchStmt:
				if node.Init != nil {
					pass.Reportf(node.Init.Pos(), "initializer in type switch: assign the value first, then switch on it")
				}
			case *ast.RangeStmt:
				if isRangeLiteral(node.X) {
					pass.Reportf(node.X.Pos(), "composite literal in range position: assign it to a named variable first")
				}
			case *ast.AssignStmt:
				if len(node.Lhs) > 1 && len(node.Rhs) > 1 && !isSwapAssignment(node) {
					pass.Reportf(node.Pos(), "chained assignment: use one assignment per target")
				}
			case *ast.DeclStmt:
				declaration, ok := node.Decl.(*ast.GenDecl)
				if !ok {
					return true
				}

				switch declaration.Tok {
				case token.CONST:
					pass.Reportf(declaration.Pos(), "function-local const: move it to the package-scope const block")
				case token.TYPE:
					pass.Reportf(declaration.Pos(), "function-local type: declare it in the file's type section")
				}
			case *ast.StructType:
				if len(node.Fields.List) == 0 || declaredStructs[node] {
					return true
				}

				pass.Reportf(node.Pos(), "anonymous struct type: declare a named package-scope type")
			}

			return true
		})
	}

	return nil, nil
}

func checkFileVarDeclarations(pass *analysis.Pass, declarations []ast.Decl) {
	for index := 1; index < len(declarations); index++ {
		previous, previousOK := declarations[index-1].(*ast.GenDecl)
		current, currentOK := declarations[index].(*ast.GenDecl)

		if previousOK && currentOK && previous.Tok == token.VAR && current.Tok == token.VAR && pass.Fset.Position(current.Pos()).Line == pass.Fset.Position(previous.End()).Line+1 {
			pass.Reportf(current.Pos(), "consecutive var declarations: combine them into a var block")
		}
	}
}

func checkStatementVarDeclarations(pass *analysis.Pass, statements []ast.Stmt) {
	for index := 1; index < len(statements); index++ {
		previous, previousOK := varDeclaration(statements[index-1])
		current, currentOK := varDeclaration(statements[index])

		if previousOK && currentOK && previous.Tok == token.VAR && current.Tok == token.VAR && pass.Fset.Position(current.Pos()).Line == pass.Fset.Position(previous.End()).Line+1 {
			pass.Reportf(current.Pos(), "consecutive var declarations: combine them into a var block")
		}
	}
}

func varDeclaration(statement ast.Stmt) (*ast.GenDecl, bool) {
	declarationStatement, ok := unlabel(statement).(*ast.DeclStmt)
	if !ok {
		return nil, false
	}

	declaration, ok := declarationStatement.Decl.(*ast.GenDecl)

	return declaration, ok
}

func isRangeLiteral(expression ast.Expr) bool {
	expression = unparen(expression)

	address, ok := expression.(*ast.UnaryExpr)
	if ok && address.Op == token.AND {
		expression = unparen(address.X)
	}

	_, isLiteral := expression.(*ast.CompositeLit)

	return isLiteral
}

func unparen(expression ast.Expr) ast.Expr {
	for {
		parentheses, ok := expression.(*ast.ParenExpr)
		if !ok {
			return expression
		}

		expression = parentheses.X
	}
}

func isSwapAssignment(assignment *ast.AssignStmt) bool {
	if assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 2 {
		return false
	}

	leftFirst := types.ExprString(unparen(assignment.Lhs[0]))
	leftSecond := types.ExprString(unparen(assignment.Lhs[1]))
	rightFirst := types.ExprString(unparen(assignment.Rhs[0]))
	rightSecond := types.ExprString(unparen(assignment.Rhs[1]))

	return leftFirst != leftSecond && leftFirst == rightSecond && leftSecond == rightFirst
}

// isCommaOK reports whether init is a two-value map lookup or type assertion.
func isCommaOK(init ast.Stmt) bool {
	assignment, ok := init.(*ast.AssignStmt)
	if !ok {
		return false
	}

	if len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
		return false
	}

	switch assignment.Rhs[0].(type) {
	case *ast.IndexExpr, *ast.TypeAssertExpr:
		return true
	}

	return false
}
