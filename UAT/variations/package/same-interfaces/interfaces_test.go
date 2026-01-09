package samepackage_test

//go:generate impgen samepackage.DataProcessor --dependency
//go:generate impgen samepackage.DataSource --dependency
//go:generate impgen samepackage.DataSink --dependency

import (
	"errors"
	"testing"

	"github.com/toejough/imptest/imptest"
)

// TestSamePackageInterfaces tests that interfaces using other interfaces
// from the same package work correctly in generated mocks.
//
// Key Requirements Met:
//  1. Same-Package Interface References: Generated mocks must handle interfaces
//     that use other interfaces from the same package in method signatures.
//  2. No Import Cycles: The generator must avoid creating import cycles when
//     mocking interfaces that reference each other.
func TestSamePackageInterfaces(t *testing.T) {
	t.Parallel()

	// Create mocks for all interfaces
	processor := MockDataProcessor(t)
	source := MockDataSource(t)
	sink := MockDataSink(t)

	// Test that we can use same-package interface types in expectations
	testData := []byte("test data")
	testErr := errors.New("test error")

	// Set up source to return test data
	go func() {
		source.Method.GetData.ExpectCalledWithExactly().InjectReturnValues(testData, nil)
	}()

	// Set up sink to accept data
	go func() {
		sink.Method.PutData.ExpectCalledWithExactly(testData).InjectReturnValues(testErr)
	}()

	// Set up processor expectation with same-package interface arguments
	go func() {
		processor.Method.Process.ExpectCalledWithMatches(
			imptest.Any(),
			imptest.Any(),
		).InjectReturnValues(testErr)
	}()

	// Call the processor
	result := processor.Mock.Process(source.Mock, sink.Mock)

	// Verify result
	if !errors.Is(result, testErr) {
		t.Errorf("Expected error %v, got %v", testErr, result)
	}
}

// TestTransformReturnsInterface tests methods that return same-package interfaces.
//
// Key Requirements Met:
//  1. Interface Return Types: Mocks correctly handle methods that return
//     other interfaces from the same package.
func TestTransformReturnsInterface(t *testing.T) {
	t.Parallel()

	processor := MockDataProcessor(t)
	inputSource := MockDataSource(t)
	outputSource := MockDataSource(t)

	// Get the interface value to return
	expectedOutput := outputSource.Mock

	// Set up processor to return a different source
	go func() {
		processor.Method.Transform.ExpectCalledWithMatches(
			imptest.Any(),
		).InjectReturnValues(expectedOutput, nil)
	}()

	// Call transform
	result, err := processor.Mock.Transform(inputSource.Mock)
	// Verify result
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != expectedOutput {
		t.Errorf("Expected output source, got different instance")
	}
}
