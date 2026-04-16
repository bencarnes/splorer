package filetree

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// setupTempDir creates a temp directory with the given subdirs and files,
// returning its path and a cleanup function.
func setupTempDir(t *testing.T, dirs []string, files []string) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range dirs {
		if err := os.Mkdir(filepath.Join(root, d), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(root, f), []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}
	return root
}

func TestLoadDir_SortOrder(t *testing.T) {
	root := setupTempDir(t,
		[]string{"Zebra", "alpha", "Beta"},
		[]string{"zoo.txt", "Apple.go", "mango.md"},
	)

	entries, err := loadDir(root)
	if err != nil {
		t.Fatalf("loadDir error: %v", err)
	}

	// Dirs come first, case-insensitively sorted.
	wantDirs := []string{"alpha", "Beta", "Zebra"}
	wantFiles := []string{"Apple.go", "mango.md", "zoo.txt"}

	var gotDirs, gotFiles []string
	for _, e := range entries {
		if e.IsDir {
			gotDirs = append(gotDirs, e.Name)
		} else {
			gotFiles = append(gotFiles, e.Name)
		}
	}

	if len(gotDirs) != len(wantDirs) {
		t.Fatalf("dirs: got %v, want %v", gotDirs, wantDirs)
	}
	for i := range wantDirs {
		if gotDirs[i] != wantDirs[i] {
			t.Errorf("dirs[%d]: got %q, want %q", i, gotDirs[i], wantDirs[i])
		}
	}
	if len(gotFiles) != len(wantFiles) {
		t.Fatalf("files: got %v, want %v", gotFiles, wantFiles)
	}
	for i := range wantFiles {
		if gotFiles[i] != wantFiles[i] {
			t.Errorf("files[%d]: got %q, want %q", i, gotFiles[i], wantFiles[i])
		}
	}
}

func TestLoadDir_Empty(t *testing.T) {
	root := setupTempDir(t, nil, nil)
	entries, err := loadDir(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestLoadDir_PermissionDenied(t *testing.T) {
	root := setupTempDir(t, nil, nil)
	restricted := filepath.Join(root, "noaccess")
	if err := os.Mkdir(restricted, 0000); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Cleanup(func() { os.Chmod(restricted, 0755) }) //nolint:errcheck

	_, err := loadDir(restricted)
	if err == nil {
		t.Error("expected error for unreadable directory, got nil")
	}
}

func TestCursorBounds_Up(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt"})
	m := New(root)
	// cursor starts at 0; moving up should stay at 0
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m2.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m2.cursor)
	}
}

func TestCursorBounds_Down(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt"})
	m := New(root)
	m.cursor = len(m.entries) - 1
	// moving down at the last entry should stay
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m2.cursor != len(m2.entries)-1 {
		t.Errorf("cursor = %d, want %d", m2.cursor, len(m2.entries)-1)
	}
}

func TestNavigateInto(t *testing.T) {
	root := setupTempDir(t, []string{"subdir"}, []string{"file.txt"})
	m := New(root)
	// Find the subdir entry.
	for i, e := range m.entries {
		if e.Name == "subdir" {
			m.cursor = i
			break
		}
	}

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m2.cwd != filepath.Join(root, "subdir") {
		t.Errorf("cwd = %q, want %q", m2.cwd, filepath.Join(root, "subdir"))
	}
	if m2.cursor != 0 {
		t.Errorf("cursor after navigate = %d, want 0", m2.cursor)
	}
}

func TestNavigateUp(t *testing.T) {
	root := setupTempDir(t, []string{"child"}, nil)
	m := New(filepath.Join(root, "child"))

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m2.cwd != root {
		t.Errorf("cwd = %q, want %q", m2.cwd, root)
	}
}

func TestNavigateUp_AtRoot(t *testing.T) {
	m := New("/")
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m2.cwd != "/" {
		t.Errorf("navigating up from / should stay at /, got %q", m2.cwd)
	}
}

func TestDoubleClickDetection_Opens(t *testing.T) {
	root := setupTempDir(t, []string{"child"}, nil)
	m := New(root)
	m.height = 24
	m.width = 80

	// Find child entry index.
	childIdx := -1
	for i, e := range m.entries {
		if e.Name == "child" {
			childIdx = i
			break
		}
	}
	if childIdx == -1 {
		t.Fatal("child entry not found")
	}

	// First click: just selects.
	m.lastClick = time.Time{} // zero = very old
	clickY := headerHeight + childIdx
	m2, _ := m.Update(tea.MouseClickMsg{
		X:      0,
		Y:      clickY,
		Button: tea.MouseLeft,
	})
	if m2.cursor != childIdx {
		t.Errorf("after first click cursor = %d, want %d", m2.cursor, childIdx)
	}
	if m2.cwd != root {
		t.Errorf("single click should not navigate, cwd = %q", m2.cwd)
	}

	// Second click within 500 ms: should navigate into child.
	m2.lastClick = time.Now().Add(-100 * time.Millisecond)
	m3, _ := m2.Update(tea.MouseClickMsg{
		X:      0,
		Y:      clickY,
		Button: tea.MouseLeft,
	})
	if m3.cwd != filepath.Join(root, "child") {
		t.Errorf("double click should navigate into child, cwd = %q", m3.cwd)
	}
}

func TestDoubleClickDetection_TooSlow(t *testing.T) {
	root := setupTempDir(t, []string{"child"}, nil)
	m := New(root)
	m.height = 24
	m.width = 80

	childIdx := 0
	clickY := headerHeight + childIdx

	// First click.
	m2, _ := m.Update(tea.MouseClickMsg{
		X:      0,
		Y:      clickY,
		Button: tea.MouseLeft,
	})

	// Simulate a click more than 500 ms later.
	m2.lastClick = time.Now().Add(-600 * time.Millisecond)
	m3, _ := m2.Update(tea.MouseClickMsg{
		X:      0,
		Y:      clickY,
		Button: tea.MouseLeft,
	})
	// Should still be in root, not navigated.
	if m3.cwd != root {
		t.Errorf("slow double click should not navigate, cwd = %q", m3.cwd)
	}
}
