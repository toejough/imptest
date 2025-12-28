package targetinterface_test

import (
	"github.com/toejough/imptest/imptest"
)

// WrapCalculator wraps a Calculator implementation for testing.
func WrapCalculator(t imptest.TestReporter, instance Calculator) *WrapCalculatorWrapper {
	imp := imptest.NewImp(t)
	wrapper := &WrapCalculatorWrapper{
		imp:      imp,
		instance: instance,
	}
	wrapper.Add = &WrapCalculatorAddMethod{wrapper: wrapper}
	wrapper.Subtract = &WrapCalculatorSubtractMethod{wrapper: wrapper}
	wrapper.Divide = &WrapCalculatorDivideMethod{wrapper: wrapper}
	return wrapper
}

// WrapCalculatorWrapper provides a fluent API for calling and verifying Calculator methods.
type WrapCalculatorWrapper struct {
	imp      *imptest.Imp
	instance Calculator
	Add      *WrapCalculatorAddMethod
	Subtract *WrapCalculatorSubtractMethod
	Divide   *WrapCalculatorDivideMethod
}

// WrapCalculatorAddMethod wraps the Add method.
type WrapCalculatorAddMethod struct {
	wrapper *WrapCalculatorWrapper
}

// CallWith calls the Add method and captures the result.
func (m *WrapCalculatorAddMethod) CallWith(a, b int) *WrapCalculatorAddCall {
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
		result = m.wrapper.instance.Add(a, b)
	}()

	call := &imptest.TargetCall{
		Imp:      m.wrapper.imp,
		Ordered:  true,
		Returned: !panicked,
		Panicked: panicked,
	}
	if !panicked {
		call.ReturnValues = []any{result}
	} else {
		call.PanicValue = panicValue
	}

	return &WrapCalculatorAddCall{TargetCall: call}
}

// WrapCalculatorAddCall wraps a TargetCall for Add with type-safe return accessors.
type WrapCalculatorAddCall struct {
	*imptest.TargetCall
}

// WrapCalculatorAddReturns provides type-safe access to Add return values.
type WrapCalculatorAddReturns struct {
	R1 int
}

// GetReturns returns type-safe return values for Add.
func (c *WrapCalculatorAddCall) GetReturns() *WrapCalculatorAddReturns {
	raw := c.TargetCall.GetReturns()
	return &WrapCalculatorAddReturns{
		R1: raw.R1.(int),
	}
}

// WrapCalculatorSubtractMethod wraps the Subtract method.
type WrapCalculatorSubtractMethod struct {
	wrapper *WrapCalculatorWrapper
}

// CallWith calls the Subtract method and captures the result.
func (m *WrapCalculatorSubtractMethod) CallWith(a, b int) *WrapCalculatorSubtractCall {
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
		result = m.wrapper.instance.Subtract(a, b)
	}()

	call := &imptest.TargetCall{
		Imp:      m.wrapper.imp,
		Ordered:  true,
		Returned: !panicked,
		Panicked: panicked,
	}
	if !panicked {
		call.ReturnValues = []any{result}
	} else {
		call.PanicValue = panicValue
	}

	return &WrapCalculatorSubtractCall{TargetCall: call}
}

// WrapCalculatorSubtractCall wraps a TargetCall for Subtract with type-safe return accessors.
type WrapCalculatorSubtractCall struct {
	*imptest.TargetCall
}

// WrapCalculatorSubtractReturns provides type-safe access to Subtract return values.
type WrapCalculatorSubtractReturns struct {
	R1 int
}

// GetReturns returns type-safe return values for Subtract.
func (c *WrapCalculatorSubtractCall) GetReturns() *WrapCalculatorSubtractReturns {
	raw := c.TargetCall.GetReturns()
	return &WrapCalculatorSubtractReturns{
		R1: raw.R1.(int),
	}
}

// WrapCalculatorDivideMethod wraps the Divide method.
type WrapCalculatorDivideMethod struct {
	wrapper *WrapCalculatorWrapper
}

// CallWith calls the Divide method and captures the result.
func (m *WrapCalculatorDivideMethod) CallWith(a, b int) *WrapCalculatorDivideCall {
	var r1 int
	var r2 error
	var panicked bool
	var panicValue any

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicValue = r
			}
		}()
		r1, r2 = m.wrapper.instance.Divide(a, b)
	}()

	call := &imptest.TargetCall{
		Imp:      m.wrapper.imp,
		Ordered:  true,
		Returned: !panicked,
		Panicked: panicked,
	}
	if !panicked {
		call.ReturnValues = []any{r1, r2}
	} else {
		call.PanicValue = panicValue
	}

	return &WrapCalculatorDivideCall{TargetCall: call}
}

// WrapCalculatorDivideCall wraps a TargetCall for Divide with type-safe return accessors.
type WrapCalculatorDivideCall struct {
	*imptest.TargetCall
}

// WrapCalculatorDivideReturns provides type-safe access to Divide return values.
type WrapCalculatorDivideReturns struct {
	R1 int
	R2 error
}

// GetReturns returns type-safe return values for Divide.
func (c *WrapCalculatorDivideCall) GetReturns() *WrapCalculatorDivideReturns {
	raw := c.TargetCall.GetReturns()
	var r2 error
	if raw.R2 != nil {
		r2 = raw.R2.(error)
	}
	return &WrapCalculatorDivideReturns{
		R1: raw.R1.(int),
		R2: r2,
	}
}
