package dotimports_test

import (
	"errors"
	"testing"

	. "github.com/toejough/imptest/UAT/variations/package/dot-imports/helpers" //nolint:revive // Dot import intentional for testing dot-import support
)

//go:generate impgen Storage --dependency
//go:generate impgen Processor --dependency

// TestDotImportedProcessor verifies multiple dot-imported interfaces work.
func TestDotImportedProcessor(t *testing.T) {
	t.Parallel()

	mock, imp := MockProcessor(t)

	// Launch goroutine
	go func() {
		_ = mock.Process("input data")
	}()

	// Verify and inject return
	call := imp.Process.ArgsEqual("input data")
	call.Return("processed data")

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
		mock, imp := MockStorage(t)

		// Launch goroutine that calls Save
		go func() {
			_ = mock.Save("key1", "value1")
		}()

		// Verify the mock received the call
		call := imp.Save.ArgsEqual("key1", "value1")
		call.Return(nil)

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
		mock, imp := MockStorage(t)

		// Launch goroutine
		go func() {
			_ = mock.Save("bad_key", "bad_value")
		}()

		// Inject error return
		testErr := errors.New("save failed")
		call := imp.Save.ArgsEqual("bad_key", "bad_value")
		call.Return(testErr)
	})

	t.Run("Load", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockStorage(t)

		// Launch goroutine
		go func() {
			_, _ = mock.Load("key2")
		}()

		// Verify and inject return values
		call := imp.Load.ArgsEqual("key2")
		call.Return("loaded_value", nil)

		// Verify args
		args := call.GetArgs()
		if args.Key != "key2" {
			t.Fatalf("expected key = 'key2', got %q", args.Key)
		}
	})

	t.Run("LoadWithError", func(t *testing.T) {
		t.Parallel()
		mock, imp := MockStorage(t)

		// Launch goroutine
		go func() {
			_, _ = mock.Load("missing_key")
		}()

		// Inject error return
		testErr := errors.New("not found")
		call := imp.Load.ArgsEqual("missing_key")
		call.Return("", testErr)
	})
}

// unexported variables.
var (
	_ Storage   = (*mockStorageImpl)(nil)
	_ Processor = (*mockProcessorImpl)(nil)
)
