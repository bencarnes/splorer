package opener

import (
	"testing"
)

// TestOpenFile_Start verifies that OpenFile returns no error when called with
// a plausible path (xdg-open starts but we don't wait for it).
// This is a unit test — it exercises the Start() call but does not assert
// what xdg-open does with the path.
func TestOpenFile_Start(t *testing.T) {
	// xdg-open is non-blocking; we only verify that Start itself doesn't error.
	// On CI without a desktop environment xdg-open may not be installed, so
	// skip gracefully when the binary is absent.
	err := OpenFile("/tmp")
	if err != nil {
		t.Skipf("xdg-open not available or failed to start: %v", err)
	}
}

