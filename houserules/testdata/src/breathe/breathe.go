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

func work() error {
	return nil
}
