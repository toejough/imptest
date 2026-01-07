package functype

// WalkFunc is a function type for walking directory entries.
// This demonstrates wrapping a named function type.
type WalkFunc func(path string, info string) error
