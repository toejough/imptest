package generate

import (
	"go/token"
	"sort"

	"github.com/dave/dst"
	detect "github.com/toejough/imptest/impgen/run/3_detect"
)

// GenerateStructDependencyCode generates dependency mock code for a struct type.
// It converts the struct type into an interface-like representation by collecting all methods
// on the struct, then delegates to the existing dependencyGenerator.
//
//nolint:revive // stutter acceptable for exported API consistency
func GenerateStructDependencyCode(
	astFiles []*dst.File,
	info GeneratorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	structWithDetails detect.StructWithDetails,
) (string, error) {
	// Collect all methods for this struct type
	methods := detect.CollectStructMethods(astFiles, fset, structWithDetails.TypeName)

	// Convert methods map to sorted slice of method names (for deterministic output)
	methodNames := make([]string, 0, len(methods))
	for name := range methods {
		methodNames = append(methodNames, name)
	}

	sort.Strings(methodNames)

	// Create an interface type with the struct's methods
	// This allows us to reuse the existing dependencyGenerator
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
	ifaceDetails := detect.IfaceWithDetails{
		Iface:         interfaceType,
		TypeParams:    structWithDetails.TypeParams,
		SourceImports: structWithDetails.SourceImports,
		IsStructType:  true, // Mark as struct-derived to generate synthetic interface in output
	}

	// Delegate to the existing dependencyGenerator
	return GenerateDependencyCode(astFiles, info, fset, pkgImportPath, pkgLoader, ifaceDetails)
}

// GenerateStructTargetCode generates target wrapper code for a struct type.
// It converts the struct type into an interface-like representation by collecting all methods
// on the struct, then delegates to the existing interfaceTargetGenerator.
//
//nolint:revive // stutter acceptable for exported API consistency
func GenerateStructTargetCode(
	astFiles []*dst.File,
	info GeneratorInfo,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	structWithDetails detect.StructWithDetails,
) (string, error) {
	// Collect all methods for this struct type
	methods := detect.CollectStructMethods(astFiles, fset, structWithDetails.TypeName)

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
	ifaceDetails := detect.IfaceWithDetails{
		Iface:         interfaceType,
		TypeParams:    structWithDetails.TypeParams,
		SourceImports: structWithDetails.SourceImports,
	}

	// Delegate to the existing interfaceTargetGenerator with isStructType=true
	// The generator will create wrappers for all methods on the struct using pointer types
	return GenerateInterfaceTargetCode(astFiles, info, fset, pkgImportPath, pkgLoader, ifaceDetails, true)
}
