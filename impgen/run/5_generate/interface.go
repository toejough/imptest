package generate

import (
	"errors"
	"fmt"
	"go/token"
	"strings"

	"github.com/dave/dst"

	detect "github.com/toejough/imptest/impgen/run/3_detect"
)

// unexported variables.
var (
	errUnsupportedEmbeddedType = errors.New("unsupported embedded type")
)

// Types

// codeGenerator Methods

// checkIfFmtNeeded pre-scans all interface methods to determine if fmt import is needed.
// fmt is no longer needed for callbacks since we use t.Fatalf instead of panic(fmt.Sprintf(...)).

// Callbacks need reflect for DeepEqual but no longer need fmt

// Early exit once we know reflect is needed

// checkIfImptestNeeded pre-scans all interface methods to determine if imptest import is needed.
// imptest is needed when any method has parameters (for ExpectArgsShould).

// Early exit once we know imptest is needed

// checkIfQualifierNeeded pre-scans to determine if the package qualifier is needed.

// checkIfReflectNeeded pre-scans all interface methods to determine if reflect import is needed.

// Early exit once we know reflect is needed

// checkIfValidForExternalUsage checks if the interface can be mocked from an external package.

// collectAdditionalImports collects all external type imports needed for the interface methods.
// It walks through all method parameters and return types, collecting package references.

// Get source imports from the first AST file

// Collect imports from all method signatures

// Convert map to slice and sort for deterministic output

// Sort by import path for consistent ordering

// collectFieldListImports collects external imports from a field list (params or returns).

// Collect direct imports from the field type

// If this is a type alias that resolves to a function type,
// also collect imports from the underlying function's parameters and returns

// collectMethodImports walks through all interface methods and collects external imports.

// Deduplicate by path

// Collect from parameters

// Collect from return types

// extractFuncParams returns information about function-typed parameters in a function.
// Supports inline function types, local type aliases, and external types.

// Resolve the parameter type to a function type if possible

// forEachMethod iterates over interface methods and calls the callback for each.
// This is safe to call without error checking because we already validated the interface
// structure during method name collection in generateImplementationCode. If an error occurs
// here, it indicates a programming error and will cause a panic in the underlying function.

// Ignore error - interface was already validated during method name collection

// generate orchestrates the code generation process after initialization.

// Pre-scan to determine if reflect import is needed

// Pre-scan to determine if imptest import is needed

// Pre-scan to determine if fmt import is needed

// Pre-scan to see if qualifier is needed

// If we have an interface name, we need the qualifier for interface verification.
// Exception: when pkgPath is empty but qualifier is set (test package case), the import
// already exists from baseGenerator, so we don't need to add it.

// generateBuilderShortcuts generates InjectResult/InjectPanic/Resolve shortcut methods on the builder.

// Validator that only checks method name

// Generate InjectResult shortcut

// Multiple return values - InjectResults

// No results - generate Resolve shortcut

// Generate InjectPanic shortcut (always available)

// generateCallStruct generates the union call struct that can hold any method call.

// generateCallStructParamFields generates the parameter fields for a call struct.

// generateCallbackInvocationMethod generates type-safe InvokeFn and ExpectReturned methods for a callback parameter.
// For a callback like fn func(path string, d fs.DirEntry, err error) error, this generates:
//   - InvokeFn(path string, d fs.DirEntry, err error) - type-safe invocation
//   - ExpectReturned(result0 error) - type-safe result verification

// Generate the callback result type (holds response from the callback)

// Add private fields for each return value

// Add panic field to track callback panics

// Add imptest.Tester field for proper test failures

// Generate the ExpectReturned method with type-safe parameters

// Generate type-safe parameters for ExpectReturned

// Check if callback panicked instead of returning

// Generate type-safe comparisons

// Use DeepEqual for non-comparable types (like functions, slices, maps)

// Keep for future optimizations (use == for comparable types)

// Generate ExpectReturnedShould with matcher support

// Check if callback panicked instead of returning

// Generate ExpectPanicWith to verify callback panics

// Generate the type-safe InvokeFn method

// Generate type-safe parameters for InvokeFn

// Create result channel with typed response

// Send typed request

// Receive typed response

// Return typed result

