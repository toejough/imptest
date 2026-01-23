package samepackage_test

//go:generate impgen DataProcessor --dependency
//go:generate impgen DataSource --dependency
//go:generate impgen DataSink --dependency

import (
	"errors"
	"testing"

	. "github.com/toejough/imptest/match" //nolint:revive // Dot import for matcher DSL
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
	processor, procImp := MockDataProcessor(t)
	source, _ := MockDataSource(t)
	sink, _ := MockDataSink(t)

	// Test that we can use same-package interface types in expectations
	testErr := errors.New("test error")

	// Set up processor expectation with same-package interface arguments
	// Note: The mock's Process method doesn't actually call source.GetData() or sink.PutData(),
	// it just receives the interface values as arguments and waits for the test to inject a return.
	go func() {
		procImp.Process.ArgsShould(
			BeAny,
			BeAny,
		).Return(testErr)
	}()

	// Call the processor - this passes the mock interfaces as arguments
	result := processor.Process(source, sink)

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

	processor, procImp := MockDataProcessor(t)
	inputSource, _ := MockDataSource(t)
	outputSource, _ := MockDataSource(t)

	// Get the interface value to return
	expectedOutput := outputSource

	// Set up processor to return a different source
	go func() {
		procImp.Transform.ArgsShould(
			BeAny,
		).Return(expectedOutput, nil)
	}()

	// Call transform
	result, err := processor.Transform(inputSource)
	// Verify result
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != expectedOutput {
		t.Errorf("Expected output source, got different instance")
	}
}
