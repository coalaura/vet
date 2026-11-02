package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLintCGO(t *testing.T) {
	if os.Getenv("CGO_ENABLED") != "1" {
		t.Skip("requires a CGO toolchain; run builder test go --cgo --no-gen")
	}

	t.Chdir("testdata/cgo")

	output, code, err := executeLint(lintOptions{Checks: "houserules,breathe", Tests: false}, "json")
	if err != nil {
		t.Fatal(err)
	}

	if code != 1 {
		t.Fatalf("exit code = %d, want 1; output:\n%s", code, output)
	}

	diagnostics, err := decodeDiagnostics(output)
	if err != nil {
		t.Fatal(err)
	}

	want := map[int]string{
		17: "houserules",
		20: "breathe",
		23: "breathe",
		26: "breathe",
		31: "houserules",
		37: "breathe",
		43: "breathe",
	}

	for _, diagnostic := range diagnostics {
		check, ok := want[diagnostic.Location.Line]
		if filepath.Base(diagnostic.Location.File) != "source.go" || !ok || diagnostic.Code != check {
			t.Errorf("unexpected diagnostic: %+v", diagnostic)

			continue
		}

		delete(want, diagnostic.Location.Line)
	}

	if len(want) != 0 {
		t.Errorf("missing diagnostics by line: %v; output:\n%s", want, output)
	}
}
