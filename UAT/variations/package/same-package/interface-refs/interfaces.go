// Package samepackage demonstrates interfaces that reference other interfaces
// from the same package in their method signatures.
package samepackage

type DataProcessor interface {
	// Process reads from a source and writes to a sink
	Process(source DataSource, sink DataSink) error

	// Transform reads from a source, transforms it, and returns a new source
	Transform(input DataSource) (DataSource, error)

	// Validate checks if a sink is valid
	Validate(sink DataSink) bool
}

type DataSink interface {
	PutData(data []byte) error
}

type DataSource interface {
	GetData() ([]byte, error)
}
