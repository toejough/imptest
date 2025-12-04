package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

func main() {
	info := getGeneratorInfo()
	fmt.Printf("Generator info: %+v\n", info)

	pkgImportPath, matchName := getPackageAndMatchName(info)
	fmt.Printf("Target package import path: %q, matchName: %q\n", pkgImportPath, matchName)

	astFiles, fset := parsePackageAST(pkgImportPath, info.pkgDir)
	fmt.Printf("Parsed %d AST files for package %q\n", len(astFiles), pkgImportPath)

	iface := getMatchingInterfaceFromAST(astFiles, matchName)
	if iface == nil {
		fmt.Printf("No interface named %q found in package %q.\n", matchName, pkgImportPath)
		return
	}

	fmt.Printf("Found interface %q in package %q:\n", matchName, pkgImportPath)
	printAstTree(iface, "  ")

	code := generateImplementationCode(iface, info, fset)
	fmt.Printf("Generated implementation code:\n%s\n", code)

	writeGeneratedCodeToFile(code, info.impName)
}

// getGeneratorInfo gathers basic information about the generator call.
func getGeneratorInfo() struct {
	cwd, pkgDir, pkgName, goFilePath, matchName, impName string
} {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	pkgName := os.Getenv("GOPACKAGE")
	goFile := os.Getenv("GOFILE")
	goFilePath := ""

	if goFile != "" {
		if filepath.IsAbs(goFile) {
			goFilePath = goFile
		} else {
			goFilePath = filepath.Join(cwd, goFile)
		}
	}

	pkgDir := cwd // assume current dir is the package dir
	matchName := ""
	impName := ""

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "--name" && i+1 < len(args) {
			impName = args[i+1]
			i++
		} else {
			matchName = args[i]
		}
	}

	return struct {
		cwd, pkgDir, pkgName, goFilePath, matchName, impName string
	}{cwd, pkgDir, pkgName, goFilePath, matchName, impName}
}

// getPackageAndMatchName determines the import path and interface name to match.
func getPackageAndMatchName(info struct {
	cwd, pkgDir, pkgName, goFilePath, matchName, impName string
},
) (string, string) {
	matchName := info.matchName
	// Check if matchName contains a dot, e.g. "run.ExampleInt"
	if dot := strings.Index(matchName, "."); dot != -1 {
		targetPkgImport := matchName[:dot]
		matchName = matchName[dot+1:]
		// Resolve the full import path for the target package
		astFiles, _ := parsePackageFiles(info.pkgDir)
		for _, fileAst := range astFiles {
			for _, imp := range fileAst.Imports {
				importPath, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					continue
				}
				// Check if the last segment matches the targetPkgImport
				parts := strings.Split(importPath, "/")
				if len(parts) > 0 && parts[len(parts)-1] == targetPkgImport {
					return importPath, matchName
				}
			}
		}

		return "", matchName
	}

	return info.pkgDir, matchName
}

// parsePackageAST loads and parses the AST for the given package import path.
func parsePackageAST(pkgImportPath, pkgDir string) ([]*ast.File, *token.FileSet) {
	if pkgImportPath == pkgDir || pkgImportPath == "" {
		return parsePackageFiles(pkgDir)
	}

	cfg := &packages.Config{Mode: packages.LoadAllSyntax}

	pkgs, err := packages.Load(cfg, pkgImportPath)
	if err != nil || len(pkgs) == 0 {
		fmt.Printf("error loading package %q: %v\n", pkgImportPath, err)
		return nil, token.NewFileSet()
	}

	return pkgs[0].Syntax, pkgs[0].Fset
}

