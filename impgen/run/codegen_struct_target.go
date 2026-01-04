//nolint:wsl_v5
package run

import (
	"go/token"
	"sort"

	"github.com/dave/dst"
)

// generateStructTargetCode generates target wrapper code for a struct type.
// It converts the struct type into an interface-like representation by collecting all methods
// on the struct, then delegates to the existing interfaceTargetGenerator.
func generateStructTargetCode(
	astFiles []*dst.File,
	info generatorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader PackageLoader,
	structWithDetails structWithDetails,
) (string, error) {
	// Collect all methods for this struct type
	methods := collectStructMethods(astFiles, fset, structWithDetails.typeName)

	// Convert methods map to sorted slice of method names (for deterministic output)
	methodNames := make([]string, 0, len(methods))
	for name := range methods {
		methodNames = append(methodNames, name)
	}
	sort.Strings(methodNames)

	// Create an interface type with the struct's methods
	// This allows us to reuse the existing interfaceTargetGenerator
	interfaceType := &dst.InterfaceType{
		Methods: &dst.FieldList{
			List: make([]*dst.Field, 0, len(methods)),
		},
	}

	// Add each method to the interface type (in sorted order for deterministic output)
	for _, methodName := range methodNames {
		funcType := methods[methodName]
		interfaceType.Methods.List = append(interfaceType.Methods.List, &dst.Field{
			Names: []*dst.Ident{{Name: methodName}},
			Type:  funcType,
		})
	}

	// Create ifaceWithDetails from the synthetic interface
	ifaceDetails := ifaceWithDetails{
		iface:         interfaceType,
		typeParams:    structWithDetails.typeParams,
		sourceImports: structWithDetails.sourceImports,
	}

	// Delegate to the existing interfaceTargetGenerator with isStructType=true
	// The generator will create wrappers for all methods on the struct using pointer types
	return generateInterfaceTargetCode(astFiles, info, fset, pkgImportPath, pkgLoader, ifaceDetails, true)
}
