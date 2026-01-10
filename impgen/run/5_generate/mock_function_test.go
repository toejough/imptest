package generate

import (
	"maps"
	"testing"

	"github.com/dave/dst"
)

func TestCollectImportFromSelector(t *testing.T) {
	t.Parallel()

	gen := &functionDependencyGenerator{}

	tests := []struct {
		name       string
		sel        *dst.SelectorExpr
		imports    []*dst.ImportSpec
		seenPaths  map[string]bool
		wantLen    int
		wantPath   string
		wantSeenK  string
		wantNilRet bool
	}{
		{
			name: "non-ident X returns nil",
			sel: &dst.SelectorExpr{
				X:   &dst.SelectorExpr{}, // Not an Ident
				Sel: &dst.Ident{Name: "Request"},
			},
			imports:    nil,
			seenPaths:  make(map[string]bool),
			wantNilRet: true,
		},
		{
			name: "match by alias",
			sel: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "h"},
				Sel: &dst.Ident{Name: "Request"},
			},
			imports: []*dst.ImportSpec{
				{Name: &dst.Ident{Name: "h"}, Path: &dst.BasicLit{Value: `"net/http"`}},
			},
			seenPaths: make(map[string]bool),
			wantLen:   1,
			wantPath:  "net/http",
			wantSeenK: "net/http",
		},
		{
			name: "match by path suffix",
			sel: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "http"},
				Sel: &dst.Ident{Name: "Request"},
			},
			imports: []*dst.ImportSpec{
				{Path: &dst.BasicLit{Value: `"net/http"`}},
			},
			seenPaths: make(map[string]bool),
			wantLen:   1,
			wantPath:  "net/http",
			wantSeenK: "net/http",
		},
		{
			name: "match by exact path",
			sel: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "fmt"},
				Sel: &dst.Ident{Name: "Println"},
			},
			imports: []*dst.ImportSpec{
				{Path: &dst.BasicLit{Value: `"fmt"`}},
			},
			seenPaths: make(map[string]bool),
			wantLen:   1,
			wantPath:  "fmt",
			wantSeenK: "fmt",
		},
		{
			name: "already seen returns nil",
			sel: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "http"},
				Sel: &dst.Ident{Name: "Request"},
			},
			imports: []*dst.ImportSpec{
				{Path: &dst.BasicLit{Value: `"net/http"`}},
			},
			seenPaths:  map[string]bool{"net/http": true},
			wantNilRet: true,
		},
		{
			name: "no match returns nil",
			sel: &dst.SelectorExpr{
				X:   &dst.Ident{Name: "unknown"},
				Sel: &dst.Ident{Name: "Type"},
			},
			imports: []*dst.ImportSpec{
				{Path: &dst.BasicLit{Value: `"net/http"`}},
			},
			seenPaths:  make(map[string]bool),
			wantNilRet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Make a copy of seenPaths for each test
			seenPaths := make(map[string]bool)
			maps.Copy(seenPaths, tt.seenPaths)

			result := gen.collectImportFromSelector(tt.sel, tt.imports, seenPaths)

			if tt.wantNilRet {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}

				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("expected %d results, got %d", tt.wantLen, len(result))
				return
			}

			if tt.wantLen > 0 && result[0].Path != tt.wantPath {
				t.Errorf("expected path %s, got %s", tt.wantPath, result[0].Path)
			}

			if tt.wantSeenK != "" && !seenPaths[tt.wantSeenK] {
				t.Errorf("expected %s to be in seenPaths", tt.wantSeenK)
			}
		})
	}
}

func TestResolveFunctionPackageInfo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		info          GeneratorInfo
		pkgImportPath string
		wantPkgPath   string
		wantQualifier string
	}{
		{
			name: "external package unqualified name",
			info: GeneratorInfo{
				InterfaceName: "ProcessOrder",
				PkgName:       "mypkg",
			},
			pkgImportPath: "github.com/example/mypkg",
			wantPkgPath:   "github.com/example/mypkg",
			wantQualifier: "mypkg",
		},
		{
			name: "local same package",
			info: GeneratorInfo{
				InterfaceName: "ProcessOrder",
				PkgName:       "mypkg",
			},
			pkgImportPath: ".",
			wantPkgPath:   "",
			wantQualifier: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Pass nil for pkgLoader since these test cases don't use it
			pkgPath, qualifier := resolveFunctionPackageInfo(tt.info, tt.pkgImportPath, nil)

			if pkgPath != tt.wantPkgPath {
				t.Errorf("resolveFunctionPackageInfo() pkgPath = %v, want %v", pkgPath, tt.wantPkgPath)
			}

			if qualifier != tt.wantQualifier {
				t.Errorf("resolveFunctionPackageInfo() qualifier = %v, want %v", qualifier, tt.wantQualifier)
			}
		})
	}
}
