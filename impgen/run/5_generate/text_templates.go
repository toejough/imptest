package generate

import (
	"bytes"
	"fmt"
	"text/template"
)

// TemplateRegistry holds all parsed text templates for code generation.
// Create a registry using NewTemplateRegistry() to initialize all templates.
type TemplateRegistry struct {
	// Dependency templates
	depHeaderTmpl          *template.Template
	depMockStructTmpl      *template.Template
	depConstructorTmpl     *template.Template
	depInterfaceMethodTmpl *template.Template
	depImplStructTmpl      *template.Template
	depImplMethodTmpl      *template.Template
	depArgsStructTmpl      *template.Template
	depCallWrapperTmpl     *template.Template
	depMethodWrapperTmpl   *template.Template
	// Target wrapper templates
	targetHeaderTmpl           *template.Template
	targetConstructorTmpl      *template.Template
	targetWrapperStructTmpl    *template.Template
	targetCallHandleStructTmpl *template.Template
	targetStartMethodTmpl      *template.Template
	targetWaitMethodTmpl       *template.Template
	targetExpectReturnsTmpl    *template.Template
	targetExpectCompletesTmpl  *template.Template
	targetExpectPanicTmpl      *template.Template
	targetReturnsStructTmpl    *template.Template
	// Interface Target wrapper templates
	interfaceTargetHeaderTmpl                 *template.Template
	interfaceTargetWrapperStructTmpl          *template.Template
	interfaceTargetConstructorTmpl            *template.Template
	interfaceTargetMethodWrapperFuncTmpl      *template.Template
	interfaceTargetMethodWrapperStructTmpl    *template.Template
	interfaceTargetMethodCallHandleStructTmpl *template.Template
	interfaceTargetMethodStartTmpl            *template.Template
	interfaceTargetMethodReturnsTmpl          *template.Template
	interfaceTargetMethodExpectReturnsTmpl    *template.Template
	interfaceTargetMethodExpectCompletesTmpl  *template.Template
	interfaceTargetMethodExpectPanicTmpl      *template.Template
	// Function dependency templates
	funcDepMockStructTmpl    *template.Template
	funcDepConstructorTmpl   *template.Template
	funcDepMethodWrapperTmpl *template.Template
}

// NewTemplateRegistry creates and initializes a new template registry with all templates parsed.
// Templates are hardcoded constants, so parsing cannot fail at runtime.
func NewTemplateRegistry() *TemplateRegistry {
	registry := &TemplateRegistry{}

	registry.parseDependencyTemplates()
	registry.parseTargetTemplates()
	registry.parseInterfaceTargetTemplates()
	registry.parseFunctionDependencyTemplates()

	return registry
}

// WriteDepArgsStruct writes the dependency args struct.
func (r *TemplateRegistry) WriteDepArgsStruct(buf *bytes.Buffer, data any) {
	err := r.depArgsStructTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute depArgsStruct template: %v", err))
	}
}

// WriteDepCallWrapper writes the dependency call wrapper.
func (r *TemplateRegistry) WriteDepCallWrapper(buf *bytes.Buffer, data any) {
	err := r.depCallWrapperTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute depCallWrapper template: %v", err))
	}
}

// WriteDepConstructor writes the dependency mock constructor.
func (r *TemplateRegistry) WriteDepConstructor(buf *bytes.Buffer, data any) {
	err := r.depConstructorTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute depConstructor template: %v", err))
	}
}

// WriteDepHeader writes the dependency mock header.
func (r *TemplateRegistry) WriteDepHeader(buf *bytes.Buffer, data any) {
	err := r.depHeaderTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute depHeader template: %v", err))
	}
}

// WriteDepImplMethod writes a dependency implementation method.
func (r *TemplateRegistry) WriteDepImplMethod(buf *bytes.Buffer, data any) {
	err := r.depImplMethodTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute depImplMethod template: %v", err))
	}
}

// WriteDepImplStruct writes the dependency implementation struct.
func (r *TemplateRegistry) WriteDepImplStruct(buf *bytes.Buffer, data any) {
	err := r.depImplStructTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute depImplStruct template: %v", err))
	}
}

// WriteDepInterfaceMethod writes the dependency Interface() method.
func (r *TemplateRegistry) WriteDepInterfaceMethod(buf *bytes.Buffer, data any) {
	err := r.depInterfaceMethodTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute depInterfaceMethod template: %v", err))
	}
}

// WriteDepMethodWrapper writes the dependency method wrapper.
func (r *TemplateRegistry) WriteDepMethodWrapper(buf *bytes.Buffer, data any) {
	err := r.depMethodWrapperTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute depMethodWrapper template: %v", err))
	}
}

