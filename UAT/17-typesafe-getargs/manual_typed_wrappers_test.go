// Manual implementation of what we want to generate
package typesafeargs_test

import (
	typesafeargs "github.com/toejough/imptest/UAT/17-typesafe-getargs"
	"github.com/toejough/imptest/imptest"
)

// CalculatorAddArgs holds typed arguments for Add method
type CalculatorAddArgs struct {
	A int
	B int
}

// CalculatorAddCall wraps DependencyCall with typed GetArgs
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

// CalculatorAddMethod wraps DependencyMethod with typed return
type CalculatorAddMethod struct {
	*imptest.DependencyMethod
}

func (m *CalculatorAddMethod) Eventually() *CalculatorAddMethod {
	return &CalculatorAddMethod{DependencyMethod: m.DependencyMethod.Eventually()}
}

func (m *CalculatorAddMethod) ExpectCalledWithExactly(a, b int) *CalculatorAddCall {
	call := m.DependencyMethod.ExpectCalledWithExactly(a, b)
	return &CalculatorAddCall{DependencyCall: call}
}

func (m *CalculatorAddMethod) ExpectCalledWithMatches(matchers ...any) *CalculatorAddCall {
	call := m.DependencyMethod.ExpectCalledWithMatches(matchers...)
	return &CalculatorAddCall{DependencyCall: call}
}

// Typesafe calculator mock
type TypesafeCalculatorMock struct {
	imp      *imptest.Imp
	Add      *CalculatorAddMethod
	Multiply *imptest.DependencyMethod // For now, only Add is typed
	Store    *imptest.DependencyMethod
}

func NewTypesafeCalculatorMock(t imptest.TestReporter) *TypesafeCalculatorMock {
	imp := imptest.NewImp(t)

	return &TypesafeCalculatorMock{
		imp:      imp,
		Add:      &CalculatorAddMethod{DependencyMethod: imptest.NewDependencyMethod(imp, "Add")},
		Multiply: imptest.NewDependencyMethod(imp, "Multiply"),
		Store:    imptest.NewDependencyMethod(imp, "Store"),
	}
}

func (m *TypesafeCalculatorMock) Interface() typesafeargs.Calculator {
	return &mockCalculatorImpl{mock: m}
}

type mockCalculatorImpl struct {
	mock *TypesafeCalculatorMock
}

func (impl *mockCalculatorImpl) Add(a, b int) int {
	call := &imptest.GenericCall{
		MethodName:   "Add",
		Args:         []any{a, b},
		ResponseChan: make(chan imptest.GenericResponse, 1),
	}
	impl.mock.imp.CallChan <- call

	resp := <-call.ResponseChan
	if resp.Type == "panic" {
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
	impl.mock.imp.CallChan <- call

	resp := <-call.ResponseChan
	if resp.Type == "panic" {
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
	impl.mock.imp.CallChan <- call

	resp := <-call.ResponseChan
	if resp.Type == "panic" {
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
