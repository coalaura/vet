package houserules

import (
	"go/ast"
	"go/token"

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

		declaredStructs := make(map[*ast.StructType]bool)

		ast.Inspect(file, func(current ast.Node) bool {
			switch node := current.(type) {
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
