package reorder

import (
	"bytes"
	"fmt"
	"go/token"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/dave/dst"
	"github.com/dave/dst/decorator"
	"github.com/dave/dst/dstutil"
)

// Section represents a declaration section in a Go file.
type Section struct {
	Name     string // e.g., "Imports", "Exported Types", "unexported functions"
	Position int    // Position in file (1-indexed)
	Expected int    // Expected position (1-indexed), 0 if section shouldn't exist
}

// SectionOrder represents the detected sections in a file and their order.
type SectionOrder struct {
	Sections []Section
}

// AnalyzeSectionOrder analyzes the current declaration order in source code.
// Returns a SectionOrder showing which sections are present and their positions.
func AnalyzeSectionOrder(src string) (*SectionOrder, error) {
	dec := decorator.NewDecorator(token.NewFileSet())

	file, err := dec.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("failed to parse source: %w", err)
	}

	// Map section names to their expected positions (1-indexed)
	//nolint:mnd // These are the canonical ordering positions from CLAUDE.md
	expectedPositions := map[string]int{
		"Imports":              1,
		"main()":               2,
		"Exported Constants":   3,
		"Exported Enums":       4,
		"Exported Variables":   5,
		"Exported Types":       6,
		"Exported Functions":   7,
		"unexported constants": 8,
		"unexported enums":     9,
		"unexported variables": 10,
		"unexported types":     11,
		"unexported functions": 12,
	}

	// Track which sections we've seen and their first occurrence position
	sectionPositions := make(map[string]int)
	currentPos := 0

	// Walk through original declarations to track section transitions
	for _, decl := range file.Decls {
		currentPos++

		sectionName := identifySection(decl)
		if sectionName == "" {
			continue
		}

		// Record first occurrence of each section
		if _, seen := sectionPositions[sectionName]; !seen {
			sectionPositions[sectionName] = currentPos
		}
	}

	// Build the section list
	sections := make([]Section, 0, len(sectionPositions))
	for name, pos := range sectionPositions {
		sections = append(sections, Section{
			Name:     name,
			Position: pos,
			Expected: expectedPositions[name],
		})
	}

	// Sort by current position
	slices.SortFunc(sections, func(a, b Section) int {
		return a.Position - b.Position
	})

	return &SectionOrder{Sections: sections}, nil
}

// File reorders declarations in a dst.File according to project conventions.
func File(file *dst.File) error {
	categorized := categorizeDeclarations(file)
	reordered := reassembleDeclarations(categorized)
	file.Decls = reordered

	return nil
}

// Source reorders declarations in Go source code according to project conventions.
// It preserves all comments and handles edge cases like iota blocks and type-method grouping.
func Source(src string) (string, error) {
	dec := decorator.NewDecorator(token.NewFileSet())

	file, err := dec.Parse(src)
	if err != nil {
		return "", fmt.Errorf("failed to parse source: %w", err)
	}

	err = File(file)
	if err != nil {
		return "", fmt.Errorf("failed to reorder: %w", err)
	}

	var buf bytes.Buffer

	res := decorator.NewRestorer()

	err = res.Fprint(&buf, file)
	if err != nil {
		return "", fmt.Errorf("failed to print: %w", err)
	}

	return buf.String(), nil
}

// categorizedDecls holds declarations organized by category.
type categorizedDecls struct {
	imports          []dst.Decl
	main             *dst.FuncDecl
	exportedConsts   []*dst.ValueSpec
	exportedEnums    []*enumGroup
	exportedVars     []*dst.ValueSpec
	exportedTypes    []*typeGroup
	exportedFuncs    []*dst.FuncDecl
	unexportedConsts []*dst.ValueSpec
	unexportedEnums  []*enumGroup
	unexportedVars   []*dst.ValueSpec
	unexportedTypes  []*typeGroup
	unexportedFuncs  []*dst.FuncDecl
}

// enumGroup pairs an enum type with its iota const block.
type enumGroup struct {
	typeName  string
	typeDecl  *dst.GenDecl
	constDecl *dst.GenDecl
}

