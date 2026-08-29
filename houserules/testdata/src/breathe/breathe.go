package breathe

func violations(ready bool, count int) {
	if ready {
		count++
	}
	if count > 0 { // want `missing blank line after control-flow block`
		count--
	}

	total := 0
	if ready { // want `missing blank line before control-flow block`
		total++
	}

	_ = total
}

func allowed(items []int) error {
	err := work()
	if err != nil {
		return err
	}

	sum := 0

	for _, item := range items {
		sum += item
	}

	_ = sum

	return nil
}

func unrelatedStatementBeforeIntroduction() error {
	stuff := something()
	example, err := thing() // want `missing blank line before statement feeding control-flow block`
	if err != nil {
		return err
	}

	_ = stuff
	_ = example

	return nil
}

func multipleIntroductions() {
	exampleA := something()
	exampleB := something()
	if exampleA > 0 && exampleB != exampleA { // want `missing blank line before control-flow block: multiple statements feed its condition`
		work()
	}
}

func invadedMultipleIntroductions() {
	work()
	exampleA := something() // want `missing blank line before statements feeding control-flow block`
	exampleB := something()
	exampleC := something()

	if exampleA > 0 && exampleB != exampleA && exampleC != exampleB {
		work()
	}
}

func allowedIntroductions() error {
	stuff := something()

	example, err := thing()
	if err != nil {
		return err
	}

	_ = stuff
	_ = example

	exampleA := something()
	exampleB := something()

	if exampleA > 0 && exampleB != exampleA {
		work()
	}

	return nil
}

func breakViolations() {
	for {
		work()
		break // want `missing blank line before break`
	}
}

func sameLineBreakViolation() {
	for {
		work()
		break // want `missing blank line before break`
	}
}

func continueViolations() {
	for {
		work()
		continue // want `missing blank line before continue`
	}
}

func sameLineContinueViolation() {
	for {
		work()
		continue // want `missing blank line before continue`
	}
}

func returnViolations() error {
	work()
	return nil // want `missing blank line before return`
}

func varBlockViolations() {
	value := 1
	var ( // want `missing blank line before var block`
		_ int
	)
	_ = value // want `missing blank line after var block`
}

func allowedBoundaries() error {
	var (
		_ int
	)

	for {
		break
	}

	for {
		continue
	}

	err := work()
	return err
}

func allowedTrailingVarBlock() {
	work()

	var (
		_ int
	)
}

func work() error {
	return nil
}

func something() int {
	return 0
}

func thing() (int, error) {
	return 0, nil
}
