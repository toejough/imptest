package reorder_test

import (
	"testing"

	"github.com/toejough/imptest/impgen/reorder"
)

//nolint:cyclop,funlen,gocognit // Test function with validation logic; complexity from comprehensive test cases
func TestAnalyzeSectionOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		validate func(*testing.T, *reorder.SectionOrder)
	}{
		{
			name: "correctly ordered code",
			input: `package example

import "fmt"

func main() {}

const Version = "1.0"

type Config struct{}

func Start() {}
`,
			wantErr: false,
			validate: func(t *testing.T, order *reorder.SectionOrder) {
				t.Helper()

				if len(order.Sections) == 0 {
					t.Error("Expected non-empty sections")
				}
				// Verify sections are in ascending order by position
				for i := 1; i < len(order.Sections); i++ {
					if order.Sections[i].Position < order.Sections[i-1].Position {
						t.Errorf("Sections not in position order: %v", order.Sections)
					}
				}
			},
		},
		{
			name: "incorrectly ordered code",
			input: `package example

func helper() {}

const Version = "1.0"

type Config struct{}

func main() {}
`,
			wantErr: false,
			validate: func(t *testing.T, order *reorder.SectionOrder) {
				t.Helper()
				// Find main() section
				var mainSection *reorder.Section

				for i := range order.Sections {
					if order.Sections[i].Name == "main()" {
						mainSection = &order.Sections[i]
						break
					}
				}

				if mainSection == nil {
					t.Error("Expected to find main() section")
					return
				}
				// main() should be at position 4 but expected at position 2
				if mainSection.Position == mainSection.Expected {
					t.Error("Expected main() to be out of order")
				}
			},
		},
		{
			name: "code with imports and multiple sections",
			input: `package example

import "fmt"

const maxRetries = 10

type server struct{}

func (s *server) start() {}

const Version = "1.0"

func NewServer() *server { return &server{} }
`,
			wantErr: false,
			validate: func(t *testing.T, order *reorder.SectionOrder) {
				t.Helper()

				sectionNames := make(map[string]bool)
				for _, sec := range order.Sections {
					sectionNames[sec.Name] = true
				}
				// Should have both exported and unexported sections
				expectedSections := []string{"Imports", "Exported Constants", "unexported constants", "unexported types"}
				for _, expected := range expectedSections {
					if !sectionNames[expected] {
						t.Errorf("Expected section %q not found", expected)
					}
				}
			},
		},
		{
			name: "invalid Go code",
			input: `package example

func 123InvalidName() {}`,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			order, err := reorder.AnalyzeSectionOrder(testCase.input)
			if (err != nil) != testCase.wantErr {
				t.Errorf("AnalyzeSectionOrder() error = %v, wantErr %v", err, testCase.wantErr)
				return
			}

			if err == nil && testCase.validate != nil {
				testCase.validate(t, order)
			}
		})
	}
}

func TestSource_BasicReordering(t *testing.T) {
	t.Parallel()

	input := `package example

func helper() {}

const Version = "1.0"

type Config struct {}

func main() {}

var Debug = false
`

	expected := `package example

func main() {}

// Exported constants.
const (
	Version = "1.0"
)

// Exported variables.
var (
	Debug = false
)

type Config struct{}

func helper() {}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

func TestSource_ConstructorGrouping(t *testing.T) {
	t.Parallel()

	input := `package example

func NewConfig() *Config {
	return &Config{}
}

type Config struct {
	timeout int
}

func (c *Config) Validate() error {
	return nil
}

func NewConfigWithTimeout(t int) *Config {
	return &Config{timeout: t}
}
`

	expected := `package example

type Config struct {
	timeout int
}

func NewConfig() *Config {
	return &Config{}
}

func NewConfigWithTimeout(t int) *Config {
	return &Config{timeout: t}
}

func (c *Config) Validate() error {
	return nil
}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

func TestSource_EnumHandling(t *testing.T) {
	t.Parallel()

	input := `package example

const (
	StatusPending Status = iota
	StatusActive
	StatusClosed
)

type Status int

const MaxRetries = 3
`

	expected := `package example

// Exported constants.
const (
	MaxRetries = 3
)

type Status int

// Status values.
const (
	StatusPending Status = iota
	StatusActive
	StatusClosed
)
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

func TestSource_ExportedUnexportedSeparation(t *testing.T) {
	t.Parallel()

	input := `package example

const maxWorkers = 10

var Debug = false

const Version = "1.0"

func helper() {}

func Start() {}

var workers = 0
`

	expected := `package example

// Exported constants.
const (
	Version = "1.0"
)

// Exported variables.
var (
	Debug = false
)

func Start() {}

// unexported constants.
const (
	maxWorkers = 10
)

// unexported variables.
var (
	workers = 0
)

func helper() {}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

func TestSource_Idempotency(t *testing.T) {
	t.Parallel()

	input := `package example

type Config struct{}

func NewConfig() *Config { return &Config{} }

const Version = "1.0"
`

	first, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("First Source() error = %v", err)
	}

	second, err := reorder.Source(first)
	if err != nil {
		t.Fatalf("Second Source() error = %v", err)
	}

	if first != second {
		t.Errorf("Not idempotent:\nFirst:\n%s\n\nSecond:\n%s", first, second)
	}
}

func TestSource_MethodOrdering(t *testing.T) {
	t.Parallel()

	input := `package example

type Server struct{}

func (s *Server) start() {}

func (s *Server) Stop() {}

func (s *Server) Start() {}

func (s *Server) shutdown() {}
`

	expected := `package example

type Server struct{}

func (s *Server) Start() {}

func (s *Server) Stop() {}

func (s *Server) shutdown() {}

func (s *Server) start() {}
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}

func TestSource_MultipleEnums(t *testing.T) {
	t.Parallel()

	input := `package example

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
)

type Status int

const (
	StatusPending Status = iota
	StatusActive
)

type Priority int
`

	expected := `package example

type Priority int

// Priority values.
const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
)

type Status int

// Status values.
const (
	StatusPending Status = iota
	StatusActive
)
`

	result, err := reorder.Source(input)
	if err != nil {
		t.Fatalf("Source() error = %v", err)
	}

	if result != expected {
		t.Errorf("Source() mismatch:\nGot:\n%s\n\nWant:\n%s", result, expected)
	}
}
