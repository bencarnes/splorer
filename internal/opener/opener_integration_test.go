//go:build integration

package opener

import (
	"os"
	"testing"
)

// TestOpenFile_Integration verifies that the platform opener starts without error.
// Run with: go test -tags integration ./internal/opener/
func TestOpenFile_Integration(t *testing.T) {
	if err := OpenFile(os.TempDir()); err != nil {
		t.Fatalf("OpenFile(%q) returned error: %v", os.TempDir(), err)
	}
}
