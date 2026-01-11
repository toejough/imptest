// Package samepackage demonstrates interfaces that reference other interfaces
// from the same package in their method signatures.
package samepackage

// DataProcessor processes data from sources and sends to sinks.
// This interface uses other interfaces from the same package.
type DataProcessor interface {
	// Process reads from a source and writes to a sink
	Process(source DataSource, sink DataSink) error

	// Transform reads from a source, transforms it, and returns a new source
	Transform(input DataSource) (DataSource, error)

	// Validate checks if a sink is valid
	Validate(sink DataSink) bool
}

// DataSink represents a destination that can receive data.
type DataSink interface {
	PutData(data []byte) error
}

// DataSource represents a source that can provide data.
type DataSource interface {
	GetData() ([]byte, error)
}
