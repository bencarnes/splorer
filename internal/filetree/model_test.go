package filetree

import (
	"os"
	"path/filepath"
	"runtime"
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
	if runtime.GOOS == "windows" {
		t.Skip("Unix mode bits don't restrict access on Windows NTFS; " +
			"an equivalent test would require manipulating ACLs")
	}
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
	m2, _ := m.SetSortOrder(SortBySize)

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
	root := filesystemRoot(t)
	m := New(root)
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m2.cwd != root {
		t.Errorf("navigating up from %q should stay put, got %q", root, m2.cwd)
	}
}

// filesystemRoot walks up from t.TempDir until filepath.Dir becomes a fixed
// point, which is the real filesystem root ("/" on Unix, e.g. "C:\" on Windows).
func filesystemRoot(t *testing.T) string {
	t.Helper()
	p := t.TempDir()
	for {
		parent := filepath.Dir(p)
		if parent == p {
			return p
		}
		p = parent
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

// ── Filesystem watcher ───────────────────────────────────────────────────────

func TestDirChangedMsg_UpdatesEntries(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt"})
	m := New(root)

	if err := os.WriteFile(filepath.Join(root, "c.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("write c.txt: %v", err)
	}
	newEntries, err := loadDir(root, SortByName)
	if err != nil {
		t.Fatalf("loadDir: %v", err)
	}

	m2, cmd := m.Update(DirChangedMsg{Dir: root, SortOrder: SortByName, Entries: newEntries})
	if len(m2.entries) != 3 {
		t.Errorf("expected 3 entries after update, got %d", len(m2.entries))
	}
	if cmd == nil {
		t.Error("DirChangedMsg should return a new watch command")
	}
}

func TestDirChangedMsg_NoChangeNoUpdate(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt"})
	m := New(root)
	before := m.entries[0].Name

	// Send identical entries — model should not change meaningfully.
	same, _ := loadDir(root, SortByName)
	m2, cmd := m.Update(DirChangedMsg{Dir: root, SortOrder: SortByName, Entries: same})
	if m2.entries[0].Name != before {
		t.Errorf("no-change DirChangedMsg should not alter entries")
	}
	if cmd == nil {
		t.Error("DirChangedMsg should still return a watch command on no-change")
	}
}

func TestDirChangedMsg_Stale_DifferentDir(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt"})
	m := New(root)

	m2, cmd := m.Update(DirChangedMsg{Dir: "/some/other/path", SortOrder: SortByName, Entries: []FileEntry{}})
	if len(m2.entries) == 0 {
		t.Error("stale DirChangedMsg (wrong dir) should not clear entries")
	}
	if cmd != nil {
		t.Error("stale message should not reschedule the watcher")
	}
}

func TestDirChangedMsg_Stale_OldSortOrder(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt"})
	m := New(root) // default SortByName

	m2, cmd := m.Update(DirChangedMsg{Dir: root, SortOrder: SortBySize, Entries: []FileEntry{}})
	if len(m2.entries) == 0 {
		t.Error("DirChangedMsg with wrong sort order should not clear entries")
	}
	if cmd != nil {
		t.Error("stale sort-order message should not reschedule the watcher")
	}
}

func TestDirChangedMsg_NilEntries_Reschedules(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt"})
	m := New(root)
	before := len(m.entries)

	// nil Entries signals a transient read error.
	m2, cmd := m.Update(DirChangedMsg{Dir: root, SortOrder: SortByName, Entries: nil})
	if len(m2.entries) != before {
		t.Error("nil-entries DirChangedMsg should not change entries")
	}
	if cmd == nil {
		t.Error("nil-entries DirChangedMsg should still reschedule watcher")
	}
}

func TestDirChangedMsg_PreservesCursor(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt"})
	m := New(root)
	m.cursor = 1 // b.txt

	newEntries := []FileEntry{
		{Name: "a.txt", Path: filepath.Join(root, "a.txt")},
		{Name: "b.txt", Path: filepath.Join(root, "b.txt")},
		{Name: "c.txt", Path: filepath.Join(root, "c.txt")},
		{Name: "d.txt", Path: filepath.Join(root, "d.txt")},
	}
	m2, _ := m.Update(DirChangedMsg{Dir: root, SortOrder: SortByName, Entries: newEntries})
	if m2.cursor != 1 {
		t.Errorf("cursor should remain on b.txt (index 1), got %d", m2.cursor)
	}
}

func TestDirChangedMsg_ClampsCursorWhenEntryGone(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt"})
	m := New(root)
	m.cursor = 2 // c.txt

	newEntries := []FileEntry{
		{Name: "a.txt", Path: filepath.Join(root, "a.txt")},
	}
	m2, _ := m.Update(DirChangedMsg{Dir: root, SortOrder: SortByName, Entries: newEntries})
	if m2.cursor != 0 {
		t.Errorf("cursor should clamp to 0 when selected entry is gone, got %d", m2.cursor)
	}
}

func TestDirGoneMsg_NavigatesToParent(t *testing.T) {
	root := setupTempDir(t, []string{"sub"}, nil)
	subPath := filepath.Join(root, "sub")
	m := New(subPath)

	if err := os.Remove(subPath); err != nil {
		t.Fatalf("remove sub: %v", err)
	}

	m2, cmd := m.Update(DirGoneMsg{Dir: subPath})
	if m2.cwd != root {
		t.Errorf("after DirGoneMsg cwd = %q, want %q", m2.cwd, root)
	}
	if cmd == nil {
		t.Error("DirGoneMsg should return a new watch command")
	}
}

func TestDirGoneMsg_Stale(t *testing.T) {
	root := setupTempDir(t, nil, nil)
	m := New(root)

	m2, cmd := m.Update(DirGoneMsg{Dir: "/nonexistent/stale/path"})
	if m2.cwd != root {
		t.Errorf("stale DirGoneMsg changed cwd to %q", m2.cwd)
	}
	if cmd != nil {
		t.Error("stale DirGoneMsg should not reschedule the watcher")
	}
}

func TestNearestExistingAncestor(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(a, "b")
	if err := os.MkdirAll(b, 0755); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}

	// Remove b only — nearest ancestor of b is a.
	if err := os.Remove(b); err != nil {
		t.Fatalf("remove b: %v", err)
	}
	if got := nearestExistingAncestor(b); got != a {
		t.Errorf("nearestExistingAncestor(%q) = %q, want %q", b, got, a)
	}

	// Remove a as well — nearest ancestor of b is now root.
	if err := os.Remove(a); err != nil {
		t.Fatalf("remove a: %v", err)
	}
	if got := nearestExistingAncestor(b); got != root {
		t.Errorf("nearestExistingAncestor(%q) after removing a = %q, want %q", b, got, root)
	}
}

func TestSetSortOrder_ReturnsWatchCmd(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt"})
	m := New(root)
	_, cmd := m.SetSortOrder(SortBySize)
	if cmd == nil {
		t.Error("SetSortOrder should return a non-nil watch command")
	}
}

// ── Multi-selection ──────────────────────────────────────────────────────────

func TestSelectionPaths_DefaultsToCursor(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt"})
	m := New(root)

	got := m.SelectionPaths()
	if len(got) != 1 || got[0] != filepath.Join(root, "a.txt") {
		t.Errorf("SelectionPaths with no multi-selection should be [cursor], got %v", got)
	}
}

func TestSelectionPaths_EmptyDir(t *testing.T) {
	root := setupTempDir(t, nil, nil)
	m := New(root)
	if got := m.SelectionPaths(); len(got) != 0 {
		t.Errorf("SelectionPaths on empty dir should be empty, got %v", got)
	}
}

func TestPlainClick_SingleSelects(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt"})
	m := New(root)
	m.height = 24
	m.width = 80

	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 1, Button: tea.MouseLeft})
	if !m.IsSelected(filepath.Join(root, "b.txt")) {
		t.Errorf("click should select b.txt")
	}
	if m.IsSelected(filepath.Join(root, "a.txt")) || m.IsSelected(filepath.Join(root, "c.txt")) {
		t.Errorf("plain click should single-select; selection set = %v", m.selected)
	}
}

