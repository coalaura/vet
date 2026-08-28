package houserules

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
)

var Breathe = &analysis.Analyzer{
	Name: "breathe",
	Doc:  "reports missing blank lines around control-flow blocks",
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

		// A comment line between the statements counts as separation.
		if nextStart > previousEnd+1 {
			continue
		}

		if nextStart == previousEnd {
			continue
		}

		if isControlFlow(previous) && previousEnd > pass.Fset.Position(previous.Pos()).Line {
			pass.Reportf(next.Pos(), "missing blank line after control-flow block")

			continue
		}

		if isControlFlow(next) && !introduces(previous, next) {
			pass.Reportf(next.Pos(), "missing blank line before control-flow block: only a statement feeding its condition may sit directly above")
		}
	}
}

// introduces reports whether previous is an assignment whose results
// appear in the header of the control-flow statement next.
func introduces(previous, next ast.Stmt) bool {
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

func isControlFlow(statement ast.Stmt) bool {
	switch unlabel(statement).(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
		return true
	}

	return false
}

func unlabel(statement ast.Stmt) ast.Stmt {
	labeled, ok := statement.(*ast.LabeledStmt)
	if ok {
		return labeled.Stmt
	}

	return statement
}
