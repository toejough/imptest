package run_test

import (
	"testing"

	"github.com/toejough/imptest"
	run "github.com/toejough/imptest/UAT/run"
)

type PrintSumImp struct {
	t              *testing.T
	callable       func(a, b int, deps run.IntOps) (int, int, string)
	testInvocation *imptest.TestInvocation
}

func NewPrintSumImp(t *testing.T, callable func(a, b int, deps run.IntOps) (int, int, string)) *PrintSumImp {
	t.Helper()

	return &PrintSumImp{
		t:        t,
		callable: callable,
	}
}

func (imp *PrintSumImp) Start(a, b int, deps run.IntOps) *PrintSumImp {
	imp.testInvocation = imptest.Start(imp.t, imp.callable, a, b, deps)
	return imp
}

func (imp *PrintSumImp) ExpectReturnedValues(v1 int, v2 int, v3 string) {
	imp.testInvocation.ExpectReturnedValues(v1, v2, v3)
}
