package rules

type count int

type zeroStruct struct {
	value int
}

func mixedZeroDeclarations() {
	var buffer [8]byte
	something := false // want `consecutive var declarations`
	var counter int    // want `consecutive var declarations`
	value := ""        // want `consecutive var declarations`

	_ = buffer
	_ = something
	_ = counter
	_ = value
}

func shortZeroDeclarations() {
	integer := 0
	floating := 0.0           // want `consecutive zero-value declarations`
	complexValue := 0i        // want `consecutive zero-value declarations`
	runeValue := '\x00'       // want `consecutive zero-value declarations`
	named := count(0)         // want `consecutive zero-value declarations`
	array := [8]byte{}        // want `consecutive zero-value declarations`
	structure := zeroStruct{} // want `consecutive zero-value declarations`

	_ = integer
	_ = floating
	_ = complexValue
	_ = runeValue
	_ = named
	_ = array
	_ = structure
}

func explicitZeroInitializers() {
	var pointer *int = nil
	ready := false      // want `consecutive var declarations`
	var integer int = 0 // want `consecutive var declarations`
	text := ""          // want `consecutive var declarations`

	_ = pointer
	_ = ready
	_ = integer
	_ = text
}

func allowedGroupedZeros() {
	var (
		buffer    [8]byte
		something bool
		counter   int
		value     string
	)

	_ = buffer
	_ = something
	_ = counter
	_ = value
}

func separatedZeroDeclarations() {
	first := 0

	second := false
	// A comment is an intentional boundary too.
	third := ""

	_ = first
	_ = second
	_ = third
}

func nonZeroInitializers() {
	integer := 1
	ready := false
	imaginary := 1i
	realPart := complex(1, 0)
	text := ""
	slice := []int{}
	array := [1]int{1}
	structure := zeroStruct{value: 1}
	mapping := map[string]int{}
	pointer := &zeroStruct{}
	value := 0

	_ = integer
	_ = ready
	_ = imaginary
	_ = realPart
	_ = text
	_ = slice
	_ = array
	_ = structure
	_ = mapping
	_ = pointer
	_ = value
}

func explicitInterfaceValuesStillGroup() {
	first := 0
	var boxed any = 0                // want `consecutive var declarations`
	second := false                  // want `consecutive var declarations`
	var structure any = zeroStruct{} // want `consecutive var declarations`
	third := ""                      // want `consecutive var declarations`

	_ = first
	_ = boxed
	_ = second
	_ = structure
	_ = third
}

func assignmentsAndCallsAreNotDeclarations() {
	first := 0
	first = 0
	second := false
	third := work()
	fourth := ""

	_ = first
	_ = second
	_ = third
	_ = fourth
}

func shadowedFalseIsNotConstant(false bool) {
	first := 0
	second := false
	third := ""

	_ = first
	_ = second
	_ = third
}

func partialRedeclaration() {
	first := 0
	first, second := 0, false // want `chained assignment`
	third := ""

	_ = first
	_ = second
	_ = third
}

func declarationsWithDependencies() {
	first := 0
	second := first
	third := false

	_ = first
	_ = second
	_ = third
}

func zeroClauseDeclarations(value int, channel chan int) {
	switch value {
	case 0:
		first := 0
		second := false // want `consecutive zero-value declarations`
		_ = first
		_ = second
	default:
		var _ int
	}

	select {
	case <-channel:
		first := ""
		var second int // want `consecutive var declarations`
		_ = first
		_ = second
	default:
		var _ int
	}
}

func labeledDeclarations(ready bool) {
	var first int
start:
	var second int
	third := false

	_ = first
	_ = second
	_ = third

	if ready {
		goto start // want `goto is not allowed`
	}
}

func nonZeroVarGroupsWithZeroShortDeclaration() {
	var first = 1
	second := false    // want `consecutive var declarations`
	var third = work() // want `consecutive var declarations`
	fourth := ""       // want `consecutive var declarations`

	_ = first
	_ = second
	_ = third
	_ = fourth
}

func explicitTypedNilInterfaceStillGroups() {
	first := 0
	var pointer any = (*int)(nil) // want `consecutive var declarations`
	second := false               // want `consecutive var declarations`

	_ = first
	_ = pointer
	_ = second
}

func groupedVarAndShortDeclaration() {
	var (
		first  int
		second bool
	)
	third := "" // want `consecutive var declarations`

	_ = first
	_ = second
	_ = third
}

func emptyVarBlock() {
	var ()
	first := 0

	_ = first
}

func allowedGroupedInterfaceValues() {
	var (
		first     int
		boxed     any = 0
		second    bool
		structure any = zeroStruct{}
		third     string
	)

	_ = first
	_ = boxed
	_ = second
	_ = structure
	_ = third
}

func shortInterfaceValuesAreNotZero() {
	first := 0
	boxed := any(0)
	second := false
	structure := any(zeroStruct{})
	third := ""
	pointer := any((*int)(nil))
	fourth := 0

	_ = first
	_ = boxed
	_ = second
	_ = structure
	_ = third
	_ = pointer
	_ = fourth
}

func explicitVarDoesNotMakeNonZeroShortsGroupable() {
	var first int
	second := 1
	var third bool
	fourth := work()
	var fifth string

	_ = first
	_ = second
	_ = third
	_ = fourth
	_ = fifth
}

func explicitVarsPreserveInitializerOrder() {
	first := 0
	var second = first + 1 // want `consecutive var declarations`
	third := false         // want `consecutive var declarations`
	var fourth = work()    // want `consecutive var declarations`
	fifth := ""            // want `consecutive var declarations`

	_ = second
	_ = third
	_ = fourth
	_ = fifth
}

func explicitFunctionInitializerStillGroups() {
	first := 0
	var callback = func() { // want `consecutive var declarations`
		work()
	}
	second := false // want `consecutive var declarations`

	_ = first
	_ = callback
	_ = second
}