// typeGroup holds a type and its associated constructors and methods.
type typeGroup struct {
	typeName          string
	typeDecl          *dst.GenDecl
	constructors      []*dst.FuncDecl
	exportedMethods   []*dst.FuncDecl
	unexportedMethods []*dst.FuncDecl
}

// categorizeDeclarations organizes all declarations by category.
//
//nolint:gocognit,gocyclo,cyclop,funlen,maintidx // Complex by nature - handles all Go declaration types
func categorizeDeclarations(file *dst.File) *categorizedDecls {
	cat := &categorizedDecls{}

	// Maps for grouping
	typeGroups := make(map[string]*typeGroup)
	enumTypes := make(map[string]bool)
	iotaBlocks := make(map[string]*dst.GenDecl)

	// First pass: collect all type names
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*dst.GenDecl); ok && genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				if tspec, ok := spec.(*dst.TypeSpec); ok {
					typeName := tspec.Name.Name
					typeGroups[typeName] = &typeGroup{typeName: typeName}
				}
			}
		}
	}

	// Second pass: categorize all declarations
	for _, decl := range file.Decls {
		switch genDecl := decl.(type) {
		case *dst.GenDecl:
			//nolint:exhaustive // We only care about IMPORT/CONST/VAR/TYPE; other tokens are intentionally ignored
			switch genDecl.Tok {
			case token.IMPORT:
				cat.imports = append(cat.imports, genDecl)
			case token.CONST:
				if isIotaBlock(genDecl) { //nolint:nestif // Categorization logic requires nested conditions
					// Extract type from first spec
					typeName := extractEnumType(genDecl)
					exported := isExported(typeName)

					iotaBlocks[typeName] = genDecl
					if exported {
						cat.exportedEnums = append(cat.exportedEnums, &enumGroup{
							typeName:  typeName,
							constDecl: genDecl,
						})
					} else {
						cat.unexportedEnums = append(cat.unexportedEnums, &enumGroup{
							typeName:  typeName,
							constDecl: genDecl,
						})
					}

					enumTypes[typeName] = true
				} else {
					// Regular const - extract specs for merging
					for _, spec := range genDecl.Specs {
						if vspec, ok := spec.(*dst.ValueSpec); ok {
							if len(vspec.Names) > 0 {
								exported := isExported(vspec.Names[0].Name)
								if exported {
									cat.exportedConsts = append(cat.exportedConsts, vspec)
								} else {
									cat.unexportedConsts = append(cat.unexportedConsts, vspec)
								}
							}
						}
					}
				}
			case token.VAR:
				// Extract specs for merging
				for _, spec := range genDecl.Specs {
					if vspec, ok := spec.(*dst.ValueSpec); ok {
						if len(vspec.Names) > 0 {
							exported := isExported(vspec.Names[0].Name)
							if exported {
								cat.exportedVars = append(cat.exportedVars, vspec)
							} else {
								cat.unexportedVars = append(cat.unexportedVars, vspec)
							}
						}
					}
				}
			case token.TYPE:
				// Extract type name
				for _, spec := range genDecl.Specs {
					if tspec, ok := spec.(*dst.TypeSpec); ok { //nolint:nestif // Type extraction requires nested type assertions
						typeName := tspec.Name.Name
						exported := isExported(typeName)

						// Create or get type group
						if typeGroups[typeName] == nil {
							typeGroups[typeName] = &typeGroup{
								typeName: typeName,
							}
						}

						typeGroups[typeName].typeDecl = genDecl

						// Add to categorized list if not an enum type
						if !enumTypes[typeName] {
							if exported {
								cat.exportedTypes = append(cat.exportedTypes, typeGroups[typeName])
							} else {
								cat.unexportedTypes = append(cat.unexportedTypes, typeGroups[typeName])
							}
						}
					}
				}
			default:
				// Other token types are ignored
			}
		case *dst.FuncDecl:
			switch {
			case genDecl.Name.Name == "main" && genDecl.Recv == nil:
				cat.main = genDecl
			case genDecl.Recv != nil:
				// Method - associate with type
				typeName := extractReceiverTypeName(genDecl.Recv)
				if typeGroups[typeName] == nil {
					typeGroups[typeName] = &typeGroup{
						typeName: typeName,
					}
				}

				methodExported := isExported(genDecl.Name.Name)
				if methodExported {
					typeGroups[typeName].exportedMethods = append(typeGroups[typeName].exportedMethods, genDecl)
				} else {
					typeGroups[typeName].unexportedMethods = append(typeGroups[typeName].unexportedMethods, genDecl)
				}
			default:
				// Standalone function or constructor
				funcName := genDecl.Name.Name
				exported := isExported(funcName)

				// Check if it's a constructor (NewTypeName pattern)
				if strings.HasPrefix(funcName, "New") { //nolint:nestif // Constructor matching requires nested logic
					suffix := funcName[3:] // Remove "New" prefix

					// Try exact match first (e.g., NewConfig → Config)
					if typeGroups[suffix] != nil {
						typeGroups[suffix].constructors = append(typeGroups[suffix].constructors, genDecl)
						continue
					}

					// Try longest prefix match (e.g., NewConfigWithTimeout → Config)
					// Sort type names by length (longest first) to get best match
					var typeNames []string
					for tn := range typeGroups {
						typeNames = append(typeNames, tn)
					}

					sort.Slice(typeNames, func(i, j int) bool {
						return len(typeNames[i]) > len(typeNames[j])
					})

					matched := false

					for _, tn := range typeNames {
						if strings.HasPrefix(suffix, tn) {
							if tg := typeGroups[tn]; tg != nil {
								tg.constructors = append(tg.constructors, genDecl)
								matched = true

								break
							}
						}
					}

					if matched {
						continue
					}
				}

				// Not a constructor, add to standalone functions
				if exported {
					cat.exportedFuncs = append(cat.exportedFuncs, genDecl)
				} else {
					cat.unexportedFuncs = append(cat.unexportedFuncs, genDecl)
				}
			}
		}
	}

	// Second pass: pair enum types with their const blocks
	for _, enumGroup := range cat.exportedEnums {
		if typeGroups[enumGroup.typeName] != nil {
			enumGroup.typeDecl = typeGroups[enumGroup.typeName].typeDecl
			// Remove from regular types
			for i, tg := range cat.exportedTypes {
				if tg.typeName == enumGroup.typeName {
					cat.exportedTypes = append(cat.exportedTypes[:i], cat.exportedTypes[i+1:]...)
					break
				}
			}
		}
	}

	for _, enumGroup := range cat.unexportedEnums {
		if typeGroups[enumGroup.typeName] != nil {
			enumGroup.typeDecl = typeGroups[enumGroup.typeName].typeDecl
			// Remove from regular types
			for i, tg := range cat.unexportedTypes {
				if tg.typeName == enumGroup.typeName {
					cat.unexportedTypes = append(cat.unexportedTypes[:i], cat.unexportedTypes[i+1:]...)
					break
				}
			}
		}
	}

	// Sort everything
	sortCategorized(cat)

	return cat
}

