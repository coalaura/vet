package breathe

func singleVarBoundaries() {
	work()
	var buffer [12]byte // want `missing blank line before var declaration`
	_ = buffer          // want `missing blank line after var declaration`
}

func initializedVarBoundaries() {
	work()
	var value = something() // want `missing blank line before var declaration`
	_ = value               // want `missing blank line after var declaration`
}

func allowedSingleVarBoundaries() {
	var _ [12]byte
}

func allowedSeparatedSingleVar() {
	work()

	var buffer [12]byte

	_ = buffer

	var _ int
}

func allowedCommentBoundaries() {
	work()
	// Keep the existing comment-as-separation convention.
	var buffer [12]byte
	// This separates the declaration from its use too.
	_ = buffer
}

func groupableDeclarationsUseHouseRules() {
	var buffer [8]byte
	something := false
	var counter int
	value := ""

	_ = buffer
	_ = something
	_ = counter
	_ = value
}

func consecutiveVarsUseHouseRules() {
	var first int
	var second = something()

	_ = first
	_ = second
}

func singleVarBeforeFeeder() {
	var _ int
	value := something() // want `missing blank line after var declaration`
	if value > 0 {
		work()
	}
}

func singleVarBeforeMultipleFeeders() {
	var _ int
	first := something() // want `missing blank line after var declaration`
	second := something()

	if first > second {
		work()
	}
}

func varClauseBoundaries(value int, channel chan int) {
	switch value {
	case 0:
		work()
		var _ int // want `missing blank line before var declaration`
	default:
		var _ int
		work() // want `missing blank line after var declaration`
	}

	select {
	case <-channel:
		work()
		var _ int // want `missing blank line before var declaration`
	default:
		var _ int
	}
}

func nonVarDeclarations() {
	work()
	const value = 1
	type count int
	_ = count(value)
}

func zeroDeclarationsBeforeFeederUseHouseRules() {
	first := 0
	second := false
	if second {
		work()
	}

	_ = first
}

func zeroDeclarationsBeforeMultipleFeedersUseHouseRules() {
	first := 0
	second := 0
	third := something()

	if second > third {
		work()
	}

	_ = first
}

func explicitInitializersUseHouseRuleGrouping() {
	first := 0
	var boxed any = 0
	second := false
	var structure any = request{}
	third := ""
	var result = work()
	fourth := 0

	_ = first
	_ = boxed
	_ = second
	_ = structure
	_ = third
	_ = result
	_ = fourth
}

func explicitFunctionInitializerUsesHouseRuleGrouping() {
	first := 0
	var callback = func() {
		work()
	}
	second := false

	_ = first
	_ = callback
	_ = second
}
