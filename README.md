# vet

A small heavily opinionated CLI wrapper for Go analyzers.

It runs:

- Go vet and additional x/tools correctness analyzers
- staticcheck, stylecheck, quickfix, and unused
- modernize analyzers
- error-handling, response-body, and duration checks
- repository house rules

Output is rendered as short, readable diagnostics.

## Build

`go build -o vet .`

## Usage

`./vet`

Analyze a different build target:

`./vet --os windows --arch amd64 ./...`

The `--os` and `--arch` options default to the operating system and architecture on which `vet` is running. They select one build target per invocation, including its build constraints and platform-specific filename suffixes.

CGO is disabled by default. Enable it with `./vet --cgo`; vet configures Zig as the C and C++ compiler for the selected target.

With no package patterns, vet analyzes `./...`. You can pass explicit package patterns and the options shown by `vet --help`. The tool forces JSON output internally and prints compact lines.

## Exit codes

- `0` no issues
- `1` diagnostics found
- `2` runtime or setup error
