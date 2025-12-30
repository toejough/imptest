package parameterized_test

import (
	"testing"

	parameterized "github.com/toejough/imptest/UAT/21-parameterized-types"
)

//go:generate go run ../../impgen --dependency parameterized.DataProcessor

func TestDataProcessor_ProcessContainer(t *testing.T) {
	t.Parallel()

	mock := MockDataProcessor(t)

	data := parameterized.Container[string]{Value: "test"}

	// Expect call with parameterized type
	go func() {
		call := mock.ProcessContainer.ExpectCalledWithExactly(data)
		call.InjectReturnValues(nil)
	}()

	// Call the mock
	err := mock.Interface().ProcessContainer(data)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestDataProcessor_ProcessPair(t *testing.T) {
	t.Parallel()

	mock := MockDataProcessor(t)

	pair := parameterized.Pair[int, bool]{Key: 42, Value: true}

	// Expect call with parameterized type
	go func() {
		call := mock.ProcessPair.ExpectCalledWithExactly(pair)
		call.InjectReturnValues("processed")
	}()

	// Call the mock
	result := mock.Interface().ProcessPair(pair)

	if result != "processed" {
		t.Fatalf("expected 'processed', got %q", result)
	}
}

func TestDataProcessor_ReturnContainer(t *testing.T) {
	t.Parallel()

	mock := MockDataProcessor(t)

	expected := parameterized.Container[int]{Value: 99}

	// Expect call and return parameterized type
	go func() {
		call := mock.ReturnContainer.ExpectCalledWithExactly()
		call.InjectReturnValues(expected)
	}()

	// Call the mock
	result := mock.Interface().ReturnContainer()

	if result.Value != 99 {
		t.Fatalf("expected Container{Value: 99}, got Container{Value: %d}", result.Value)
	}
}
