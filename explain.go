package main

import (
	"go/scanner"
	"go/token"
	"io"
	"strings"
)

const (
	ansiReset       = "\x1b[0m"
	ansiBoldCyan    = "\x1b[1;36m"
	ansiBoldRed     = "\x1b[1;31m"
	ansiBoldGreen   = "\x1b[1;32m"
	ansiCyan        = "\x1b[36m"
	ansiGreen       = "\x1b[32m"
	ansiYellow      = "\x1b[33m"
	ansiMagenta     = "\x1b[35m"
	ansiBrightRed   = "\x1b[91m"
	ansiFunction    = "\x1b[3;94m"
	ansiNamedType   = "\x1b[3;33m"
	ansiBrightBlack = "\x1b[90m"
)

type explainPage struct {
	name        string
	description string
	sections    []explainSection
}

type explainSection struct {
	title       string
	description string
	bad         string
	good        string
}

var houseRulesExplainPage = explainPage{
	name:        "House Rules",
	description: "Project-specific Go conventions that keep declarations and control flow explicit.",
	sections: []explainSection{
		{
			title:       "Assign Before Control Flow",
			description: "Assign values before if, switch and type switch statements. Two-value map lookups and type assertions in if statements are allowed.",
			bad: code(
				"if err := run(); err != nil {",
				"\treturn err",
				"}",
			),
			good: code(
				"err := run()",
				"if err != nil {",
				"\treturn err",
				"}",
			),
		},
		{
			title:       "No Goto",
			description: "Every goto statement is disallowed. Use structured control flow such as loops, conditionals and returns. Labeled break and continue remain allowed.",
			bad: code(
				"retry:",
				"work()",
				"",
				"if retryNeeded() {",
				"\tgoto retry",
				"}",
			),
			good: code(
				"for {",
				"\twork()",
				"",
				"\tif !retryNeeded() {",
				"\t\tbreak",
				"\t}",
				"}",
			),
		},
		{
			title:       "One Assignment Per Target",
			description: "When assigning separate values, give each target its own statement. Var declarations, including specs inside var blocks, use one variable per specification. Multiple results from one expression and intentional two-target swaps with stable target expressions remain allowed. Preserve evaluation order and original values when splitting assignments; use temporaries when necessary.",
			bad:         "first, second := int32(3), int32(4)",
			good: code(
				"first := int32(3)",
				"second := int32(4)",
			),
		},
		{
			title:       "Name Values Before Range",
			description: "Assign composite literals to a descriptive name before ranging over them.",
			bad: code(
				`for name := range map[string]entry{"first": {}} {`,
				"\tuse(name)",
				"}",
			),
			good: code(
				`entries := map[string]entry{"first": {}}`,
				"",
				"for name := range entries {",
				"\tuse(name)",
				"}",
			),
		},
		{
			title:       "Declare Types And Constants At Package Scope",
			description: "Keep constants and types out of functions and give struct types a package-scope name.",
			bad: code(
				"func run() {",
				"\tconst timeout = 5",
				"\ttype result struct{ value string }",
				"\tvar payload struct{ value string }",
				"}",
			),
			good: code(
				"const timeout = 5",
				"",
				"type result struct {",
				"\tvalue string",
				"}",
				"",
				"func run() {",
				"\tvar payload result",
				"}",
			),
		},
		{
			title:       "Group Consecutive Variables",
			description: "Combine adjacent explicit var declarations and zero-value short declarations into one var block. Preserve explicit var types, initializers and evaluation order, even for non-zero values or runtime calls. Short declarations qualify only when every target is new and every initializer is provably zero, including typed nil conversions. An interface containing a typed nil pointer is not zero.",
			bad: code(
				"var buffer [8]byte",
				"ready := false",
				"var boxed any = 0",
				"value := \"\"",
			),
			good: code(
				"var (",
				"\tbuffer [8]byte",
				"\tready  bool",
				"\tboxed  any = 0",
				"\tvalue  string",
				")",
			),
		},
	},
}

