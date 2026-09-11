package main

import (
	"os"
	"path/filepath"
	"testing"
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
