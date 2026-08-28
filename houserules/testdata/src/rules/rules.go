package rules

type namedCase struct {
	Name string
}

func violations(lookup map[string]int) {
	const width = 0.25 // want `function-local const`

	type local struct{ Name string } // want `function-local type`

	var payload struct { // want `anonymous struct type`
		Name string
	}

	if err := work(); err != nil { // want `initializer in if statement`
		_ = err
	}

	switch mode := width; mode { // want `initializer in switch statement`
	default:
	}

	for name := range map[string]namedCase{"first": {}} { // want `composite literal in range position`
		_ = name
	}

	_ = payload
	_ = lookup
}

func allowed(lookup map[string]int, value any) {
	if count, ok := lookup["key"]; ok {
		_ = count
	}

	if text, ok := value.(string); ok {
		_ = text
	}

	set := map[string]struct{}{"first": {}}

	_ = set
}

func work() error {
	return nil
}
