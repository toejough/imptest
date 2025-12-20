package noncomparable

// DataProcessor handles types that don't support the '==' operator (slices and maps).
type DataProcessor interface {
	ProcessSlice(data []string) int
	ProcessMap(config map[string]int) bool
}

// RunProcessor uses the DataProcessor interface with non-comparable types.
func RunProcessor(p DataProcessor) {
	_ = p.ProcessSlice([]string{"a", "b", "c"})
	_ = p.ProcessMap(map[string]int{"threshold": 10})
}