// containsIota checks if an expression contains iota.
func containsIota(expr dst.Expr) bool {
	found := false

	dstutil.Apply(expr, func(c *dstutil.Cursor) bool {
		if ident, ok := c.Node().(*dst.Ident); ok {
			if ident.Name == "iota" {
				found = true
				return false
			}
		}

		return true
	}, nil)

	return found
}

// extractEnumType extracts the type name from the first spec in an iota block.
func extractEnumType(decl *dst.GenDecl) string {
	if len(decl.Specs) == 0 {
		return ""
	}

	vspec, ok := decl.Specs[0].(*dst.ValueSpec)
	if !ok || vspec.Type == nil {
		return ""
	}

	return extractTypeName(vspec.Type)
}

// extractReceiverTypeName extracts the type name from a method receiver.
func extractReceiverTypeName(recv *dst.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}

	return extractTypeName(recv.List[0].Type)
}

// extractTypeName extracts the type name from a type expression.
func extractTypeName(expr dst.Expr) string {
	switch typeExpr := expr.(type) {
	case *dst.Ident:
		return typeExpr.Name
	case *dst.SelectorExpr:
		return typeExpr.Sel.Name
	case *dst.StarExpr:
		return extractTypeName(typeExpr.X)
	case *dst.IndexExpr:
		return extractTypeName(typeExpr.X)
	case *dst.IndexListExpr:
		return extractTypeName(typeExpr.X)
	}

	return ""
}

