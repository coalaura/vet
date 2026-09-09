package houserules

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

func checkVarSpecifications(pass *analysis.Pass, declaration *ast.GenDecl) {
	for _, specification := range declaration.Specs {
		value := specification.(*ast.ValueSpec)
		if len(value.Names) > 1 && len(value.Values) != 1 {
			pass.Reportf(value.Pos(), "multiple variables in declaration: use one variable per specification, preserving evaluation order and original values")
		}
	}
}

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

	if isNilInitializer(pass, expression) {
		return true
	}

	value := pass.TypesInfo.Types[expression]

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

func isNilInitializer(pass *analysis.Pass, expression ast.Expr) bool {
	expression = unparen(expression)

	value := pass.TypesInfo.Types[expression]
	if value.IsNil() {
		return true
	}

	conversion, ok := expression.(*ast.CallExpr)
	if !ok || len(conversion.Args) != 1 || !pass.TypesInfo.Types[unparen(conversion.Fun)].IsType() {
		return false
	}

	// A type parameter's underlying interface is a constraint, not the value's
	// runtime representation. Leave generic conversions to explicit var specs.
	if _, parameter := types.Unalias(value.Type).(*types.TypeParam); parameter {
		return false
	}

	switch destination := value.Type.Underlying().(type) {
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature:
	case *types.Basic:
		if destination.Kind() != types.UnsafePointer {
			return false
		}
	case *types.Interface:
		argument := pass.TypesInfo.Types[unparen(conversion.Args[0])]
		_, interfaceArgument := argument.Type.Underlying().(*types.Interface)

		// Boxing a typed nil pointer/slice/etc. creates a non-nil interface.
		if !argument.IsNil() && !interfaceArgument {
			return false
		}
	default:
		return false
	}

	return isNilInitializer(pass, conversion.Args[0])
}
