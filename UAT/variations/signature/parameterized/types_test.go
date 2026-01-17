package parameterized_test

import (
	"testing"

	"github.com/toejough/imptest/UAT/variations/signature/parameterized"
)

//go:generate impgen parameterized.DataProcessor --dependency

func TestDataProcessor_ProcessContainer(t *testing.T) {
	t.Parallel()

	mock, imp := MockDataProcessor(t)

	data := parameterized.Container[string]{Value: "test"}

	// Expect call with parameterized type
	go func() {
		call := imp.ProcessContainer.ArgsEqual(data)
		call.Return(nil)
	}()

	// Call the mock
	err := mock.ProcessContainer(data)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestDataProcessor_ProcessPair(t *testing.T) {
	t.Parallel()

	mock, imp := MockDataProcessor(t)

	pair := parameterized.Pair[int, bool]{Key: 42, Value: true}

	// Expect call with parameterized type
	go func() {
		call := imp.ProcessPair.ArgsEqual(pair)
		call.Return("processed")
	}()

	// Call the mock
	result := mock.ProcessPair(pair)

	if result != "processed" {
		t.Fatalf("expected 'processed', got %q", result)
	}
}

func TestDataProcessor_ReturnContainer(t *testing.T) {
	t.Parallel()

	mock, imp := MockDataProcessor(t)

	expected := parameterized.Container[int]{Value: 99}

	// Expect call and return parameterized type
	go func() {
		call := imp.ReturnContainer.Called()
		call.Return(expected)
	}()

	// Call the mock
	result := mock.ReturnContainer()

	if result.Value != 99 {
		t.Fatalf("expected Container{Value: 99}, got Container{Value: %d}", result.Value)
	}
}
