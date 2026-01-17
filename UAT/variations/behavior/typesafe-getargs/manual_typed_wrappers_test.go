// Manual implementation of what we want to generate
package typesafeargs_test

import (
	"github.com/toejough/imptest"
	typesafeargs "github.com/toejough/imptest/UAT/variations/behavior/typesafe-getargs"
)

type CalculatorAddArgs struct {
	A int
	B int
}

type CalculatorAddCall struct {
	*imptest.DependencyCall
}

func (c *CalculatorAddCall) GetArgs() CalculatorAddArgs {
	raw := c.RawArgs()

	return CalculatorAddArgs{
		A: raw[0].(int), //nolint:forcetypeassert // Manual example code - type is guaranteed by mock setup
		B: raw[1].(int), //nolint:forcetypeassert // Manual example code - type is guaranteed by mock setup
	}
}

type CalculatorAddMethod struct {
	*imptest.DependencyMethod

	// Eventually is the async version of this method for concurrent code.
	Eventually *CalculatorAddMethod
}

func (m *CalculatorAddMethod) ArgsEqual(a, b int) *CalculatorAddCall {
	call := m.DependencyMethod.ArgsEqual(a, b)
	return &CalculatorAddCall{DependencyCall: call}
}

func (m *CalculatorAddMethod) ArgsShould(matchers ...any) *CalculatorAddCall {
	call := m.DependencyMethod.ArgsShould(matchers...)
	return &CalculatorAddCall{DependencyCall: call}
}

type TypesafeCalculatorImp struct {
	Add      *CalculatorAddMethod
	Multiply *imptest.DependencyMethod // For now, only Add is typed
	Store    *imptest.DependencyMethod
}

// NewTypesafeCalculatorMock creates a mock Calculator and returns (mock, expectation handle).
func NewTypesafeCalculatorMock(
	t imptest.TestReporter,
) (typesafeargs.Calculator, *TypesafeCalculatorImp) {
	ctrl := imptest.GetOrCreateImp(t)
	imp := &TypesafeCalculatorImp{
		Add:      newCalculatorAddMethod(imptest.NewDependencyMethod(ctrl, "Add")),
		Multiply: imptest.NewDependencyMethod(ctrl, "Multiply"),
		Store:    imptest.NewDependencyMethod(ctrl, "Store"),
	}
	mock := &mockCalculatorImpl{ctrl: ctrl}

	return mock, imp
}

// unexported constants.
const (
	responseTypePanic = "panic"
)

type mockCalculatorImpl struct {
	ctrl *imptest.Imp
}

func (impl *mockCalculatorImpl) Add(a, b int) int {
	call := &imptest.GenericCall{
		MethodName:   "Add",
		Args:         []any{a, b},
		ResponseChan: make(chan imptest.GenericResponse, 1),
	}
	impl.ctrl.CallChan <- call

	resp := <-call.ResponseChan
	if resp.Type == responseTypePanic {
		panic(resp.PanicValue)
	}

	var result1 int

	if len(resp.ReturnValues) > 0 {
		if value, ok := resp.ReturnValues[0].(int); ok {
			result1 = value
		}
	}

	return result1
}

func (impl *mockCalculatorImpl) Multiply(x, y int) int {
	call := &imptest.GenericCall{
		MethodName:   "Multiply",
		Args:         []any{x, y},
		ResponseChan: make(chan imptest.GenericResponse, 1),
	}
	impl.ctrl.CallChan <- call

	resp := <-call.ResponseChan
	if resp.Type == responseTypePanic {
		panic(resp.PanicValue)
	}

	var result1 int

	if len(resp.ReturnValues) > 0 {
		if value, ok := resp.ReturnValues[0].(int); ok {
			result1 = value
		}
	}

	return result1
}

func (impl *mockCalculatorImpl) Store(key string, value any) error {
	call := &imptest.GenericCall{
		MethodName:   "Store",
		Args:         []any{key, value},
		ResponseChan: make(chan imptest.GenericResponse, 1),
	}
	impl.ctrl.CallChan <- call

	resp := <-call.ResponseChan
	if resp.Type == responseTypePanic {
		panic(resp.PanicValue)
	}

	var result1 error

	if len(resp.ReturnValues) > 0 {
		if value, ok := resp.ReturnValues[0].(error); ok {
			result1 = value
		}
	}

	return result1
}

func newCalculatorAddMethod(depMethod *imptest.DependencyMethod) *CalculatorAddMethod {
	m := &CalculatorAddMethod{DependencyMethod: depMethod}
	m.Eventually = &CalculatorAddMethod{DependencyMethod: depMethod.AsEventually()}

	return m
}
