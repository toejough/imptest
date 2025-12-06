package imptest_test

import (
	"fmt"
	"testing"

	"github.com/toejough/imptest"
)

// Helper to capture test failures.
type mockT struct {
	testing.T

	failed bool
	msg    string
}

func (m *mockT) Fatalf(format string, args ...any) {
	m.failed = true
	m.msg = fmt.Sprintf(format, args...)
	// In a real test we'd stop here, but for testing our test helper we just record it
	panic("mockT failed: " + m.msg)
}

func (m *mockT) Helper() {}

func TestStart_Validation(t *testing.T) {
	t.Parallel()

	t.Run("Panics if fn is not a function", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic, got none")
			}
		}()

		imptest.Start(t, "not a function")
	})

	t.Run("Panics if args count mismatch", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic, got none")
			}
		}()

		fn := func(_ int) {}
		imptest.Start(t, fn) // Missing arg
	})

	t.Run("Panics if arg type mismatch", func(t *testing.T) {
		t.Parallel()

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic, got none")
			}
		}()

		fn := func(_ int) {}
		imptest.Start(t, fn, "not an int")
	})
}

func TestStart_Success(t *testing.T) {
	t.Parallel()

	t.Run("Function with args and returns", func(t *testing.T) {
		t.Parallel()

		fn := func(a, b int) int { return a + b }
		inv := imptest.Start(t, fn, 2, 3)
		inv.ExpectReturnedValues(5)
	})

	t.Run("Function with multiple returns", func(t *testing.T) {
		t.Parallel()

		fn := func() (int, string) { return 42, "foo" }
		inv := imptest.Start(t, fn)
		inv.ExpectReturnedValues(42, "foo")
	})

	t.Run("Function that panics", func(t *testing.T) {
		t.Parallel()

		fn := func() { panic("oops") }
		inv := imptest.Start(t, fn)
		inv.ExpectPanicWith("oops")
	})
}

func TestExpectReturnedValues_WrongEventType(t *testing.T) {
	t.Parallel()

	// 1. Wrong event type (Panic instead of Return)
	t.Run("Fails when event is panic", func(t *testing.T) {
		t.Parallel()

		fn := func() { panic("oops") }
		inv := imptest.Start(t, fn)

		mock := &mockT{}
		// We have to wait for the event to be ready before swapping?
		// No, GetResponse waits. Swapping immediately is safe because
		// Expect* calls GetResponse using inv.t only for reporting?
		// No, Expect* calls GetResponse, then uses t to report errors.

		// Race condition warning: Start runs in goroutine.
		// inv.t is used ONLY in Expect* methods. Start uses the passed t (which becomes inv.t, but we overwrite it).
		// Start does NOT use inv.t. It uses 't' param passed to it.
		// So overwriting inv.t is safe.

		inv.T = mock

		// Run Expect in a way that doesn't kill the test process (mock captures Fatalf)
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

		inv.ExpectReturnedValues(1)
	})
}

func TestExpectReturnedValues_ValueMismatch(t *testing.T) {
	t.Parallel()

	// 2. Wrong number of values
	t.Run("Fails when count mismatch", func(t *testing.T) {
		t.Parallel()

		fn := func() int { return 1 }
		inv := imptest.Start(t, fn)

		mock := &mockT{}
		inv.T = mock

		defer func() {
			_ = recover()

			if !mock.failed {
				t.Error("Expected failure")
			}
		}()

		inv.ExpectReturnedValues(1, 2)
	})

	// 3. Values mismatch
	t.Run("Fails when values mismatch", func(t *testing.T) {
		t.Parallel()

		fn := func() int { return 1 }
		inv := imptest.Start(t, fn)

		mock := &mockT{}
		inv.T = mock

		defer func() {
			_ = recover()

			if !mock.failed {
				t.Error("Expected failure")
			}
		}()

		inv.ExpectReturnedValues(2)
	})
}

func TestExpectPanicWith_Failures(t *testing.T) {
	t.Parallel()

	// 1. Return instead of Panic
	t.Run("Fails when event is return", func(t *testing.T) {
		t.Parallel()

		fn := func() {}
		inv := imptest.Start(t, fn)

		mock := &mockT{}
		inv.T = mock

		defer func() {
			_ = recover()

			if !mock.failed {
				t.Error("Expected failure")
			}
		}()

		inv.ExpectPanicWith("err")
	})

	// 2. Panic value mismatch
	t.Run("Fails when panic value mismatch", func(t *testing.T) {
		t.Parallel()

		fn := func() { panic("foo") }
		inv := imptest.Start(t, fn)

		mock := &mockT{}
		inv.T = mock

		defer func() {
			_ = recover()

			if !mock.failed {
				t.Error("Expected failure")
			}
		}()

		inv.ExpectPanicWith("bar")
	})
}

func TestResponse_Methods(t *testing.T) {
	t.Parallel()

	t.Run("Type returns correct values", func(t *testing.T) {
		t.Parallel()

		r1 := &imptest.TestResponse{EventType: imptest.ReturnEvent}
		if r1.Type() != imptest.ReturnEvent {
			t.Errorf("Expected ReturnEvent, got %v", r1.Type())
		}

		r2 := &imptest.TestResponse{EventType: imptest.PanicEvent}
		if r2.Type() != imptest.PanicEvent {
			t.Errorf("Expected PanicEvent, got %v", r2.Type())
		}

		r3 := &imptest.TestResponse{} // Empty
		if r3.Type() != "stub" {
			t.Errorf("Expected stub, got %v", r3.Type())
		}
	})

	t.Run("AsReturn returns values", func(t *testing.T) {
		t.Parallel()

		retVal := imptest.TestReturn{1, "a"}
		r := &imptest.TestResponse{ReturnVal: &retVal}

		got := r.AsReturn()
		if len(got) != 2 || got[0] != 1 || got[1] != "a" {
			t.Errorf("AsReturn failed, got %v", got)
		}
	})

	t.Run("AsReturn returns empty on nil", func(t *testing.T) {
		t.Parallel()

		r := &imptest.TestResponse{}

		got := r.AsReturn()
		if len(got) != 0 {
			t.Errorf("Expected empty return, got %v", got)
		}
	})
}

func TestGetResponse_Caching(t *testing.T) {
	t.Parallel()

	fn := func() int { return 123 }
	inv := imptest.Start(t, fn)

	// First call waits for channel
	resp1 := inv.GetResponse()
	if resp1.Type() != imptest.ReturnEvent {
		t.Errorf("Expected ReturnEvent")
	}

	// Second call should use cached 'returned' field
	// We can verify this by modifying the internal channel or struct if we really wanted,
	// but purely observing behavior: it should return same result immediately.
	resp2 := inv.GetResponse()
	if resp2 != nil && resp2.Type() != imptest.ReturnEvent {
		t.Errorf("Expected ReturnEvent on second call")
	}

	// Check pointers (optional, depending on implementation detail)
	// The implementation returns a new &TestResponse struct each time but points to same data
}

// To hit the "panic" branch of GetResponse caching.
func TestGetResponse_Caching_Panic(t *testing.T) {
	t.Parallel()

	fn := func() { panic("err") }
	inv := imptest.Start(t, fn)

	resp1 := inv.GetResponse()
	if resp1.Type() != imptest.PanicEvent {
		t.Errorf("Expected PanicEvent")
	}

	resp2 := inv.GetResponse()
	if resp2.Type() != imptest.PanicEvent {
		t.Errorf("Expected PanicEvent cached")
	}
}