// generateCallbackHelper generates a helper function to invoke a callback with dynamic arguments.
// generateCallbackRequestResponseStructs generates type-safe request and response structs
// for a callback parameter. For a callback like fn func(path string, d fs.DirEntry, err error) error:
//   - TreeWalkerImpWalkCallFnRequest with fields for each parameter plus ResultChan
//   - TreeWalkerImpWalkCallFnResponse with fields for each return value

// Generate request struct

// Add fields for each callback parameter

// Capitalize the parameter name for the field

// Add result channel

// Generate response struct

// Add fields for each return value

// Add panic field to capture callback panics

// generateConstructor generates the New{ImpName} constructor function.

// generateExpectArgsAre generates the type-safe ExpectArgsAre method on the builder.

// Method signature

// Validator function

// GetCall and return

// generateExpectArgsShould generates the matcher-based ExpectArgsShould method on the builder.

// Method signature - all params are 'any'

// Validator function

// GetCall and return

// generateExpectCallIsStruct generates the struct for expecting specific method calls.

// generateGetCurrentCallMethod generates the GetCurrentCall method that returns the current or next call.

// generateHeader writes the package declaration and imports for the generated file.

// generateInjectPanicMethod generates the InjectPanic method for simulating panics.

// generateInjectResultMethod generates the InjectResult method for methods with a single return value.

// generateInjectResultsMethod generates the InjectResults method for methods with multiple return values.

// generateInterfaceVerification generates a compile-time check that the mock implements the interface.

// generateMainStruct generates the main implementation struct that handles test call tracking.

// generateMethodBuilder generates the builder struct and all its methods for a single interface method.

// Generate builder struct

// Generate ExpectCallIs.MethodName() -> returns builder

// Only generate ExpectArgs methods if the method has parameters

// Generate ExpectArgsAre (type-safe)

// Generate ExpectArgsShould (matcher-based)

// Generate shortcut InjectResult/InjectPanic/Resolve

// generateMethodBuilders generates builder structs and methods for each interface method.

// generateMethodCallStruct generates the call struct for a specific method, which tracks the method call parameters.

// Generate callback request/response structs first (they need to exist before the call struct references them)

// Add callback coordination channels with typed request channels

// Add imptest.Tester field for passing to callback results

// generateMethodResponseMethods generates the InjectResult, InjectResults, InjectPanic, and Resolve methods
// for a call struct.

// Generate callback invocation methods if there are func-typed parameters

// Compute the request/response type names (same logic as in generateCallbackRequestResponseStructs)

// generateMethodResponseStruct generates the response struct for a method, which holds return values or panic data.

// generateMethodStructs generates the call and response structs for each interface method.

// generateMockMethod generates a single mock method that creates a call, sends it to the imp, and handles the response.

// Only write return statement if there are no callbacks (otherwise it's in the select)

// generateMockMethods generates the mock methods that implement the interface on the mock struct.

// generateMockStruct generates the mock struct that wraps the implementation.

// generateResolveMethod generates the Resolve method for methods with no return values.

// generateResponseStructResultFields generates the result fields for a response struct.

// generateTimedStruct generates the struct and method for timed call expectations.

// getSourceImports returns combined import specs from all AST files.
// This collects imports from all files including source and generated files
// to ensure type resolution and import collection work correctly.

// methodBuilderName returns the builder struct name for a method (e.g. "MyImpAddBuilder").

// methodCallName returns the call struct name for a method (e.g. "MyImpDoSomethingCall").

// methodTemplateData returns template data for a specific method.

// renderField renders a single field with its name and type.

// Names

// Type

// renderFieldList renders a *dst.FieldList as Go code for return types.

// templateData returns common template data for this generator.
// The result is cached after the first call to avoid redundant struct construction.

// writeCallStructField writes a single field assignment for a call struct initialization.

// writeCallStructFields writes the field assignments for initializing a call struct.

// writeComparisonCheck writes either an equality or matcher-based comparison check.
// When useMatcher is true, uses imptest.MatchValue for flexible matching.
// When useMatcher is false, uses == or reflect.DeepEqual for equality checks.

// writeExpectArgsAreChecks writes parameter equality checks for ExpectArgsAre.

// writeExpectArgsShouldChecks writes matcher-based checks for ExpectArgsShould.

