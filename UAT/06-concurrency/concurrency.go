package concurrency

import (
	"sync"
	"time"
)

// SlowService represents a dependency that might be called concurrently.
type SlowService interface {
	DoA(id int) string
	DoB(id int) string
}

// RunConcurrent calls DoA and DoB in separate goroutines.
// It purposefully introduces a small delay for DoB to increase the chance
// that they arrive "out of order" relative to a sequential test.
func RunConcurrent(svc SlowService, id int) []string {
	var wg sync.WaitGroup

	results := make([]string, 2)

	wg.Add(2)

	go func() {
		defer wg.Done()
		// Small delay to make ordering non-deterministic or different from test expectation
		time.Sleep(50 * time.Millisecond)

		results[1] = svc.DoB(id)
	}()

	go func() {
		defer wg.Done()

		results[0] = svc.DoA(id)
	}()

	wg.Wait()

	return results
}
