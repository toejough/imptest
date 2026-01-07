package safety

// CriticalDependency represents a service that might crash.
type CriticalDependency interface {
	DoWork()
}

// SafeRunner attempts to perform work and recovers from any panics.
func SafeRunner(dep CriticalDependency) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()

	dep.DoWork()

	return true
}

// UnsafeRunner performs work but doesn't recover from panics.
func UnsafeRunner(dep CriticalDependency) {
	dep.DoWork()
}
