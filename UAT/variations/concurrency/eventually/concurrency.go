// Package concurrency demonstrates testing code that calls dependencies concurrently.
package concurrency

import (
	"sync"
	"time"
)

type SlowService interface {
	DoA(id int) string
	DoB(id int) string
}

// RunConcurrent calls DoA and DoB in separate goroutines.
// It purposefully introduces a small delay for DoB to increase the chance
// that they arrive "out of order" relative to a sequential test.
func RunConcurrent(svc SlowService, serviceID int) []string {
	const (
		numTasks = 2
		delay    = 50
	)

	var waitGroup sync.WaitGroup

	results := make([]string, numTasks)

	waitGroup.Add(numTasks)

	go func() {
		defer waitGroup.Done()
		// Small delay to make ordering non-deterministic or different from test expectation
		time.Sleep(delay * time.Millisecond)

		results[1] = svc.DoB(serviceID)
	}()

	go func() {
		defer waitGroup.Done()

		results[0] = svc.DoA(serviceID)
	}()

	waitGroup.Wait()

	return results
}
