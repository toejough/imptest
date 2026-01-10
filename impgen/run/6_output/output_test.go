package output

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

func TestWriteGeneratedCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		code         string
		impName      string
		pkgName      string
		goFile       string
		writerErr    error
		wantFilename string
		wantErr      bool
	}{
		{
			name:         "regular package adds .go suffix",
			code:         "package foo\n",
			impName:      "MyType",
			pkgName:      "foo",
			goFile:       "source.go",
			wantFilename: "generated_MyType.go",
		},
		{
			name:         "impName already has .go suffix",
			code:         "package foo\n",
			impName:      "MyType.go",
			pkgName:      "foo",
			goFile:       "source.go",
			wantFilename: "generated_MyType.go",
		},
		{
			name:         "test package adds _test suffix",
			code:         "package foo_test\n",
			impName:      "MyType",
			pkgName:      "foo_test",
			goFile:       "source.go",
			wantFilename: "generated_MyType_test.go",
		},
		{
			name:         "test file adds _test suffix",
			code:         "package foo\n",
			impName:      "MyType",
			pkgName:      "foo",
			goFile:       "source_test.go",
			wantFilename: "generated_MyType_test.go",
		},
		{
			name:         "impName already has _test suffix in test package",
			code:         "package foo_test\n",
			impName:      "MyType_test",
			pkgName:      "foo_test",
			goFile:       "source.go",
			wantFilename: "generated_MyType_test.go",
		},
		{
			name:         "impName with .go suffix in test package",
			code:         "package foo_test\n",
			impName:      "MyType.go",
			pkgName:      "foo_test",
			goFile:       "source.go",
			wantFilename: "generated_MyType_test.go",
		},
		{
			name:      "write error returns error",
			code:      "package foo\n",
			impName:   "MyType",
			pkgName:   "foo",
			goFile:    "source.go",
			writerErr: errors.New("write failed"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			writer := newMockWriter()
			writer.writeErr = tt.writerErr
			out := &bytes.Buffer{}

			getEnv := func(key string) string {
				if key == "GOFILE" {
					return tt.goFile
				}

				return ""
			}

			err := WriteGeneratedCode(tt.code, tt.impName, tt.pkgName, getEnv, writer, out)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.wantFilename != "" {
				if _, ok := writer.writtenFiles[tt.wantFilename]; !ok {
					t.Errorf("expected file %s to be written, got files: %v", tt.wantFilename, writer.writtenFiles)
				}
			}
		})
	}
}

// mockWriter is a test mock for the Writer interface.
type mockWriter struct {
	writtenFiles map[string][]byte
	writeErr     error
}

func (m *mockWriter) WriteFile(name string, data []byte, _ os.FileMode) error {
	if m.writeErr != nil {
		return m.writeErr
	}

	m.writtenFiles[name] = data

	return nil
}

func newMockWriter() *mockWriter {
	return &mockWriter{writtenFiles: make(map[string][]byte)}
}
