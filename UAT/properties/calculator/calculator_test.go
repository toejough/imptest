// Package calculator_test demonstrates property-based testing with imptest and rapid.
package calculator_test

import (
	callable "github.com/toejough/imptest/UAT/core/wrapper-function"
)

// Ensure import is used.
var _ = callable.NewCalculator

// Generate wrappers for Calculator methods.
//go:generate impgen callable.Calculator.Add --target
//go:generate impgen callable.Calculator.Multiply --target
//go:generate impgen callable.Calculator.Divide --target
//go:generate impgen callable.Calculator.ProcessValue --target
