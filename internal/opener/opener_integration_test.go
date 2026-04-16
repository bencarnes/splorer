//go:build integration

package opener

import "testing"

// TestOpenFile_Integration verifies that xdg-open starts without error.
// Run with: go test -tags integration ./internal/opener/
func TestOpenFile_Integration(t *testing.T) {
	if err := OpenFile("/tmp"); err != nil {
		t.Fatalf("OpenFile(/tmp) returned error: %v", err)
	}
}
