package houserules

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

const (
	mutexNone = iota
	mutexLock
	mutexUnlock
	mutexDeferredUnlock
)

func mutexSpacingBoundary(pass *analysis.Pass, statements []ast.Stmt, index int) string {
	previous := mutexOperation(pass, statements[index-1])
	next := mutexOperation(pass, statements[index])

	if previous == next {
		return ""
	}

	if previous == mutexDeferredUnlock {
		return "after deferred mutex unlock"
	}

	if next == mutexLock {
		return "before mutex lock"
	}

	if previous == mutexUnlock {
		return "after mutex unlock"
	}

	if previous == mutexLock && index > 1 && adjacentMutexOperation(pass, statements[index-2], statements[index-1], mutexLock) {
		return "after mutex lock group"
	}

	if next == mutexUnlock && index+1 < len(statements) && adjacentMutexOperation(pass, statements[index+1], statements[index], mutexUnlock) {
		return "before mutex unlock group"
	}

	return ""
}

func adjacentMutexOperation(pass *analysis.Pass, candidate, current ast.Stmt, operation int) bool {
	if mutexOperation(pass, candidate) != operation {
		return false
	}

	if candidate.Pos() < current.Pos() {
		return pass.Fset.Position(current.Pos()).Line <= pass.Fset.Position(candidate.End()).Line+1
	}

	return pass.Fset.Position(candidate.Pos()).Line <= pass.Fset.Position(current.End()).Line+1
}

func mutexOperation(pass *analysis.Pass, statement ast.Stmt) int {
	switch node := unlabel(statement).(type) {
	case *ast.DeferStmt:
		// Keep lock/defer together, but separate the protected work that follows.
		operation := mutexCallOperation(pass, node.Call)
		if operation == mutexUnlock {
			return mutexDeferredUnlock
		}
	case *ast.ExprStmt:
		call, ok := unparen(node.X).(*ast.CallExpr)
		if ok {
			return mutexCallOperation(pass, call)
		}
	}

	// Asynchronous calls and stored method values are not mutex boundaries.
	return mutexNone
}

func mutexCallOperation(pass *analysis.Pass, call *ast.CallExpr) int {
	selector, ok := unparen(call.Fun).(*ast.SelectorExpr)
	if !ok {
		return mutexNone
	}

	selection := pass.TypesInfo.Selections[selector]
	if selection == nil {
		return mutexNone
	}

	if selection.Kind() != types.MethodVal && selection.Kind() != types.MethodExpr {
		return mutexNone
	}

	// Method expressions pass the receiver explicitly; bound methods do not.
	var argumentCount int

	if selection.Kind() == types.MethodExpr {
		argumentCount = 1
	}

	if len(call.Args) != argumentCount {
		return mutexNone
	}

	method := selection.Obj()
	if method.Pkg() == nil || method.Pkg().Path() != "sync" {
		return mutexNone
	}

	switch method.Name() {
	case "Lock", "RLock":
		return mutexLock
	case "Unlock", "RUnlock":
		return mutexUnlock
	}

	return mutexNone
}
