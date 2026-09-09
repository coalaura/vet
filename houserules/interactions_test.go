package houserules_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/coalaura/vet/houserules"

	"golang.org/x/tools/go/analysis"
)

type ruleInteraction struct {
	name   string
	before string
	after  string
	want   []string
}

func TestRuleInteractions(t *testing.T) {
	cases := []ruleInteraction{
		{
			name: "grouping takes precedence over function literal spacing",
			before: `func example() {
	first := 0
	var callback = func() {}

	_ = first
	_ = callback
}`,
			after: `func example() {
	var (
		first int
		callback = func() {}
	)

	_ = first
	_ = callback
}`,
			want: []string{"consecutive var declarations"},
		},
		{
			name: "typed nil declarations group without spacing diagnostics",
			before: `func example() {
	pointer := (*int)(nil)
	slice := []int(nil)

	_ = pointer
	_ = slice
}`,
			after: `func example() {
	var (
		pointer *int
		slice []int
	)

	_ = pointer
	_ = slice
}`,
			want: []string{"consecutive zero-value declarations"},
		},
		{
			name: "one diagnostic at a function literal feeder boundary",
			before: `func example() {
	callback := func() {}
	value := 1
	if value > 0 {
		callback()
	}
}`,
			after: `func example() {
	callback := func() {}

	value := 1
	if value > 0 {
		callback()
	}
}`,
			want: []string{"missing blank line after function literal"},
		},
		{
			name: "split variable specifications remain a compact var block",
			before: `func example() {
	var first, second = 1, 2

	_ = first
	_ = second
}`,
			after: `func example() {
	var (
		first = 1
		second = 2
	)

	_ = first
	_ = second
}`,
			want: []string{"multiple variables in declaration"},
		},
	}

	for index := range cases {
		testCase := &cases[index]

		t.Run(testCase.name, func(t *testing.T) {
			checkRuleSnippet(t, testCase.before, testCase.want)
			checkRuleSnippet(t, testCase.after, nil)
		})
	}
}

func checkRuleSnippet(t *testing.T, source string, want []string) {
	t.Helper()

	fileSet := token.NewFileSet()

	file, err := parser.ParseFile(fileSet, "interaction.go", "package interaction\n"+source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	information := &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
	configuration := types.Config{}
	files := []*ast.File{file}

	checkedPackage, err := configuration.Check("interaction", fileSet, files, information)
	if err != nil {
		t.Fatal(err)
	}

	diagnostics := make([]string, 0, len(want))

	pass := &analysis.Pass{
		Fset:      fileSet,
		Files:     files,
		Pkg:       checkedPackage,
		TypesInfo: information,
		Report: func(diagnostic analysis.Diagnostic) {
			diagnostics = append(diagnostics, diagnostic.Message)
		},
	}

	analyzers := []*analysis.Analyzer{houserules.Analyzer, houserules.Breathe}

	for _, analyzer := range analyzers {
		pass.Analyzer = analyzer

		_, err = analyzer.Run(pass)
		if err != nil {
			t.Fatal(err)
		}
	}

	if len(diagnostics) != len(want) {
		t.Fatalf("diagnostics = %q, want %q\nsource:\n%s", diagnostics, want, source)
	}

	for index, expected := range want {
		if !strings.Contains(diagnostics[index], expected) {
			t.Errorf("diagnostic = %q, want substring %q", diagnostics[index], expected)
		}
	}
}
