package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/urfave/cli/v3"
	"honnef.co/go/tools/lintcmd"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
	"honnef.co/go/tools/unused"

	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/modernize"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
)

type Location struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

type Related struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

type Diagnostic struct {
	Location Location  `json:"location"`
	Related  []Related `json:"related,omitempty"`
	Code     string    `json:"code"`
	Severity string    `json:"severity,omitempty"`
	Message  string    `json:"message"`
}

var Version = "dev"

func main() {
	var exitCode int

	app := &cli.Command{
		Name:            "vet",
		Usage:           "run analyzers and pretty-print diagnostics",
		Version:         Version,
		SkipFlagParsing: true,
		Action: func(_ context.Context, c *cli.Command) error {
			code, err := run(c.Args().Slice())
			if err != nil {
				return err
			}

			exitCode = code

			return nil
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "vet: %v\n", err)

		os.Exit(2)
	}

	os.Exit(exitCode)
}

func run(rawArgs []string) (int, error) {
	cmd := newLintCommand()

	cmd.ParseFlags(forceJSONFormat(rawArgs))

	out, code, err := captureCommandOutput(cmd.Execute)
	if err != nil {
		return 2, err
	}

	if !looksLikeJSONStream(out) {
		_, err = os.Stdout.Write(out)
		if err != nil {
			return 2, fmt.Errorf("write analyzer output: %w", err)
		}

		return code, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}

	issued, err := renderDiagnostics(cwd, out)
	if err != nil {
		_, err = os.Stdout.Write(out)
		if err != nil {
			return 2, fmt.Errorf("decode diagnostics: %v; write raw output: %w", err, err)
		}

		return code, nil
	}

	if issued {
		return 1, nil
	}

	return code, nil
}

func newLintCommand() *lintcmd.Command {
	cmd := lintcmd.NewCommand("vet")
	cmd.SetVersion("1.0.0", "v1.0.0")

	// go vet analyzers
	cmd.AddBareAnalyzers(
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		framepointer.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
	)

	// staticcheck analyzers
	cmd.AddAnalyzers(simple.Analyzers...)
	cmd.AddAnalyzers(staticcheck.Analyzers...)
	cmd.AddAnalyzers(stylecheck.Analyzers...)
	cmd.AddAnalyzers(unused.Analyzer)

	// modernize analyzers
	cmd.AddBareAnalyzers(modernize.Suite...)

	return cmd
}

func forceJSONFormat(rawArgs []string) []string {
	args := make([]string, 0, len(rawArgs)+2)

	args = append(args, "-f", "json")

	for i := 0; i < len(rawArgs); i++ {
		arg := rawArgs[i]

		switch {
		case arg == "-f":
			if i+1 < len(rawArgs) {
				i++
			}

			continue
		case strings.HasPrefix(arg, "-f="):
			continue
		}

		args = append(args, arg)
	}

	return args
}

func captureCommandOutput(run func() int) ([]byte, int, error) {
	rd, wr, err := os.Pipe()
	if err != nil {
		return nil, 2, fmt.Errorf("create output pipe: %w", err)
	}

	stdout := os.Stdout

	os.Stdout = wr

	defer func() {
		os.Stdout = stdout
	}()

	var (
		wg      sync.WaitGroup
		buf     bytes.Buffer
		copyErr error
	)

	wg.Go(func() {
		_, copyErr = io.Copy(&buf, rd)
	})

	code := run()

	closeErr := wr.Close()

	wg.Wait()

	readCloseErr := rd.Close()

	if closeErr != nil {
		return nil, code, fmt.Errorf("close output pipe writer: %w", closeErr)
	}

	if copyErr != nil {
		return nil, code, fmt.Errorf("read analyzer output: %w", copyErr)
	}

	if readCloseErr != nil {
		return nil, code, fmt.Errorf("close output pipe reader: %w", readCloseErr)
	}

	return buf.Bytes(), code, nil
}

func renderDiagnostics(cwd string, out []byte) (bool, error) {
	dec := json.NewDecoder(bytes.NewReader(out))

	var issued bool

	for {
		var diag Diagnostic

		err := dec.Decode(&diag)
		if err != nil {
			if err == io.EOF {
				return issued, nil
			}

			return false, fmt.Errorf("decode diagnostic stream: %w", err)
		}

		issued = true

		file := relPath(cwd, diag.Location.File)
		line := diag.Location.Line
		col := diag.Location.Column

		color := "\033[36m"

		if diag.Severity == "error" || strings.HasPrefix(diag.Code, "compile") || diag.Code == "config" {
			color = "\033[31m"
		}

		_, err = fmt.Fprintf(os.Stdout, "   %s-> %s:%d:%d\033[0m \033[90m(%s)\033[0m %s\n", color, file, line, col, diag.Code, diag.Message)
		if err != nil {
			return false, fmt.Errorf("write diagnostic: %w", err)
		}

		for _, rel := range diag.Related {
			rf := relPath(cwd, rel.Location.File)

			_, err = fmt.Fprintf(os.Stdout, "   \033[90m-> %s:%d:%d\033[0m %s\n", rf, rel.Location.Line, rel.Location.Column, rel.Message)
			if err != nil {
				return false, fmt.Errorf("write related diagnostic: %w", err)
			}
		}
	}
}

func looksLikeJSONStream(b []byte) bool {
	b = bytes.TrimSpace(b)

	return len(b) > 0 && b[0] == '{'
}

func relPath(cwd, p string) string {
	if cwd == "" {
		return p
	}

	r, err := filepath.Rel(cwd, p)
	if err == nil && !strings.HasPrefix(r, "..") {
		return r
	}

	return p
}