var breatheExplainPage = explainPage{
	name:        "Breathe",
	description: "Blank lines or intervening comment lines separate setup, control flow, function literals, returns, branches, var declarations and mutex sections. Each statement boundary gets one prioritized spacing diagnostic; ordinary section boundaries take precedence over feeder-specific spacing.",
	sections: []explainSection{
		{
			title:       "Before Control Flow",
			description: "Always separate for, switch and select blocks from preceding work. This does not apply to if.",
			bad: code(
				"amount := total",
				"for i := range amount {",
				"\tcount += i",
				"}",
			),
			good: code(
				"amount := total",
				"",
				"for i := range amount {",
				"\tcount += i",
				"}",
			),
		},
		{
			title:       "Single If Feeder",
			description: "An if may sit directly below one assignment feeding its condition, keeping it connected. A blank line is also allowed. Field assignments may accompany conditions using related fields or methods of the same receiver.",
			bad: code(
				"work()",
				"ready := isReady()",
				"if ready {",
				"\ttotal++",
				"}",
			),
			good: code(
				"work()",
				"",
				"ready := isReady()",
				"if ready {",
				"\ttotal++",
				"}",
			),
		},
		{
			title:       "Multiple Introductions",
			description: "When several statements feed a condition, separate the group from surrounding work and from the control-flow block.",
			bad: code(
				"work()",
				"first := left()",
				"second := right()",
				"if first < second {",
				"\twork()",
				"}",
			),
			good: code(
				"work()",
				"",
				"first := left()",
				"second := right()",
				"",
				"if first < second {",
				"\twork()",
				"}",
			),
		},
		{
			title:       "After Control Flow",
			description: "Always leave a blank line after for, switch and select blocks. Multi-line if blocks require one too.",
			bad: code(
				"for ready() {",
				"\twork()",
				"}",
				"finish()",
			),
			good: code(
				"for ready() {",
				"\twork()",
				"}",
				"",
				"finish()",
			),
		},
		{
			title:       "Before Returns And Branches",
			description: "Separate return, break and continue from preceding work. An assignment feeding a return value may sit directly above it.",
			bad: code(
				"work()",
				"return nil",
			),
			good: code(
				"work()",
				"",
				"return nil",
			),
		},
		{
			title:       "Around Var Declarations",
			description: "Separate every var declaration, including single declarations and initialized declarations, from statements on both sides. Block and clause boundaries need no extra blank line. Adjacent groupable declarations are handled by the house rule instead.",
			bad: code(
				"value := calculate()",
				"var buffer [12]byte",
				"use(value)",
			),
			good: code(
				"value := calculate()",
				"",
				"var buffer [12]byte",
				"",
				"use(value)",
			),
		},
		{
			title:       "Around Mutex Sections",
			description: "Separate synchronous sync.Mutex, sync.RWMutex and sync.Locker lock calls from preceding work, and unlock calls from following work. A single lock or unlock may touch the protected statements.",
			bad: code(
				"prepare()",
				"mutex.Lock()",
				"work()",
				"mutex.Unlock()",
				"finish()",
			),
			good: code(
				"prepare()",
				"",
				"mutex.Lock()",
				"work()",
				"mutex.Unlock()",
				"",
				"finish()",
			),
		},
		{
			title:       "After Deferred Mutex Unlocks",
			description: "A lock and its deferred unlock may stay together, but separate the deferred unlock from following work. Consecutive deferred unlocks may form one group with separation after the last defer. Block and clause boundaries need no extra blank line.",
			bad: code(
				"mutex.Lock()",
				"defer mutex.Unlock()",
				"work()",
			),
			good: code(
				"mutex.Lock()",
				"defer mutex.Unlock()",
				"",
				"work()",
			),
		},
		{
			title:       "Around Mutex Groups",
			description: "Consecutive locks or consecutive unlocks form a group. Separate groups of two or more calls on both sides, including from the protected statements. Block and clause boundaries need no extra blank line.",
			bad: code(
				"first.Lock()",
				"second.RLock()",
				"work()",
				"second.RUnlock()",
				"first.Unlock()",
			),
			good: code(
				"first.Lock()",
				"second.RLock()",
				"",
				"work()",
				"",
				"second.RUnlock()",
				"first.Unlock()",
			),
		},
		{
			title:       "Around Function Literals",
			description: "Non-empty function literal bodies and the statements containing them need room to breathe. Empty bodies may remain on one line.",
			bad: code(
				"ready := check(func(Item) bool { return true })",
				"if ready {",
				"\twork()",
				"}",
			),
			good: code(
				"ready := check(func(Item) bool {",
				"\treturn true",
				"})",
				"",
				"if ready {",
				"\twork()",
				"}",
			),
		},
		{
			title:       "Outside Conditions",
			description: "Assign a function literal before using it in an if or for condition.",
			bad: code(
				"if check(func(Item) bool {",
				"\treturn true",
				"}) {",
				"\twork()",
				"}",
			),
			good: code(
				"accept := func(Item) bool {",
				"\treturn true",
				"}",
				"",
				"if check(accept) {",
				"\twork()",
				"}",
			),
		},
	},
}

func writeAnalyzerHelp(writer io.Writer, analyzer string) (bool, error) {
	var page explainPage

	switch analyzer {
	case "houserules":
		page = houseRulesExplainPage
	case "breathe":
		page = breatheExplainPage
	default:
		return false, nil
	}

	_, err := io.WriteString(writer, renderExplainPage(page))

	return true, err
}

func code(lines ...string) string {
	return strings.Join(lines, "\n")
}

func renderExplainPage(page explainPage) string {
	var output strings.Builder

	writeStyled(&output, ansiBoldCyan, strings.ToUpper(page.name))
	output.WriteString("\n\n")
	output.WriteString(page.description)
	output.WriteString("\n")

	for _, section := range page.sections {
		output.WriteString("\n")
		writeStyled(&output, ansiBoldCyan, strings.ToUpper(section.title))
		output.WriteString("\n\n")
		output.WriteString(section.description)
		output.WriteString("\n\n")
		writeStyled(&output, ansiBoldRed, "BAD")
		output.WriteString("\n")
		writeCode(&output, section.bad)
		output.WriteString("\n")
		writeStyled(&output, ansiBoldGreen, "GOOD")
		output.WriteString("\n")
		writeCode(&output, section.good)
	}

	return output.String()
}

