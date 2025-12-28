package targetfunction_test

import (
	"testing"
	"time"

	imptest "github.com/toejough/imptest/imptest/v2"
)

// AsyncAdd simulates an asynchronous add operation
func AsyncAdd(a, b int, resultChan chan<- int) {
	go func() {
		time.Sleep(10 * time.Millisecond)
		resultChan <- a + b
	}()
}

// TestTargetFunction_Unordered_Exact_Returns demonstrates Eventually mode
// for asynchronous/concurrent code where we don't know the exact timing
func TestTargetFunction_Unordered_Exact_Returns(t *testing.T) {
	imp := imptest.NewImp(t)
	target := imptest.NewTargetFunction(imp, AsyncAdd)

	resultChan := make(chan int, 1)

	// Call the async function
	call := target.CallWith(2, 3, resultChan)

	// Switch to unordered mode - will wait for matching interaction
	call = call.Eventually()

	// This will wait for the goroutine to complete, queueing any
	// interactions that don't match
	call.ExpectReturnsEqual()
}

// ConcurrentOps performs operations that may complete in any order
func ConcurrentOps(op1, op2 func()) {
	go op1()
	go op2()
}

// TestTargetFunction_Unordered_GetReturns demonstrates Eventually with GetReturns
func TestTargetFunction_Unordered_GetReturns(t *testing.T) {
	imp := imptest.NewImp(t)

	done1 := make(chan bool)
	done2 := make(chan bool)

	op1 := func() {
		time.Sleep(5 * time.Millisecond)
		close(done1)
	}
	op2 := func() {
		time.Sleep(10 * time.Millisecond)
		close(done2)
	}

	target := imptest.NewTargetFunction(imp, ConcurrentOps)
	call := target.CallWith(op1, op2)

	// Wait for completion without caring about order
	call = call.Eventually()
	call.GetReturns()

	// Verify ops completed
	select {
	case <-done1:
	case <-time.After(100 * time.Millisecond):
		t.Error("op1 did not complete")
	}
	select {
	case <-done2:
	case <-time.After(100 * time.Millisecond):
		t.Error("op2 did not complete")
	}
}
