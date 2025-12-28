package dependencyinterface_test

import (
	"errors"
	"testing"

	imptest "github.com/toejough/imptest/imptest/v2"
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

// TestDependencyInterface_Ordered_Exact_Args demonstrates mocking an interface dependency
// with ordered expectations and exact argument matching
func TestDependencyInterface_Ordered_Exact_Args(t *testing.T) {
	imp := imptest.NewImp(t)

	// Create mock for the dependency interface
	storeMock := imptest.NewDependencyInterface[DataStore](imp)

	// Expect Get to be called with exact arguments
	call := storeMock.Get.ExpectCalledWithExactly(42)

	// Inject the return values
	call.InjectReturnValues("test data", nil)

	// Execute the function under test with the mock
	svc := &Service{store: storeMock.Interface()}
	result, err := svc.LoadAndProcess(42)

	// Verify the business logic result
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "processed: test data" {
		t.Errorf("expected 'processed: test data', got %q", result)
	}
}

// TestDependencyInterface_Ordered_Matcher_Args demonstrates matcher validation
func TestDependencyInterface_Ordered_Matcher_Args(t *testing.T) {
	imp := imptest.NewImp(t)
	storeMock := imptest.NewDependencyInterface[DataStore](imp)

	// Expect Get with argument matching a condition
	call := storeMock.Get.ExpectCalledWithMatches(imptest.Satisfies(func(v any) bool {
		id, ok := v.(int)
		return ok && id > 0
	}))

	call.InjectReturnValues("data", nil)

	svc := &Service{store: storeMock.Interface()}
	result, err := svc.LoadAndProcess(99)

	if err != nil || result != "processed: data" {
		t.Errorf("unexpected result: %q, %v", result, err)
	}
}

// TestDependencyInterface_Ordered_InjectError demonstrates injecting errors
func TestDependencyInterface_Ordered_InjectError(t *testing.T) {
	imp := imptest.NewImp(t)
	storeMock := imptest.NewDependencyInterface[DataStore](imp)

	call := storeMock.Get.ExpectCalledWithExactly(42)

	// Inject an error return
	expectedErr := errors.New("not found")
	call.InjectReturnValues("", expectedErr)

	svc := &Service{store: storeMock.Interface()}
	result, err := svc.LoadAndProcess(42)

	if err != expectedErr {
		t.Errorf("expected error 'not found', got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

// TestDependencyInterface_Ordered_GetArgs demonstrates getting actual arguments
func TestDependencyInterface_Ordered_GetArgs(t *testing.T) {
	imp := imptest.NewImp(t)
	storeMock := imptest.NewDependencyInterface[DataStore](imp)

	call := storeMock.Save.ExpectCalledWithExactly(42, "processed: input")
	call.InjectReturnValues(nil)

	svc := &Service{store: storeMock.Interface()}
	err := svc.SaveProcessed(42, "input")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Get the actual arguments
	args := call.GetArgs()
	if args.A1 != 42 {
		t.Errorf("expected id 42, got %d", args.A1)
	}
	if args.A2 != "processed: input" {
		t.Errorf("expected 'processed: input', got %q", args.A2)
	}
}

// TestDependencyInterface_Ordered_InjectPanic demonstrates injecting a panic
func TestDependencyInterface_Ordered_InjectPanic(t *testing.T) {
	imp := imptest.NewImp(t)
	storeMock := imptest.NewDependencyInterface[DataStore](imp)

	call := storeMock.Get.ExpectCalledWithExactly(42)

	// Inject a panic
	call.InjectPanicValue("database connection lost")

	// Expect panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic, but none occurred")
		} else if r != "database connection lost" {
			t.Errorf("expected panic 'database connection lost', got %v", r)
		}
	}()

	svc := &Service{store: storeMock.Interface()}
	svc.LoadAndProcess(42)
}

// TestDependencyInterface_Ordered_MultipleMethod demonstrates multiple method calls
func TestDependencyInterface_Ordered_MultipleMethod(t *testing.T) {
	imp := imptest.NewImp(t)
	storeMock := imptest.NewDependencyInterface[DataStore](imp)

	// Expect multiple ordered calls
	call1 := storeMock.Get.ExpectCalledWithExactly(1)
	call1.InjectReturnValues("data1", nil)

	call2 := storeMock.Get.ExpectCalledWithExactly(2)
	call2.InjectReturnValues("data2", nil)

	svc := &Service{store: storeMock.Interface()}

	// First call
	result1, _ := svc.LoadAndProcess(1)
	if result1 != "processed: data1" {
		t.Errorf("expected 'processed: data1', got %q", result1)
	}

	// Second call
	result2, _ := svc.LoadAndProcess(2)
	if result2 != "processed: data2" {
		t.Errorf("expected 'processed: data2', got %q", result2)
	}
}
