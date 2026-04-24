package opener

import (
	"os"
	"testing"
)

// TestOpenFile_Start verifies that OpenFile returns no error when called with
// a plausible path. The underlying opener (xdg-open on Unix, cmd /c start on
// Windows) is non-blocking; we only verify that Start itself doesn't error.
// On CI without a desktop environment the opener may not be available, so
// skip gracefully when it fails.
func TestOpenFile_Start(t *testing.T) {
	err := OpenFile(os.TempDir())
	if err != nil {
		t.Skipf("OpenFile not available or failed to start: %v", err)
	}
}