// parsePackageFiles reads and parses all Go files in the package directory.
func parsePackageFiles(pkgDir string) ([]*ast.File, *token.FileSet) {
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		fmt.Printf("  error reading package dir %q: %v\n", pkgDir, err)
		return nil, token.NewFileSet()
	}

	var files []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if len(name) > 3 && name[len(name)-3:] == ".go" && name != "generated.go" {
			files = append(files, filepath.Join(pkgDir, name))
		}
	}

	fset := token.NewFileSet()

	astFiles := make([]*ast.File, 0, len(files))

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("  error reading file %q: %v\n", file, err)
			continue
		}

		parsedFile, err := parser.ParseFile(fset, file, data, parser.ParseComments)
		if err != nil {
			fmt.Printf("  error parsing file %q: %v\n", file, err)
			continue
		}

		astFiles = append(astFiles, parsedFile)
	}

	return astFiles, fset
}

// getMatchingInterfaceFromAST finds the interface by name in the ASTs.
func getMatchingInterfaceFromAST(astFiles []*ast.File, matchName string) *ast.InterfaceType {
	for _, fileAst := range astFiles {
		var found *ast.InterfaceType

		ast.Inspect(fileAst, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if ok {
				if iface, ok2 := ts.Type.(*ast.InterfaceType); ok2 && ts.Name.Name == matchName {
					found = iface
					return false
				}
			}

			return true
		})

		if found != nil {
			return found
		}
	}

	return nil
}