func writeCode(output *strings.Builder, source string) {
	highlighted := highlightGo(source)

	for line := range strings.SplitSeq(highlighted, "\n") {
		output.WriteString("  ")
		output.WriteString(line)
		output.WriteString("\n")
	}
}

func highlightGo(source string) string {
	fileSet := token.NewFileSet()

	file := fileSet.AddFile("example.go", -1, len(source))

	var lexer scanner.Scanner

	lexer.Init(file, []byte(source), nil, scanner.ScanComments)

	var (
		output strings.Builder

		previousOffset int
		declaredTypes  = make(map[string]bool)
		previousToken  = token.ILLEGAL
	)

	for {
		position, scannedToken, literal := lexer.Scan()
		if scannedToken == token.EOF {
			break
		}

		if scannedToken == token.SEMICOLON && literal == "\n" {
			continue
		}

		if literal == "" {
			literal = scannedToken.String()
		}

		offset := file.Offset(position)

		if scannedToken == token.IDENT && previousToken == token.TYPE {
			declaredTypes[literal] = true
		}

		output.WriteString(source[previousOffset:offset])

		style := tokenStyle(source, offset, literal, scannedToken, previousToken, declaredTypes)

		writeStyled(&output, style, literal)

		previousOffset = offset + len(literal)
		previousToken = scannedToken
	}

	output.WriteString(source[previousOffset:])

	return output.String()
}

func tokenStyle(source string, offset int, literal string, scannedToken, previousToken token.Token, declaredTypes map[string]bool) string {
	switch {
	case scannedToken.IsKeyword():
		return ansiMagenta
	case isPredeclaredValue(literal, scannedToken):
		return ansiBrightRed
	case isBuiltinType(literal, scannedToken):
		return ansiMagenta
	case declaredTypes[literal] || isTypeLiteral(source, offset+len(literal), scannedToken):
		return ansiNamedType
	case previousToken == token.CONST && scannedToken == token.IDENT:
		return ansiYellow
	case isFunctionName(source, offset+len(literal), scannedToken):
		return ansiFunction
	case scannedToken == token.STRING || scannedToken == token.CHAR:
		return ansiGreen
	case scannedToken == token.INT || scannedToken == token.FLOAT || scannedToken == token.IMAG:
		return ansiYellow
	case scannedToken == token.COMMENT:
		return ansiBrightBlack
	case isOperator(scannedToken):
		return ansiCyan
	case isBracket(scannedToken):
		return ansiBrightRed
	default:
		return ""
	}
}

func isFunctionName(source string, offset int, scannedToken token.Token) bool {
	if scannedToken != token.IDENT {
		return false
	}

	remainder := strings.TrimLeft(source[offset:], " \t\r\n")

	return strings.HasPrefix(remainder, "(")
}

func isTypeLiteral(source string, offset int, scannedToken token.Token) bool {
	if scannedToken != token.IDENT {
		return false
	}

	remainder := strings.TrimLeft(source[offset:], " \t\r\n")

	return strings.HasPrefix(remainder, "{")
}

func isPredeclaredValue(literal string, scannedToken token.Token) bool {
	if scannedToken != token.IDENT {
		return false
	}

	switch literal {
	case "false", "iota", "nil", "true":
		return true
	default:
		return false
	}
}

func isBuiltinType(literal string, scannedToken token.Token) bool {
	if scannedToken != token.IDENT {
		return false
	}

	switch literal {
	case "any", "bool", "byte", "comparable", "complex64", "complex128", "error", "float32", "float64", "int", "int8", "int16", "int32", "int64", "rune", "string", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	default:
		return false
	}
}

func isOperator(scannedToken token.Token) bool {
	switch scannedToken {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM,
		token.AND, token.OR, token.XOR, token.SHL, token.SHR, token.AND_NOT,
		token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN,
		token.AND_ASSIGN, token.OR_ASSIGN, token.XOR_ASSIGN, token.SHL_ASSIGN, token.SHR_ASSIGN, token.AND_NOT_ASSIGN,
		token.LAND, token.LOR, token.ARROW, token.INC, token.DEC,
		token.EQL, token.LSS, token.GTR, token.ASSIGN, token.NOT, token.NEQ, token.LEQ, token.GEQ, token.DEFINE:
		return true
	default:
		return false
	}
}

func isBracket(scannedToken token.Token) bool {
	switch scannedToken {
	case token.LPAREN, token.LBRACK, token.LBRACE, token.RPAREN, token.RBRACK, token.RBRACE:
		return true
	default:
		return false
	}
}

func writeStyled(output *strings.Builder, style, text string) {
	if style == "" {
		output.WriteString(text)

		return
	}

	output.WriteString(style)
	output.WriteString(text)
	output.WriteString(ansiReset)
}
