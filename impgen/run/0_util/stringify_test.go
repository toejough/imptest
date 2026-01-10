//nolint:testpackage // Needs access to unexported stringifyStructType for whitebox testing
package astutil

import (
	"testing"

	"github.com/dave/dst"
)

// TestStringifyExpr verifies that StringifyExpr correctly converts
// DST expression nodes to their Go source code representation.
//
//nolint:funlen // table-driven test with comprehensive test cases
func TestStringifyExpr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    dst.Expr
		expected string
	}{
		{
			name:     "nil expression",
			input:    nil,
			expected: "",
		},
		{
			name:     "ident",
			input:    &dst.Ident{Name: "MyType"},
			expected: "MyType",
		},
		{
			name:     "basic lit",
			input:    &dst.BasicLit{Value: "42"},
			expected: "42",
		},
		{
			name: "selector expr",
			input: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "time"},
				Sel: &dst.Ident{Name: "Duration"},
			},
			expected: "time.Duration",
		},
		{
			name: "star expr",
			input: &dst.StarExpr{
				X: &dst.Ident{Name: "string"},
			},
			expected: "*string",
		},
		{
			name: "slice type",
			input: &dst.ArrayType{
				Elt: &dst.Ident{Name: "int"},
			},
			expected: "[]int",
		},
		{
			name: "array type with length",
			input: &dst.ArrayType{
				Len: &dst.BasicLit{Value: "10"},
				Elt: &dst.Ident{Name: "byte"},
			},
			expected: "[10]byte",
		},
		{
			name: "map type",
			input: &dst.MapType{
				Key:   &dst.Ident{Name: "string"},
				Value: &dst.Ident{Name: "int"},
			},
			expected: "map[string]int",
		},
		{
			name: "bidirectional chan",
			input: &dst.ChanType{
				Dir:   dst.SEND | dst.RECV,
				Value: &dst.Ident{Name: "int"},
			},
			expected: "chan int",
		},
		{
			name: "send-only chan",
			input: &dst.ChanType{
				Dir:   dst.SEND,
				Value: &dst.Ident{Name: "string"},
			},
			expected: "chan<- string",
		},
		{
			name: "receive-only chan",
			input: &dst.ChanType{
				Dir:   dst.RECV,
				Value: &dst.Ident{Name: "error"},
			},
			expected: "<-chan error",
		},
		{
			name: "empty interface",
			input: &dst.InterfaceType{
				Methods: &dst.FieldList{},
			},
			expected: "interface{}",
		},
		{
			name: "interface with single method",
			input: &dst.InterfaceType{
				Methods: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Read"}},
							Type: &dst.FuncType{
								Params: &dst.FieldList{
									List: []*dst.Field{
										{Type: &dst.ArrayType{Elt: &dst.Ident{Name: "byte"}}},
									},
								},
								Results: &dst.FieldList{
									List: []*dst.Field{
										{Type: &dst.Ident{Name: "int"}},
										{Type: &dst.Ident{Name: "error"}},
									},
								},
							},
						},
					},
				},
			},
			expected: "interface{ Read([]byte) (int, error) }",
		},
		{
			name: "interface with multiple methods",
			input: &dst.InterfaceType{
				Methods: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Read"}},
							Type: &dst.FuncType{
								Params:  &dst.FieldList{},
								Results: nil,
							},
						},
						{
							Names: []*dst.Ident{{Name: "Write"}},
							Type: &dst.FuncType{
								Params:  &dst.FieldList{},
								Results: nil,
							},
						},
					},
				},
			},
			expected: "interface{\n\tRead()\n\tWrite()\n}",
		},
		{
			name: "interface with embedded type",
			input: &dst.InterfaceType{
				Methods: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: nil,
							Type:  &dst.Ident{Name: "Reader"},
						},
					},
				},
			},
			expected: "interface{ Reader }",
		},
		{
			name: "func type no params no results",
			input: &dst.FuncType{
				Params: &dst.FieldList{},
			},
			expected: "func()",
		},
		{
			name: "func type with params and single result",
			input: &dst.FuncType{
				Params: &dst.FieldList{
					List: []*dst.Field{
						{Type: &dst.Ident{Name: "int"}},
						{Type: &dst.Ident{Name: "string"}},
					},
				},
				Results: &dst.FieldList{
					List: []*dst.Field{
						{Type: &dst.Ident{Name: "error"}},
					},
				},
			},
			expected: "func(int, string) error",
		},
		{
			name: "func type with multiple results",
			input: &dst.FuncType{
				Params: &dst.FieldList{},
				Results: &dst.FieldList{
					List: []*dst.Field{
						{Type: &dst.Ident{Name: "int"}},
						{Type: &dst.Ident{Name: "error"}},
					},
				},
			},
			expected: "func() (int, error)",
		},
		{
			name: "ellipsis",
			input: &dst.Ellipsis{
				Elt: &dst.Ident{Name: "string"},
			},
			expected: "...string",
		},
		{
			name: "index expr (generic with single type param)",
			input: &dst.IndexExpr{
				X:     &dst.Ident{Name: "List"},
				Index: &dst.Ident{Name: "int"},
			},
			expected: "List[int]",
		},
		{
			name: "index list expr (generic with multiple type params)",
			input: &dst.IndexListExpr{
				X: &dst.Ident{Name: "Map"},
				Indices: []dst.Expr{
					&dst.Ident{Name: "string"},
					&dst.Ident{Name: "int"},
				},
			},
			expected: "Map[string, int]",
		},
		{
			name: "paren expr",
			input: &dst.ParenExpr{
				X: &dst.Ident{Name: "int"},
			},
			expected: "(int)",
		},
		{
			name:     "unknown type falls back to type name",
			input:    &dst.BadExpr{},
			expected: "*dst.BadExpr",
		},
	}

	for _, tt := range tests { //nolint:varnamelen // tt is idiomatic in Go tests
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := StringifyExpr(tt.input)
			if got != tt.expected {
				t.Errorf("StringifyExpr() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestStringifyStructType verifies that stringifyStructType correctly converts
// DST StructType nodes to their Go source code representation, preserving all
// field information including names, types, and tags.
//
// This is a regression test for Issue #34 where struct literals were reduced
// to "struct{}" regardless of their actual field definitions.
//
//nolint:funlen,maintidx // table-driven test with many comprehensive test cases
func TestStringifyStructType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    *dst.StructType
		expected string
	}{
		{
			name: "empty struct",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{},
				},
			},
			expected: "struct{}",
		},
		{
			name: "single field",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Timeout"}},
							Type:  &dst.Ident{Name: "int"},
						},
					},
				},
			},
			expected: "struct{ Timeout int }",
		},
		{
			name: "multiple fields with same type on one line",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Host"}, {Name: "Port"}},
							Type:  &dst.Ident{Name: "string"},
						},
					},
				},
			},
			expected: "struct{ Host, Port string }",
		},
		{
			name: "multiple fields with different types",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Debug"}},
							Type:  &dst.Ident{Name: "bool"},
						},
						{
							Names: []*dst.Ident{{Name: "Level"}},
							Type:  &dst.Ident{Name: "int"},
						},
						{
							Names: []*dst.Ident{{Name: "Name"}},
							Type:  &dst.Ident{Name: "string"},
						},
					},
				},
			},
			expected: "struct{ Debug bool; Level int; Name string }",
		},
		{
			name: "nested struct",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Config"}},
							Type: &dst.StructType{
								Fields: &dst.FieldList{
									List: []*dst.Field{
										{
											Names: []*dst.Ident{{Name: "Host"}},
											Type:  &dst.Ident{Name: "string"},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "struct{ Config struct{ Host string } }",
		},
		{
			name: "pointer field",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Next"}},
							Type: &dst.StarExpr{
								X: &dst.Ident{Name: "Node"},
							},
						},
					},
				},
			},
			expected: "struct{ Next *Node }",
		},
		{
			name: "slice field",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Items"}},
							Type: &dst.ArrayType{
								Elt: &dst.Ident{Name: "string"},
							},
						},
					},
				},
			},
			expected: "struct{ Items []string }",
		},
		{
			name: "map field",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Meta"}},
							Type: &dst.MapType{
								Key:   &dst.Ident{Name: "string"},
								Value: &dst.Ident{Name: "int"},
							},
						},
					},
				},
			},
			expected: "struct{ Meta map[string]int }",
		},
		{
			name: "function field",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Handler"}},
							Type: &dst.FuncType{
								Params: &dst.FieldList{
									List: []*dst.Field{
										{Type: &dst.Ident{Name: "int"}},
									},
								},
								Results: &dst.FieldList{
									List: []*dst.Field{
										{Type: &dst.Ident{Name: "error"}},
									},
								},
							},
						},
					},
				},
			},
			expected: "struct{ Handler func(int) error }",
		},
		{
			name: "complex combination",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Name"}},
							Type:  &dst.Ident{Name: "string"},
						},
						{
							Names: []*dst.Ident{{Name: "Data"}},
							Type: &dst.MapType{
								Key: &dst.Ident{Name: "string"},
								Value: &dst.InterfaceType{
									Methods: &dst.FieldList{},
								},
							},
						},
						{
							Names: []*dst.Ident{{Name: "Next"}},
							Type: &dst.StarExpr{
								X: &dst.Ident{Name: "Node"},
							},
						},
					},
				},
			},
			expected: "struct{ Name string; Data map[string]interface{}; Next *Node }",
		},
		{
			name: "embedded field (no field name)",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: nil,
							Type:  &dst.Ident{Name: "Reader"},
						},
					},
				},
			},
			expected: "struct{ Reader }",
		},
		{
			name: "multiple embedded fields",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: nil,
							Type:  &dst.Ident{Name: "Reader"},
						},
						{
							Names: nil,
							Type:  &dst.Ident{Name: "Writer"},
						},
					},
				},
			},
			expected: "struct{ Reader; Writer }",
		},
		{
			name: "struct with tags",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Name"}},
							Type:  &dst.Ident{Name: "string"},
							Tag:   &dst.BasicLit{Value: "`json:\"name\"`"},
						},
					},
				},
			},
			expected: "struct{ Name string `json:\"name\"` }",
		},
		{
			name: "multiple fields with tags",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "ID"}},
							Type:  &dst.Ident{Name: "int"},
							Tag:   &dst.BasicLit{Value: "`json:\"id\"`"},
						},
						{
							Names: []*dst.Ident{{Name: "Name"}},
							Type:  &dst.Ident{Name: "string"},
							Tag:   &dst.BasicLit{Value: "`json:\"name\"`"},
						},
					},
				},
			},
			expected: "struct{ ID int `json:\"id\"`; Name string `json:\"name\"` }",
		},
		{
			name: "channel field",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Ch"}},
							Type: &dst.ChanType{
								Dir:   dst.SEND | dst.RECV,
								Value: &dst.Ident{Name: "int"},
							},
						},
					},
				},
			},
			expected: "struct{ Ch chan int }",
		},
		{
			name: "array field (fixed size)",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Buffer"}},
							Type: &dst.ArrayType{
								Len: &dst.BasicLit{Value: "256"},
								Elt: &dst.Ident{Name: "byte"},
							},
						},
					},
				},
			},
			expected: "struct{ Buffer [256]byte }",
		},
		{
			name: "qualified type field",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Timestamp"}},
							Type: &dst.SelectorExpr{
								X:   &dst.Ident{Name: "time"},
								Sel: &dst.Ident{Name: "Time"},
							},
						},
					},
				},
			},
			expected: "struct{ Timestamp time.Time }",
		},
		{
			name: "interface literal field",
			input: &dst.StructType{
				Fields: &dst.FieldList{
					List: []*dst.Field{
						{
							Names: []*dst.Ident{{Name: "Handler"}},
							Type: &dst.InterfaceType{
								Methods: &dst.FieldList{
									List: []*dst.Field{
										{
											Names: []*dst.Ident{{Name: "Handle"}},
											Type: &dst.FuncType{
												Params: &dst.FieldList{
													List: []*dst.Field{
														{Type: &dst.Ident{Name: "string"}},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expected: "struct{ Handler interface{ Handle(string) } }",
		},
	}

	for _, tt := range tests { //nolint:varnamelen // tt is idiomatic in Go tests
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := stringifyStructType(tt.input)
			if got != tt.expected {
				t.Errorf("stringifyStructType() = %q, want %q", got, tt.expected)
			}
		})
	}
}
