package imptest

import "testing"

// TestGenericCallDone tests GenericCall.Done.
func TestGenericCallDone(t *testing.T) {
	t.Parallel()

	call := &GenericCall{}
	if call.Done() {
		t.Error("expected Done() to return false initially")
	}

	call.MarkDone()

	if !call.Done() {
		t.Error("expected Done() to return true after MarkDone()")
	}
}

// TestGenericCallName tests GenericCall.Name.
func TestGenericCallName(t *testing.T) {
	t.Parallel()

	call := &GenericCall{MethodName: "TestMethod"}
	if call.Name() != "TestMethod" {
		t.Errorf("expected Name() to return 'TestMethod', got %q", call.Name())
	}
}

// TestImpFatalf tests that Imp.Fatalf delegates to the underlying test reporter.
func TestImpFatalf(t *testing.T) {
	t.Parallel()

	called := false
	mockReporter := &mockTestReporter{
		fatal: func(_ string, _ ...any) {
			called = true
		},
	}

	imp := &Imp{t: mockReporter}
	imp.Fatalf("test message")

	if !called {
		t.Error("expected Fatalf to be called on underlying reporter")
	}
}

// TestTesterAdapterFatalf tests that testerAdapter.Fatalf delegates correctly.
func TestTesterAdapterFatalf(t *testing.T) {
	t.Parallel()

	called := false
	mockReporter := &mockTestReporter{
		fatal: func(_ string, _ ...any) {
			called = true
		},
	}

	adapter := &testerAdapter{t: mockReporter}
	adapter.Fatalf("test")

	if !called {
		t.Error("expected Fatalf to be called on underlying reporter")
	}
}

type mockTestReporter struct {
	fatal func(string, ...any)
}

func (m *mockTestReporter) Fatalf(format string, args ...any) {
	if m.fatal != nil {
		m.fatal(format, args...)
	}
}

func (m *mockTestReporter) Helper() {}
