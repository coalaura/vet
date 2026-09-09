package rules

func forwardGoto() {
	goto done // want `goto is not allowed`

done:
	return
}

func backwardGoto(ready bool) {
start:
	work()

	if ready {
		goto start // want `goto is not allowed`
	}
}

func nestedGotos(value int, channel chan int) {
	switch value {
	case 0:
		goto done // want `goto is not allowed`
	}

	select {
	case <-channel:
		goto done // want `goto is not allowed`
	default:
	}

done:
	return
}

func functionLiteralGoto() {
	_ = func() {
		goto done // want `goto is not allowed`

	done:
		return
	}
}

func allowedOtherBranches(value int) {
outer:
	for {
		switch value {
		case 0:
			continue outer
		case 1:
			fallthrough
		default:
			break outer
		}
	}
}
