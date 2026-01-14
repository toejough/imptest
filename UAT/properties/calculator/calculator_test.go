// Package calculator_test demonstrates property-based testing with imptest and rapid.
package calculator_test

import (
	callable "github.com/toejough/imptest/UAT/core/wrapper-function"
)

// unexported variables.
var (
	_ = callable.NewCalculator
)
