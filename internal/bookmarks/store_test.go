package bookmarks

import (
	"os"
	"path/filepath"
	"testing"
)

// withConfigDir redirects XDG_CONFIG_HOME to a temp directory for the test.
func withConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func TestLoad_MissingFile(t *testing.T) {
	withConfigDir(t)
	bmarks, err := Load()
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(bmarks) != 0 {
		t.Errorf("expected empty slice, got %v", bmarks)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	withConfigDir(t)

	input := []Bookmark{
		{Name: "Home", Path: "/home/user"},
		{Name: "Projects", Path: "/home/user/projects"},
	}

	if err := Save(input); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(got) != len(input) {
		t.Fatalf("len = %d, want %d", len(got), len(input))
	}
	for i, want := range input {
		if got[i].Name != want.Name || got[i].Path != want.Path {
			t.Errorf("bookmark[%d] = {%q, %q}, want {%q, %q}",
				i, got[i].Name, got[i].Path, want.Name, want.Path)
		}
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	dir := withConfigDir(t)
	splorer := filepath.Join(dir, "splorer")

	if _, err := os.Stat(splorer); !os.IsNotExist(err) {
		t.Skip("directory already exists, skipping creation check")
	}

	if err := Save([]Bookmark{{Name: "test", Path: "/tmp"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(splorer); err != nil {
		t.Errorf("config directory not created: %v", err)
	}
}

func TestLoad_CorruptJSON(t *testing.T) {
	dir := withConfigDir(t)
	path := filepath.Join(dir, "splorer", "bookmarks.json")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{corrupt"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	bmarks, err := Load()
	if err == nil {
		t.Error("expected error for corrupt JSON, got nil")
	}
	if len(bmarks) != 0 {
		t.Errorf("expected nil/empty on error, got %v", bmarks)
	}
}

func TestSave_EmptySlice(t *testing.T) {
	withConfigDir(t)

	if err := Save([]Bookmark{}); err != nil {
		t.Fatalf("Save empty: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected 0 bookmarks, got %d", len(got))
	}
}
