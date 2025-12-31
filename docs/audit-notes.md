# UAT Audit Notes for TAXONOMY.md

## UAT-01: basic-interface-mocking
- **Primary Feature**: Core interface mocking functionality
- **Types**: Interface (`Ops`, `CustomOps`)
- **Package**: Local package (basic)
- **Signature**: Variadic params (`Notify(msg string, ids ...int)`), zero params (`Finish()`)
- **Generate Directives**: `--dependency` on interfaces
- **Notes**: Tests basic mocking with custom naming

## UAT-02: callable-wrappers
- **Primary Feature**: Function and method wrapping
- **Types**: Interface (`ExternalService`), Function (`BusinessLogic`), Methods (`Calculator.Add`, `.Multiply`, `.Divide`, `.ProcessValue`)
- **Package**: Local package (callable)
- **Signature**: Multiple params, multiple returns, single return
- **Generate Directives**: `--dependency` for interface, `--target` for function and methods
- **Notes**: Demonstrates both targets and dependencies together

## UAT-03: non-comparable-arguments
- **Primary Feature**: Handling non-comparable types (slices, maps)
- **Types**: Interface (`DataProcessor`)
- **Package**: Local package (noncomparable)
- **Signature**: Slice args, map args (uses DeepEqual)
- **Generate Directives**: `--dependency`
- **Notes**: Auto-detection of non-comparable types

## UAT-04: error-and-panic-handling
- **Primary Feature**: Panic injection and verification
- **Types**: Interface (`CriticalDependency`), Functions (`SafeRunner`, `UnsafeRunner`)
- **Package**: Local package (safety)
- **Signature**: Error returns, panic behavior
- **Generate Directives**: `--dependency` for interface, `--target` for functions
- **Notes**: Tests `InjectPanic()` and `ExpectPanic()`

## UAT-05: advanced-matching
- **Primary Feature**: Gomega matchers integration
- **Types**: Interface (`ComplexService`)
- **Package**: Local package (matching)
- **Signature**: Various return types
- **Generate Directives**: `--dependency`
- **Notes**: Uses `ExpectReturnsMatch()` with Gomega matchers

## UAT-06: concurrency
- **Primary Feature**: Eventually() for async testing
- **Types**: Interface (`SlowService`)
- **Package**: Local package (concurrency)
- **Signature**: Standard interface methods
- **Generate Directives**: `--dependency`
- **Notes**: Tests concurrent call ordering with `Eventually()`

## UAT-07: generics
- **Primary Feature**: Generic functions and interfaces
- **Types**: Generic interface (`Repository[T]`), Generic function (`ProcessItem[T]`)
- **Package**: Local package (generics)
- **Signature**: Type parameters, generic methods
- **Generate Directives**: Generated files exist (MockRepository, WrapProcessItem) but no visible directives in source
- **Notes**: Tests type parameter support

## UAT-08: embedded-interfaces
- **Primary Feature**: Embedded interface expansion
- **Types**: Embedded interface (`ReadCloser` embeds `io.Reader` + `io.Closer`)
- **Package**: Local package (embedded), references stdlib (io)
- **Signature**: Standard interface methods
- **Generate Directives**: `--dependency`
- **Notes**: Auto-expands embedded interfaces

## UAT-09: edge-zero-returns
- **Primary Feature**: Functions with no return values
- **Types**: Function (`ProcessData`)
- **Package**: Local package (zero_returns)
- **Signature**: Multiple params, zero return values
- **Generate Directives**: Generated file exists (WrapProcessData) but no visible directive
- **Notes**: Uses `ExpectCompletes()` instead of `ExpectReturnsEqual()`

## UAT-10: edge-many-params
- **Primary Feature**: Functions with many parameters
- **Types**: Interface (`ManyParams`)
- **Package**: Local package (many_params)
- **Signature**: 10 parameters (tests param naming beyond A-H)
- **Generate Directives**: Generated file exists (MockManyParams) but no visible directive
- **Notes**: Tests parameter handling for 10 params (a, b, c, d, e, f, g, h, i, j)

## UAT-11: package-name-conflicts
- **Primary Feature**: Import aliasing for package name conflicts
- **Types**: Interfaces (`Scheduler` in timeconflict package, `Timer` in stdlib time)
- **Package**: Local package with name shadowing stdlib
- **Signature**: Standard interface methods
- **Generate Directives**: `--dependency` on both local and stdlib types
- **Notes**: Local type named `Time` alongside `time.Time` from stdlib

## UAT-12: whitebox-testing
- **Primary Feature**: Same-package testing (whitebox)
- **Types**: Interface (`Ops`)
- **Package**: Same package as test (no `_test` suffix in package name)
- **Signature**: Standard interface methods
- **Generate Directives**: `impgen Ops --dependency` (no package prefix)
- **Notes**: Tests when code and tests in same package

## UAT-13: external-type-imports
- **Primary Feature**: External type parameters in signatures
- **Types**: Interface (`FileHandler`)
- **Package**: Local package (externalimports) with external type refs
- **Signature**: Uses external types (e.g., `fs.DirEntry`, pointers)
- **Generate Directives**: `--dependency`
- **Notes**: Auto-imports external types

## UAT-14: same-package-interfaces
- **Primary Feature**: Multiple interfaces in same package
- **Types**: Interfaces (`DataProcessor`, `DataSource`, `DataSink`)
- **Package**: Same package (samepackage)
- **Signature**: Standard interface methods
- **Generate Directives**: `--dependency` on multiple interfaces
- **Notes**: Tests multiple mocks from one package

