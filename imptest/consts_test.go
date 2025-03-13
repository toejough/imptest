package imptest //nolint:testpackage
// This package is intentionally a white-box test

import (
	"testing"
	"time"
)

// TestConsts tests consts to be exactly what I set them at. It's a little silly, but I don't see any other way to make
// the mutation tester chill out about these values. They're not important as exact numbers, so without this test, the
// mutation tester rightly calls them out as being flexible. The thing is, I don't care.
func TestConsts(t *testing.T) {
	t.Parallel()

	if constInvalidIndex != -1 {
		t.Fatal("the invalid index needs to be -1, but it's not.")
	}

	if constDefaultActivityBufferSize != 100 {
		t.Fatal("the default activity buffer size needs to be 100, but it's not.")
	}

	if defaultResolutionMaxDuration != 100*time.Millisecond {
		t.Fatal("the default resolution max duration needs to be 100 milliseconds, but it's not.")
	}
}
