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

`./vet ./...`

You can pass package patterns and the options shown by `vet --help`. The tool forces JSON output internally and prints compact lines.

## Exit codes

- `0` no issues
- `1` diagnostics found
- `2` runtime or setup error
