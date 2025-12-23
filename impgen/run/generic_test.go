package run_test

import (
	"io"
	"strings"
	"testing"

	"github.com/toejough/imptest/impgen/run"
)

func TestRunCallable_GenericFunction(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Generic function with type parameters
	sourceCode := `package run

func GenericFunc[T any, U comparable](item T, key U) (T, U) {
	return item, key
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.GenericFunc", "--name", "GenericFuncImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_GenericFuncImp.go"]
	if !ok {
		t.Fatal("Expected GenericFuncImp.go to be created")
	}

	contentStr := string(content)

	// Verify generic function type parameters are rendered correctly in callable
	expected := []string{
		"type GenericFuncImp[T any, U comparable] struct",
		"type GenericFuncImpReturn[T any, U comparable] struct",
		"func NewGenericFuncImp[T any, U comparable]",
		"callable func(item T, key U) (T, U)", // Function field should NOT have type params
		"func (s *GenericFuncImp[T, U]) Start(item T, key U)",
		"func (s *GenericFuncImp[T, U]) ExpectReturnedValues",
		"Result0 T", // Type parameters in return struct should not be qualified
		"Result1 U",
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}

	// Verify the callable field does NOT have type parameters on the function itself
	if strings.Contains(contentStr, "callable   func[") {
		t.Error("callable field should not have type parameters on the function type")
		t.Logf("Generated code:\n%s", contentStr)
	}
}

func TestRunCallable_GenericMultiParams(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Function with multiple type parameters and complex instantiation
	sourceCode := `package run

type MyContainer[T any, U any] struct {
	first  T
	second U
}

func ProcessMulti[T any, U any](c *MyContainer[T, U]) {}
func ProcessMultiList[T any, U any](c MyContainer[T, U]) {}
func ProcessExportedParam[T any](c MyContainer[T, MyExportedType]) {}
type MyExportedContainer[T any] struct {
	item T
}
func ProcessExportedBase[T any](c MyExportedContainer[T]) {}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.ProcessMulti", "--name", "ProcessMultiImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	argsList := []string{"impgen", "run.ProcessMultiList", "--name", "ProcessMultiListImp"}

	err = run.Run(argsList, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	argsExported := []string{"impgen", "run.ProcessExportedParam", "--name", "ProcessExportedParamImp"}

	err = run.Run(argsExported, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	argsExportedBase := []string{"impgen", "run.ProcessExportedBase", "--name", "ProcessExportedBaseImp"}

	err = run.Run(argsExportedBase, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, ok := mockFS.files["generated_ProcessMultiListImp.go"]; !ok {
		t.Fatal("Expected ProcessMultiListImp.go to be created")
	}
}

func TestRunCallable_GenericExported(t *testing.T) {
	t.Parallel()

	// Function with exported generic types/parameters
	sourceCode := `package run
import "fmt"

type MyExportedType struct { Value int }
type MyContainer[T any, U any] struct { first T; second U }
type MyExportedContainer[T any] struct { item T }

func ProcessExportedParam[T any](c MyContainer[T, MyExportedType]) {}
func ProcessExportedBase[T any](c MyExportedContainer[T]) {}
func ProcessExportedSelector[T any](c MyContainer[T, fmt.Stringer]) {}
func ProcessIndexExported[T any](c map[T]MyExportedType) {}
func ProcessMultiExported[T any](c MyContainer[MyExportedType, T]) {}
type secret int
func ProcessMultiUnexportedParam[T any](c MyContainer[T, secret]) {}
func ProcessMapValue[T comparable](m map[T]MyExportedType) {}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	tests := []struct {
		name    string
		symbol  string
		wantErr bool
	}{
		{"exported param", "run.ProcessExportedParam", false},
		{"exported base", "run.ProcessExportedBase", false},
		{"exported selector", "run.ProcessExportedSelector", false},
		{"exported map value", "run.ProcessIndexExported", false},
		{"exported generic param", "run.ProcessMultiExported", false},
		{"unexported generic param", "run.ProcessMultiUnexportedParam", true},
		{"exported map value in func", "run.ProcessMapValue", false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockFS := NewMockFileSystem()

			args := []string{"impgen", testCase.symbol, "--name", "Imp"}

			err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("Run failed for %s: %v", testCase.symbol, err)
			}
		})
	}
}

func TestRunCallable_ExportedGenericWithExportedParam(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Exported generic type with exported type parameter
	sourceCode := `package run

type MyExportedType struct {
	Value int
}

type MyContainer[T any, U any] struct {
	first  T
	second U
}

func ProcessContainer(c *MyContainer[MyExportedType, string]) string {
	return "processed"
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.ProcessContainer", "--name", "ProcessContainerImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ProcessContainerImp.go"]
	if !ok {
		t.Fatal("Expected ProcessContainerImp.go to be created")
	}

	contentStr := string(content)

	// Should import run package because MyExportedType is exported
	if !strings.Contains(contentStr, `run "github.com/toejough/imptest/UAT/run"`) {
		t.Error("Expected import of run package for exported type parameter")
		t.Logf("Generated code:\n%s", contentStr)
	}
}

func TestRunCallable_ChannelTypes(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Function with various channel types
	sourceCode := `package run

type MyType struct {
	Value int
}

func ProcessChannels(in <-chan MyType, out chan<- string, bidir chan int) {
	// process channels
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.ProcessChannels", "--name", "ProcessChannelsImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ProcessChannelsImp.go"]
	if !ok {
		t.Fatal("Expected ProcessChannelsImp.go to be created")
	}

	contentStr := string(content)

	// Verify channel types are handled correctly
	expected := []string{
		"<-chan run.MyType", // Receive-only channel with qualified type
		"chan<- string",     // Send-only channel
		"chan int",          // Bidirectional channel
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}

func TestRunCallable_FunctionTypeParameter(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Function with function type parameter that has exported types
	sourceCode := `package run

type CustomData struct {
	Value string
}

func ProcessWithCallback(data CustomData, handler func(CustomData) error) error {
	return handler(data)
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.ProcessWithCallback", "--name", "ProcessWithCallbackImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_ProcessWithCallbackImp.go"]
	if !ok {
		t.Fatal("Expected ProcessWithCallbackImp.go to be created")
	}

	contentStr := string(content)

	// Verify function type with exported parameter is handled correctly
	expected := []string{
		"func(run.CustomData) error", // Function type with qualified exported type
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}

func TestRunCallable_FunctionTypeReturnValue(t *testing.T) {
	t.Parallel()

	mockFS := NewMockFileSystem()

	// Function with function type that returns an exported type
	sourceCode := `package run

type Result struct {
	Success bool
}

func CreateHandler(id string) func() Result {
	return func() Result {
		return Result{Success: true}
	}
}
`
	mockPkgLoader := NewMockPackageLoader()
	mockPkgLoader.AddPackageFromSource(".", localPackageSource)
	mockPkgLoader.AddPackageFromSource("github.com/toejough/imptest/UAT/run", sourceCode)

	args := []string{"impgen", "run.CreateHandler", "--name", "CreateHandlerImp"}

	err := run.Run(args, envWithPkgName, mockFS, mockPkgLoader, io.Discard)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	content, ok := mockFS.files["generated_CreateHandlerImp.go"]
	if !ok {
		t.Fatal("Expected CreateHandlerImp.go to be created")
	}

	contentStr := string(content)

	// Verify function return type with exported type is handled correctly
	expected := []string{
		"func() run.Result", // Function type with qualified exported return type
	}
	for _, exp := range expected {
		if !strings.Contains(contentStr, exp) {
			t.Errorf("Expected generated code to contain %q", exp)
			t.Logf("Generated code:\n%s", contentStr)
		}
	}
}
