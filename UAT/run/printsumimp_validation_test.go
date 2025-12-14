package run_test

import (
	"fmt"
	"testing"

	"github.com/toejough/imptest/UAT/run"
)

// mockT captures test failures for testing ExpectReturnedValues/ExpectPanicWith behavior.
type mockT struct {
	testing.T

	failed bool
	msg    string
}

func (m *mockT) Fatalf(format string, args ...any) {
	m.failed = true
	m.msg = fmt.Sprintf(format, args...)
	panic("mockT failed: " + m.msg)
}

func (m *mockT) Helper() {}

// TestPrintSumImp_Start_Success tests that Start correctly handles functions.
func TestPrintSumImp_Start_Success(t *testing.T) {
	t.Parallel()

	t.Run("Function with args and returns", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(2, 3, imp.Mock)

		// Inject expected calls
		imp.ExpectCallTo.Add(2, 3).InjectResult(5)
		imp.ExpectCallTo.Format(5).InjectResult("5")
		imp.ExpectCallTo.Print("5").Resolve()

		printSumImp.ExpectReturnedValues(2, 3, "5")
	})

	t.Run("Function that panics", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(1, 2, imp.Mock)

		// Inject panic
		imp.ExpectCallTo.Add(1, 2).InjectPanic("oops")

		printSumImp.ExpectPanicWith("oops")
	})
}

// TestPrintSumImp_ExpectReturnedValues_WrongEventType tests ExpectReturnedValues when event is panic.
func TestPrintSumImp_ExpectReturnedValues_WrongEventType(t *testing.T) {
	t.Parallel()

	t.Run("Fails when event is panic", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(1, 2, imp.Mock)

		// Inject panic so we get a panic event
		imp.ExpectCallTo.Add(1, 2).InjectPanic("oops")

		// Swap test interface to capture failure
		mock := &mockT{}
		printSumImp.t = mock

		defer func() {
			if r := recover(); r != nil {
				_ = r // expected panic from mockT
			}

			if !mock.failed {
				t.Error("Expected failure but got none")
			}

			if mock.msg == "" {
				t.Error("Expected failure message")
			}
		}()

		printSumImp.ExpectReturnedValues(1, 2, "3")
	})
}

// TestPrintSumImp_ExpectReturnedValues_ValueMismatch tests value mismatch failures.
func TestPrintSumImp_ExpectReturnedValues_ValueMismatch(t *testing.T) {
	t.Parallel()

	t.Run("Fails when values mismatch", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(2, 3, imp.Mock)

		// Inject calls
		imp.ExpectCallTo.Add(2, 3).InjectResult(5)
		imp.ExpectCallTo.Format(5).InjectResult("5")
		imp.ExpectCallTo.Print("5").Resolve()

		// Swap test interface
		mock := &mockT{}
		printSumImp.t = mock

		defer func() {
			_ = recover()

			if !mock.failed {
				t.Error("Expected failure")
			}
		}()

		// Wrong third value
		printSumImp.ExpectReturnedValues(2, 3, "wrong")
	})
}

// TestPrintSumImp_ExpectPanicWith_Failures tests ExpectPanicWith failure cases.
func TestPrintSumImp_ExpectPanicWith_Failures(t *testing.T) {
	t.Parallel()

	t.Run("Fails when event is return", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(2, 3, imp.Mock)

		// Inject normal return
		imp.ExpectCallTo.Add(2, 3).InjectResult(5)
		imp.ExpectCallTo.Format(5).InjectResult("5")
		imp.ExpectCallTo.Print("5").Resolve()

		mock := &mockT{}
		printSumImp.t = mock

		defer func() {
			_ = recover()

			if !mock.failed {
				t.Error("Expected failure")
			}
		}()

		printSumImp.ExpectPanicWith("err")
	})

	t.Run("Fails when panic value mismatch", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(1, 2, imp.Mock)

		// Inject panic with different value
		imp.ExpectCallTo.Add(1, 2).InjectPanic("foo")

		mock := &mockT{}
		printSumImp.t = mock

		defer func() {
			_ = recover()

			if !mock.failed {
				t.Error("Expected failure")
			}
		}()

		printSumImp.ExpectPanicWith("bar")
	})
}

