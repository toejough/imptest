package matching

// Data is a complex struct where we might only care about matching some fields.
type Data struct {
	ID        int
	Payload   string
	Timestamp int64
}

// ComplexService is an interface taking a complex struct.
type ComplexService interface {
	Process(d Data) bool
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
