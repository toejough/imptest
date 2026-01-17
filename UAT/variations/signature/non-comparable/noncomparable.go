// Package noncomparable demonstrates mocking interfaces with non-comparable
// parameter types like slices and maps.
package noncomparable

type DataProcessor interface {
	ProcessSlice(data []string) int
	ProcessMap(config map[string]int) bool
}

// RunProcessor uses the DataProcessor interface with non-comparable types.
func RunProcessor(p DataProcessor) {
	const threshold = 10

	_ = p.ProcessSlice([]string{"a", "b", "c"})
	_ = p.ProcessMap(map[string]int{"threshold": threshold})
}
