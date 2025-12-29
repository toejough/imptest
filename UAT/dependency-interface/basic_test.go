// Package dependencyinterface_test demonstrates mocking interface dependencies with imptest v2.
//
// Test Taxonomy Coverage:
//
//	What:     Target x | Dependency ✓
//	Type:     Function x | Interface ✓
//	Mode:     Ordered ✓ | Unordered x
//	Matching: Exact ✓ | Matcher ✓
//	Outcome:  Return ✓ | Panic ✓
//	Source:   Type ✓ | Definition x
//
// Mock Sources (interface types used for code generation):
//
//	MockDataStore ← type DataStore interface
package dependencyinterface_test

import (
	"errors"
	"testing"

	"github.com/onsi/gomega"
	"github.com/toejough/imptest/imptest"
)

// DataStore defines storage operations
type DataStore interface {
	Get(id int) (string, error)
	Save(id int, data string) error
	Delete(id int) error
}

// Service uses a DataStore dependency
type Service struct {
	store DataStore
}

func (s *Service) LoadAndProcess(id int) (string, error) {
	data, err := s.store.Get(id)
	if err != nil {
		return "", err
	}

	return "processed: " + data, nil
}

func (s *Service) SaveProcessed(id int, input string) error {
	processed := "processed: " + input
	return s.store.Save(id, processed)
}

// TestDependencyInterface_Ordered_Exact_Args demonstrates the conversational pattern
// with ordered expectations and exact argument matching
func TestDependencyInterface_Ordered_Exact_Args(t *testing.T) {
	t.Parallel()

	imp := imptest.NewImp(t)

	// Create mock for the dependency interface
	store := MockDataStore(imp)

	// Create service with mock
	svc := &Service{store: store.Interface()}

	// Start execution (runs in goroutine)
	result := WrapLoadAndProcess(imp, svc.LoadAndProcess).Start(42)

	// THEN verify the call and inject response (the "conversation")
	call := store.Get.ExpectCalledWithExactly(42)
	call.InjectReturnValues("test data", nil)

	// Verify the result
	result.ExpectReturnsEqual("processed: test data", nil)
}

// TestDependencyInterface_Ordered_GetArgs demonstrates getting actual arguments
func TestDependencyInterface_Ordered_GetArgs(t *testing.T) {
	t.Parallel()

	imp := imptest.NewImp(t)
	store := MockDataStore(imp)
	svc := &Service{store: store.Interface()}

	result := WrapSaveProcessed(imp, svc.SaveProcessed).Start(42, "input")

	call := store.Save.ExpectCalledWithExactly(42, "processed: input")
	call.InjectReturnValues(nil)

	result.ExpectReturnsEqual(nil)

	// Get the actual arguments
	args := call.GetArgs()

	if args.A1 != 42 {
		t.Errorf("expected id 42, got %d", args.A1)
	}

	if args.A2 != "processed: input" {
		t.Errorf("expected 'processed: input', got %q", args.A2)
	}
}

// TestDependencyInterface_Ordered_InjectError demonstrates injecting errors
func TestDependencyInterface_Ordered_InjectError(t *testing.T) {
	t.Parallel()

	imp := imptest.NewImp(t)
	store := MockDataStore(imp)
	svc := &Service{store: store.Interface()}

	result := WrapLoadAndProcess(imp, svc.LoadAndProcess).Start(42)

	expectedErr := errors.New("not found")
	call := store.Get.ExpectCalledWithExactly(42)
	call.InjectReturnValues("", expectedErr)

	result.ExpectReturnsEqual("", expectedErr)
}

// TestDependencyInterface_Ordered_InjectPanic demonstrates injecting a panic
func TestDependencyInterface_Ordered_InjectPanic(t *testing.T) {
	t.Parallel()

	imp := imptest.NewImp(t)
	store := MockDataStore(imp)
	svc := &Service{store: store.Interface()}

	result := WrapLoadAndProcess(imp, svc.LoadAndProcess).Start(42)

	call := store.Get.ExpectCalledWithExactly(42)
	call.InjectPanicValue("database connection lost")

	result.ExpectPanicEquals("database connection lost")
}

// TestDependencyInterface_Ordered_Matcher_Args demonstrates matcher validation
func TestDependencyInterface_Ordered_Matcher_Args(t *testing.T) {
	t.Parallel()

	imp := imptest.NewImp(t)
	store := MockDataStore(imp)
	svc := &Service{store: store.Interface()}

	result := WrapLoadAndProcess(imp, svc.LoadAndProcess).Start(99)

	// Expect Get with argument matching a condition using gomega matcher
	call := store.Get.ExpectCalledWithMatches(gomega.BeNumerically(">", 0))
	call.InjectReturnValues("data", nil)

	result.ExpectReturnsEqual("processed: data", nil)
}

// TestDependencyInterface_Ordered_MultipleMethod demonstrates multiple method calls
func TestDependencyInterface_Ordered_MultipleMethod(t *testing.T) {
	t.Parallel()

	imp := imptest.NewImp(t)
	store := MockDataStore(imp)
	svc := &Service{store: store.Interface()}

	// First call
	result1 := WrapLoadAndProcess(imp, svc.LoadAndProcess).Start(1)

	call1 := store.Get.ExpectCalledWithExactly(1)
	call1.InjectReturnValues("data1", nil)

	result1.ExpectReturnsEqual("processed: data1", nil)

	// Second call
	result2 := WrapLoadAndProcess(imp, svc.LoadAndProcess).Start(2)

	call2 := store.Get.ExpectCalledWithExactly(2)
	call2.InjectReturnValues("data2", nil)

	result2.ExpectReturnsEqual("processed: data2", nil)
}