// generateImplementationCode creates the Go code for the interface implementation.
func generateImplementationCode(identifiedInterface *ast.InterfaceType, info struct {
	cwd, pkgDir, pkgName, goFilePath, matchName, impName string
}, fset *token.FileSet,
) string {
	impName := info.impName
	if impName == "" {
		impName = "interfaceImplementation"
	}

	mockName := impName + "Mock"

	var buf bytes.Buffer
	buf.WriteString("package main\n\n")
	buf.WriteString("// Code generated by generate.go. DO NOT EDIT.\n\n")
	buf.WriteString("import \"sync\"\n")
	buf.WriteString("import \"testing\"\n")
	buf.WriteString("import \"time\"\n\n")

	// Generate the Mock struct with a reference to the main struct
	buf.WriteString(fmt.Sprintf("type %s struct {\n", mockName))
	buf.WriteString(fmt.Sprintf("\timp *%s\n", impName))
	buf.WriteString("}\n\n")

	// Generate the main implementation struct with Mock field and channel
	callName := impName + "Call"
	expectCallToName := impName + "ExpectCallTo"
	timedName := impName + "Timed"

	buf.WriteString(fmt.Sprintf("type %s struct {\n", impName))
	buf.WriteString("\tt *testing.T\n")
	buf.WriteString(fmt.Sprintf("\tMock *%s\n", mockName))
	buf.WriteString(fmt.Sprintf("\tcallChan chan *%s\n", callName))
	buf.WriteString(fmt.Sprintf("\tExpectCallTo *%s\n", expectCallToName))
	buf.WriteString(fmt.Sprintf("\tcurrentCall *%s\n", callName))
	buf.WriteString(fmt.Sprintf("\tcallQueue []*%s\n", callName))
	buf.WriteString("\tqueueLock sync.Mutex\n")
	buf.WriteString("}\n\n")

	// Generate method-specific call structs first
	var methodNames []string

	for _, field := range identifiedInterface.Methods.List {
		if len(field.Names) == 0 {
			continue
		}

		for _, methodName := range field.Names {
			ftype, ok := field.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			// Collect method name for later use
			methodNames = append(methodNames, methodName.Name)

			// Generate method-specific call struct (e.g., IntOpsImpAddCall)
			methodCallName := impName + methodName.Name + "Call"
			buf.WriteString(fmt.Sprintf("type %s struct {\n", methodCallName))
			// Add response channel field
			buf.WriteString(fmt.Sprintf("\tresponseChan chan %sResponse\n", methodCallName))
			buf.WriteString("\tdone bool\n")

			// Generate fields for parameters
			if ftype.Params != nil && len(ftype.Params.List) > 0 {
				// Count total number of parameters
				totalParams := 0

				for _, param := range ftype.Params.List {
					if len(param.Names) > 0 {
						totalParams += len(param.Names)
					} else {
						totalParams++
					}
				}

				paramIndex := 0

				for _, param := range ftype.Params.List {
					paramType := exprToString(fset, param.Type)
					if len(param.Names) > 0 {
						// Named parameters
						for _, name := range param.Names {
							buf.WriteString(fmt.Sprintf("\t%s %s\n", name.Name, paramType))
						}
					} else {
						// Unnamed parameters - generate names
						fieldName := generateParamName(paramIndex, paramType, totalParams)
						buf.WriteString(fmt.Sprintf("\t%s %s\n", fieldName, paramType))

						paramIndex++
					}
				}
			}

			buf.WriteString("}\n\n")

			// Generate response type for this method call
			buf.WriteString(fmt.Sprintf("type %sResponse struct {\n", methodCallName))
			buf.WriteString("\tType string // \"return\", \"panic\", or \"resolve\"\n")

			if ftype.Results != nil && len(ftype.Results.List) > 0 {
				// Add fields for return values
				returnIndex := 0

				for _, result := range ftype.Results.List {
					resultType := exprToString(fset, result.Type)
					if len(result.Names) > 0 {
						for _, name := range result.Names {
							buf.WriteString(fmt.Sprintf("\t%s %s\n", name.Name, resultType))

							returnIndex++
						}
					} else {
						buf.WriteString(fmt.Sprintf("\tResult%d %s\n", returnIndex, resultType))
						returnIndex++
					}
				}
			}

			buf.WriteString("\tPanicValue interface{}\n")
			buf.WriteString("}\n\n")

			// Generate InjectResult, InjectPanic, and Resolve methods for the method call struct
			if ftype.Results != nil && len(ftype.Results.List) > 0 {
				// Methods WITH return values - only allow "return" and "panic"
				// Count total return values
				totalReturns := 0

				for _, result := range ftype.Results.List {
					if len(result.Names) > 0 {
						totalReturns += len(result.Names)
					} else {
						totalReturns++
					}
				}

				if totalReturns == 1 {
					// Single return value - generate InjectResult
					resultType := exprToString(fset, ftype.Results.List[0].Type)
					buf.WriteString(fmt.Sprintf("func (c *%s) InjectResult(result %s) {\n", methodCallName, resultType))
					buf.WriteString("\tc.done = true\n")
					buf.WriteString(fmt.Sprintf("\tc.responseChan <- %sResponse{Type: \"return\"", methodCallName))

					if len(ftype.Results.List[0].Names) > 0 {
						buf.WriteString(fmt.Sprintf(", %s: result", ftype.Results.List[0].Names[0].Name))
					} else {
						buf.WriteString(", Result0: result")
					}

					buf.WriteString("}\n")
					buf.WriteString("}\n")
				} else {
					// Multiple return values - generate InjectResults
					buf.WriteString(fmt.Sprintf("func (c *%s) InjectResults(", methodCallName))

					returnIndex := 0

					var returnParamNames []string

					for _, result := range ftype.Results.List {
						resultType := exprToString(fset, result.Type)
						if len(result.Names) > 0 {
							for _, name := range result.Names {
								if returnIndex > 0 {
									buf.WriteString(", ")
								}

								buf.WriteString(fmt.Sprintf("%s %s", name.Name, resultType))
								returnParamNames = append(returnParamNames, name.Name)
								returnIndex++
							}
						} else {
							if returnIndex > 0 {
								buf.WriteString(", ")
							}

							paramName := fmt.Sprintf("result%d", returnIndex)
							buf.WriteString(fmt.Sprintf("%s %s", paramName, resultType))
							returnParamNames = append(returnParamNames, paramName)
							returnIndex++
						}
					}

					buf.WriteString(") {\n")
					buf.WriteString("\tc.done = true\n")
					buf.WriteString(fmt.Sprintf("\tresp := %sResponse{Type: \"return\"", methodCallName))

					returnIndex = 0

					for _, result := range ftype.Results.List {
						if len(result.Names) > 0 {
							for _, name := range result.Names {
								buf.WriteString(fmt.Sprintf(", %s: %s", name.Name, returnParamNames[returnIndex]))
								returnIndex++
							}
						} else {
							buf.WriteString(fmt.Sprintf(", Result%d: %s", returnIndex, returnParamNames[returnIndex]))
							returnIndex++
						}
					}

					buf.WriteString("}\n")
					buf.WriteString("\tc.responseChan <- resp\n")
					buf.WriteString("}\n")
				}
				// Generate InjectPanic for methods with return values
				buf.WriteString(fmt.Sprintf("func (c *%s) InjectPanic(msg interface{}) {\n", methodCallName))
				buf.WriteString("\tc.done = true\n")
				buf.WriteString(fmt.Sprintf("\tc.responseChan <- %sResponse{Type: \"panic\", PanicValue: msg}\n", methodCallName))
				buf.WriteString("}\n")
			} else {
				// Methods WITHOUT return values - only allow "resolve" and "panic"
				// Generate Resolve
				buf.WriteString(fmt.Sprintf("func (c *%s) Resolve() {\n", methodCallName))
				buf.WriteString("\tc.done = true\n")
				buf.WriteString(fmt.Sprintf("\tc.responseChan <- %sResponse{Type: \"resolve\"}\n", methodCallName))
				buf.WriteString("}\n")
				// Generate InjectPanic for methods without return values
				buf.WriteString(fmt.Sprintf("func (c *%s) InjectPanic(msg interface{}) {\n", methodCallName))
				buf.WriteString("\tc.done = true\n")
				buf.WriteString(fmt.Sprintf("\tc.responseChan <- %sResponse{Type: \"panic\", PanicValue: msg}\n", methodCallName))
				buf.WriteString("}\n")
			}

			buf.WriteString("\n")
		}
	}

	// Generate Mock method implementations that intercept calls and send them on the channel
	for _, field := range identifiedInterface.Methods.List {
		if len(field.Names) == 0 {
			continue
		}

		for _, methodName := range field.Names {
			ftype, ok := field.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			methodCallName := impName + methodName.Name + "Call"

			// Build parameter list with names for function signature
			var paramNames []string

			if ftype.Params != nil && len(ftype.Params.List) > 0 {
				// Count total parameters
				totalParams := 0

				for _, param := range ftype.Params.List {
					if len(param.Names) > 0 {
						totalParams += len(param.Names)
					} else {
						totalParams++
					}
				}

				paramIndex := 0

				for _, param := range ftype.Params.List {
					if len(param.Names) > 0 {
						for _, name := range param.Names {
							paramNames = append(paramNames, name.Name)
						}
					} else {
						// Generate a parameter name
						paramName := fmt.Sprintf("param%d", paramIndex)
						paramNames = append(paramNames, paramName)
						paramIndex++
					}
				}
			}

			// Generate the Mock method signature with parameter names
			buf.WriteString(fmt.Sprintf("func (m *%s) ", mockName))
			buf.WriteString(methodName.Name)
			buf.WriteString("(")

			if ftype.Params != nil && len(ftype.Params.List) > 0 {
				paramNameIndex := 0

				for i, param := range ftype.Params.List {
					if i > 0 {
						buf.WriteString(", ")
					}

					paramType := exprToString(fset, param.Type)
					if len(param.Names) > 0 {
						for j, name := range param.Names {
							if j > 0 {
								buf.WriteString(", ")
							}

							buf.WriteString(fmt.Sprintf("%s %s", name.Name, paramType))

							paramNameIndex++
						}
					} else {
						buf.WriteString(fmt.Sprintf("%s %s", paramNames[paramNameIndex], paramType))
						paramNameIndex++
					}
				}
			}

			buf.WriteString(")")
			buf.WriteString(renderFieldList(fset, ftype.Results, false))
			buf.WriteString(" {\n")

			// Create response channel
			buf.WriteString(fmt.Sprintf("\tresponseChan := make(chan %sResponse, 1)\n", methodCallName))
			buf.WriteString("\n")

			// Create the method-specific call struct with parameters
			buf.WriteString(fmt.Sprintf("\tcall := &%s{\n", methodCallName))
			buf.WriteString("\t\tresponseChan: responseChan,\n")

			// Populate call struct fields with parameters
			if ftype.Params != nil && len(ftype.Params.List) > 0 {
				paramNameIndex := 0

				for _, param := range ftype.Params.List {
					paramType := exprToString(fset, param.Type)
					if len(param.Names) > 0 {
						for _, name := range param.Names {
							buf.WriteString(fmt.Sprintf("\t\t%s: %s,\n", name.Name, name.Name))

							paramNameIndex++
						}
					} else {
						// Unnamed parameters - use generated field name and parameter name
						totalParams := 0

						for _, p := range ftype.Params.List {
							if len(p.Names) > 0 {
								totalParams += len(p.Names)
							} else {
								totalParams++
							}
						}
						// Calculate which unnamed parameter index this is
						unnamedIndex := 0

						for _, p := range ftype.Params.List {
							if len(p.Names) == 0 {
								if p == param {
									break
								}

								unnamedIndex++
							}
						}

						fieldName := generateParamName(unnamedIndex, paramType, totalParams)
						buf.WriteString(fmt.Sprintf("\t\t%s: %s,\n", fieldName, paramNames[paramNameIndex]))
						paramNameIndex++
					}
				}
			}

			buf.WriteString("\t}\n\n")

			// Create the Call struct and set the appropriate field
			buf.WriteString(fmt.Sprintf("\tcallEvent := &%s{\n", callName))
			buf.WriteString(fmt.Sprintf("\t\t%s: call,\n", methodName.Name))
			buf.WriteString("\t}\n\n")

			// Send on channel
			buf.WriteString("\tm.imp.callChan <- callEvent\n\n")

			// Wait for response
			buf.WriteString("\tresp := <-responseChan\n\n")

			// Handle response - panic if panic, otherwise always return
			buf.WriteString("\tif resp.Type == \"panic\" {\n")
			buf.WriteString("\t\tpanic(resp.PanicValue)\n")
			buf.WriteString("\t}\n\n")

			// Always return (with values for methods with return values, without for methods without)
			if ftype.Results != nil && len(ftype.Results.List) > 0 {
				// Methods WITH return values - return the values from the response
				buf.WriteString("\treturn")

				returnIndex := 0

				for _, result := range ftype.Results.List {
					if len(result.Names) > 0 {
						for _, name := range result.Names {
							if returnIndex > 0 {
								buf.WriteString(", ")
							}

							buf.WriteString(" resp." + name.Name)

							returnIndex++
						}
					} else {
						if returnIndex > 0 {
							buf.WriteString(", ")
						}

						buf.WriteString(fmt.Sprintf(" resp.Result%d", returnIndex))
						returnIndex++
					}
				}

				buf.WriteString("\n")
			} else {
				// Methods WITHOUT return values - just return
				buf.WriteString("\treturn\n")
			}

			buf.WriteString("}\n\n")
		}
	}

	// Generate the Call struct with fields for each method-specific call struct
	buf.WriteString(fmt.Sprintf("type %s struct {\n", callName))

	for _, methodName := range methodNames {
		methodCallName := impName + methodName + "Call"
		buf.WriteString(fmt.Sprintf("\t%s *%s\n", methodName, methodCallName))
	}

	buf.WriteString("}\n\n")
	// Generate Name() method that returns the method name based on which field is non-nil
	buf.WriteString(fmt.Sprintf("func (c *%s) Name() string {\n", callName))

	for _, methodName := range methodNames {
		buf.WriteString(fmt.Sprintf("\tif c.%s != nil {\n", methodName))
		buf.WriteString(fmt.Sprintf("\t\treturn %q\n", methodName))
		buf.WriteString("\t}\n")
	}

	buf.WriteString("\treturn \"\"\n")
	buf.WriteString("}\n\n")

	// Generate Done() method
	buf.WriteString(fmt.Sprintf("func (c *%s) Done() bool {\n", callName))

	for _, methodName := range methodNames {
		buf.WriteString(fmt.Sprintf("\tif c.%s != nil {\n", methodName))
		buf.WriteString(fmt.Sprintf("\t\treturn c.%s.done\n", methodName))
		buf.WriteString("\t}\n")
	}

	buf.WriteString("\treturn false\n")
	buf.WriteString("}\n\n")

	// Generate As[MethodName] methods on the Call struct
	for _, methodName := range methodNames {
		methodCallName := impName + methodName + "Call"
		buf.WriteString(fmt.Sprintf("func (c *%s) As%s() *%s { return c.%s }\n\n", callName, methodName, methodCallName, methodName))
	}

	// Generate ExpectCallTo struct and methods
	buf.WriteString(fmt.Sprintf("type %s struct {\n", expectCallToName))
	buf.WriteString(fmt.Sprintf("\timp *%s\n", impName))
	buf.WriteString("\ttimeout time.Duration\n")
	buf.WriteString("}\n\n")

	// Generate ExpectCallTo methods for each interface method
	for _, field := range identifiedInterface.Methods.List {
		if len(field.Names) == 0 {
			continue
		}

		for _, methodName := range field.Names {
			ftype, ok := field.Type.(*ast.FuncType)
			if !ok {
				continue
			}

			methodCallName := impName + methodName.Name + "Call"

			// Build parameter list with names for function signature
			var paramNames []string

			if ftype.Params != nil && len(ftype.Params.List) > 0 {
				// Count total parameters
				totalParams := 0

				for _, param := range ftype.Params.List {
					if len(param.Names) > 0 {
						totalParams += len(param.Names)
					} else {
						totalParams++
					}
				}

				paramIndex := 0

				for _, param := range ftype.Params.List {
					if len(param.Names) > 0 {
						for _, name := range param.Names {
							paramNames = append(paramNames, name.Name)
						}
					} else {
						// Generate a parameter name
						paramName := fmt.Sprintf("param%d", paramIndex)
						paramNames = append(paramNames, paramName)
						paramIndex++
					}
				}
			}

			// Generate method signature with parameter names
			buf.WriteString(fmt.Sprintf("func (e *%s) ", expectCallToName))
			buf.WriteString(methodName.Name)
			buf.WriteString("(")

			if ftype.Params != nil && len(ftype.Params.List) > 0 {
				paramNameIndex := 0

				for i, param := range ftype.Params.List {
					if i > 0 {
						buf.WriteString(", ")
					}

					paramType := exprToString(fset, param.Type)
					if len(param.Names) > 0 {
						for j, name := range param.Names {
							if j > 0 {
								buf.WriteString(", ")
							}

							buf.WriteString(fmt.Sprintf("%s %s", name.Name, paramType))

							paramNameIndex++
						}
					} else {
						buf.WriteString(fmt.Sprintf("%s %s", paramNames[paramNameIndex], paramType))
						paramNameIndex++
					}
				}
			}

			buf.WriteString(")")
			buf.WriteString(fmt.Sprintf(" *%s {\n", methodCallName))

			// Create validator function
			buf.WriteString(fmt.Sprintf("\tvalidator := func(c *%s) bool {\n", callName))
			buf.WriteString(fmt.Sprintf("\t\tif c.Name() != %q {\n", methodName.Name))
			buf.WriteString("\t\t\treturn false\n")
			buf.WriteString("\t\t}\n")

			// Validate the args match
			if ftype.Params != nil && len(ftype.Params.List) > 0 {
				buf.WriteString(fmt.Sprintf("\t\tmethodCall := c.As%s()\n", methodName.Name))

				paramNameIndex := 0

				for _, param := range ftype.Params.List {
					paramType := exprToString(fset, param.Type)
					if len(param.Names) > 0 {
						for _, name := range param.Names {
							buf.WriteString(fmt.Sprintf("\t\tif methodCall.%s != %s {\n", name.Name, name.Name))
							buf.WriteString("\t\t\treturn false\n")
							buf.WriteString("\t\t}\n")

							paramNameIndex++
						}
					} else {
						// Unnamed parameters - need to get the field name
						totalParams := 0

						for _, p := range ftype.Params.List {
							if len(p.Names) > 0 {
								totalParams += len(p.Names)
							} else {
								totalParams++
							}
						}
						// Calculate which unnamed parameter index this is
						unnamedIndex := 0

						for _, p := range ftype.Params.List {
							if len(p.Names) == 0 {
								if p == param {
									break
								}

								unnamedIndex++
							}
						}

						fieldName := generateParamName(unnamedIndex, paramType, totalParams)
						buf.WriteString(fmt.Sprintf("\t\tif methodCall.%s != %s {\n", fieldName, paramNames[paramNameIndex]))
						buf.WriteString("\t\t\treturn false\n")
						buf.WriteString("\t\t}\n")

						paramNameIndex++
					}
				}
			}

			buf.WriteString("\t\treturn true\n")
			buf.WriteString("\t}\n\n")

			// Call GetCall
			buf.WriteString("\tcall := e.imp.GetCall(e.timeout, validator)\n")
			buf.WriteString(fmt.Sprintf("\treturn call.As%s()\n", methodName.Name))
			buf.WriteString("}\n\n")
		}
	}

	// Generate Timed struct
	buf.WriteString(fmt.Sprintf("type %s struct {\n", timedName))
	buf.WriteString(fmt.Sprintf("\tExpectCallTo *%s\n", expectCallToName))
	buf.WriteString("}\n\n")

	// Generate Within method
	buf.WriteString(fmt.Sprintf("func (i *%s) Within(d time.Duration) *%s {\n", impName, timedName))
	buf.WriteString(fmt.Sprintf("\treturn &%s{\n", timedName))
	buf.WriteString(fmt.Sprintf("\t\tExpectCallTo: &%s{imp: i, timeout: d},\n", expectCallToName))
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	// Generate GetCall method
	buf.WriteString(fmt.Sprintf("func (i *%s) GetCall(d time.Duration, validator func(*%s) bool) *%s {\n", impName, callName, callName))
	buf.WriteString("\ti.queueLock.Lock()\n")
	buf.WriteString("\tdefer i.queueLock.Unlock()\n\n")

	// Check queue first
	buf.WriteString("\tfor index, call := range i.callQueue {\n")
	buf.WriteString("\t\tif validator(call) {\n")
	buf.WriteString("\t\t\t// Remove from queue\n")
	buf.WriteString("\t\t\ti.callQueue = append(i.callQueue[:index], i.callQueue[index+1:]...)\n")
	buf.WriteString("\t\t\treturn call\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n\n")

	// Setup timeout
	buf.WriteString("\tvar timeoutChan <-chan time.Time\n")
	buf.WriteString("\tif d > 0 {\n")
	buf.WriteString("\t\ttimeoutChan = time.After(d)\n")
	buf.WriteString("\t}\n\n")

	buf.WriteString("\tfor {\n")
	buf.WriteString("\t\tselect {\n")
	buf.WriteString("\t\tcase call := <-i.callChan:\n")
	buf.WriteString("\t\t\tif validator(call) {\n")
	buf.WriteString("\t\t\t\treturn call\n")
	buf.WriteString("\t\t\t}\n")
	buf.WriteString("\t\t\t// Queue it\n")
	buf.WriteString("\t\t\ti.callQueue = append(i.callQueue, call)\n")
	buf.WriteString("\t\tcase <-timeoutChan:\n")
	buf.WriteString("\t\t\ti.t.Fatalf(\"timeout waiting for call matching validator\")\n")
	buf.WriteString("\t\t\treturn nil\n")
	buf.WriteString("\t\t}\n")
	buf.WriteString("\t}\n")
	buf.WriteString("}\n\n")

	// Generate GetCurrentCall method - now uses GetCall with no timeout and always-true validator
	buf.WriteString(fmt.Sprintf("func (i *%s) GetCurrentCall() *%s {\n", impName, callName))
	buf.WriteString("\tif i.currentCall != nil && !i.currentCall.Done() {\n")
	buf.WriteString("\t\treturn i.currentCall\n")
	buf.WriteString("\t}\n")
	buf.WriteString(fmt.Sprintf("\ti.currentCall = i.GetCall(0, func(c *%s) bool { return true })\n", callName))
	buf.WriteString("\treturn i.currentCall\n")
	buf.WriteString("}\n\n")

	// New[impName] constructor with *testing.T arg
	buf.WriteString(fmt.Sprintf("func New%s(t *testing.T) *%s {\n", impName, impName))
	buf.WriteString(fmt.Sprintf("\timp := &%s{\n", impName))
	buf.WriteString("\t\tt: t,\n")
	buf.WriteString(fmt.Sprintf("\t\tcallChan: make(chan *%s, 1),\n", callName))
	buf.WriteString("\t}\n")
	buf.WriteString(fmt.Sprintf("\timp.Mock = &%s{imp: imp}\n", mockName))
	buf.WriteString(fmt.Sprintf("\timp.ExpectCallTo = &%s{imp: imp}\n", expectCallToName))
	buf.WriteString("\treturn imp\n")
	buf.WriteString("}\n\n")

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		fmt.Printf("error formatting generated code: %v\n", err)
		return buf.String()
	}

	return string(formatted)
}

