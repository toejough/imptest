// Package matching demonstrates mocking interfaces with complex struct parameters
// and using matchers for partial field validation.
package matching

type ComplexService interface {
	Process(d Data) bool
}

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
