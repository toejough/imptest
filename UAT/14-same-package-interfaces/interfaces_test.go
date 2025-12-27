package samepackage_test

//go:generate ../../bin/impgen DataProcessor
//go:generate ../../bin/impgen DataSource
//go:generate ../../bin/impgen DataSink

import (
	"errors"
	"testing"

	samepackage "github.com/toejough/imptest/UAT/14-same-package-interfaces"
)

// TestSamePackageInterfaces tests that interfaces using other interfaces
// from the same package work correctly in generated mocks.
//
//nolint:varnamelen // Standard Go testing convention
func TestSamePackageInterfaces(t *testing.T) {
	t.Parallel()

	// Create mocks for all interfaces
	processor := NewDataProcessorImp(t)
	source := NewDataSourceImp(t)
	sink := NewDataSinkImp(t)

	// Test that we can use same-package interface types in expectations
	testData := []byte("test data")
	testErr := errors.New("test error")

	// Set up source to return test data
	go func() {
		source.ExpectCallIs.GetData().InjectResults(testData, nil)
	}()

	// Set up sink to accept data
	go func() {
		sink.ExpectCallIs.PutData().ExpectArgsAre(testData).InjectResult(testErr)
	}()

	// Set up processor expectation with same-package interface arguments
	go func() {
		call := processor.ExpectCallIs.Process().ExpectArgsShould(
			samepackage.DataSource(source.Mock),
			samepackage.DataSink(sink.Mock),
		)
		call.InjectResult(testErr)
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
//nolint:varnamelen // Standard Go testing convention
func TestTransformReturnsInterface(t *testing.T) {
	t.Parallel()

	processor := NewDataProcessorImp(t)
	inputSource := NewDataSourceImp(t)
	outputSource := NewDataSourceImp(t)

	// Set up processor to return a different source
	go func() {
		call := processor.ExpectCallIs.Transform().ExpectArgsShould(
			samepackage.DataSource(inputSource.Mock),
		)
		call.InjectResults(outputSource.Mock, nil)
	}()

	// Call transform
	result, err := processor.Mock.Transform(inputSource.Mock)
	// Verify result
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != outputSource.Mock {
		t.Errorf("Expected output source, got different instance")
	}
}
