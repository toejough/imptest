package orderedvsmode

// Service demonstrates ordered vs eventually call matching modes.
// Used to test that ordered mode fails fast on mismatches while
// eventually mode queues mismatches and waits for matches.
type Service interface {
	// OperationA represents the first operation in a sequence.
	OperationA(id int) error

	// OperationB represents the second operation in a sequence.
	OperationB(id int) error

	// OperationC represents the third operation in a sequence.
	OperationC(id int) error
}