func TestShiftClick_RangeSelectsFromAnchor(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt", "d.txt"})
	m := New(root)
	m.height = 24
	m.width = 80

	// Anchor on a.txt, then shift-click on c.txt — selection should span a..c.
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 0, Button: tea.MouseLeft})
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 2, Button: tea.MouseLeft, Mod: tea.ModShift})

	want := []string{
		filepath.Join(root, "a.txt"),
		filepath.Join(root, "b.txt"),
		filepath.Join(root, "c.txt"),
	}
	got := m.SelectionPaths()
	if len(got) != len(want) {
		t.Fatalf("range select length = %d, want %d (got %v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("range[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestCtrlClick_TogglesIndividualEntries(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt"})
	m := New(root)
	m.height = 24
	m.width = 80

	// Select a.txt, then ctrl-click c.txt → both selected.
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 0, Button: tea.MouseLeft})
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 2, Button: tea.MouseLeft, Mod: tea.ModCtrl})
	if !m.IsSelected(filepath.Join(root, "a.txt")) || !m.IsSelected(filepath.Join(root, "c.txt")) {
		t.Errorf("ctrl-click should add to selection: %v", m.selected)
	}
	if m.IsSelected(filepath.Join(root, "b.txt")) {
		t.Errorf("b.txt should not be selected: %v", m.selected)
	}

	// Ctrl-click c.txt again → toggles off.
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 2, Button: tea.MouseLeft, Mod: tea.ModCtrl})
	if m.IsSelected(filepath.Join(root, "c.txt")) {
		t.Errorf("second ctrl-click should remove c.txt; selection = %v", m.selected)
	}
}