// WriteDepMockStruct writes the dependency mock struct.
func (r *TemplateRegistry) WriteDepMockStruct(buf *bytes.Buffer, data any) {
	err := r.depMockStructTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute depMockStruct template: %v", err))
	}
}

// WriteFuncDepConstructor writes the function dependency mock constructor.
func (r *TemplateRegistry) WriteFuncDepConstructor(buf *bytes.Buffer, data any) {
	err := r.funcDepConstructorTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute funcDepConstructor template: %v", err))
	}
}

// WriteFuncDepFuncMethod writes the function dependency Func() method.

// WriteFuncDepMethodWrapper writes the function dependency method wrapper.
func (r *TemplateRegistry) WriteFuncDepMethodWrapper(buf *bytes.Buffer, data any) {
	err := r.funcDepMethodWrapperTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute funcDepMethodWrapper template: %v", err))
	}
}

// WriteFuncDepMockStruct writes the function dependency mock struct.
func (r *TemplateRegistry) WriteFuncDepMockStruct(buf *bytes.Buffer, data any) {
	err := r.funcDepMockStructTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute funcDepMockStruct template: %v", err))
	}
}

// WriteInterfaceTargetConstructor writes the interface target constructor.
func (r *TemplateRegistry) WriteInterfaceTargetConstructor(buf *bytes.Buffer, data any) {
	err := r.interfaceTargetConstructorTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute interfaceTargetConstructor template: %v", err))
	}
}

// WriteInterfaceTargetHeader writes the interface target header.
func (r *TemplateRegistry) WriteInterfaceTargetHeader(buf *bytes.Buffer, data any) {
	err := r.interfaceTargetHeaderTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute interfaceTargetHeader template: %v", err))
	}
}

// WriteInterfaceTargetMethodCallHandleStruct writes the interface target method call handle struct.
func (r *TemplateRegistry) WriteInterfaceTargetMethodCallHandleStruct(buf *bytes.Buffer, data any) {
	err := r.interfaceTargetMethodCallHandleStructTmpl.Execute(buf, data)
	if err != nil {
		panic(
			fmt.Sprintf(
				"failed to execute interfaceTargetMethodCallHandleStruct template: %v",
				err,
			),
		)
	}
}

// WriteInterfaceTargetMethodExpectCompletes writes the interface target method ExpectCompletes method.
func (r *TemplateRegistry) WriteInterfaceTargetMethodExpectCompletes(buf *bytes.Buffer, data any) {
	err := r.interfaceTargetMethodExpectCompletesTmpl.Execute(buf, data)
	if err != nil {
		panic(
			fmt.Sprintf("failed to execute interfaceTargetMethodExpectCompletes template: %v", err),
		)
	}
}

// WriteInterfaceTargetMethodExpectPanic writes the interface target method ExpectPanic methods.
func (r *TemplateRegistry) WriteInterfaceTargetMethodExpectPanic(buf *bytes.Buffer, data any) {
	err := r.interfaceTargetMethodExpectPanicTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute interfaceTargetMethodExpectPanic template: %v", err))
	}
}

// WriteInterfaceTargetMethodExpectReturns writes the interface target method ExpectReturns methods.
func (r *TemplateRegistry) WriteInterfaceTargetMethodExpectReturns(buf *bytes.Buffer, data any) {
	err := r.interfaceTargetMethodExpectReturnsTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute interfaceTargetMethodExpectReturns template: %v", err))
	}
}

// WriteInterfaceTargetMethodReturns writes the interface target method returns struct and methods.
func (r *TemplateRegistry) WriteInterfaceTargetMethodReturns(buf *bytes.Buffer, data any) {
	err := r.interfaceTargetMethodReturnsTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute interfaceTargetMethodReturns template: %v", err))
	}
}

// WriteInterfaceTargetMethodStart writes the interface target method Start method.
func (r *TemplateRegistry) WriteInterfaceTargetMethodStart(buf *bytes.Buffer, data any) {
	err := r.interfaceTargetMethodStartTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute interfaceTargetMethodStart template: %v", err))
	}
}

// WriteInterfaceTargetMethodWrapperFunc writes the interface target method wrapper function.
func (r *TemplateRegistry) WriteInterfaceTargetMethodWrapperFunc(buf *bytes.Buffer, data any) {
	err := r.interfaceTargetMethodWrapperFuncTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute interfaceTargetMethodWrapperFunc template: %v", err))
	}
}