// writeInjectResultsArgs writes the argument list for InjectResults call.

// writeInjectResultsParams writes the parameter list for InjectResults method and returns the result names.

// Write parameters using shared formatter with proper type qualification

// Build names array for return

// writeInjectResultsResponseFields writes the response struct field assignments for InjectResults.

// writeMethodParams writes the method parameters in the form "name type, name2 type2".

// writeMethodParamsAsAny writes method parameters with all types as 'any'.

// writeMethodParamsWithFormatter writes method parameters using a custom type formatter.
// The typeFormatter function receives the qualified type string and returns the formatted type to use.
// This allows writing params as "name actualType" or "name any" with the same iteration logic.

// Indices not used when using visitParams

// writeMethodSignature writes the method name and parameters (e.g., "MethodName(a int, b string)").

// writeMockMethodCallCreation writes the response channel and call struct creation.

// Create callback channels with typed requests

// Add callback channels to call struct

// Add testing.TB for callback result verification

// writeMockMethodEventDispatch writes the call event creation and dispatch to the imp.

// writeMockMethodResponseHandling writes the response reception and panic handling.

// No callbacks - simple response handling

// With callbacks - loop until final response

// Generate case for each callback channel with type-safe invocation

// Invoke callback directly with typed request fields

// Use the field name from the request struct

// Send typed response back

// Generate case for final response

// Return early if there are results

// writeMockMethodSignature writes the mock method signature and opening brace.

// writeNamedParamFields writes fields for named parameters.

// writeParamChecks writes parameter comparison checks.
// When useMatcher is true, uses imptest.MatchValue for flexible matching.
// When useMatcher is false, uses == or reflect.DeepEqual for equality checks.

// writeReturnStatement writes the return statement for a mock method.

// writeReturnValues writes all return values from the response struct.

// writeUnnamedParamField writes a field for an unnamed parameter.

