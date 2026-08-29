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
