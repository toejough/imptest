package imptest

import "errors"

// unexported variables.
var (
	errTypeMismatch = errors.New("type mismatch")
)