## UAT-15: callback-visitor
- **Primary Feature**: Callback extraction and function type wrapping
- **Types**: Interface (`TreeWalker`), Function (`CountFiles`), Function type (`WalkFunc`)
- **Package**: Local package (visitor)
- **Signature**: Function parameters (callbacks), function types
- **Generate Directives**: `--dependency` on interface, `--target` on function and func type
- **Notes**: Tests callback invocation via `GetArgs()` and function type wrapping

## UAT-16: function-type-wrapping
- **Primary Feature**: Wrapping named function types (local)
- **Types**: Named function type (`WalkFunc`)
- **Package**: Local package (functype)
- **Signature**: Function type signature
- **Generate Directives**: `--target functype.WalkFunc`
- **Notes**: Local function type wrapping

## UAT-17: typesafe-getargs
- **Primary Feature**: Type-safe GetArgs() extraction
- **Types**: Interface (`Calculator`)
- **Package**: Local package (typesafeargs)
- **Signature**: Mixed parameter types (int, string, any)
- **Generate Directives**: Manual test (no generated files, manually written wrappers)
- **Notes**: Tests type-safe argument extraction from dependency calls

## UAT-18: external-function-types
- **Primary Feature**: Wrapping named function types from external packages
- **Types**: Named function type (`http.HandlerFunc`)
- **Package**: External stdlib (net/http)
- **Signature**: Function type signature, zero returns
- **Generate Directives**: `impgen http.HandlerFunc --target`
- **Notes**: External stdlib function type wrapping

## UAT-19: interface-external-func-type
- **Primary Feature**: Interface methods with external function type parameters
- **Types**: Interface (`HTTPMiddleware`)
- **Package**: Local package (middleware) with stdlib type refs
- **Signature**: Method with `http.HandlerFunc` parameter
- **Generate Directives**: `--dependency`
- **Notes**: Combines external func types with interface mocking

## UAT-20: channel-types
- **Primary Feature**: Channel type parameters and returns
- **Types**: Interface (`ChannelHandler`)
- **Package**: Local package (channels)
- **Signature**: Channel params/returns (`chan T`, `<-chan T`, `chan<- T`)
- **Generate Directives**: `--dependency`
- **Notes**: Tests all channel direction variants

## UAT-21: parameterized-types
- **Primary Feature**: Constrained generic types
- **Types**: Interface (`DataProcessor`)
- **Package**: Local package (parameterized)
- **Signature**: Generic with constraints (e.g., `[T Numeric]`)
- **Generate Directives**: `--dependency`
- **Notes**: Tests type constraints on generics

## UAT-22: test-package-import
- **Primary Feature**: External module type imports
- **Types**: Interface (`Service`)
- **Package**: External package from different module
- **Signature**: Standard interface methods
- **Generate Directives**: `--dependency Service`
- **Notes**: Tests importing from third-party modules

## Gaps Identified

### Untested Features (? in matrices)
1. **Generic methods** - Methods on generic structs (e.g., `Container[T].Get`)
2. **Named returns** - Functions with named return values
3. **Methods with variadic params** - No specific UAT found
4. **Methods with zero returns** - No specific UAT found
5. **Named func types with variadic** - No specific UAT found

### Missing UAT Details (need verification)
- ~~UAT-09: Need to check actual types and directives~~ ✅ Verified
- ~~UAT-10: Need to check actual types and directives~~ ✅ Verified
- ~~UAT-17: Need to check actual types and directives~~ ✅ Verified

### Features Tested but Not in Original Matrix Design
- Gomega matcher integration (UAT-05)
- Eventually() concurrency support (UAT-06)
- Callback extraction with GetArgs() (UAT-15)

## Matrix Coverage Summary

### Capability Matrix (Types) Coverage
- ✅ Package function (UAT-02, UAT-04, UAT-15)
- ✅ Struct method (UAT-02)
- ✅ Named func type (local) (UAT-15, UAT-16)
- ✅ Named func type (external) (UAT-18)
- ✅ Generic function (UAT-07)
- ❓ Generic method (untested)
- ✅ Interface (UAT-01, UAT-02, multiple others)
- ✅ Embedded interface (UAT-08)
- ✅ Generic interface (UAT-07, UAT-21)
- ✅ External interface (implied by UAT-08 using io.Reader/Closer)
- ✅ Interface with func params (UAT-19)

### Package Variations Coverage
- ✅ Same package (UAT-12, UAT-14)
- ✅ Local package (most UATs)
- ✅ External package (UAT-22)
- ✅ Stdlib package (UAT-08, UAT-18, UAT-19)
- ✅ Package name conflicts (UAT-11)
- ✅ Aliased imports (tested via conflict resolution in UAT-11)

### Signature Variations Coverage

**Basic:**
- ✅ Multiple parameters (UAT-10 if confirmed)
- ✅ Zero parameters (UAT-01 `Finish()`)
- ✅ Zero returns (UAT-09 if confirmed, UAT-18)
- ✅ Multiple returns (UAT-02 `Calculator.Divide`)
- ❓ Named returns (need to verify)
- ✅ Variadic parameters (UAT-01 `Notify`)

**Nillable:**
- ✅ Pointers (UAT-13)
- ✅ Slices (UAT-03)
- ✅ Maps (UAT-03)
- ✅ Interfaces (UAT-02 error returns)
- ✅ Channels (UAT-20)
- ✅ Functions (UAT-15 callback params, UAT-19)

**Advanced:**
- ✅ Generic type params (UAT-07, UAT-21)
- ✅ Non-comparable types (UAT-03)
- ✅ External types (UAT-13, UAT-18, UAT-19)
- ✅ Named func types (UAT-15, UAT-16, UAT-18, UAT-19)
