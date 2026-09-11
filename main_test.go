package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"honnef.co/go/tools/sarif"
)

type spacingFixCase struct {
	name    string
	source  string
	want    string
	fixes   map[int]spacingFix
	wantFix int
}

func TestApplySpacingFixes(t *testing.T) {
	cases := []spacingFixCase{
		{
			name:   "insert LF blank line",
			source: "func example() {\n\twork()\n\treturn\n}\n",
			want:   "func example() {\n\twork()\n\n\treturn\n}\n",
			fixes: map[int]spacingFix{
				3: spacingFixInsert,
			},
			wantFix: 1,
		},
		{
			name:   "insert CRLF blank line",
			source: "func example() {\r\n\twork()\r\n\treturn\r\n}\r\n",
			want:   "func example() {\r\n\twork()\r\n\r\n\treturn\r\n}\r\n",
			fixes: map[int]spacingFix{
				3: spacingFixInsert,
			},
			wantFix: 1,
		},
		{
			name:   "remove adjacent blank line",
			source: "func example() {\n\terr := work()\n\n\tif err != nil {\n\t\treturn\n\t}\n}\n",
			want:   "func example() {\n\terr := work()\n\tif err != nil {\n\t\treturn\n\t}\n}\n",
			fixes: map[int]spacingFix{
				4: spacingFixRemove,
			},
			wantFix: 1,
		},
		{
			name:   "leave non-adjacent blank line",
			source: "func example() {\n\terr := work()\n\n\t// Explain the check.\n\tif err != nil {\n\t\treturn\n\t}\n}\n",
			want:   "func example() {\n\terr := work()\n\n\t// Explain the check.\n\tif err != nil {\n\t\treturn\n\t}\n}\n",
			fixes: map[int]spacingFix{
				5: spacingFixRemove,
			},
		},
	}

	for index := range cases {
		testCase := &cases[index]

		t.Run(testCase.name, func(t *testing.T) {
			updated, fixed := applySpacingFixes([]byte(testCase.source), testCase.fixes)
			if string(updated) != testCase.want {
				t.Errorf("updated source:\n%s\nwant:\n%s", updated, testCase.want)
			}

			if fixed != testCase.wantFix {
				t.Errorf("fixed = %d, want %d", fixed, testCase.wantFix)
			}
		})
	}
}

func TestApplyAutomaticFixes(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "example.go")

	source := "package example\n\nfunc example() {\n\twork()\n\treturn\n}\n"

	err := os.WriteFile(path, []byte(source), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	diagnostics := []Diagnostic{
		{
			Location: Location{File: path, Line: 5, Column: 2},
			Code:     "breathe",
			Message:  "missing blank line before return: only a statement feeding its result may sit directly above",
		},
		{
			Location: Location{File: path, Line: 4, Column: 2},
			Code:     "SA0000",
			Message:  "missing blank line should not make another analyzer fixable",
		},
	}

	fixed, err := applyAutomaticFixes(diagnostics)
	if err != nil {
		t.Fatal(err)
	}

	if fixed != 1 {
		t.Errorf("fixed = %d, want 1", fixed)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := "package example\n\nfunc example() {\n\twork()\n\n\treturn\n}\n"
	if string(contents) != want {
		t.Errorf("updated source:\n%s\nwant:\n%s", contents, want)
	}
}

func TestApplySuggestedFixes(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)

	path := filepath.Join(directory, "example.go")
	source := "package example\nvar first = value == true\nvar second = value != false\n"

	err := os.WriteFile(path, []byte(source), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	firstFix := sarif.Fix{
		ArtifactChanges: []sarif.ArtifactChange{
			{
				ArtifactLocation: sarif.ArtifactLocation{URI: "example.go"},
				Replacements: []sarif.Replacement{
					{
						DeletedRegion:   sarif.Region{StartLine: 2, StartColumn: 13, EndLine: 2, EndColumn: 26},
						InsertedContent: sarif.ArtifactContent{Text: "value"},
					},
				},
			},
		},
	}
	alternativeFix := sarif.Fix{
		ArtifactChanges: []sarif.ArtifactChange{
			{
				ArtifactLocation: sarif.ArtifactLocation{URI: "example.go"},
				Replacements: []sarif.Replacement{
					{
						DeletedRegion:   sarif.Region{StartLine: 3, StartColumn: 14, EndLine: 3, EndColumn: 28},
						InsertedContent: sarif.ArtifactContent{Text: "value"},
					},
				},
			},
		},
	}
	log := sarif.Log{
		Runs: []sarif.Run{
			{
				Results: []sarif.Result{
					{Fixes: []sarif.Fix{firstFix}},
					{Fixes: []sarif.Fix{firstFix}},
					{Fixes: []sarif.Fix{alternativeFix, alternativeFix}},
				},
			},
		},
	}

	output, err := json.Marshal(log)
	if err != nil {
		t.Fatal(err)
	}

	fixed, err := applySuggestedFixes(output)
	if err != nil {
		t.Fatal(err)
	}

	if fixed != 1 {
		t.Errorf("fixed = %d, want 1", fixed)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := "package example\nvar first = value\nvar second = value != false\n"
	if string(contents) != want {
		t.Errorf("updated source:\n%s\nwant:\n%s", contents, want)
	}
}

func TestApplySuggestedFixesSkipsConflicts(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)

	path := filepath.Join(directory, "example.go")
	source := "package example\nvar value = true\n"

	err := os.WriteFile(path, []byte(source), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	result := sarif.Result{
		Fixes: []sarif.Fix{
			{
				ArtifactChanges: []sarif.ArtifactChange{
					{
						ArtifactLocation: sarif.ArtifactLocation{URI: "example.go"},
						Replacements: []sarif.Replacement{
							{
								DeletedRegion:   sarif.Region{StartLine: 2, StartColumn: 5, EndLine: 2, EndColumn: 10},
								InsertedContent: sarif.ArtifactContent{Text: "other"},
							},
							{
								DeletedRegion:   sarif.Region{StartLine: 2, StartColumn: 7, EndLine: 2, EndColumn: 12},
								InsertedContent: sarif.ArtifactContent{Text: "conflict"},
							},
						},
					},
				},
			},
		},
	}
	log := sarif.Log{Runs: []sarif.Run{{Results: []sarif.Result{result}}}}

	output, err := json.Marshal(log)
	if err != nil {
		t.Fatal(err)
	}

	fixed, err := applySuggestedFixes(output)
	if err != nil {
		t.Fatal(err)
	}

	if fixed != 0 {
		t.Errorf("fixed = %d, want 0", fixed)
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(contents) != source {
		t.Errorf("conflicting fix changed source:\n%s", contents)
	}
}
