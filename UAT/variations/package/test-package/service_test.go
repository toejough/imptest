package testpkgimport_test

import (
	"testing"
)

//go:generate impgen Service --dependency

func TestService_Execute(t *testing.T) {
	t.Parallel()

	mock := MockService(t)

	input := "test input"
	expectedOutput := "processed output"

	// Expect call with specific input
	go func() {
		call := mock.Method.Execute.ExpectCalledWithExactly(input)
		call.InjectReturnValues(expectedOutput, nil)
	}()

	// Call the mock
	output, err := mock.Mock.Execute(input)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if output != expectedOutput {
		t.Fatalf("expected %q, got %q", expectedOutput, output)
	}
}

func TestService_Validate(t *testing.T) {
	t.Parallel()

	mock := MockService(t)

	input := "valid input"

	// Expect call and return true
	go func() {
		call := mock.Method.Validate.ExpectCalledWithExactly(input)
		call.InjectReturnValues(true)
	}()

	// Call the mock
	result := mock.Mock.Validate(input)

	if !result {
		t.Fatalf("expected true, got false")
	}
}
