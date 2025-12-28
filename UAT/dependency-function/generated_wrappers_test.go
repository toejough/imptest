package dependencyfunction_test

import (
	"github.com/toejough/imptest/imptest"
)

// WrapProcessData wraps the ProcessData function for testing.
func WrapProcessData(t imptest.TestReporter, fn func(int, Fetcher) (string, error)) *WrapProcessDataWrapper {
	return &WrapProcessDataWrapper{
		imp: imptest.NewImp(t),
		fn:  fn,
	}
}

// WrapProcessDataWrapper provides a fluent API for calling and verifying ProcessData.
type WrapProcessDataWrapper struct {
	imp *imptest.Imp
	fn  func(int, Fetcher) (string, error)
}

// CallWith calls the wrapped function and captures the result.
func (w *WrapProcessDataWrapper) CallWith(id int, fetcher Fetcher) *WrapProcessDataCall {
	var r1 string
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
		r1, r2 = w.fn(id, fetcher)
	}()

	call := &imptest.TargetCall{
		Imp:      w.imp,
		Ordered:  true,
		Returned: !panicked,
		Panicked: panicked,
	}
	if !panicked {
		call.ReturnValues = []any{r1, r2}
	} else {
		call.PanicValue = panicValue
	}

	return &WrapProcessDataCall{TargetCall: call}
}

// WrapProcessDataCall wraps a TargetCall with type-safe return accessors.
type WrapProcessDataCall struct {
	*imptest.TargetCall
}

// WrapProcessDataReturns provides type-safe access to return values.
type WrapProcessDataReturns struct {
	R1 string
	R2 error
}

// GetReturns returns type-safe return values.
func (c *WrapProcessDataCall) GetReturns() *WrapProcessDataReturns {
	raw := c.TargetCall.GetReturns()
	var r2 error
	if raw.R2 != nil {
		r2 = raw.R2.(error)
	}
	return &WrapProcessDataReturns{
		R1: raw.R1.(string),
		R2: r2,
	}
}
