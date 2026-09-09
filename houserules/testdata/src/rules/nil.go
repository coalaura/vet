package rules

import "unsafe"

type nilPointer *int

func typedNilDeclarations() {
	pointer := (*int)(nil)
	slice := []int(nil)                  // want `consecutive zero-value declarations`
	mapping := map[string]int(nil)       // want `consecutive zero-value declarations`
	channel := (chan int)(nil)           // want `consecutive zero-value declarations`
	function := (func())(nil)            // want `consecutive zero-value declarations`
	interfaced := any(nil)               // want `consecutive zero-value declarations`
	nested := any(error(nil))            // want `consecutive zero-value declarations`
	named := nilPointer((*int)(nil))     // want `consecutive zero-value declarations`
	unsafePointer := unsafe.Pointer(nil) // want `consecutive zero-value declarations`

	_ = pointer
	_ = slice
	_ = mapping
	_ = channel
	_ = function
	_ = interfaced
	_ = nested
	_ = named
	_ = unsafePointer
}

func boxedTypedNilsAreNotZero() {
	pointer := (*int)(nil)
	boxedPointer := any((*int)(nil))
	slice := []int(nil)
	boxedSlice := any([]int(nil))
	mapping := map[string]int(nil)
	boxedMap := any(map[string]int(nil))
	channel := (chan int)(nil)
	boxedChannel := any((chan int)(nil))
	function := (func())(nil)
	boxedFunction := any((func())(nil))

	_ = pointer
	_ = boxedPointer
	_ = slice
	_ = boxedSlice
	_ = mapping
	_ = boxedMap
	_ = channel
	_ = boxedChannel
	_ = function
	_ = boxedFunction
}

func nilFunctionCallDoesNotGroup(convert func(*int) *int) {
	first := (*int)(nil)
	second := convert(nil)
	third := []int(nil)

	_ = first
	_ = second
	_ = third
}

func shadowedNilDoesNotGroup(nil *int) {
	first := []int{}
	second := (*int)(nil)
	third := false

	_ = first
	_ = second
	_ = third
}

func genericBoxedNilDoesNotGroup[Pointer ~*int]() {
	first := 0
	second := any(Pointer(nil))
	third := false

	_ = first
	_ = second
	_ = third
}