// identifySection determines which section a declaration belongs to.
//
//nolint:gocognit,cyclop,funlen,nestif,varnamelen // Complex type checking is inherent to declaration categorization
func identifySection(decl dst.Decl) string {
	switch d := decl.(type) {
	case *dst.GenDecl:
		if d.Tok == token.IMPORT {
			return "Imports"
		}

		if d.Tok == token.CONST {
			if isIotaBlock(d) {
				typeName := extractEnumType(d)
				if isExported(typeName) {
					return "Exported Enums"
				}

				return "unexported enums"
			}
			// Check if it's a merged const block
			if len(d.Specs) > 0 {
				if vspec, ok := d.Specs[0].(*dst.ValueSpec); ok {
					if len(vspec.Names) > 0 {
						if isExported(vspec.Names[0].Name) {
							return "Exported Constants"
						}

						return "unexported constants"
					}
				}
			}
		}

		if d.Tok == token.VAR {
			if len(d.Specs) > 0 {
				if vspec, ok := d.Specs[0].(*dst.ValueSpec); ok {
					if len(vspec.Names) > 0 {
						if isExported(vspec.Names[0].Name) {
							return "Exported Variables"
						}

						return "unexported variables"
					}
				}
			}
		}

		if d.Tok == token.TYPE {
			if len(d.Specs) > 0 {
				if tspec, ok := d.Specs[0].(*dst.TypeSpec); ok {
					if isExported(tspec.Name.Name) {
						return "Exported Types"
					}

					return "unexported types"
				}
			}
		}
	case *dst.FuncDecl:
		if d.Name.Name == "main" && d.Recv == nil {
			return "main()"
		}
		// Skip methods (they're part of type groups)
		if d.Recv != nil {
			typeName := extractReceiverTypeName(d.Recv)
			if isExported(typeName) {
				return "Exported Types"
			}

			return "unexported types"
		}

		if isExported(d.Name.Name) {
			return "Exported Functions"
		}

		return "unexported functions"
	}

	return ""
}

// isExported checks if a name is exported (starts with uppercase).
func isExported(name string) bool {
	if name == "" {
		return false
	}

	r := []rune(name)[0]

	return unicode.IsUpper(r)
}

// isIotaBlock checks if a const block uses iota.
func isIotaBlock(decl *dst.GenDecl) bool {
	if decl.Tok != token.CONST {
		return false
	}

	for _, spec := range decl.Specs {
		vspec, ok := spec.(*dst.ValueSpec)
		if !ok {
			continue
		}

		if slices.ContainsFunc(vspec.Values, containsIota) {
			return true
		}
	}

	return false
}

// mergeConstSpecs creates a single const block from multiple specs.
func mergeConstSpecs(specs []*dst.ValueSpec, comment string) *dst.GenDecl {
	dstSpecs := make([]dst.Spec, 0, len(specs))

	for _, spec := range specs {
		// Clear any existing decorations from the spec
		spec.Decs.Before = dst.NewLine
		spec.Decs.After = dst.NewLine
		dstSpecs = append(dstSpecs, spec)
	}

	decl := &dst.GenDecl{
		Tok:    token.CONST,
		Lparen: true, // Force parentheses
		Specs:  dstSpecs,
	}
	decl.Decs.Before = dst.EmptyLine
	decl.Decs.Start.Append("// " + comment)

	return decl
}

// mergeVarSpecs creates a single var block from multiple specs.
func mergeVarSpecs(specs []*dst.ValueSpec, comment string) *dst.GenDecl {
	dstSpecs := make([]dst.Spec, 0, len(specs))

	for _, spec := range specs {
		// Clear any existing decorations from the spec
		spec.Decs.Before = dst.NewLine
		spec.Decs.After = dst.NewLine
		dstSpecs = append(dstSpecs, spec)
	}

	decl := &dst.GenDecl{
		Tok:    token.VAR,
		Lparen: true, // Force parentheses
		Specs:  dstSpecs,
	}
	decl.Decs.Before = dst.EmptyLine
	decl.Decs.Start.Append("// " + comment)

	return decl
}

