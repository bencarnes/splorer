package associations

import (
	"os"
	"path/filepath"
	"testing"
)

// withConfigDir redirects the os.UserConfigDir resolution by temporarily
// setting XDG_CONFIG_HOME to a temp path, then restores it on cleanup.
func withConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func TestLoad_MissingFile(t *testing.T) {
	withConfigDir(t)
	assocs, err := Load()
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(assocs) != 0 {
		t.Errorf("expected empty map, got %v", assocs)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	withConfigDir(t)

	input := map[string]string{
		".pdf": "evince",
		".mp3": "vlc",
		".png": "eog",
	}

	if err := Save(input); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for k, want := range input {
		if got[k] != want {
			t.Errorf("assocs[%q] = %q, want %q", k, got[k], want)
		}
	}
	if len(got) != len(input) {
		t.Errorf("len = %d, want %d", len(got), len(input))
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := withConfigDir(t)
	splorer := filepath.Join(dir, "splorer")

	if _, err := os.Stat(splorer); !os.IsNotExist(err) {
		t.Skip("directory already exists, skipping creation check")
	}

	if err := Save(map[string]string{".txt": "nano"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(splorer); err != nil {
		t.Errorf("config directory not created: %v", err)
	}
}

func TestLoad_CorruptJSON(t *testing.T) {
	dir := withConfigDir(t)
	path := filepath.Join(dir, "splorer", "openers.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{corrupt"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	assocs, err := Load()
	if err == nil {
		t.Error("expected error for corrupt JSON, got nil")
	}
	if len(assocs) != 0 {
		t.Errorf("expected empty map on error, got %v", assocs)
	}
}
