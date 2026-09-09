package houserules

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

func isVarDeclaration(statement ast.Stmt) bool {
	declaration, ok := varDeclaration(statement)

	return ok && declaration.Tok == token.VAR
}

func canGroupVarStatements(pass *analysis.Pass, previous, next ast.Stmt) bool {
	// Moving a declaration across a label can change goto validity.
	_, previousLabeled := previous.(*ast.LabeledStmt)
	_, nextLabeled := next.(*ast.LabeledStmt)

	if previousLabeled || nextLabeled {
		return false
	}

	return isGroupableVarStatement(pass, previous) && isGroupableVarStatement(pass, next)
}

func isGroupableVarStatement(pass *analysis.Pass, statement ast.Stmt) bool {
	declaration, ok := varDeclaration(statement)
	if ok {
		// Explicit var specs can move intact: retain their types, initializers
		// and evaluation order, even when their values are not zero.
		return declaration.Tok == token.VAR && len(declaration.Specs) > 0
	}

	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != len(assignment.Rhs) {
		return false
	}

	for index, target := range assignment.Lhs {
		name, ok := target.(*ast.Ident)
		if !ok || !isZeroInitializer(pass, name, assignment.Rhs[index]) {
			return false
		}
	}

	return len(assignment.Lhs) > 0
}

func isZeroInitializer(pass *analysis.Pass, name *ast.Ident, expression ast.Expr) bool {
	variable, ok := pass.TypesInfo.Defs[name].(*types.Var)
	if !ok || name.Name == "_" {
		return false
	}

	expression = unparen(expression)

	value := pass.TypesInfo.Types[expression]
	if value.IsNil() {
		return true
	}

	// The destination type matters: an interface containing 0 is not nil.
	_, basic := variable.Type().Underlying().(*types.Basic)
	if basic && value.Value != nil {
		switch value.Value.Kind() {
		case constant.Bool:
			return !constant.BoolVal(value.Value)
		case constant.String:
			return constant.StringVal(value.Value) == ""
		case constant.Int, constant.Float:
			return constant.Sign(value.Value) == 0
		case constant.Complex:
			return constant.Sign(constant.Real(value.Value)) == 0 && constant.Sign(constant.Imag(value.Value)) == 0
		}
	}

	literal, ok := expression.(*ast.CompositeLit)
	if !ok || len(literal.Elts) != 0 {
		return false
	}

	// Empty slices and maps are non-nil, unlike their zero values.
	switch variable.Type().Underlying().(type) {
	case *types.Array, *types.Struct:
		return true
	}

	return false
}