// reassembleDeclarations builds the final ordered declaration list.
//
//nolint:gocognit,cyclop,funlen // Complex by design - assembles all declaration categories in correct order
func reassembleDeclarations(cat *categorizedDecls) []dst.Decl {
	const extraCapacity = 10 // Extra capacity for main + merged const/var blocks

	// Pre-allocate with estimated capacity
	estimatedSize := len(cat.imports) + len(cat.exportedConsts) + len(cat.exportedEnums) +
		len(cat.exportedVars) + len(cat.exportedTypes) + len(cat.exportedFuncs) +
		len(cat.unexportedConsts) + len(cat.unexportedEnums) + len(cat.unexportedVars) +
		len(cat.unexportedTypes) + len(cat.unexportedFuncs) + extraCapacity

	decls := make([]dst.Decl, 0, estimatedSize)

	// Imports
	decls = append(decls, cat.imports...)

	// main() if present
	if cat.main != nil {
		decls = append(decls, cat.main)
	}

	// Exported constants (merged)
	if len(cat.exportedConsts) > 0 {
		constDecl := mergeConstSpecs(cat.exportedConsts, "Exported constants.")
		decls = append(decls, constDecl)
	}

	// Exported enums (type + const block pairs)
	for _, enumGrp := range cat.exportedEnums {
		if enumGrp.typeDecl != nil {
			enumGrp.typeDecl.Decs.Before = dst.EmptyLine
			decls = append(decls, enumGrp.typeDecl)
		}
		// Add comment header (clear existing first to avoid duplicates)
		enumGrp.constDecl.Decs.Start = nil
		enumGrp.constDecl.Decs.Before = dst.EmptyLine
		enumGrp.constDecl.Decs.Start.Append(fmt.Sprintf("// %s values.", enumGrp.typeName))
		decls = append(decls, enumGrp.constDecl)
	}

	// Exported variables (merged)
	if len(cat.exportedVars) > 0 {
		varDecl := mergeVarSpecs(cat.exportedVars, "Exported variables.")
		decls = append(decls, varDecl)
	}

	// Exported types (with constructors and methods)
	for _, typeGrp := range cat.exportedTypes {
		if typeGrp.typeDecl != nil {
			typeGrp.typeDecl.Decs.Before = dst.EmptyLine
			decls = append(decls, typeGrp.typeDecl)
		}

		for _, ctor := range typeGrp.constructors {
			ctor.Decs.Before = dst.EmptyLine
			decls = append(decls, ctor)
		}

		for _, method := range typeGrp.exportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}

		for _, method := range typeGrp.unexportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}
	}

	// Exported standalone functions
	for _, fn := range cat.exportedFuncs {
		fn.Decs.Before = dst.EmptyLine
		decls = append(decls, fn)
	}

	// Unexported constants (merged)
	if len(cat.unexportedConsts) > 0 {
		constDecl := mergeConstSpecs(cat.unexportedConsts, "unexported constants.")
		decls = append(decls, constDecl)
	}

	// Unexported enums (type + const block pairs)
	for _, enumGrp := range cat.unexportedEnums {
		if enumGrp.typeDecl != nil {
			enumGrp.typeDecl.Decs.Before = dst.EmptyLine
			decls = append(decls, enumGrp.typeDecl)
		}
		// Add comment header (clear existing first to avoid duplicates)
		enumGrp.constDecl.Decs.Start = nil
		enumGrp.constDecl.Decs.Before = dst.EmptyLine
		enumGrp.constDecl.Decs.Start.Append(fmt.Sprintf("// %s values.", enumGrp.typeName))
		decls = append(decls, enumGrp.constDecl)
	}

	// Unexported variables (merged)
	if len(cat.unexportedVars) > 0 {
		varDecl := mergeVarSpecs(cat.unexportedVars, "unexported variables.")
		decls = append(decls, varDecl)
	}

	// Unexported types (with constructors and methods)
	for _, typeGrp := range cat.unexportedTypes {
		if typeGrp.typeDecl != nil {
			typeGrp.typeDecl.Decs.Before = dst.EmptyLine
			decls = append(decls, typeGrp.typeDecl)
		}

		for _, ctor := range typeGrp.constructors {
			ctor.Decs.Before = dst.EmptyLine
			decls = append(decls, ctor)
		}

		for _, method := range typeGrp.exportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}

		for _, method := range typeGrp.unexportedMethods {
			method.Decs.Before = dst.EmptyLine
			decls = append(decls, method)
		}
	}

	// Unexported standalone functions
	for _, fn := range cat.unexportedFuncs {
		fn.Decs.Before = dst.EmptyLine
		decls = append(decls, fn)
	}

	return decls
}

