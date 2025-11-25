package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/toejough/imptest/POC-gen-from-black-box-test/run"
)

//go:generate go run generate.go run.ExampleInt
//type exampleInt run.ExampleInt

//var _ = (exampleInt)(&mockExampleInt{})

// mockExampleInt is a mock implementation of run.ExampleInt for testing.
type mockExampleInt struct {
	printCalled     bool
	printArg        string
	addCalled       bool
	addA, addB      int
	addResult       int
	transformCalled bool
	transformArg    float64
	transformErr    error
	transformData   []byte
}

func (m *mockExampleInt) Print(s string) {
	m.printCalled = true
	m.printArg = s
}

func (m *mockExampleInt) Add(a, b int) int {
	m.addCalled = true
	m.addA = a
	m.addB = b
	return m.addResult
}

func (m *mockExampleInt) Transform(f float64) (error, []byte) {
	m.transformCalled = true
	m.transformArg = f
	return m.transformErr, m.transformData
}

func Test_runExample(t *testing.T) {
	mock := &mockExampleInt{
		addResult:     42,
		transformErr:  errors.New("mock error"),
		transformData: []byte{9, 8, 7},
	}

	// Capture os.Stdout
	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		io.Copy(&buf, r)
		close(done)
	}()

	run.RunExample(mock)

	w.Close()
	os.Stdout = origStdout
	<-done

	// Check that methods were called with expected arguments
	if !mock.printCalled || mock.printArg != "test" {
		t.Errorf("print not called as expected")
	}
	if !mock.addCalled || mock.addA != 2 || mock.addB != 3 {
		t.Errorf("add not called as expected")
	}
	if !mock.transformCalled || mock.transformArg != 1.23 {
		t.Errorf("transform not called as expected")
	}

	// Check output
	output := buf.String()
	if !strings.Contains(output, "add result: 42") {
		t.Errorf("expected add result in output, got: %q", output)
	}
	if !strings.Contains(output, "transform result: mock error [9 8 7]") {
		t.Errorf("expected transform result in output, got: %q", output)
	}
}
