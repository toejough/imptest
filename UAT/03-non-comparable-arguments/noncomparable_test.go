package noncomparable_test

import (
	"testing"

	noncomparable "github.com/toejough/imptest/UAT/03-non-comparable-arguments"
)

//go:generate go run ../../impgen/main.go noncomparable.DataProcessor --name DataProcessorImp

func TestNonComparableArguments(t *testing.T) {
	t.Parallel()

	mock := NewDataProcessorImp(t)

	go noncomparable.RunProcessor(mock.Mock)

	// Intercept ProcessSlice with a slice argument.
	// imptest automatically uses reflect.DeepEqual for slices.
	mock.ExpectCallIs.ProcessSlice().ExpectArgsAre([]string{"a", "b", "c"}).InjectResult(3)

	// Intercept ProcessMap with a map argument.
	// imptest automatically uses reflect.DeepEqual for maps.
	mock.ExpectCallIs.ProcessMap().ExpectArgsAre(map[string]int{"threshold": 10}).InjectResult(true)
}