// forEachInterfaceMethod iterates over interface methods and calls the callback for each,
// expanding embedded interfaces.
func forEachInterfaceMethod(
	iface *dst.InterfaceType,
	astFiles []*dst.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	callback func(methodName string, ftype *dst.FuncType),
) error {
	for _, field := range iface.Methods.List {
		err := interfaceProcessFieldMethods(
			field,
			astFiles,
			fset,
			pkgImportPath,
			pkgLoader,
			callback,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

// forEachParamField iterates over parameter fields, handling both named and unnamed parameters.
// It calls the action callback for each field with the computed field name and parameter name.

// Entry Point - Public

// generateImplementationCode generates the complete mock implementation code for an interface.

// Private Functions

// interfaceCollectMethodNames collects all method names from an interface, including embedded ones.
func interfaceCollectMethodNames(
	iface *dst.InterfaceType, astFiles []*dst.File, fset *token.FileSet,
	pkgImportPath string, pkgLoader detect.PackageLoader,
) ([]string, error) {
	var methodNames []string

	err := forEachInterfaceMethod(
		iface, astFiles, fset, pkgImportPath, pkgLoader,
		func(methodName string, _ *dst.FuncType) {
			methodNames = append(methodNames, methodName)
		},
	)
	if err != nil {
		return nil, err
	}

	return methodNames, nil
}

// interfaceExpandEmbedded expands an embedded interface by loading its definition and recursively processing methods.
func interfaceExpandEmbedded(
	embeddedType dst.Expr,
	astFiles []*dst.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	callback func(methodName string, ftype *dst.FuncType),
) error {
	embeddedName, embeddedPkgPath, err := resolveEmbeddedType(
		embeddedType,
		astFiles,
		pkgImportPath,
		pkgLoader,
	)
	if err != nil {
		return err
	}

	// Load the embedded interface's package files
	embeddedAstFiles, embeddedFset := astFiles, fset
	if embeddedPkgPath != pkgImportPath {
		embeddedAstFiles, embeddedFset, _, err = pkgLoader.Load(embeddedPkgPath)
		if err != nil {
			return fmt.Errorf(
				"failed to load external embedded interface package %s: %w",
				embeddedPkgPath,
				err,
			)
		}
	}

	// Find and recursively process the embedded interface
	embeddedIface, err := detect.GetMatchingInterfaceFromAST(
		embeddedAstFiles,
		embeddedName,
		embeddedPkgPath,
	)
	if err != nil {
		return fmt.Errorf("failed to find embedded interface %s: %w", embeddedName, err)
	}

	return forEachInterfaceMethod(
		embeddedIface.Iface, embeddedAstFiles, embeddedFset, embeddedPkgPath, pkgLoader, callback,
	)
}

// interfaceProcessFieldMethods handles a single field in an interface's method list.
func interfaceProcessFieldMethods(
	field *dst.Field,
	astFiles []*dst.File,
	fset *token.FileSet,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
	callback func(methodName string, ftype *dst.FuncType),
) error {
	// Handle embedded interfaces (they have no names)
	if !hasFieldNames(field) {
		return interfaceExpandEmbedded(
			field.Type,
			astFiles,
			fset,
			pkgImportPath,
			pkgLoader,
			callback,
		)
	}

	// Skip non-function types (shouldn't happen in a valid interface, but be safe)
	ftype, ok := field.Type.(*dst.FuncType)
	if !ok {
		return nil
	}

	// Process each method name with the same function type
	for _, methodName := range field.Names {
		callback(methodName.Name, ftype)
	}

	return nil
}

// resolveEmbeddedType extracts the interface name and package path from an embedded type expression.
func resolveEmbeddedType(
	embeddedType dst.Expr,
	astFiles []*dst.File,
	pkgImportPath string,
	pkgLoader detect.PackageLoader,
) (name, pkgPath string, err error) {
	switch typ := embeddedType.(type) {
	case *dst.Ident:
		return typ.Name, pkgImportPath, nil
	case *dst.SelectorExpr:
		pkgIdent, ok := typ.X.(*dst.Ident)
		if !ok {
			return "", "", fmt.Errorf("%w: %T", errUnsupportedEmbeddedType, typ.X)
		}

		importPath, err := detect.FindImportPath(astFiles, pkgIdent.Name, pkgLoader)
		if err != nil {
			return "", "", fmt.Errorf(
				"failed to find import path for embedded interface %s.%s: %w",
				pkgIdent.Name, typ.Sel.Name, err,
			)
		}

		return typ.Sel.Name, importPath, nil
	default:
		return "", "", fmt.Errorf("%w: %T", errUnsupportedEmbeddedType, embeddedType)
	}
}

// resolvePackageInfo resolves the package path and qualifier for an interface.
// Handles special case of test packages needing to import the non-test version.
func resolvePackageInfo(
	info GeneratorInfo,
	pkgLoader detect.PackageLoader,
) (pkgPath, qualifier string, err error) {
	pkgPath, qualifier, err = GetPackageInfo(
		info.InterfaceName,
		pkgLoader,
		info.PkgName,
	)
	if errors.Is(err, ErrNotPackageReference) {
		// Not a package reference (e.g., Counter.Inc) - no import needed
		return "", "", nil
	}

	if err != nil {
		return "", "", err
	}

	// Special case: when in a test package (e.g., "imptest_test") and the interface
	// has no package qualifier (GetPackageInfo returned empty), the interface is from
	// the non-test version of this package. We need to import it with its full path.
	if qualifier == "" && strings.HasSuffix(info.PkgName, "_test") {
		basePkgPath, baseQualifier := resolveTestPackageImport(pkgLoader, info.PkgName)
		if basePkgPath != "" {
			return basePkgPath, baseQualifier, nil
		}
	}

	return pkgPath, qualifier, nil
}

// resolveTestPackageImport resolves the import path for the non-test version of a test package.
func resolveTestPackageImport(
	pkgLoader detect.PackageLoader,
	pkgName string,
) (pkgPath, qualifier string) {
	// Strip _test suffix to get the base package name
	basePkgName := strings.TrimSuffix(pkgName, "_test")

	// Load the non-test package to get its import path
	basePkgFiles, baseFset, _, err := pkgLoader.Load(".")
	if err != nil || len(basePkgFiles) == 0 {
		return "", ""
	}

	// Get the import path from the package's own declaration
	path, err := detect.GetImportPathFromFiles(basePkgFiles, baseFset, "")
	if err != nil || path == "" {
		return "", ""
	}

	return path, basePkgName
}
