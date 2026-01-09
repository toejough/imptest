package dotimports_test

import (
	"errors"
	"testing"

	. "github.com/toejough/imptest/UAT/variations/package/dot-imports/helpers"
)

//go:generate impgen Storage --dependency
//go:generate impgen Processor --dependency

// TestDotImportedProcessor verifies multiple dot-imported interfaces work.
func TestDotImportedProcessor(t *testing.T) {
	t.Parallel()

	mock := MockProcessor(t)

	// Launch goroutine
	go func() {
		_ = mock.Mock.Process("input data")
	}()

	// Verify and inject return
	call := mock.Method.Process.ExpectCalledWithExactly("input data")
	call.InjectReturnValues("processed data")

	// Verify args
	args := call.GetArgs()
	if args.Input != "input data" {
		t.Fatalf("expected input = 'input data', got %q", args.Input)
	}
}

// TestDotImportedStorage demonstrates that impgen can generate mocks for
// interfaces available via dot imports.
//
//nolint:funlen // Comprehensive test with multiple sub-tests
func TestDotImportedStorage(t *testing.T) {
	t.Parallel()

	t.Run("Save", func(t *testing.T) {
		t.Parallel()
		mock := MockStorage(t)

		// Launch goroutine that calls Save
		go func() {
			_ = mock.Mock.Save("key1", "value1")
		}()

		// Verify the mock received the call
		call := mock.Method.Save.ExpectCalledWithExactly("key1", "value1")
		call.InjectReturnValues(nil)

		// Verify args can be retrieved
		args := call.GetArgs()
		if args.Key != "key1" {
			t.Fatalf("expected key = 'key1', got %q", args.Key)
		}

		if args.Value != "value1" {
			t.Fatalf("expected value = 'value1', got %q", args.Value)
		}
	})

	t.Run("SaveWithError", func(t *testing.T) {
		t.Parallel()
		mock := MockStorage(t)

		// Launch goroutine
		go func() {
			_ = mock.Mock.Save("bad_key", "bad_value")
		}()

		// Inject error return
		testErr := errors.New("save failed")
		call := mock.Method.Save.ExpectCalledWithExactly("bad_key", "bad_value")
		call.InjectReturnValues(testErr)
	})

	t.Run("Load", func(t *testing.T) {
		t.Parallel()
		mock := MockStorage(t)

		// Launch goroutine
		go func() {
			_, _ = mock.Mock.Load("key2")
		}()

		// Verify and inject return values
		call := mock.Method.Load.ExpectCalledWithExactly("key2")
		call.InjectReturnValues("loaded_value", nil)

		// Verify args
		args := call.GetArgs()
		if args.Key != "key2" {
			t.Fatalf("expected key = 'key2', got %q", args.Key)
		}
	})

	t.Run("LoadWithError", func(t *testing.T) {
		t.Parallel()
		mock := MockStorage(t)

		// Launch goroutine
		go func() {
			_, _ = mock.Mock.Load("missing_key")
		}()

		// Inject error return
		testErr := errors.New("not found")
		call := mock.Method.Load.ExpectCalledWithExactly("missing_key")
		call.InjectReturnValues("", testErr)
	})
}

// unexported variables.
var (
	_ Storage   = (*mockStorageImpl)(nil)
	_ Processor = (*mockProcessorImpl)(nil)
)
