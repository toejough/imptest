// Manual implementation of what we want to generate
package typesafeargs_test

import (
	"github.com/toejough/imptest"
	typesafeargs "github.com/toejough/imptest/UAT/variations/behavior/typesafe-getargs"
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

	// Eventually is the async version of this method for concurrent code.
	Eventually *CalculatorAddMethod
}

func (m *CalculatorAddMethod) ExpectCalledWithExactly(a, b int) *CalculatorAddCall {
	call := m.DependencyMethod.ExpectCalledWithExactly(a, b)
	return &CalculatorAddCall{DependencyCall: call}
}

func (m *CalculatorAddMethod) ExpectCalledWithMatches(matchers ...any) *CalculatorAddCall {
	call := m.DependencyMethod.ExpectCalledWithMatches(matchers...)
	return &CalculatorAddCall{DependencyCall: call}
}

// TypesafeCalculatorMockHandle is the test handle for Calculator.
type TypesafeCalculatorMockHandle struct {
	Mock       typesafeargs.Calculator
	Method     *TypesafeCalculatorMockMethods
	Controller *imptest.Imp
}

// TypesafeCalculatorMockMethods holds method wrappers for setting expectations.
type TypesafeCalculatorMockMethods struct {
	Add      *CalculatorAddMethod
	Multiply *imptest.DependencyMethod // For now, only Add is typed
	Store    *imptest.DependencyMethod
}

func NewTypesafeCalculatorMock(t imptest.TestReporter) *TypesafeCalculatorMockHandle {
	ctrl := imptest.GetOrCreateImp(t)
	methods := &TypesafeCalculatorMockMethods{
		Add:      newCalculatorAddMethod(imptest.NewDependencyMethod(ctrl, "Add")),
		Multiply: imptest.NewDependencyMethod(ctrl, "Multiply"),
		Store:    imptest.NewDependencyMethod(ctrl, "Store"),
	}
	handle := &TypesafeCalculatorMockHandle{
		Method:     methods,
		Controller: ctrl,
	}
	handle.Mock = &mockCalculatorImpl{handle: handle}

	return handle
}

// unexported constants.
const (
	responseTypePanic = "panic"
)

type mockCalculatorImpl struct {
	handle *TypesafeCalculatorMockHandle
}

func (impl *mockCalculatorImpl) Add(a, b int) int {
	call := &imptest.GenericCall{
		MethodName:   "Add",
		Args:         []any{a, b},
		ResponseChan: make(chan imptest.GenericResponse, 1),
	}
	impl.handle.Controller.CallChan <- call

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
	impl.handle.Controller.CallChan <- call

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
	impl.handle.Controller.CallChan <- call

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
	m.Eventually = &CalculatorAddMethod{DependencyMethod: depMethod.Eventually}

	return m
}