func TestPlainClick_ResetsRangeAfterCtrlClicks(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt"})
	m := New(root)
	m.height = 24
	m.width = 80

	// Build up a multi-selection, then plain-click should reset to that one row.
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 0, Button: tea.MouseLeft})
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 2, Button: tea.MouseLeft, Mod: tea.ModCtrl})
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 1, Button: tea.MouseLeft})

	got := m.SelectionPaths()
	if len(got) != 1 || got[0] != filepath.Join(root, "b.txt") {
		t.Errorf("plain click should reset to single selection [b.txt], got %v", got)
	}
}

func TestNavigateInto_ClearsSelection(t *testing.T) {
	root := setupTempDir(t, []string{"sub"}, []string{"a.txt"})
	m := New(root)
	m.height = 24
	m.width = 80

	// Select something, then navigate into the subdir — selection should reset.
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 0, Button: tea.MouseLeft})
	if len(m.selected) == 0 {
		t.Fatal("precondition: should have a selection")
	}
	for i, e := range m.entries {
		if e.Name == "sub" {
			m.cursor = i
			break
		}
	}
	m, _ = m.activate()
	if len(m.selected) != 0 {
		t.Errorf("selection should clear on navigate; got %v", m.selected)
	}
}

func TestApplyEntryRefresh_DropsMissingSelections(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt"})
	m := New(root)
	m.height = 24
	m.width = 80
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 0, Button: tea.MouseLeft})
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 1, Button: tea.MouseLeft, Mod: tea.ModCtrl})
	if len(m.selected) != 2 {
		t.Fatalf("precondition: 2 selections, got %d", len(m.selected))
	}

	// Simulate watcher: only a.txt remains.
	newEntries := []FileEntry{
		{Name: "a.txt", Path: filepath.Join(root, "a.txt")},
	}
	m, _ = m.Update(DirChangedMsg{Dir: root, SortOrder: m.sortOrder, Entries: newEntries})

	if len(m.selected) != 1 {
		t.Errorf("selection should prune missing entries; got %v", m.selected)
	}
	if !m.IsSelected(filepath.Join(root, "a.txt")) {
		t.Errorf("a.txt should remain selected")
	}
}

func TestSpace_TogglesCursorSelection(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt"})
	m := New(root)
	m.height = 24
	m.width = 80

	// Toggle cursor (a.txt) on, then move to b.txt and toggle on.
	m, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})

	if !m.IsSelected(filepath.Join(root, "a.txt")) || !m.IsSelected(filepath.Join(root, "b.txt")) {
		t.Errorf("space should toggle cursor row; selected = %v", m.selected)
	}
	if m.IsSelected(filepath.Join(root, "c.txt")) {
		t.Errorf("c.txt should not be selected: %v", m.selected)
	}

	// Toggle a.txt off again by moving back and pressing space.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	m, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if m.IsSelected(filepath.Join(root, "a.txt")) {
		t.Errorf("second space should toggle a.txt off; selected = %v", m.selected)
	}
}

func TestShiftDown_ExtendsSelection(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt", "d.txt"})
	m := New(root)
	m.height = 24
	m.width = 80

	// Cursor starts on a.txt; shift+down twice should select a..c inclusive.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift})

	want := []string{
		filepath.Join(root, "a.txt"),
		filepath.Join(root, "b.txt"),
		filepath.Join(root, "c.txt"),
	}
	got := m.SelectionPaths()
	if len(got) != len(want) {
		t.Fatalf("shift+down extension length = %d, want %d (got %v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("extend[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestShiftUp_ContractsSelection(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt", "c.txt"})
	m := New(root)
	m.height = 24
	m.width = 80

	// Move cursor down, then shift+down to extend, then shift+up to contract.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})           // cursor → b
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}) // anchor=b, extend to c
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift})   // contract back to b

	got := m.SelectionPaths()
	if len(got) != 1 || got[0] != filepath.Join(root, "b.txt") {
		t.Errorf("contract should leave just anchor selected; got %v", got)
	}
}

func TestClearSelection(t *testing.T) {
	root := setupTempDir(t, nil, []string{"a.txt", "b.txt"})
	m := New(root)
	m.height = 24
	m.width = 80
	m, _ = m.Update(tea.MouseClickMsg{Y: headerHeight + 0, Button: tea.MouseLeft})
	m = m.ClearSelection()
	if len(m.selected) != 0 || m.anchorPath != "" {
		t.Errorf("ClearSelection should drop set and anchor; selected=%v anchor=%q",
			m.selected, m.anchorPath)
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
