package embedded

import (
	"fmt"
	"io"
)

// Closer is a local interface.
type Closer interface {
	Close() error
}

// ReadCloser embeds an external (io.Reader) and a local (Closer) interface.
type ReadCloser interface {
	io.Reader
	Closer
}

// ProcessStream reads from a ReadCloser and then closes it.

func ProcessStream(readCloser ReadCloser) (int, error) {
	const bufSize = 10

	buf := make([]byte, bufSize)

	bytesRead, err := readCloser.Read(buf)
	if err != nil {
		_ = readCloser.Close()

		return 0, fmt.Errorf("read failed: %w", err)
	}

	err = readCloser.Close()
	if err != nil {
		return bytesRead, fmt.Errorf("close failed: %w", err)
	}

	return bytesRead, nil
}
