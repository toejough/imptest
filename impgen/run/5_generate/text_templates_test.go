//nolint:testpackage // Tests internal functions
package generate

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// Use the correct data structure expected by the template

// Verify output contains expected content

// Test with no params

//nolint:tparallel,paralleltest // Tests panic behavior which needs sequential execution per subtest
func TestTemplateRegistry_WritePanicPaths(t *testing.T) {
	t.Parallel()

	registry := NewTemplateRegistry()

	for _, testCase := range allTemplateWriteTests() {
		t.Run(testCase.name, func(t *testing.T) {
			buf := &bytes.Buffer{}

			defer func() {
				recovered := recover()
				if recovered == nil {
					t.Errorf("%s: expected panic for invalid template data", testCase.name)
					return
				}

				msg, isString := recovered.(string)
				if !isString {
					t.Errorf("%s: expected string panic, got %T", testCase.name, recovered)
					return
				}

				if !strings.Contains(msg, testCase.panicMsg) {
					t.Errorf("%s: unexpected panic message: %s", testCase.name, msg)
				}
			}()

			testCase.writeFunc(registry, buf)
		})
	}
}

// WriteTargetWaitMethod has an empty template - should not panic and produce no output

// templateWriteTest defines a template write function and its expected panic message prefix.
type templateWriteTest struct {
	name      string
	writeFunc func(registry *TemplateRegistry, buf *bytes.Buffer)
	panicMsg  string
}

// allTemplateWriteTests returns test cases for all template Write functions.
// WriteTargetWaitMethod is intentionally omitted - template has empty content and won't panic.
func allTemplateWriteTests() []templateWriteTest {
	methodNames := []string{
		"WriteDepArgsStruct", "WriteDepCallWrapper", "WriteDepConstructor", "WriteDepHeader",
		"WriteDepImplMethod", "WriteDepImplStruct", "WriteDepInterfaceMethod", "WriteDepMethodWrapper",
		"WriteDepMockStruct", "WriteFuncDepConstructor", "WriteFuncDepMethodWrapper",
		"WriteFuncDepMockStruct", "WriteInterfaceTargetConstructor", "WriteInterfaceTargetHeader",
		"WriteInterfaceTargetMethodCallHandleStruct", "WriteInterfaceTargetMethodExpectCompletes",
		"WriteInterfaceTargetMethodExpectPanic", "WriteInterfaceTargetMethodExpectReturns",
		"WriteInterfaceTargetMethodReturns", "WriteInterfaceTargetMethodStart",
		"WriteInterfaceTargetMethodWrapperFunc", "WriteInterfaceTargetMethodWrapperStruct",
		"WriteInterfaceTargetWrapperStruct", "WriteTargetCallHandleStruct", "WriteTargetConstructor",
		"WriteTargetExpectCompletes", "WriteTargetExpectPanic", "WriteTargetExpectReturns",
		"WriteTargetHeader", "WriteTargetReturnsStruct", "WriteTargetStartMethod",
		"WriteTargetWrapperStruct",
	}

	tests := make([]templateWriteTest, 0, len(methodNames))

	for _, methodName := range methodNames {
		// Derive panicMsg by removing "Write" prefix and lowercasing first char
		panicMsg := strings.ToLower(methodName[5:6]) + methodName[6:]
		tests = append(tests, templateWriteTest{
			name:      methodName,
			writeFunc: makeWriteFunc(methodName),
			panicMsg:  panicMsg,
		})
	}

	return tests
}

func makeWriteFunc(methodName string) func(*TemplateRegistry, *bytes.Buffer) {
	return func(r *TemplateRegistry, b *bytes.Buffer) {
		method := reflect.ValueOf(r).MethodByName(methodName)
		method.Call([]reflect.Value{reflect.ValueOf(b), reflect.ValueOf(struct{}{})})
	}
}
