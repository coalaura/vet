package houserules

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

func assignmentTargetUsesExpression(pass *analysis.Pass, destination, target ast.Expr) bool {
	// Assigning a location reads its receiver, index or pointer, but does not
	// read the stored value simply by naming the destination.
	switch expression := unparen(destination).(type) {
	case *ast.SelectorExpr:
		return nodeUsesExpression(pass, expression.X, target, true)
	case *ast.IndexExpr:
		return nodeUsesExpression(pass, expression.X, target, true) || nodeUsesExpression(pass, expression.Index, target, true)
	case *ast.StarExpr:
		return nodeUsesExpression(pass, expression.X, target, true)
	}

	return false
}

func nodeUsesExpression(pass *analysis.Pass, node ast.Node, target ast.Expr, relatedReceiver bool) bool {
	if node == nil {
		return false
	}

	targetSelector, _ := unparen(target).(*ast.SelectorExpr)
	found := false

	ast.Inspect(node, func(current ast.Node) bool {
		if found {
			return false
		}

		if _, isFunction := current.(*ast.FuncLit); isFunction {
			return false
		}

		expression, ok := current.(ast.Expr)
		if ok && sameExpression(pass, expression, target) {
			found = true

			return false
		}

		// Conditions deliberately allow related fields/methods of the same
		// receiver. Return values require an exact dependency instead.
		selector, ok := current.(*ast.SelectorExpr)
		if relatedReceiver && targetSelector != nil && ok && sameExpression(pass, selector.X, targetSelector.X) {
			found = true
		}

		return !found
	})

	return found
}

// sameExpression compares resolved names and expression structure without
// formatting ASTs. Calls and receives cannot identify a stable location:
// repeating either can produce different values or have side effects.
func sameExpression(pass *analysis.Pass, left, right ast.Expr) bool {
	left = unparen(left)
	right = unparen(right)

	leftValue := pass.TypesInfo.Types[left]
	rightValue := pass.TypesInfo.Types[right]

	if leftValue.Value != nil && rightValue.Value != nil && types.Identical(leftValue.Type, rightValue.Type) {
		return constant.Compare(leftValue.Value, token.EQL, rightValue.Value)
	}

	switch expression := left.(type) {
	case *ast.Ident:
		other, ok := right.(*ast.Ident)
		if !ok || expression.Name == "_" || other.Name == "_" {
			return false
		}

		object := pass.TypesInfo.ObjectOf(expression)

		return object != nil && object == pass.TypesInfo.ObjectOf(other)
	case *ast.SelectorExpr:
		other, ok := right.(*ast.SelectorExpr)
		if !ok || !sameExpression(pass, expression.X, other.X) {
			return false
		}

		object := pass.TypesInfo.ObjectOf(expression.Sel)

		return object != nil && object == pass.TypesInfo.ObjectOf(other.Sel)
	case *ast.IndexExpr:
		other, ok := right.(*ast.IndexExpr)

		return ok && sameExpression(pass, expression.X, other.X) && sameExpression(pass, expression.Index, other.Index)
	case *ast.StarExpr:
		other, ok := right.(*ast.StarExpr)

		return ok && sameExpression(pass, expression.X, other.X)
	case *ast.UnaryExpr:
		other, ok := right.(*ast.UnaryExpr)

		return ok && expression.Op != token.ARROW && expression.Op == other.Op && sameExpression(pass, expression.X, other.X)
	case *ast.BinaryExpr:
		other, ok := right.(*ast.BinaryExpr)

		return ok && expression.Op == other.Op && sameExpression(pass, expression.X, other.X) && sameExpression(pass, expression.Y, other.Y)
	case *ast.CallExpr:
		other, ok := right.(*ast.CallExpr)
		if !ok || len(expression.Args) != 1 || len(other.Args) != 1 {
			return false
		}

		// A type conversion is stable when its argument is stable; a function
		// call with identical spelling need not return the same value twice.
		leftConversion := pass.TypesInfo.Types[unparen(expression.Fun)].IsType()
		rightConversion := pass.TypesInfo.Types[unparen(other.Fun)].IsType()

		return leftConversion && rightConversion && types.Identical(leftValue.Type, rightValue.Type) && sameExpression(pass, expression.Args[0], other.Args[0])
	}

	return false
}