// TestPrintSumImp_Response_Methods tests the Response struct methods.
func TestPrintSumImp_Response_Methods(t *testing.T) {
	t.Parallel()

	t.Run("Type returns ReturnEvent for returns", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(2, 3, imp.Mock)

		imp.ExpectCallTo.Add(2, 3).InjectResult(5)
		imp.ExpectCallTo.Format(5).InjectResult("5")
		imp.ExpectCallTo.Print("5").Resolve()

		resp := printSumImp.GetResponse()
		if resp.Type() != returnEventType {
			t.Errorf("Expected ReturnEvent, got %v", resp.Type())
		}
	})

	t.Run("Type returns PanicEvent for panics", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(1, 2, imp.Mock)

		imp.ExpectCallTo.Add(1, 2).InjectPanic("err")

		resp := printSumImp.GetResponse()
		if resp.Type() != panicEventType {
			t.Errorf("Expected PanicEvent, got %v", resp.Type())
		}
	})

	t.Run("AsReturn returns values", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(2, 3, imp.Mock)

		imp.ExpectCallTo.Add(2, 3).InjectResult(5)
		imp.ExpectCallTo.Format(5).InjectResult("5")
		imp.ExpectCallTo.Print("5").Resolve()

		resp := printSumImp.GetResponse()
		got := resp.AsReturn()

		if len(got) != 3 {
			t.Fatalf("Expected 3 values, got %d", len(got))
		}

		if got[0] != 2 || got[1] != 3 || got[2] != "5" {
			t.Errorf("AsReturn failed, got %v", got)
		}
	})
}

// TestPrintSumImp_GetResponse_Caching tests that GetResponse caches results.
func TestPrintSumImp_GetResponse_Caching(t *testing.T) {
	t.Parallel()

	t.Run("Caches return values", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(2, 3, imp.Mock)

		imp.ExpectCallTo.Add(2, 3).InjectResult(5)
		imp.ExpectCallTo.Format(5).InjectResult("5")
		imp.ExpectCallTo.Print("5").Resolve()

		// First call waits for result
		resp1 := printSumImp.GetResponse()
		if resp1.Type() != returnEventType {
			t.Errorf("Expected ReturnEvent")
		}

		// Second call should use cached value
		resp2 := printSumImp.GetResponse()
		if resp2.Type() != returnEventType {
			t.Errorf("Expected ReturnEvent on second call")
		}
	})

	t.Run("Caches panic values", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(1, 2, imp.Mock)

		imp.ExpectCallTo.Add(1, 2).InjectPanic("err")

		resp1 := printSumImp.GetResponse()
		if resp1.Type() != panicEventType {
			t.Errorf("Expected PanicEvent")
		}

		resp2 := printSumImp.GetResponse()
		if resp2.Type() != panicEventType {
			t.Errorf("Expected PanicEvent cached")
		}
	})
}

// TestPrintSumImp_ExpectReturnedValues_Caching tests that ExpectReturnedValues uses cached values.
func TestPrintSumImp_ExpectReturnedValues_Caching(t *testing.T) {
	t.Parallel()

	t.Run("Uses cached return value", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(2, 3, imp.Mock)

		imp.ExpectCallTo.Add(2, 3).InjectResult(5)
		imp.ExpectCallTo.Format(5).InjectResult("5")
		imp.ExpectCallTo.Print("5").Resolve()

		// First call caches the result
		printSumImp.ExpectReturnedValues(2, 3, "5")
		// Second call should use cached result (no deadlock/block)
		printSumImp.ExpectReturnedValues(2, 3, "5")
	})

	t.Run("Fails on cached panic", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(1, 2, imp.Mock)

		imp.ExpectCallTo.Add(1, 2).InjectPanic("err")

		// First call caches the panic
		_ = printSumImp.GetResponse()

		mock := &mockT{}
		printSumImp.t = mock

		defer func() {
			_ = recover()

			if !mock.failed {
				t.Error("Expected failure")
			}
		}()

		// Now ExpectReturnedValues should fail because we have a cached panic
		printSumImp.ExpectReturnedValues(1, 2, "3")
	})
}

// TestPrintSumImp_ExpectPanicWith_Caching tests that ExpectPanicWith uses cached values.
func TestPrintSumImp_ExpectPanicWith_Caching(t *testing.T) {
	t.Parallel()

	t.Run("Uses cached panic value", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(1, 2, imp.Mock)

		imp.ExpectCallTo.Add(1, 2).InjectPanic("err")

		// First call caches the result
		printSumImp.ExpectPanicWith("err")
		// Second call should use cached result
		printSumImp.ExpectPanicWith("err")
	})

	t.Run("Fails on cached return", func(t *testing.T) {
		t.Parallel()

		imp := NewIntOpsImp(t)
		printSumImp := NewPrintSumImp(t, run.PrintSum).Start(2, 3, imp.Mock)

		imp.ExpectCallTo.Add(2, 3).InjectResult(5)
		imp.ExpectCallTo.Format(5).InjectResult("5")
		imp.ExpectCallTo.Print("5").Resolve()

		// First call caches the return
		_ = printSumImp.GetResponse()

		mock := &mockT{}
		printSumImp.t = mock

		defer func() {
			_ = recover()

			if !mock.failed {
				t.Error("Expected failure")
			}
		}()

		// Now ExpectPanicWith should fail because we have a cached return
		printSumImp.ExpectPanicWith("err")
	})
}
