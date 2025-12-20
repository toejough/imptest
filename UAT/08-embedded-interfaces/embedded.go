package embedded

import "io"

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
func ProcessStream(rc ReadCloser) (int, error) {
	buf := make([]byte, 10)

	n, err := rc.Read(buf)
	if err != nil {
		_ = rc.Close()
		return 0, err
	}

	err = rc.Close()

	return n, err
}
