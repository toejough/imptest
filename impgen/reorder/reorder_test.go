package reorder_test

import (
	"testing"

	"github.com/toejough/imptest/impgen/reorder"
)

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