// WriteInterfaceTargetMethodWrapperStruct writes the interface target method wrapper struct.
func (r *TemplateRegistry) WriteInterfaceTargetMethodWrapperStruct(buf *bytes.Buffer, data any) {
	err := r.interfaceTargetMethodWrapperStructTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute interfaceTargetMethodWrapperStruct template: %v", err))
	}
}

// WriteInterfaceTargetWrapperStruct writes the interface target wrapper struct.
func (r *TemplateRegistry) WriteInterfaceTargetWrapperStruct(buf *bytes.Buffer, data any) {
	err := r.interfaceTargetWrapperStructTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute interfaceTargetWrapperStruct template: %v", err))
	}
}

// WriteTargetCallHandleStruct writes the target call handle struct.
func (r *TemplateRegistry) WriteTargetCallHandleStruct(buf *bytes.Buffer, data any) {
	err := r.targetCallHandleStructTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute targetCallHandleStruct template: %v", err))
	}
}

// WriteTargetConstructor writes the target wrapper constructor.
func (r *TemplateRegistry) WriteTargetConstructor(buf *bytes.Buffer, data any) {
	err := r.targetConstructorTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute targetConstructor template: %v", err))
	}
}

// WriteTargetExpectCompletes writes the target ExpectCompletes method.
func (r *TemplateRegistry) WriteTargetExpectCompletes(buf *bytes.Buffer, data any) {
	err := r.targetExpectCompletesTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute targetExpectCompletes template: %v", err))
	}
}

// WriteTargetExpectPanic writes the target ExpectPanic methods.
func (r *TemplateRegistry) WriteTargetExpectPanic(buf *bytes.Buffer, data any) {
	err := r.targetExpectPanicTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute targetExpectPanic template: %v", err))
	}
}

// WriteTargetExpectReturns writes the target ExpectReturns methods.
func (r *TemplateRegistry) WriteTargetExpectReturns(buf *bytes.Buffer, data any) {
	err := r.targetExpectReturnsTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute targetExpectReturns template: %v", err))
	}
}

// WriteTargetHeader writes the target wrapper header.
func (r *TemplateRegistry) WriteTargetHeader(buf *bytes.Buffer, data any) {
	err := r.targetHeaderTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute targetHeader template: %v", err))
	}
}

// WriteTargetReturnsStruct writes the target returns struct.
func (r *TemplateRegistry) WriteTargetReturnsStruct(buf *bytes.Buffer, data any) {
	err := r.targetReturnsStructTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute targetReturnsStruct template: %v", err))
	}
}

// WriteTargetStartMethod writes the target Start method.
func (r *TemplateRegistry) WriteTargetStartMethod(buf *bytes.Buffer, data any) {
	err := r.targetStartMethodTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute targetStartMethod template: %v", err))
	}
}

// WriteTargetWaitMethod writes the target Wait method.
// Note: This template is currently empty, so Execute cannot fail.
func (r *TemplateRegistry) WriteTargetWaitMethod(buf *bytes.Buffer, data any) {
	_ = r.targetWaitMethodTmpl.Execute(buf, data)
}

// WriteTargetWrapperStruct writes the target wrapper struct.
func (r *TemplateRegistry) WriteTargetWrapperStruct(buf *bytes.Buffer, data any) {
	err := r.targetWrapperStructTmpl.Execute(buf, data)
	if err != nil {
		panic(fmt.Sprintf("failed to execute targetWrapperStruct template: %v", err))
	}
}

// parseDependencyTemplates parses all dependency mock templates.
func (r *TemplateRegistry) parseDependencyTemplates() {
	templates := []struct {
		target  **template.Template
		name    string
		content string
	}{
		{&r.depHeaderTmpl, "depHeader", tmplDepHeader},
		{&r.depMockStructTmpl, "depMockStruct", tmplDepMockStruct},
		{&r.depConstructorTmpl, "depConstructor", tmplDepConstructor},
		{&r.depInterfaceMethodTmpl, "depInterfaceMethod", tmplDepInterfaceMethod},
		{&r.depImplStructTmpl, "depImplStruct", tmplDepImplStruct},
		{&r.depImplMethodTmpl, "depImplMethod", tmplDepImplMethod},
		{&r.depArgsStructTmpl, "depArgsStruct", tmplDepArgsStruct},
		{&r.depCallWrapperTmpl, "depCallWrapper", tmplDepCallWrapper},
		{&r.depMethodWrapperTmpl, "depMethodWrapper", tmplDepMethodWrapper},
	}

	parseTemplateList(templates)
}

