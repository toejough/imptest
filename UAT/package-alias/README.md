# Package Alias Tests

This directory verifies that **code generation correctly handles package imports** in various scenarios.

## Test Coverage

### Package Alias Scenarios

| Scenario | Example | Resolution |
|----------|---------|------------|
| **Single-word** | `import "time"` | Package alias is `time` |
| **Final-segment** | `import "github.com/foo/bar"` | Package alias is `bar` |
| **Obscured** | Package at path has different name | Use actual package name |
| **Aliased** | `import nick "github.com/foo/bar"` | Use alias `nick` |

### Framework Import Rules

Generated code must:
1. Use package aliases as they appear in the source file
2. Generate code into the same package as the source
3. Use the same type names and package aliases as source
4. Prefix framework imports with `_` to avoid conflicts

## Key Concepts

### Why Package Aliases Matter
When generating wrappers for types from external packages, the generator must correctly:
- Detect the package alias used in the source code
- Use that same alias in the generated code
- Avoid conflicts with the source code's imports

### Example: Single-Word Import
```go
// Source code
import "time"

type TimeService interface {
    Now() time.Time
}

// Generated code must use "time" not "_time" or something else
// because that's what the source code uses
```

### Example: Framework Imports
```go
// Generated code
import (
    _testing "testing"           // _ prefix avoids conflict
    _imptest "github.com/toejough/imptest/imptest/v2"
)
```
