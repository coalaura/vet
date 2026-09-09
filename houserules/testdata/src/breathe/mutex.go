package breathe

import "sync"

type unrelatedLock struct{}

type embeddedMutex struct {
	sync.Mutex
}

type mutexAlias = sync.Mutex

type functionLock struct {
	Lock   func()
	Unlock func()
}

func (unrelatedLock) Lock()   {}
func (unrelatedLock) Unlock() {}

func singleMutexViolations(mutex *sync.Mutex) {
	work()
	mutex.Lock() // want `missing blank line before mutex lock`
	work()
	mutex.Unlock()
	work() // want `missing blank line after mutex unlock`
}

func allowedSingleMutex(mutex *sync.Mutex, ready bool) {
	mutex.Lock()
	work()
	mutex.Unlock()

	if ready {
		mutex.Lock()
		work()
		mutex.Unlock()
	}
}

func groupedMutexViolations(first *sync.Mutex, second *sync.RWMutex) {
	work()
	first.Lock() // want `missing blank line before mutex lock`
	second.RLock()
	work()           // want `missing blank line after mutex lock group`
	second.RUnlock() // want `missing blank line before mutex unlock group`
	first.Unlock()
	work() // want `missing blank line after mutex unlock`
}

func allowedGroupedMutex(first *sync.Mutex, second *sync.RWMutex) {
	work()

	first.Lock()
	second.RLock()

	work()

	second.RUnlock()
	first.Unlock()

	work()
}

func allowedEmptyCriticalSection(first *sync.Mutex, second *sync.RWMutex) {
	first.Lock()
	first.Unlock()

	first.Lock()
	second.Lock()

	second.Unlock()
	first.Unlock()
}

func separatedLocksAreNotAGroup(first, second *sync.Mutex) {
	first.Lock()

	second.Lock()
	work()
	second.Unlock()

	first.Unlock()
}

func commentSeparatedLocks(first, second *sync.Mutex) {
	first.Lock()
	// The second lock has its own boundary.
	second.Lock()
	work()
	second.Unlock()
	// Release the outer lock.
	first.Unlock()
}

func allowedDeferredUnlock(mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()

	work()
}

func deferredUnlockNeedsTrailingBoundary(mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()
	work() // want `missing blank line after deferred mutex unlock`
}

func deferredUnlockStillNeedsLeadingBoundary(mutex *sync.Mutex) {
	work()
	mutex.Lock() // want `missing blank line before mutex lock`
	defer mutex.Unlock()

	work()
}

func unrelatedLockMethods(lock unrelatedLock) {
	work()
	lock.Lock()
	work()
	lock.Unlock()
	work()
}

func mutexMethodValues(mutex *sync.Mutex) {
	work()
	lock := mutex.Lock
	lock()
	work()
}

func asynchronousMutexCalls(mutex *sync.Mutex) {
	work()
	go mutex.Lock()
	work()
	go mutex.Unlock()
	work()
}

func promotedMutexMethods(mutex *embeddedMutex) {
	work()
	mutex.Lock() // want `missing blank line before mutex lock`
	work()
	mutex.Unlock()
	work() // want `missing blank line after mutex unlock`
}

func aliasedMutexMethods(mutex *mutexAlias) {
	work()
	mutex.Lock() // want `missing blank line before mutex lock`
	work()
	mutex.Unlock()
	work() // want `missing blank line after mutex unlock`
}

func lockerInterfaceMethods(lock sync.Locker) {
	work()
	lock.Lock() // want `missing blank line before mutex lock`
	work()
	lock.Unlock()
	work() // want `missing blank line after mutex unlock`
}

func mutexClauseBoundaries(mutex *sync.RWMutex, value int, channel chan int) {
	switch value {
	case 0:
		work()
		mutex.RLock() // want `missing blank line before mutex lock`
		work()
		mutex.RUnlock()
	default:
		mutex.Lock()
		work()
		mutex.Unlock()
		work() // want `missing blank line after mutex unlock`
	}

	select {
	case <-channel:
		work()
		mutex.Lock() // want `missing blank line before mutex lock`
		work()
		mutex.Unlock()
	default:
		mutex.Lock()
		mutex.Unlock()
	}
}

func adjacentCriticalSections(mutex *sync.Mutex) {
	mutex.Lock()
	mutex.Unlock()
	mutex.Lock() // want `missing blank line before mutex lock`
	mutex.Unlock()
}

func allowedProtectedFeeder(mutex *sync.Mutex) {
	mutex.Lock()
	value := something()
	if value > 0 {
		work()
	}

	mutex.Unlock()
}

func allowedProtectedMultipleFeeders(mutex *sync.Mutex) {
	mutex.Lock()
	first := something()
	second := something()

	if first > second {
		work()
	}

	mutex.Unlock()
}

func protectedFeederAfterGroup(first, second *sync.Mutex) {
	first.Lock()
	second.Lock()
	value := something() // want `missing blank line after mutex lock group`
	if value > 0 {
		work()
	}

	second.Unlock()
	first.Unlock()
}

func mutexMethodExpressions(mutex *sync.Mutex) {
	work()
	(*sync.Mutex).Lock(mutex) // want `missing blank line before mutex lock`
	work()
	(*sync.Mutex).Unlock(mutex)
	work() // want `missing blank line after mutex unlock`
}

func parenthesizedMutexMethods(mutex *sync.Mutex) {
	work()
	(mutex.Lock)() // want `missing blank line before mutex lock`
	work()
	(mutex.Unlock)()
	work() // want `missing blank line after mutex unlock`
}

func mutexFunctionFields(lock functionLock) {
	work()
	lock.Lock()
	work()
	lock.Unlock()
	work()
}

func deferredReadUnlockNeedsTrailingBoundary(mutex *sync.RWMutex) {
	mutex.RLock()
	defer mutex.RUnlock()
	work() // want `missing blank line after deferred mutex unlock`
}

func deferredLockerUnlockNeedsTrailingBoundary(lock sync.Locker) {
	lock.Lock()
	defer lock.Unlock()
	work() // want `missing blank line after deferred mutex unlock`
}

func deferredMethodExpressionNeedsTrailingBoundary(mutex *sync.Mutex) {
	mutex.Lock()
	defer (*sync.Mutex).Unlock(mutex)
	work() // want `missing blank line after deferred mutex unlock`
}

func allowedDeferredUnlockAtBlockEnd(mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()
}

func allowedDeferredUnlockGroup(first *sync.Mutex, second *sync.RWMutex) {
	first.Lock()
	second.RLock()

	defer second.RUnlock()
	defer first.Unlock()

	work()
}

func deferredUnlockGroupNeedsTrailingBoundary(first, second *sync.Mutex) {
	first.Lock()
	second.Lock()

	defer second.Unlock()
	defer first.Unlock()
	work() // want `missing blank line after deferred mutex unlock`
}

func deferredUnlockBeforeFeeder(mutex *sync.Mutex) {
	mutex.Lock()
	defer mutex.Unlock()
	value := something() // want `missing blank line after deferred mutex unlock`
	if value > 0 {
		work()
	}
}

func unrelatedDeferredUnlock(lock unrelatedLock, fields functionLock) {
	defer lock.Unlock()
	work()
	defer fields.Unlock()
	work()
}

func deferredLockIsNotAnUnlock(mutex *sync.Mutex) {
	defer mutex.Lock()
	work()
}
