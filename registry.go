package imptest

// GetOrCreateImp returns the Imp for the given test, creating one if needed.
// Multiple calls with the same TestReporter return the same Imp instance.
// This enables coordination between mocks and wrappers in the same test.
func GetOrCreateImp(t TestReporter) *Imp {
	// TODO: implement registry lookup
	panic("not implemented")
}

// Wait blocks until all async expectations registered under t are satisfied.
// This is the package-level wait that coordinates across all mocks/wrappers
// sharing the same TestReporter.
func Wait(t TestReporter) {
	// TODO: implement
	panic("not implemented")
}
