package testpkgimport_test

import (
	"testing"
)

//go:generate go run ../../impgen --dependency Service

func TestService_Execute(t *testing.T) {
	mock := MockService(t)

	input := "test input"
	expectedOutput := "processed output"

	// Expect call with specific input
	go func() {
		call := mock.Execute.ExpectCalledWithExactly(input)
		call.InjectReturnValues(expectedOutput, nil)
	}()

	// Call the mock
	output, err := mock.Interface().Execute(input)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if output != expectedOutput {
		t.Fatalf("expected %q, got %q", expectedOutput, output)
	}
}

func TestService_Validate(t *testing.T) {
	mock := MockService(t)

	input := "valid input"

	// Expect call and return true
	go func() {
		call := mock.Validate.ExpectCalledWithExactly(input)
		call.InjectReturnValues(true)
	}()

	// Call the mock
	result := mock.Interface().Validate(input)

	if !result {
		t.Fatalf("expected true, got false")
	}
}
