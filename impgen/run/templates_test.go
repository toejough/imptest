//nolint:testpackage // Need same package to test unexported executeTemplate
package run

import (
	"strings"
	"testing"
	"text/template"
)

func TestExecuteTemplate(t *testing.T) {
	t.Parallel()

	t.Run("successful execution", func(t *testing.T) {
		t.Parallel()

		tmpl := mustParse("test", "Hello {{.Name}}")
		data := struct{ Name string }{Name: "World"}

		result := executeTemplate(tmpl, data)

		if result != "Hello World" {
			t.Errorf("expected 'Hello World', got %q", result)
		}
	})

	t.Run("template execution error", func(t *testing.T) {
		t.Parallel()

		// Create a template that will fail during execution
		// by referencing a field that doesn't exist on a strict type
		tmpl := template.Must(template.New("test").Option("missingkey=error").Parse("Hello {{.NonExistent}}"))
		data := struct{ Name string }{Name: "World"}

		// This should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic but got none")
			} else {
				msg, ok := r.(string)
				if !ok {
					t.Errorf("expected panic message to be string, got %T", r)
					return
				}

				if !strings.Contains(msg, "template execution failed") {
					t.Errorf("expected panic message to contain 'template execution failed', got %q", msg)
				}
			}
		}()

		executeTemplate(tmpl, data)
	})
}
