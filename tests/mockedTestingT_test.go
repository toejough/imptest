package imptest_test

import (
	"fmt"
)

// MockedTestingT.
func newMockedTestingT() *mockedTestingT { return &mockedTestingT{failure: ""} }

type mockedTestingT struct{ failure string }

func (mt *mockedTestingT) Fatalf(message string, args ...any) {
	mt.failure = fmt.Sprintf(message, args...)
}
func (mt *mockedTestingT) Helper()         {}
func (mt *mockedTestingT) Failed() bool    { return mt.failure != "" }
func (mt *mockedTestingT) Failure() string { return mt.failure }
