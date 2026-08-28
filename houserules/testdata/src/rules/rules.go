package rules

type namedCase struct {
	Name string
}

var packageFirst int
var packageSecond int // want `consecutive var declarations`

func violations(lookup map[string]int) {
	const width = 0.25 // want `function-local const`

	type local struct{ Name string } // want `function-local type`

	var payload struct { // want `anonymous struct type`
		Name string
	}

	var localFirst int
	var localSecond int // want `consecutive var declarations`

	wantMain, wantOffhand := int32(3), int32(4) // want `chained assignment`

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
	_ = localFirst
	_ = localSecond
	_ = wantMain
	_ = wantOffhand
}

func allowed(lookup map[string]int, value any) {
	if count, ok := lookup["key"]; ok {
		_ = count
	}

	if text, ok := value.(string); ok {
		_ = text
	}

	set := map[string]struct{}{"first": {}}
	first, second := pair()

	var (
		third  int
		fourth int
	)

	_ = set
	_ = first
	_ = second
	_ = third
	_ = fourth
}

func work() error {
	return nil
}

func pair() (int, int) {
	return 0, 0
}
