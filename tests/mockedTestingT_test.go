package imptest_test

import (
	"fmt"
	"runtime"
	"sync"
)

// MockedTestingT.
func newMockedTestingT() *mockedTestingT { return &mockedTestingT{failure: ""} }

type mockedTestingT struct{ failure string }

func (mt *mockedTestingT) Fatalf(message string, args ...any) {
	mt.failure = fmt.Sprintf(message, args...)

	runtime.Goexit()
}
func (mt *mockedTestingT) Helper()         {}
func (mt *mockedTestingT) Failed() bool    { return mt.failure != "" }
func (mt *mockedTestingT) Failure() string { return mt.failure }

// TODO: Wrap everywhere we use mockedTestingT, so we can catch Fatalf calls.
func (mt *mockedTestingT) Wrap(wrapped func()) {
	waitgroup := &sync.WaitGroup{}
	waitgroup.Add(1)

	go func() {
		defer waitgroup.Done()
		wrapped()
	}()
	waitgroup.Wait()
}
