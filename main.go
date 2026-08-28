package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/urfave/cli/v3"
	"honnef.co/go/tools/lintcmd"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
	"honnef.co/go/tools/unused"

	"github.com/charithe/durationcheck"
	"github.com/coalaura/builder/goenv"
	"github.com/coalaura/vet/houserules"
	"github.com/polyfloyd/go-errorlint/errorlint"
	"github.com/timakin/bodyclose/passes/bodyclose"

	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/modernize"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	vetsuite "golang.org/x/tools/go/analysis/suite/vet"
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

type lintOptions struct {
	Packages    []string
	Checks      string
	Explain     string
	Fail        string
	GoVersion   string
	GOARCH      string
	GOOS        string
	Tags        string
	ListChecks  bool
	ShowIgnored bool
	Tests       bool
	CGO         bool
}

var Version = "dev"

func main() {
	var (
		exitCode     int
		printVersion bool

		checks      = "inherit"
		cgoEnabled  bool
		explain     string
		fail        = "all"
		goVersion   = "module"
		listChecks  bool
		showIgnored bool
		tags        string
		targetArch  = runtime.GOARCH
		targetOS    = runtime.GOOS
		tests       = true
	)

	app := &cli.Command{
		Name:                   "vet",
		Usage:                  "run analyzers and pretty-print diagnostics",
		UsageText:              "vet [options] [packages]",
		Version:                Version,
		HideVersion:            true,
		UseShortOptionHandling: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "arch",
				Usage:       "target architecture (GOARCH)",
				Value:       runtime.GOARCH,
				Destination: &targetArch,
			},
			&cli.BoolFlag{
				Name:        "cgo",
				Usage:       "enable cgo with reproducible compiler settings",
				Destination: &cgoEnabled,
			},
			&cli.StringFlag{
				Name:        "checks",
				Usage:       "comma-separated list of checks to enable",
				Value:       "inherit",
				Destination: &checks,
			},
			&cli.StringFlag{
				Name:        "explain",
				Usage:       "print description of check",
				Destination: &explain,
			},
			&cli.StringFlag{
				Name:        "fail",
				Usage:       "comma-separated list of checks that can cause a non-zero exit status",
				Value:       "all",
				Destination: &fail,
			},
			&cli.StringFlag{
				Name:        "go",
				Usage:       "target Go version in the format '1.x' or 'module'",
				Value:       "module",
				Destination: &goVersion,
			},
			&cli.BoolFlag{
				Name:        "list-checks",
				Usage:       "list all available checks",
				Destination: &listChecks,
			},
			&cli.StringFlag{
				Name:        "os",
				Usage:       "target operating system (GOOS)",
				Value:       runtime.GOOS,
				Destination: &targetOS,
			},
			&cli.BoolFlag{
				Name:        "show-ignored",
				Usage:       "don't filter ignored diagnostics",
				Destination: &showIgnored,
			},
			&cli.StringFlag{
				Name:        "tags",
				Usage:       "list of build tags",
				Destination: &tags,
			},
			&cli.BoolFlag{
				Name:        "tests",
				Usage:       "include tests",
				Value:       true,
				Destination: &tests,
			},
			&cli.BoolFlag{
				Name:        "version",
				Aliases:     []string{"v"},
				Usage:       "print version and exit",
				Destination: &printVersion,
			},
		},
		Action: func(_ context.Context, c *cli.Command) error {
			if printVersion {
				_, err := fmt.Fprintln(os.Stdout, Version)
				if err != nil {
					return fmt.Errorf("write version: %w", err)
				}

				exitCode = 0

				return nil
			}

			code, err := run(lintOptions{
				CGO:         cgoEnabled,
				Checks:      checks,
				Explain:     explain,
				Fail:        fail,
				GoVersion:   goVersion,
				GOARCH:      targetArch,
				GOOS:        targetOS,
				ListChecks:  listChecks,
				ShowIgnored: showIgnored,
				Tags:        tags,
				Tests:       tests,
				Packages:    c.Args().Slice(),
			})

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

func run(opts lintOptions) (int, error) {
	tags, err := setBuildTarget(opts.GOOS, opts.GOARCH, opts.CGO, opts.Tags)
	if err != nil {
		return 2, err
	}

	opts.Tags = tags

	cmd := newLintCommand()

	cmd.ParseFlags(forceJSONFormat(buildLintArgs(opts)))

	out, code, err := captureCommandOutput(cmd.Execute)
	if err != nil {
		return 2, err
	}

	if code == 0 && len(bytes.TrimSpace(out)) == 0 {
		_, err = fmt.Fprintln(os.Stdout, "\x1b[32m::\x1b[0m no issues found")
		if err != nil {
			return 2, fmt.Errorf("write success message: %w", err)
		}

		return code, nil
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

	diagnosticCount, err := renderDiagnostics(cwd, out)
	if err != nil {
		_, writeErr := os.Stdout.Write(out)
		if writeErr != nil {
			return 2, fmt.Errorf(
				"decode diagnostics: %v; write raw output: %w",
				err,
				writeErr,
			)
		}

		return code, nil
	}

	if diagnosticCount == 0 {
		_, err = fmt.Fprintln(os.Stdout, "\x1b[32m::\x1b[0m no issues found")
		if err != nil {
			return 2, fmt.Errorf("write success message: %w", err)
		}

		return 0, nil
	}

	return code, nil
}

func setBuildTarget(targetOS, targetArch string, cgoEnabled bool, tags string) (string, error) {
	if targetOS == "" {
		return "", errors.New("GOOS must not be empty")
	}

	if targetArch == "" {
		return "", errors.New("GOARCH must not be empty")
	}

	config := goenv.Prepare(goenv.Options{
		Tags: strings.Split(tags, ","),
		OS:   targetOS,
		Arch: targetArch,
		CGO:  cgoEnabled,
	})

	for key, value := range config.Env {
		var err error

		if value == "" {
			err = os.Unsetenv(key)
		} else {
			err = os.Setenv(key, value)
		}

		if err != nil {
			return "", fmt.Errorf("set build environment %s: %w", key, err)
		}
	}

	return buildTags(config.BuildFlags), nil
}

func buildTags(buildFlags []string) string {
	for index, flag := range buildFlags {
		if flag == "-tags" && index+1 < len(buildFlags) {
			return buildFlags[index+1]
		}
	}

	return ""
}

func buildLintArgs(opts lintOptions) []string {
	args := make([]string, 0, 16+len(opts.Packages))

	if opts.Checks != "" && opts.Checks != "inherit" {
		args = append(args, "-checks", opts.Checks)
	}

	if opts.Explain != "" {
		args = append(args, "-explain", opts.Explain)
	}

	if opts.Fail != "" && opts.Fail != "all" {
		args = append(args, "-fail", opts.Fail)
	}

	if opts.GoVersion != "" && opts.GoVersion != "module" {
		args = append(args, "-go", opts.GoVersion)
	}

	if opts.ListChecks {
		args = append(args, "-list-checks")
	}

	if opts.ShowIgnored {
		args = append(args, "-show-ignored")
	}

	if opts.Tags != "" {
		args = append(args, "-tags", opts.Tags)
	}

	if !opts.Tests {
		args = append(args, "-tests=false")
	}

	if len(opts.Packages) == 0 {
		args = append(args, "./...")
	} else {
		args = append(args, opts.Packages...)
	}

	return args
}

func newLintCommand() *lintcmd.Command {
	cmd := lintcmd.NewCommand("vet")

	cmd.SetVersion(strings.TrimPrefix(Version, "v"), Version)

	// Standard Go vet
	cmd.AddBareAnalyzers(vetsuite.Suite...)

	// Additional x/tools correctness analyzers
	cmd.AddBareAnalyzers(
		nilness.Analyzer,
		unusedwrite.Analyzer,
		atomicalign.Analyzer,
		deepequalerrors.Analyzer,
		reflectvaluecompare.Analyzer,
		sortslice.Analyzer,
	)

	// Staticcheck
	cmd.AddAnalyzers(simple.Analyzers...)
	cmd.AddAnalyzers(staticcheck.Analyzers...)
	cmd.AddAnalyzers(stylecheck.Analyzers...)
	cmd.AddAnalyzers(unused.Analyzer)

	// Staticcheck quick fixes
	cmd.AddAnalyzers(quickfix.Analyzers...)

	// Modern Go patterns
	cmd.AddBareAnalyzers(modernize.Suite...)

	// Error handling correctness
	cmd.AddBareAnalyzers(
		errorlint.NewAnalyzer(),
	)

	// Resource correctness
	cmd.AddBareAnalyzers(
		bodyclose.Analyzer,
	)

	// Suspicious duration arithmetic
	cmd.AddBareAnalyzers(
		durationcheck.Analyzer,
	)

	// House rules
	cmd.AddBareAnalyzers(
		houserules.Analyzer,
		houserules.Breathe,
	)

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

func renderDiagnostics(cwd string, out []byte) (int, error) {
	diagnostics, err := decodeDiagnostics(out)
	if err != nil {
		return 0, err
	}

	if len(diagnostics) == 0 {
		return 0, nil
	}

	fileMap := make(map[string][]Diagnostic, len(diagnostics))
	generatedFiles := make(map[string]bool)

	for _, diag := range diagnostics {
		generated, checked := generatedFiles[diag.Location.File]
		if !checked {
			generated = isGeneratedFile(diag.Location.File)
			generatedFiles[diag.Location.File] = generated
		}

		if generated {
			continue
		}

		path := relPath(cwd, diag.Location.File)

		fileMap[path] = append(fileMap[path], diag)
	}

	diagnosticCount := 0

	files := slices.Sorted(maps.Keys(fileMap))

	for _, file := range files {
		_, err = fmt.Fprintf(os.Stdout, " \033[36m%s\033[0m\n", file)
		if err != nil {
			return 0, fmt.Errorf("write file header: %w", err)
		}

		diags := fileMap[file]
		diagnosticCount += len(diags)

		slices.SortFunc(diags, func(diagA, diagB Diagnostic) int {
			return cmp.Or(
				cmp.Compare(diagA.Location.Line, diagB.Location.Line),
				cmp.Compare(diagA.Location.Column, diagB.Location.Column),
				cmp.Compare(diagA.Code, diagB.Code),
				cmp.Compare(diagA.Message, diagB.Message),
			)
		})

		var maxLoc int

		for _, diag := range diags {
			loc := fmt.Sprintf("%d:%d:", diag.Location.Line, diag.Location.Column)

			maxLoc = max(maxLoc, len(loc))
		}

		for _, diag := range diags {
			loc := fmt.Sprintf("%d:%d:", diag.Location.Line, diag.Location.Column)

			_, err = fmt.Fprintf(os.Stdout, "   \033[97m%-*s\033[0m  \033[3m%s\033[0m \033[90m(%s)\033[0m\n", maxLoc, loc, diag.Message, diag.Code)
			if err != nil {
				return 0, fmt.Errorf("write diagnostic: %w", err)
			}

			for _, related := range diag.Related {
				rf := relPath(cwd, related.Location.File)

				var loc string

				if rf == file {
					loc = fmt.Sprintf("%d:%d:", related.Location.Line, related.Location.Column)
				} else {
					loc = fmt.Sprintf("%s:%d:%d:", rf, related.Location.Line, related.Location.Column)
				}

				_, err = fmt.Fprintf(os.Stdout, "      \033[97m→ %s\033[0m  \033[3m%s\033[0m\n", loc, related.Message)
				if err != nil {
					return 0, fmt.Errorf("write related diagnostic: %w", err)
				}
			}
		}
	}

	return diagnosticCount, nil
}

func isGeneratedFile(path string) bool {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		return false
	}

	return ast.IsGenerated(file)
}

func decodeDiagnostics(out []byte) ([]Diagnostic, error) {
	dec := json.NewDecoder(bytes.NewReader(out))

	var diags []Diagnostic

	for {
		var d Diagnostic

		err := dec.Decode(&d)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("decode diagnostic stream: %w", err)
		}

		diags = append(diags, d)
	}

	return diags, nil
}

func looksLikeJSONStream(b []byte) bool {
	b = bytes.TrimSpace(b)

	return len(b) > 0 && b[0] == '{'
}

func relPath(cwd, path string) string {
	if cwd == "" {
		return path
	}

	relative, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}

	parentPrefix := ".." + string(os.PathSeparator)
	outside := relative == ".." || strings.HasPrefix(relative, parentPrefix)

	if outside {
		return path
	}

	return relative
}
