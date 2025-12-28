package targetfunction_test

import (
	"github.com/toejough/imptest/imptest"
)

// WrapBinaryOp wraps a BinaryOp function for testing.
func WrapBinaryOp(t imptest.TestReporter, fn BinaryOp) *WrapBinaryOpWrapper {
	return &WrapBinaryOpWrapper{
		imp: imptest.NewImp(t),
		fn:  fn,
	}
}

// WrapBinaryOpWrapper provides a fluent API for calling and verifying BinaryOp.
type WrapBinaryOpWrapper struct {
	imp *imptest.Imp
	fn  BinaryOp
}

// CallWith calls the wrapped function and captures the result.
func (w *WrapBinaryOpWrapper) CallWith(a, b int) *WrapBinaryOpCall {
	var result int
	var panicked bool
	var panicValue any

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicValue = r
			}
		}()
		result = w.fn(a, b)
	}()

	call := &imptest.TargetCall{
		Imp:      w.imp,
		Ordered:  true,
		Returned: !panicked,
		Panicked: panicked,
	}
	if !panicked {
		call.ReturnValues = []any{result}
	} else {
		call.PanicValue = panicValue
	}

	return &WrapBinaryOpCall{TargetCall: call}
}

// WrapBinaryOpCall wraps a TargetCall with type-safe return accessors.
type WrapBinaryOpCall struct {
	*imptest.TargetCall
}

// GetReturns returns the captured return values in a type-safe struct.
type WrapBinaryOpReturns struct {
	R1 int
}

// GetReturns returns type-safe return values.
func (c *WrapBinaryOpCall) GetReturns() *WrapBinaryOpReturns {
	raw := c.TargetCall.GetReturns()
	return &WrapBinaryOpReturns{
		R1: raw.R1.(int),
	}
}

// WrapAdd wraps the Add function for testing.
func WrapAdd(t imptest.TestReporter, fn func(int, int) int) *WrapAddWrapper {
	return &WrapAddWrapper{
		imp: imptest.NewImp(t),
		fn:  fn,
	}
}

// WrapAddWrapper provides a fluent API for calling and verifying Add.
type WrapAddWrapper struct {
	imp *imptest.Imp
	fn  func(int, int) int
}

// CallWith calls the wrapped function and captures the result.
func (w *WrapAddWrapper) CallWith(a, b int) *WrapAddCall {
	var result int
	var panicked bool
	var panicValue any

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicValue = r
			}
		}()
		result = w.fn(a, b)
	}()

	call := &imptest.TargetCall{
		Imp:      w.imp,
		Ordered:  true,
		Returned: !panicked,
		Panicked: panicked,
	}
	if !panicked {
		call.ReturnValues = []any{result}
	} else {
		call.PanicValue = panicValue
	}

	return &WrapAddCall{TargetCall: call}
}

// WrapAddCall wraps a TargetCall with type-safe return accessors.
type WrapAddCall struct {
	*imptest.TargetCall
}

// GetReturns returns the captured return values in a type-safe struct.
type WrapAddReturns struct {
	R1 int
}

// GetReturns returns type-safe return values.
func (c *WrapAddCall) GetReturns() *WrapAddReturns {
	raw := c.TargetCall.GetReturns()
	return &WrapAddReturns{
		R1: raw.R1.(int),
	}
}

// WrapDivide wraps the Divide function for testing.
func WrapDivide(t imptest.TestReporter, fn func(int, int) int) *WrapDivideWrapper {
	return &WrapDivideWrapper{
		imp: imptest.NewImp(t),
		fn:  fn,
	}
}

// WrapDivideWrapper provides a fluent API for calling and verifying Divide.
type WrapDivideWrapper struct {
	imp *imptest.Imp
	fn  func(int, int) int
}

// CallWith calls the wrapped function and captures the result.
func (w *WrapDivideWrapper) CallWith(a, b int) *WrapDivideCall {
	var result int
	var panicked bool
	var panicValue any

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicValue = r
			}
		}()
		result = w.fn(a, b)
	}()

	call := &imptest.TargetCall{
		Imp:      w.imp,
		Ordered:  true,
		Returned: !panicked,
		Panicked: panicked,
	}
	if !panicked {
		call.ReturnValues = []any{result}
	} else {
		call.PanicValue = panicValue
	}

	return &WrapDivideCall{TargetCall: call}
}

// WrapDivideCall wraps a TargetCall.
type WrapDivideCall struct {
	*imptest.TargetCall
}

// WrapConcurrent wraps the Concurrent function for testing.
func WrapConcurrent(t imptest.TestReporter, fn func(int) int) *WrapConcurrentWrapper {
	return &WrapConcurrentWrapper{
		imp: imptest.NewImp(t),
		fn:  fn,
	}
}

// WrapConcurrentWrapper provides a fluent API for calling and verifying Concurrent.
type WrapConcurrentWrapper struct {
	imp *imptest.Imp
	fn  func(int) int
}

// CallWith calls the wrapped function and captures the result.
func (w *WrapConcurrentWrapper) CallWith(i int) *WrapConcurrentCall {
	var result int
	var panicked bool
	var panicValue any

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicValue = r
			}
		}()
		result = w.fn(i)
	}()

	call := &imptest.TargetCall{
		Imp:      w.imp,
		Ordered:  true,
		Returned: !panicked,
		Panicked: panicked,
	}
	if !panicked {
		call.ReturnValues = []any{result}
	} else {
		call.PanicValue = panicValue
	}

	return &WrapConcurrentCall{TargetCall: call}
}

// WrapConcurrentCall wraps a TargetCall with type-safe return accessors.
type WrapConcurrentCall struct {
	*imptest.TargetCall
}

// GetReturns returns the captured return values in a type-safe struct.
type WrapConcurrentReturns struct {
	R1 int
}

// GetReturns returns type-safe return values.
func (c *WrapConcurrentCall) GetReturns() *WrapConcurrentReturns {
	raw := c.TargetCall.GetReturns()
	return &WrapConcurrentReturns{
		R1: raw.R1.(int),
	}
}
