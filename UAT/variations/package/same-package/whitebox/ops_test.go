//nolint:testpackage // Intentional whitebox testing of unexported methods
package whitebox

import "testing"

//go:generate impgen Ops --dependency

// TestProcessWithOps demonstrates whitebox testing.
//
// Key Requirements Met:
//  1. Test is in the same package as the code under test (package whitebox)
//  2. Test file has _test.go suffix but same package name
//  3. Generated mock should be in generated_OpsImp_test.go (with _test.go suffix)
//  4. Can test unexported methods because we're in the same package
//  5. No import qualifier needed for the Ops interface (same package)
func TestProcessWithOps(t *testing.T) {
	t.Parallel()

	mock, imp := MockOps(t)

	go func() {
		result := ProcessWithOps(mock, 5)
		_ = result // Use the result to avoid unused variable warning
	}()

	// Can test unexported method because we're in the same package
	imp.internalMethod.ExpectCalledWithExactly(5).InjectReturnValues(10)
	imp.PublicMethod.ExpectCalledWithExactly(10).InjectReturnValues(20)
}
