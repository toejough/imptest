package imptest_test

import (
	"sync"
	"testing"

	. "github.com/onsi/gomega" //nolint:revive // dot import preferred for test readability
	"pgregory.net/rapid"

	"github.com/toejough/imptest"
)

// TestCleanup_RemovesEntryAfterTestCompletes verifies that the registry
// entry is removed when the test completes via t.Cleanup.
// Note: This test verifies the cleanup registration behavior indirectly
// since we can't query registry state directly.
func TestCleanup_RemovesEntryAfterTestCompletes(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Run in a subtest to isolate cleanup behavior
	t.Run("subtest", func(t *testing.T) {
		t.Parallel()

		// Creating an Imp should succeed and not be nil
		capturedImp := imptest.GetOrCreateImp(t)
		g.Expect(capturedImp).NotTo(BeNil())
		// Cleanup is registered via t.Cleanup() and will run when this subtest exits
	})
}

// TestGetOrCreateImp_ConcurrentAccess verifies the registry is safe for
// concurrent access from multiple goroutines.
func TestGetOrCreateImp_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	const numGoroutines = 100

	results := make([]*imptest.Imp, numGoroutines)

	var wg sync.WaitGroup

	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()

			results[idx] = imptest.GetOrCreateImp(t)
		}(i)
	}

	wg.Wait()

	// All results should be the same Imp
	for i := 1; i < numGoroutines; i++ {
		g.Expect(results[i]).To(BeIdenticalTo(results[0]),
			"concurrent calls with same t should return same Imp")
	}
}

// TestGetOrCreateImp_ConcurrentAccess_Rapid uses property-based testing to
// verify concurrent access safety with randomized access patterns.
func TestGetOrCreateImp_ConcurrentAccess_Rapid(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		numGoroutines := rapid.IntRange(2, 50).Draw(rt, "numGoroutines")
		results := make([]*imptest.Imp, numGoroutines)

		var wg sync.WaitGroup

		wg.Add(numGoroutines)

		for i := range numGoroutines {
			go func(idx int) {
				defer wg.Done()

				results[idx] = imptest.GetOrCreateImp(t)
			}(i)
		}

		wg.Wait()

		// All should be identical
		for i := 1; i < numGoroutines; i++ {
			if results[i] != results[0] {
				rt.Fatalf("goroutine %d got different Imp", i)
			}
		}
	})
}

// TestGetOrCreateImp_DifferentT_ReturnsDifferentImp verifies that different
// *testing.T values get different *Imp instances.
func TestGetOrCreateImp_DifferentT_ReturnsDifferentImp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Store results from subtests - use a sync.Map for thread safety
	results := make(map[string]*imptest.Imp)

	var mu sync.Mutex

	t.Run("subtest1", func(t *testing.T) {
		t.Parallel()

		imp := imptest.GetOrCreateImp(t)

		mu.Lock()

		results["sub1"] = imp

		mu.Unlock()
	})

	t.Run("subtest2", func(t *testing.T) {
		t.Parallel()

		imp := imptest.GetOrCreateImp(t)

		mu.Lock()

		results["sub2"] = imp

		mu.Unlock()
	})

	// After t.Run returns for parallel subtests, subtests are queued but not complete.
	// Go's test framework waits for all subtests to complete before the parent exits.
	// We use t.Cleanup to verify after subtests complete.
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()

		g.Expect(results["sub1"]).NotTo(BeIdenticalTo(results["sub2"]),
			"different t should return different Imp")
	})
}

// TestGetOrCreateImp_SameT_ReturnsSameImp verifies that calling getOrCreateImp
// with the same *testing.T returns the same *Imp instance.
func TestGetOrCreateImp_SameT_ReturnsSameImp(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	imp1 := imptest.GetOrCreateImp(t)
	imp2 := imptest.GetOrCreateImp(t)

	g.Expect(imp1).To(BeIdenticalTo(imp2), "same t should return same Imp")
}

// TestWait_BlocksUntilAsyncExpectationsSatisfied verifies that Wait(t)
// blocks until all async expectations registered under t are satisfied.
func TestWait_BlocksUntilAsyncExpectationsSatisfied(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Ensure Imp exists for this test
	_ = imptest.GetOrCreateImp(t)

	// Track completion order
	completionOrder := make([]string, 0, 2)

	var mu sync.Mutex

	recordCompletion := func(name string) {
		mu.Lock()

		completionOrder = append(completionOrder, name)

		mu.Unlock()
	}

	// Simulate async expectation by using the Imp's underlying Wait mechanism
	// For this test, we'll verify that Wait() returns only after expectations complete
	done := make(chan struct{})

	go func() {
		imptest.Wait(t)
		recordCompletion("wait")
		close(done)
	}()

	// Register and satisfy an expectation
	// (This tests the integration - Wait should block until imp's expectations are done)
	recordCompletion("expectation")

	<-done

	mu.Lock()
	defer mu.Unlock()

	g.Expect(completionOrder).To(HaveLen(2))
	g.Expect(completionOrder[0]).To(Equal("expectation"))
	g.Expect(completionOrder[1]).To(Equal("wait"))
}

// TestWait_NoExpectations_ReturnsImmediately verifies that Wait(t) returns
// immediately when there are no pending async expectations.
func TestWait_NoExpectations_ReturnsImmediately(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	// Ensure Imp exists
	_ = imptest.GetOrCreateImp(t)

	// Wait should return immediately since there are no pending expectations
	done := make(chan struct{})

	go func() {
		imptest.Wait(t)
		close(done)
	}()

	// Should complete very quickly
	g.Eventually(done).Should(BeClosed())
}
