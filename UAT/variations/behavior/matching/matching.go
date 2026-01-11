// Package matching demonstrates mocking interfaces with complex struct parameters
// and using matchers for partial field validation.
package matching

// ComplexService is an interface taking a complex struct.
type ComplexService interface {
	Process(d Data) bool
}

// Data is a complex struct where we might only care about matching some fields.
type Data struct {
	ID        int
	Payload   string
	Timestamp int64
}

// UseService is a function that uses the ComplexService.
func UseService(svc ComplexService, payload string) {
	const (
		id        = 123
		timestamp = 1600000000
	)

	svc.Process(Data{
		ID:        id,
		Payload:   payload,
		Timestamp: timestamp,
	})
}
