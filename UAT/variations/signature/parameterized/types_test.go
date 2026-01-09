package parameterized_test

import (
	"testing"

	"github.com/toejough/imptest/UAT/variations/signature/parameterized"
)

//go:generate impgen parameterized.DataProcessor --dependency

func TestDataProcessor_ProcessContainer(t *testing.T) {
	t.Parallel()

	mock := MockDataProcessor(t)

	data := parameterized.Container[string]{Value: "test"}

	// Expect call with parameterized type
	go func() {
		call := mock.Method.ProcessContainer.ExpectCalledWithExactly(data)
		call.InjectReturnValues(nil)
	}()

	// Call the mock
	err := mock.Mock.ProcessContainer(data)
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
		call := mock.Method.ProcessPair.ExpectCalledWithExactly(pair)
		call.InjectReturnValues("processed")
	}()

	// Call the mock
	result := mock.Mock.ProcessPair(pair)

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
		call := mock.Method.ReturnContainer.ExpectCalledWithExactly()
		call.InjectReturnValues(expected)
	}()

	// Call the mock
	result := mock.Mock.ReturnContainer()

	if result.Value != 99 {
		t.Fatalf("expected Container{Value: 99}, got Container{Value: %d}", result.Value)
	}
}
