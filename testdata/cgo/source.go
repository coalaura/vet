package cgo

/*
static void touch(void *value) {}
static void *pointer;
*/
import "C"

import (
	"sync"
	"unsafe"
)

func Example() int {
	var value C.int

	if number := 1; number > 0 {
		value = C.int(number)
	}
	if value > 0 {
		C.touch(unsafe.Pointer(&value))
	}
	for range 1 {
		C.touch(unsafe.Pointer(&value))
	}
	return int(value)
}

func Typed(lock *sync.Mutex) {
	first := 0
	second := 0

	first, second = second, first

	lock.Lock()
	defer lock.Unlock()
	println(first, second)
}

func ErrorBoundary() error {
	err := error(nil)

	if err != nil {
		return err
	}

	return nil
}

func CPointer() (any, int) {
	value := any(C.pointer)
	count := 0

	if C.pointer != nil {
		return value, count
	}

	return nil, 0
}