// parseFunctionDependencyTemplates parses all function dependency mock templates.
func (r *TemplateRegistry) parseFunctionDependencyTemplates() {
	templates := []struct {
		target  **template.Template
		name    string
		content string
	}{
		{&r.funcDepMockStructTmpl, "funcDepMockStruct", tmplFuncDepMockStruct},
		{&r.funcDepConstructorTmpl, "funcDepConstructor", tmplFuncDepConstructor},
		{&r.funcDepMethodWrapperTmpl, "funcDepMethodWrapper", tmplFuncDepMethodWrapper},
	}

	parseTemplateList(templates)
}

// parseInterfaceTargetTemplates parses all interface target wrapper templates.
func (r *TemplateRegistry) parseInterfaceTargetTemplates() {
	templates := []struct {
		target  **template.Template
		name    string
		content string
	}{
		{&r.interfaceTargetHeaderTmpl, "interfaceTargetHeader", tmplInterfaceTargetHeader},
		{
			&r.interfaceTargetWrapperStructTmpl,
			"interfaceTargetWrapperStruct",
			tmplInterfaceTargetWrapperStruct,
		},
		{
			&r.interfaceTargetConstructorTmpl,
			"interfaceTargetConstructor",
			tmplInterfaceTargetConstructor,
		},
		{
			&r.interfaceTargetMethodWrapperFuncTmpl,
			"interfaceTargetMethodWrapperFunc",
			tmplInterfaceTargetMethodWrapperFunc,
		},
		{
			&r.interfaceTargetMethodWrapperStructTmpl,
			"interfaceTargetMethodWrapperStruct",
			tmplInterfaceTargetMethodWrapperStruct,
		},
		{
			&r.interfaceTargetMethodCallHandleStructTmpl,
			"interfaceTargetMethodCallHandleStruct",
			tmplInterfaceTargetMethodCallHandleStruct,
		},
		{
			&r.interfaceTargetMethodStartTmpl,
			"interfaceTargetMethodStart",
			tmplInterfaceTargetMethodStart,
		},
		{
			&r.interfaceTargetMethodReturnsTmpl,
			"interfaceTargetMethodReturns",
			tmplInterfaceTargetMethodReturns,
		},
		{
			&r.interfaceTargetMethodExpectReturnsTmpl,
			"interfaceTargetMethodExpectReturns",
			tmplInterfaceTargetMethodExpectReturns,
		},
		{
			&r.interfaceTargetMethodExpectCompletesTmpl,
			"interfaceTargetMethodExpectCompletes",
			tmplInterfaceTargetMethodExpectCompletes,
		},
		{
			&r.interfaceTargetMethodExpectPanicTmpl,
			"interfaceTargetMethodExpectPanic",
			tmplInterfaceTargetMethodExpectPanic,
		},
	}

	parseTemplateList(templates)
}

// parseTargetTemplates parses all target wrapper templates.
func (r *TemplateRegistry) parseTargetTemplates() {
	templates := []struct {
		target  **template.Template
		name    string
		content string
	}{
		{&r.targetHeaderTmpl, "targetHeader", tmplTargetHeader},
		{&r.targetConstructorTmpl, "targetConstructor", tmplTargetConstructor},
		{&r.targetWrapperStructTmpl, "targetWrapperStruct", tmplTargetWrapperStruct},
		{&r.targetCallHandleStructTmpl, "targetCallHandleStruct", tmplTargetCallHandleStruct},
		{&r.targetStartMethodTmpl, "targetStartMethod", tmplTargetStartMethod},
		{&r.targetWaitMethodTmpl, "targetWaitMethod", ""},
		{&r.targetExpectReturnsTmpl, "targetExpectReturns", tmplTargetExpectReturns},
		{&r.targetExpectCompletesTmpl, "targetExpectCompletes", tmplTargetExpectCompletes},
		{&r.targetExpectPanicTmpl, "targetExpectPanic", tmplTargetExpectPanic},
		{&r.targetReturnsStructTmpl, "targetReturnsStruct", tmplTargetReturnsStruct},
	}

	parseTemplateList(templates)
}

// parseTemplate is a helper function that parses a template using template.Must().
// Templates are hardcoded constants, so parsing cannot fail at runtime.
// Panics if the template is invalid (programming error, caught at startup).
func parseTemplate(name, content string) *template.Template {
	return template.Must(template.New(name).Parse(content))
}

// parseTemplateList parses a list of templates and assigns them to their targets.
// Uses template.Must() internally, so panics on invalid templates.
func parseTemplateList(templates []struct {
	target  **template.Template
	name    string
	content string
},
) {
	for _, def := range templates {
		*def.target = parseTemplate(def.name, def.content)
	}
}
