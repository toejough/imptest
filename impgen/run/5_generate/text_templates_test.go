//nolint:testpackage // Tests internal functions
package generate

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseTemplate_Error(t *testing.T) {
	t.Parallel()

	// Test with invalid template syntax - unclosed action
	_, err := parseTemplate("test", "{{.Invalid")
	if err == nil {
		t.Error("parseTemplate() expected error for invalid template, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse test template") {
		t.Errorf(
			"parseTemplate() error = %v, want error containing 'failed to parse test template'",
			err,
		)
	}
}

// Use the correct data structure expected by the template

// Verify output contains expected content

// Test with no params

//nolint:tparallel,paralleltest // Tests panic behavior which needs sequential execution per subtest
func TestTemplateRegistry_WritePanicPaths(t *testing.T) {
	t.Parallel()

	registry, err := NewTemplateRegistry()
	if err != nil {
		t.Fatalf("failed to create template registry: %v", err)
	}

	for _, tc := range allTemplateWriteTests() {
		t.Run(tc.name, func(t *testing.T) {
			buf := &bytes.Buffer{}

			defer func() {
				recovered := recover()
				if recovered == nil {
					t.Errorf("%s: expected panic for invalid template data", tc.name)
					return
				}

				msg, isString := recovered.(string)
				if !isString {
					t.Errorf("%s: expected string panic, got %T", tc.name, recovered)
					return
				}

				if !strings.Contains(msg, tc.panicMsg) {
					t.Errorf("%s: unexpected panic message: %s", tc.name, msg)
				}
			}()

			tc.writeFunc(registry, buf)
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
func allTemplateWriteTests() []templateWriteTest {
	return []templateWriteTest{
		{
			"WriteDepArgsStruct",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteDepArgsStruct(b, struct{}{}) },
			"depArgsStruct",
		},
		{
			"WriteDepCallWrapper",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteDepCallWrapper(b, struct{}{}) },
			"depCallWrapper",
		},
		{
			"WriteDepConstructor",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteDepConstructor(b, struct{}{}) },
			"depConstructor",
		},
		{
			"WriteDepHeader",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteDepHeader(b, struct{}{}) },
			"depHeader",
		},
		{
			"WriteDepImplMethod",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteDepImplMethod(b, struct{}{}) },
			"depImplMethod",
		},
		{
			"WriteDepImplStruct",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteDepImplStruct(b, struct{}{}) },
			"depImplStruct",
		},
		{
			"WriteDepInterfaceMethod",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteDepInterfaceMethod(b, struct{}{}) },
			"depInterfaceMethod",
		},
		{
			"WriteDepMethodWrapper",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteDepMethodWrapper(b, struct{}{}) },
			"depMethodWrapper",
		},
		{
			"WriteDepMockStruct",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteDepMockStruct(b, struct{}{}) },
			"depMockStruct",
		},
		{
			"WriteFuncDepConstructor",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteFuncDepConstructor(b, struct{}{}) },
			"funcDepConstructor",
		},
		{
			"WriteFuncDepMethodWrapper",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteFuncDepMethodWrapper(b, struct{}{}) },
			"funcDepMethodWrapper",
		},
		{
			"WriteFuncDepMockStruct",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteFuncDepMockStruct(b, struct{}{}) },
			"funcDepMockStruct",
		},
		{
			"WriteInterfaceTargetConstructor",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteInterfaceTargetConstructor(b, struct{}{}) },
			"interfaceTargetConstructor",
		},
		{
			"WriteInterfaceTargetHeader",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteInterfaceTargetHeader(b, struct{}{}) },
			"interfaceTargetHeader",
		},
		{"WriteInterfaceTargetMethodCallHandleStruct", func(r *TemplateRegistry, b *bytes.Buffer) {
			r.WriteInterfaceTargetMethodCallHandleStruct(b, struct{}{})
		}, "interfaceTargetMethodCallHandleStruct"},
		{
			"WriteInterfaceTargetMethodExpectCompletes",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteInterfaceTargetMethodExpectCompletes(b, struct{}{}) },
			"interfaceTargetMethodExpectCompletes",
		},
		{
			"WriteInterfaceTargetMethodExpectPanic",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteInterfaceTargetMethodExpectPanic(b, struct{}{}) },
			"interfaceTargetMethodExpectPanic",
		},
		{
			"WriteInterfaceTargetMethodExpectReturns",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteInterfaceTargetMethodExpectReturns(b, struct{}{}) },
			"interfaceTargetMethodExpectReturns",
		},
		{
			"WriteInterfaceTargetMethodReturns",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteInterfaceTargetMethodReturns(b, struct{}{}) },
			"interfaceTargetMethodReturns",
		},
		{
			"WriteInterfaceTargetMethodStart",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteInterfaceTargetMethodStart(b, struct{}{}) },
			"interfaceTargetMethodStart",
		},
		{
			"WriteInterfaceTargetMethodWrapperFunc",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteInterfaceTargetMethodWrapperFunc(b, struct{}{}) },
			"interfaceTargetMethodWrapperFunc",
		},
		{
			"WriteInterfaceTargetMethodWrapperStruct",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteInterfaceTargetMethodWrapperStruct(b, struct{}{}) },
			"interfaceTargetMethodWrapperStruct",
		},
		{
			"WriteInterfaceTargetWrapperStruct",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteInterfaceTargetWrapperStruct(b, struct{}{}) },
			"interfaceTargetWrapperStruct",
		},
		{
			"WriteTargetCallHandleStruct",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteTargetCallHandleStruct(b, struct{}{}) },
			"targetCallHandleStruct",
		},
		{
			"WriteTargetConstructor",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteTargetConstructor(b, struct{}{}) },
			"targetConstructor",
		},
		{
			"WriteTargetExpectCompletes",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteTargetExpectCompletes(b, struct{}{}) },
			"targetExpectCompletes",
		},
		{
			"WriteTargetExpectPanic",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteTargetExpectPanic(b, struct{}{}) },
			"targetExpectPanic",
		},
		{
			"WriteTargetExpectReturns",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteTargetExpectReturns(b, struct{}{}) },
			"targetExpectReturns",
		},
		{
			"WriteTargetHeader",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteTargetHeader(b, struct{}{}) },
			"targetHeader",
		},
		{
			"WriteTargetReturnsStruct",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteTargetReturnsStruct(b, struct{}{}) },
			"targetReturnsStruct",
		},
		{
			"WriteTargetStartMethod",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteTargetStartMethod(b, struct{}{}) },
			"targetStartMethod",
		},
		// WriteTargetWaitMethod intentionally omitted - template has empty content and won't panic
		{
			"WriteTargetWrapperStruct",
			func(r *TemplateRegistry, b *bytes.Buffer) { r.WriteTargetWrapperStruct(b, struct{}{}) },
			"targetWrapperStruct",
		},
	}
}
