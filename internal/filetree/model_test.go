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

	entries, err := loadDir(root, SortByName)
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
	entries, err := loadDir(root, SortByName)
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

	_, err := loadDir(restricted, SortByName)
	if err == nil {
		t.Error("expected error for unreadable directory, got nil")
	}
}

// ── Sort order ───────────────────────────────────────────────────────────────

func TestLoadDir_SortByTime(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt"})

	// Give each file a distinct mtime so the order is deterministic.
	base := int64(1_000_000)
	for i, name := range []string{"a.txt", "b.txt", "c.txt"} {
		ts := base + int64(i)*1000
		if err := os.Chtimes(filepath.Join(root, name),
			timeFromUnix(ts), timeFromUnix(ts)); err != nil {
			t.Fatalf("chtimes %s: %v", name, err)
		}
	}

	entries, err := loadDir(root, SortByTime)
	if err != nil {
		t.Fatalf("loadDir: %v", err)
	}

	// Newest first: c.txt, b.txt, a.txt
	want := []string{"c.txt", "b.txt", "a.txt"}
	names := fileNames(entries)
	for i, w := range want {
		if i >= len(names) || names[i] != w {
			t.Errorf("SortByTime[%d] = %q, want %q (all: %v)", i, names[i], w, names)
		}
	}
}

func TestLoadDir_SortBySize(t *testing.T) {
	root := t.TempDir()
	// Write files with known sizes: big=100 bytes, mid=50 bytes, small=10 bytes.
	for name, size := range map[string]int{"big.txt": 100, "mid.txt": 50, "small.txt": 10} {
		if err := os.WriteFile(filepath.Join(root, name),
			make([]byte, size), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	entries, err := loadDir(root, SortBySize)
	if err != nil {
		t.Fatalf("loadDir: %v", err)
	}

	// Largest first.
	want := []string{"big.txt", "mid.txt", "small.txt"}
	names := fileNames(entries)
	for i, w := range want {
		if i >= len(names) || names[i] != w {
			t.Errorf("SortBySize[%d] = %q, want %q (all: %v)", i, names[i], w, names)
		}
	}
}

func TestLoadDir_SortByType(t *testing.T) {
	root := setupTempDir(t, nil, []string{"b.go", "a.go", "z.txt", "m.txt", "noext"})

	entries, err := loadDir(root, SortByType)
	if err != nil {
		t.Fatalf("loadDir: %v", err)
	}

	// .go files first (alphabetically), then .txt, then no-extension last.
	want := []string{"a.go", "b.go", "m.txt", "z.txt", "noext"}
	names := fileNames(entries)
	for i, w := range want {
		if i >= len(names) || names[i] != w {
			t.Errorf("SortByType[%d] = %q, want %q (all: %v)", i, names[i], w, names)
		}
	}
}

func TestLoadDir_DirsAlwaysFirst(t *testing.T) {
	root := setupTempDir(t, []string{"zdir"}, []string{"a.txt"})

	for _, so := range AllSortOrders {
		entries, err := loadDir(root, so)
		if err != nil {
			t.Fatalf("loadDir(%v): %v", so, err)
		}
		if len(entries) < 2 {
			t.Fatalf("expected at least 2 entries")
		}
		if !entries[0].IsDir {
			t.Errorf("SortOrder=%v: first entry should be a directory, got %q", so, entries[0].Name)
		}
	}
}

func TestSetSortOrder_ResortsCurrent(t *testing.T) {
	root := t.TempDir()
	for name, size := range map[string]int{"big.txt": 100, "small.txt": 5} {
		if err := os.WriteFile(filepath.Join(root, name),
			make([]byte, size), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	m := New(root) // default SortByName: big.txt, small.txt
	m2 := m.SetSortOrder(SortBySize)

	if m2.CurrentSortOrder() != SortBySize {
		t.Errorf("CurrentSortOrder = %v, want SortBySize", m2.CurrentSortOrder())
	}
	if len(m2.entries) == 0 {
		t.Fatal("no entries after SetSortOrder")
	}
	if m2.entries[0].Name != "big.txt" {
		t.Errorf("after SortBySize first entry = %q, want big.txt", m2.entries[0].Name)
	}
	// Cursor should reset to 0.
	if m2.cursor != 0 {
		t.Errorf("cursor after SetSortOrder = %d, want 0", m2.cursor)
	}
}

func TestSortOrder_Label(t *testing.T) {
	cases := map[SortOrder]string{
		SortByName: "Name",
		SortByTime: "Timestamp",
		SortBySize: "Size",
		SortByType: "Type",
	}
	for so, want := range cases {
		if got := so.Label(); got != want {
			t.Errorf("SortOrder(%d).Label() = %q, want %q", so, got, want)
		}
	}
}

// fileNames returns just the Name field of each entry (files only).
func fileNames(entries []FileEntry) []string {
	var out []string
	for _, e := range entries {
		if !e.IsDir {
			out = append(out, e.Name)
		}
	}
	return out
}

// timeFromUnix converts a Unix timestamp to time.Time (used in mtime tests).
func timeFromUnix(sec int64) time.Time {
	return time.Unix(sec, 0)
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

func TestSelectedPath_WithEntries(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt"})
	m := New(root)
	// cursor starts at 0, first entry is a.txt
	got := m.SelectedPath()
	want := filepath.Join(root, "a.txt")
	if got != want {
		t.Errorf("SelectedPath() = %q, want %q", got, want)
	}

	// Move cursor to second entry.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	got = m.SelectedPath()
	want = filepath.Join(root, "b.txt")
	if got != want {
		t.Errorf("SelectedPath() after down = %q, want %q", got, want)
	}
}

func TestSelectedPath_EmptyDir(t *testing.T) {
	root := setupTempDir(t, nil, nil)
	m := New(root)
	got := m.SelectedPath()
	if got != root {
		t.Errorf("SelectedPath() on empty dir = %q, want %q (CWD)", got, root)
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
