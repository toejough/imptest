package interfaceliteral_test

import (
	"errors"
	"testing"

	"github.com/toejough/imptest"
)

//go:generate impgen DataProcessor --dependency

// TestInterfaceLiteralParameters demonstrates that impgen correctly handles
// interface literals in method signatures.
//
//nolint:funlen // Comprehensive test with multiple sub-tests and helper types
func TestInterfaceLiteralParameters(t *testing.T) {
	t.Parallel()

	t.Run("SingleMethod", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockDataProcessor(t)

		obj := &simpleGetter{value: "test data"}

		// Launch goroutine that calls the method with interface literal parameter
		go func() {
			_ = mock.Process(obj)
		}()

		// Verify the mock received the call and inject return value
		call := imp.Process.ExpectCalledWithExactly(obj)
		call.InjectReturnValues("processed")

		// Verify args can be retrieved
		args := call.GetArgs()
		if args.Obj.Get() != "test data" {
			t.Fatalf("expected obj.Get() = 'test data', got %q", args.Obj.Get())
		}
	})

	t.Run("MultipleMethod", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockDataProcessor(t)

		obj := &multiMethodObject{value: 42}

		// Launch goroutine that calls the method
		go func() {
			_ = mock.Transform(obj)
		}()

		// Verify and inject return value
		call := imp.Transform.ExpectCalledWithExactly(obj)
		call.InjectReturnValues(100)

		// Verify args can be retrieved and methods can be called
		args := call.GetArgs()
		if args.Obj.GetValue() != 42 {
			t.Fatalf("expected obj.GetValue() = 42, got %d", args.Obj.GetValue())
		}
	})

	t.Run("WithError", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockDataProcessor(t)

		validator := &simpleValidator{shouldFail: true}

		// Launch goroutine that calls the method
		go func() {
			_ = mock.Validate(validator)
		}()

		// Verify and inject error return
		testErr := errors.New("mock validation error")
		call := imp.Validate.ExpectCalledWithExactly(validator)
		call.InjectReturnValues(testErr)

		// Verify validator can be called
		args := call.GetArgs()

		err := args.Validator.Check("test")
		if err == nil {
			t.Fatal("expected validator.Check to return error")
		}
	})

	t.Run("ReturnType", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockDataProcessor(t)

		// Launch goroutine that calls the method
		go func() {
			result := mock.ProcessWithReturn("input")
			_ = result.Result() // Use the returned interface literal
		}()

		// Verify and inject return value
		returnObj := &resultProvider{result: "output"}
		call := imp.ProcessWithReturn.ExpectCalledWithExactly("input")
		call.InjectReturnValues(returnObj)

		// Verify args
		args := call.GetArgs()
		if args.Input != "input" {
			t.Fatalf("expected input = 'input', got %q", args.Input)
		}
	})

	t.Run("WithMatchers", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockDataProcessor(t)

		obj := &simpleGetter{value: "matcher test"}

		// Launch goroutine
		go func() {
			_ = mock.Process(obj)
		}()

		// Use matcher for interface literal parameter
		call := imp.Process.ExpectCalledWithMatches(imptest.Any())
		call.InjectReturnValues("matched result")

		// Verify we can still get args
		args := call.GetArgs()
		if args.Obj.Get() != "matcher test" {
			t.Fatalf("expected obj.Get() = 'matcher test', got %q", args.Obj.Get())
		}
	})
}

// multiMethodObject implements the multi-method interface literal.
type multiMethodObject struct {
	value int
}

func (m *multiMethodObject) GetValue() int {
	return m.value
}

func (m *multiMethodObject) SetValue(v int) {
	m.value = v
}

// resultProvider implements the result interface literal.
type resultProvider struct {
	result string
}

func (r *resultProvider) Result() string {
	return r.result
}

// simpleGetter implements the single-method interface literal.
type simpleGetter struct {
	value string
}

func (g *simpleGetter) Get() string {
	return g.value
}

// simpleValidator implements the validation interface literal.
type simpleValidator struct {
	shouldFail bool
}

func (v *simpleValidator) Check(_ string) error {
	if v.shouldFail {
		return errors.New("validation failed")
	}

	return nil
}