// writeGeneratedCodeToFile writes the generated code to <impName>.go.
func writeGeneratedCodeToFile(code string, impName string) {
	filename := "generated.go"
	if impName != "" {
		filename = impName + ".go"
	}

	err := os.WriteFile(filename, []byte(code), 0o644)
	if err != nil {
		fmt.Printf("error writing %s: %v\n", filename, err)
		return
	}

	fmt.Printf("%s written successfully.\n", filename)
}

// printAstTree recursively prints the AST node tree with indentation.
func printAstTree(node any, indent string) {
	switch typedNode := node.(type) {
	case nil:
		return
	case *ast.Ident:
		typeName := fmt.Sprintf("%T", typedNode)
		fmt.Printf("%s%s (Name: %q)\n", indent, typeName, typedNode.Name)

		return
	case ast.Node:
		typeName := fmt.Sprintf("%T", typedNode)
		fmt.Printf("%s%s\n", indent, typeName)
		indent2 := indent + "  "

		ast.Inspect(typedNode, func(child ast.Node) bool {
			if child != typedNode && child != nil {
				printAstTree(child, indent2)
				return false
			}

			return true
		})
	}
}

// renderFieldList renders a *ast.FieldList as Go code (params/results).
func renderFieldList(fset *token.FileSet, fieldList *ast.FieldList, isParams bool) string {
	if fieldList == nil || len(fieldList.List) == 0 {
		if isParams {
			return "()"
		}

		return ""
	}

	var buf bytes.Buffer
	buf.WriteString("(")

	for i, field := range fieldList.List {
		if i > 0 {
			buf.WriteString(", ")
		}
		// Names
		for j, name := range field.Names {
			if j > 0 {
				buf.WriteString(", ")
			}

			buf.WriteString(name.Name)
		}
		// Type
		if len(field.Names) > 0 {
			buf.WriteString(" ")
		}

		buf.WriteString(exprToString(fset, field.Type))
	}

	buf.WriteString(")")

	return buf.String()
}

// exprToString renders an ast.Expr to Go code.
func exprToString(fset *token.FileSet, expr ast.Expr) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, expr)

	return buf.String()
}

// generateParamName generates a field name for an unnamed parameter
// Uses common conventions: single string -> "S", single int -> "Input", multiple -> "A", "B", "C", etc.
func generateParamName(index int, paramType string, totalParams int) string {
	// Remove common prefixes/suffixes for comparison
	normalized := strings.TrimSpace(paramType)

	// Single parameter cases
	if totalParams == 1 {
		if normalized == "string" {
			return "S"
		}

		if normalized == "int" {
			return "Input"
		}
	}

	// Multiple parameters - use A, B, C, etc.
	names := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	if index < len(names) {
		return names[index]
	}

	// Fallback
	return fmt.Sprintf("Arg%d", index)
}
