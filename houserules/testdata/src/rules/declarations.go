package rules

var packageLeft, packageRight = 1, 2 // want `multiple variables in declaration`

var (
	packageReady, packageDone               bool // want `multiple variables in declaration`
	packageFirstResult, packageSecondResult = pair()
)

func independentVarInitializers() {
	var first, second = 1, 2 // want `multiple variables in declaration`

	var (
		third, fourth = 3, 4 // want `multiple variables in declaration`
		fifth, sixth  int    // want `multiple variables in declaration`
	)

	_ = first
	_ = second
	_ = third
	_ = fourth
	_ = fifth
	_ = sixth
}

func allowedSingleExpressionDeclarations(values map[string]int, input any) {
	var first, second = pair()

	var value, found = values["key"]

	var number, valid = input.(int)

	_ = first
	_ = second
	_ = value
	_ = found
	_ = number
	_ = valid
}

func allowedParenthesizedCommaOK(values map[string]int, input any) {
	if value, ok := (values["key"]); ok {
		_ = value
	}

	if value, ok := (input.(int)); ok {
		_ = value
	}
}

func parenthesizedOrdinaryInitializer() {
	if err := (work()); err != nil { // want `initializer in if statement`
		_ = err
	}
}

func allowedIntentionalSwaps(values []int, firstIndex, secondIndex int) {
	values[0], values[1] = values[1], values[0]
	values[(firstIndex)], values[secondIndex+1] = values[(secondIndex+1)], values[firstIndex]
	values[int(firstIndex)], values[secondIndex] = values[secondIndex], values[int(firstIndex)]
}

func nonSwapAssignments(first, second int) {
	first, second = second, 1     // want `chained assignment`
	first, second = first, second // want `chained assignment`
	first, first = first, first   // want `chained assignment`
	_, second = second, first     // want `chained assignment`
}

func unstableSwapTargets(values []int, next func() int, indices chan int) {
	values[next()], values[0] = values[0], values[next()]       // want `chained assignment`
	values[<-indices], values[0] = values[0], values[<-indices] // want `chained assignment`
}