// sortCategorized sorts all categorized declarations alphabetically.
func sortCategorized(cat *categorizedDecls) {
	// Sort const specs by name
	sort.Slice(cat.exportedConsts, func(i, j int) bool {
		return cat.exportedConsts[i].Names[0].Name < cat.exportedConsts[j].Names[0].Name
	})
	sort.Slice(cat.unexportedConsts, func(i, j int) bool {
		return cat.unexportedConsts[i].Names[0].Name < cat.unexportedConsts[j].Names[0].Name
	})

	// Sort var specs by name
	sort.Slice(cat.exportedVars, func(i, j int) bool {
		return cat.exportedVars[i].Names[0].Name < cat.exportedVars[j].Names[0].Name
	})
	sort.Slice(cat.unexportedVars, func(i, j int) bool {
		return cat.unexportedVars[i].Names[0].Name < cat.unexportedVars[j].Names[0].Name
	})

	// Sort enum groups by type name
	sort.Slice(cat.exportedEnums, func(i, j int) bool {
		return cat.exportedEnums[i].typeName < cat.exportedEnums[j].typeName
	})
	sort.Slice(cat.unexportedEnums, func(i, j int) bool {
		return cat.unexportedEnums[i].typeName < cat.unexportedEnums[j].typeName
	})

	// Sort type groups by type name
	sort.Slice(cat.exportedTypes, func(i, j int) bool {
		return cat.exportedTypes[i].typeName < cat.exportedTypes[j].typeName
	})
	sort.Slice(cat.unexportedTypes, func(i, j int) bool {
		return cat.unexportedTypes[i].typeName < cat.unexportedTypes[j].typeName
	})

	// Sort within each type group
	for _, typeGrp := range cat.exportedTypes {
		sort.Slice(typeGrp.constructors, func(i, j int) bool {
			return typeGrp.constructors[i].Name.Name < typeGrp.constructors[j].Name.Name
		})
		sort.Slice(typeGrp.exportedMethods, func(i, j int) bool {
			return typeGrp.exportedMethods[i].Name.Name < typeGrp.exportedMethods[j].Name.Name
		})
		sort.Slice(typeGrp.unexportedMethods, func(i, j int) bool {
			return typeGrp.unexportedMethods[i].Name.Name < typeGrp.unexportedMethods[j].Name.Name
		})
	}

	for _, typeGrp := range cat.unexportedTypes {
		sort.Slice(typeGrp.constructors, func(i, j int) bool {
			return typeGrp.constructors[i].Name.Name < typeGrp.constructors[j].Name.Name
		})
		sort.Slice(typeGrp.exportedMethods, func(i, j int) bool {
			return typeGrp.exportedMethods[i].Name.Name < typeGrp.exportedMethods[j].Name.Name
		})
		sort.Slice(typeGrp.unexportedMethods, func(i, j int) bool {
			return typeGrp.unexportedMethods[i].Name.Name < typeGrp.unexportedMethods[j].Name.Name
		})
	}

	// Sort standalone functions
	sort.Slice(cat.exportedFuncs, func(i, j int) bool {
		return cat.exportedFuncs[i].Name.Name < cat.exportedFuncs[j].Name.Name
	})
	sort.Slice(cat.unexportedFuncs, func(i, j int) bool {
		return cat.unexportedFuncs[i].Name.Name < cat.unexportedFuncs[j].Name.Name
	})
}
