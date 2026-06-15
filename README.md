# vet

A small CLI wrapper for Go analyzers.

It runs:
- go vet analyzers
- staticcheck, stylecheck and unused
- modernize analyzers

Output is rendered as short, readable diagnostics.

## Build

`go build -o vet .`

## Usage

`./vet ./...`

You can pass normal package patterns and analyzer args. The tool forces JSON output internally and prints compact lines.

## Exit codes

- `0` no issues
- `1` diagnostics found
- `2` runtime or setup error
