package breathe

type feederState struct {
	Count int
	Other int
}

func unrelatedFieldCondition(state feederState, Count int) {
	Count = 1
	if state.Count > 0 { // want `missing blank line before control-flow block`
		work()
	}
}

func unrelatedFieldReturn(state feederState, Count int) int {
	Count = 1
	return state.Count // want `missing blank line before return`
}

func allowedFieldReturn(state feederState) int {
	(state).Count = 1
	return state.Count
}

func allowedIndexReturn(values []int, index int) int {
	values[index+1] = 1
	return values[(index + 1)]
}

func allowedMapReturn(values map[string]int) int {
	values["key"] = 1
	return values["key"]
}

func allowedPointerReturn(value *int) int {
	*value = 1
	return *value
}

func differentIndexReturn(values []int) int {
	values[0] = 1
	return values[1] // want `missing blank line before return`
}

func differentFieldReturn(state feederState) int {
	state.Count = 1
	return state.Other // want `missing blank line before return`
}

func shadowedCondition(ready bool, values map[string]bool) {
	ready = false
	if ready, ok := values["key"]; ok && ready { // want `missing blank line before control-flow block`
		work()
	}
}

func overwrittenCondition(ready, ok bool, values map[string]bool) {
	ready = false
	if ready, ok = values["key"]; ok && ready { // want `missing blank line before control-flow block`
		work()
	}
}

func overwrittenReceiverCondition(state feederState, ok bool, values map[string]feederState) {
	state.Count = 1
	if state, ok = values["key"]; ok && state.Count > 0 { // want `missing blank line before control-flow block`
		work()
	}
}

func overwrittenIndexCondition(index int, ok bool, values []int, indices map[string]int) {
	values[index] = 1
	if index, ok = indices["key"]; ok && values[index] > 0 { // want `missing blank line before control-flow block`
		work()
	}
}

func allowedInitializerDestinationDependency(index int, ok bool, values []int, mapping map[string]int) {
	index = 1
	if values[index], ok = mapping["key"]; ok {
		work()
	}
}

func shadowedReceiverCondition(state feederState, values map[string]feederState) {
	state.Count = 1
	if state, ok := values["key"]; ok && state.Other > 0 { // want `missing blank line before control-flow block`
		work()
	}
}

func allowedInitializerDependency(key string, values map[string]bool) {
	key = "key"
	if ready, ok := (values[key]); ok && ready {
		work()
	}
}

func allowedRelatedReceiverCondition(state feederState) {
	state.Count = 1
	if state.Other > 0 {
		work()
	}
}

func allowedIncrementCondition(index int, input []int) {
	index++
	if index == len(input) {
		work()
	}
}

func unrelatedIncrementCondition(index, limit int) {
	index++
	if limit > 0 { // want `missing blank line before control-flow block`
		work()
	}
}

func allowedSeparatedFeeder() {
	work()
	value := something()

	if value > 0 {
		work()
	}
}

func functionLiteralBeforeSingleFeeder() {
	callback := func() {}
	value := something() // want `missing blank line after function literal`
	if value > 0 {
		work()
	}

	_ = callback
}

func functionLiteralBeforeMultipleFeeders() {
	callback := func() {}
	first := something() // want `missing blank line after function literal`
	second := something()

	if first > second {
		work()
	}

	_ = callback
}

func functionLiteralAndTwoDistinctBoundaries() {
	callback := func() {}
	first := something() // want `missing blank line after function literal`
	second := something()
	if first > second { // want `missing blank line before control-flow block: multiple statements feed its condition`
		work()
	}

	_ = callback
}

func allowedSeparatedFunctionLiteralFeeder() {
	callback := func() {}

	value := something()
	if value > 0 {
		work()
	}

	_ = callback
}
