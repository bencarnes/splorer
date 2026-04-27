package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bjcarnes/splorer/internal/filetree"
	"github.com/bjcarnes/splorer/internal/menubar"
)

// isQuitCmd reports whether cmd, when invoked, returns tea.QuitMsg.
func isQuitCmd(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// newModel constructs a root app Model rooted in the current working dir.
// Tests only care about key routing, not the contents of the directory.
func newModel(t *testing.T) Model {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return New(cwd)
}

// asModel re-asserts a tea.Model back to the concrete app.Model so tests can
// inspect internal flags after an Update.
func asModel(t *testing.T, tm tea.Model) Model {
	t.Helper()
	m, ok := tm.(Model)
	if !ok {
		t.Fatalf("expected app.Model, got %T", tm)
	}
	return m
}

// CWD is read by main.go to write the final navigated directory to the
// shell-wrapper's temp file on exit. It must reflect the filetree's current
// directory both at construction time and after navigation.
func TestModel_CWD(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	m := New(cwd)
	if got := m.CWD(); got != cwd {
		t.Errorf("initial CWD() = %q, want %q", got, cwd)
	}

	// After navigating to the parent dir, CWD must follow.
	parent := filepath.Dir(cwd)
	if parent == cwd {
		t.Skip("already at filesystem root; no parent to navigate to")
	}
	ft, err := m.filetree.NavigateTo(parent)
	if err != nil {
		t.Fatalf("NavigateTo(%q): %v", parent, err)
	}
	m.filetree = ft
	if got := m.CWD(); got != parent {
		t.Errorf("after navigate CWD() = %q, want %q", got, parent)
	}
}

func TestMainScreen_QQuits(t *testing.T) {
	m := newModel(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if !isQuitCmd(cmd) {
		t.Error("pressing q on the main screen should return tea.Quit")
	}
}

func TestMainScreen_EscQuits(t *testing.T) {
	m := newModel(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !isQuitCmd(cmd) {
		t.Error("pressing Esc on the main screen should return tea.Quit")
	}
}

// When an overlay is open, q must not quit — it should reach the overlay so
// the user can type it as part of text input (e.g. a search pattern).
func TestSearchOpen_QDoesNotQuit(t *testing.T) {
	m := newModel(t)
	tm, _ := m.Update(openSearchByNameMsg{})
	m = asModel(t, tm)
	if !m.searchOpen {
		t.Fatal("search did not open")
	}

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if isQuitCmd(cmd) {
		t.Error("q should not quit while the search overlay is open")
	}
}

// When an overlay is open, Esc must not quit — it should reach the overlay so
// the dialog/overlay can close itself.
func TestSortDialogOpen_EscDoesNotQuit(t *testing.T) {
	m := newModel(t)
	tm, _ := m.Update(openSortMsg{})
	m = asModel(t, tm)
	if !m.sortDlgOpen {
		t.Fatal("sort dialog did not open")
	}

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if isQuitCmd(cmd) {
		t.Error("Esc should not quit while the sort dialog is open")
	}
}

// Alt+F must now open the Find dropdown, not the name-search view directly.
func TestAltF_OpensDropdown(t *testing.T) {
	m := newModel(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModAlt})
	if cmd == nil {
		t.Fatal("Alt+F produced no command")
	}
	// The menubar emits OpenDropdownMsg via a cmd; replay it so the root
	// model can react.
	tm, _ := m.Update(cmd())
	m = asModel(t, tm)

	if !m.dropdownOpen {
		t.Error("Alt+F should open the Find dropdown")
	}
	if m.searchOpen || m.csrchOpen {
		t.Error("neither search view should open directly from Alt+F anymore")
	}
}

// Pressing 'n' with the dropdown open opens the name-search view.
func TestDropdown_NSelectsByName(t *testing.T) {
	m := openFindDropdown(t, newModel(t))

	tm, cmd := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	m = asModel(t, tm)
	if m.dropdownOpen {
		t.Error("dropdown should close on sub-item activation")
	}
	if cmd == nil {
		t.Fatal("activation must emit a command")
	}

	tm2, _ := m.Update(cmd())
	m = asModel(t, tm2)
	if !m.searchOpen {
		t.Error("'n' should open the name-search view")
	}
	if m.csrchOpen {
		t.Error("content search must not open from 'n'")
	}
}

// Pressing 'c' with the dropdown open opens the content-search view.
func TestDropdown_CSelectsByContent(t *testing.T) {
	m := openFindDropdown(t, newModel(t))

	tm, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m = asModel(t, tm)
	if cmd == nil {
		t.Fatal("activation must emit a command")
	}
	tm2, _ := m.Update(cmd())
	m = asModel(t, tm2)
	if !m.csrchOpen {
		t.Error("'c' should open the content-search view")
	}
	if m.searchOpen {
		t.Error("name search must not open from 'c'")
	}
}

// Esc in the dropdown closes it without opening any search view and without
// quitting the app.
func TestDropdown_EscClosesAndDoesNotQuit(t *testing.T) {
	m := openFindDropdown(t, newModel(t))

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if isQuitCmd(cmd) {
		t.Error("Esc with dropdown open must not quit the app")
	}
	// Re-dispatching the Esc-returned command (if any) should not open a
	// search view; dropdown closure is silent.
}

// TestWatcherMsg_PassesThroughSearchOverlay verifies that DirChangedMsg
// reaches the file tree even while the search overlay is open.
func TestWatcherMsg_PassesThroughSearchOverlay(t *testing.T) {
	m := newModel(t)

	// The test directory must have at least one entry so we can observe it
	// being cleared by the simulated watcher update.
	if m.filetree.SelectedPath() == m.CWD() {
		t.Skip("test directory is empty; cannot verify entry update through overlay")
	}

	// Open the search overlay.
	tm, _ := m.Update(openSearchByNameMsg{})
	m = asModel(t, tm)
	if !m.searchOpen {
		t.Fatal("search overlay did not open")
	}

	// Simulate the watcher reporting an empty directory.
	msg := filetree.DirChangedMsg{
		Dir:       m.CWD(),
		SortOrder: m.filetree.CurrentSortOrder(),
		Entries:   []filetree.FileEntry{},
	}
	tm2, _ := m.Update(msg)
	m = asModel(t, tm2)

	// An empty directory returns CWD as the selected path.
	if got := m.filetree.SelectedPath(); got != m.CWD() {
		t.Errorf("SelectedPath after empty DirChangedMsg = %q, want CWD %q", got, m.CWD())
	}
	// The overlay must remain open — the watcher message must not close it.
	if !m.searchOpen {
		t.Error("search overlay should remain open after a watcher message")
	}
}

// openFindDropdown is a test helper that drives a model to the
// dropdown-open state and returns it.
func openFindDropdown(t *testing.T, m Model) Model {
	t.Helper()
	tm, _ := m.Update(menubar.OpenDropdownMsg{Index: 0})
	m = asModel(t, tm)
	if !m.dropdownOpen {
		t.Fatal("precondition: dropdown should be open")
	}
	return m
}

// ── Manipulate (delete / copy / cut / paste) ─────────────────────────────────

// newModelIn constructs a root app Model rooted in the given directory.
func newModelIn(t *testing.T, cwd string) Model {
	t.Helper()
	return New(cwd)
}

// makeFile writes `content` to a file under root and returns the path.
func makeFile(t *testing.T, root, name, content string) string {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// confirmYes drives the open confirmation dialog to OK.
func confirmYes(t *testing.T, m Model) Model {
	t.Helper()
	if !m.confirmDlgOpen {
		t.Fatalf("expected confirm dialog open")
	}
	tm, _ := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	return asModel(t, tm)
}

// confirmNo drives the open confirmation dialog to Cancel.
func confirmNo(t *testing.T, m Model) Model {
	t.Helper()
	if !m.confirmDlgOpen {
		t.Fatalf("expected confirm dialog open")
	}
	tm, _ := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	return asModel(t, tm)
}

func TestDelete_ConfirmDialogOpens(t *testing.T) {
	root := t.TempDir()
	makeFile(t, root, "a.txt", "x")
	m := newModelIn(t, root)

	tm, _ := m.Update(manipulateMsg{op: manipulateDelete})
	m = asModel(t, tm)
	if !m.confirmDlgOpen {
		t.Fatal("delete should open the confirm dialog")
	}
	if m.pendingOp != manipulateDelete {
		t.Errorf("pendingOp = %v, want manipulateDelete", m.pendingOp)
	}
}

func TestDelete_RemovesFileOnConfirm(t *testing.T) {
	root := t.TempDir()
	target := makeFile(t, root, "a.txt", "x")
	m := newModelIn(t, root)

	tm, _ := m.Update(manipulateMsg{op: manipulateDelete})
	m = asModel(t, tm)
	m = confirmYes(t, m)

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("a.txt should be removed; stat err = %v", err)
	}
	if m.confirmDlgOpen {
		t.Error("confirm dialog should be closed after Y")
	}
	if m.pendingOp != manipulateNone {
		t.Errorf("pendingOp should reset; got %v", m.pendingOp)
	}
}

func TestDelete_CancelLeavesFileAlone(t *testing.T) {
	root := t.TempDir()
	target := makeFile(t, root, "a.txt", "x")
	m := newModelIn(t, root)

	tm, _ := m.Update(manipulateMsg{op: manipulateDelete})
	m = asModel(t, tm)
	m = confirmNo(t, m)

	if _, err := os.Stat(target); err != nil {
		t.Errorf("a.txt should still exist after cancel: %v", err)
	}
}

func TestCopyPaste_DuplicatesFileIntoCwd(t *testing.T) {
	srcRoot := t.TempDir()
	src := makeFile(t, srcRoot, "a.txt", "alpha")
	destRoot := t.TempDir()

	m := newModelIn(t, srcRoot)

	// Select a.txt explicitly via ctrl-click so the test doesn't depend on
	// the cursor's default position.
	tm, _ := m.Update(tea.MouseClickMsg{Y: 2, Button: tea.MouseLeft, Mod: tea.ModCtrl})
	m = asModel(t, tm)
	if len(m.filetree.SelectionPaths()) != 1 {
		t.Fatalf("expected 1 selected, got %d", len(m.filetree.SelectionPaths()))
	}

	tm, _ = m.Update(manipulateMsg{op: manipulateCopy})
	m = asModel(t, tm)
	m = confirmYes(t, m)
	if len(m.clipboard) != 1 || m.clipboard[0] != src {
		t.Errorf("clipboard = %v, want [%s]", m.clipboard, src)
	}
	if m.clipboardMode != clipCopy {
		t.Errorf("clipboardMode = %v, want clipCopy", m.clipboardMode)
	}

	// Navigate to dest dir.
	ft, err := m.filetree.NavigateTo(destRoot)
	if err != nil {
		t.Fatalf("navigate: %v", err)
	}
	m.filetree = ft

	tm, _ = m.Update(manipulateMsg{op: manipulatePaste})
	m = asModel(t, tm)
	m = confirmYes(t, m)

	// File should be in both source and destination.
	if _, err := os.Stat(src); err != nil {
		t.Errorf("source missing after copy+paste: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(destRoot, "a.txt")); err != nil || string(got) != "alpha" {
		t.Errorf("dest copy = %q, %v", got, err)
	}
	// Copy mode persists so the user can paste again.
	if m.clipboardMode != clipCopy || len(m.clipboard) != 1 {
		t.Errorf("clipboard should persist after copy-paste: %v / mode=%v", m.clipboard, m.clipboardMode)
	}
}

func TestCutPaste_MovesFileAndClearsClipboard(t *testing.T) {
	srcRoot := t.TempDir()
	src := makeFile(t, srcRoot, "a.txt", "alpha")
	destRoot := t.TempDir()

	m := newModelIn(t, srcRoot)
	tm, _ := m.Update(tea.MouseClickMsg{Y: 2, Button: tea.MouseLeft, Mod: tea.ModCtrl})
	m = asModel(t, tm)

	tm, _ = m.Update(manipulateMsg{op: manipulateCut})
	m = asModel(t, tm)
	m = confirmYes(t, m)

	ft, err := m.filetree.NavigateTo(destRoot)
	if err != nil {
		t.Fatalf("navigate: %v", err)
	}
	m.filetree = ft

	tm, _ = m.Update(manipulateMsg{op: manipulatePaste})
	m = asModel(t, tm)
	m = confirmYes(t, m)

	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source should be moved; stat err = %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(destRoot, "a.txt")); err != nil || string(got) != "alpha" {
		t.Errorf("dest after cut+paste = %q, %v", got, err)
	}
	if len(m.clipboard) != 0 || m.clipboardMode != clipNone {
		t.Errorf("cut+paste should clear clipboard; got %v mode=%v", m.clipboard, m.clipboardMode)
	}
}

func TestPaste_NoClipboard_DoesNothing(t *testing.T) {
	root := t.TempDir()
	makeFile(t, root, "a.txt", "x")
	m := newModelIn(t, root)
	tm, _ := m.Update(manipulateMsg{op: manipulatePaste})
	m = asModel(t, tm)
	if m.confirmDlgOpen {
		t.Error("paste with empty clipboard must not open the confirm dialog")
	}
}

func TestDeleteKey_OpensConfirmDialog(t *testing.T) {
	root := t.TempDir()
	makeFile(t, root, "a.txt", "x")
	m := newModelIn(t, root)
	tm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDelete})
	m = asModel(t, tm)
	if !m.confirmDlgOpen || m.pendingOp != manipulateDelete {
		t.Errorf("Delete key should start a delete op; confirmOpen=%v pendingOp=%v",
			m.confirmDlgOpen, m.pendingOp)
	}
}

func TestCtrlC_StartsCopy(t *testing.T) {
	root := t.TempDir()
	makeFile(t, root, "a.txt", "x")
	m := newModelIn(t, root)
	tm, _ := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	m = asModel(t, tm)
	if !m.confirmDlgOpen || m.pendingOp != manipulateCopy {
		t.Errorf("Ctrl+C should start copy; confirmOpen=%v pendingOp=%v",
			m.confirmDlgOpen, m.pendingOp)
	}
}

// ── Rename ──────────────────────────────────────────────────────────────────

func TestRename_ConfirmDialogOpensWithSingleEntry(t *testing.T) {
	root := t.TempDir()
	makeFile(t, root, "a.txt", "x")
	m := newModelIn(t, root)

	tm, _ := m.Update(manipulateMsg{op: manipulateRename})
	m = asModel(t, tm)
	if !m.renameDlgOpen {
		t.Fatal("rename should open the rename dialog with one entry selected")
	}
	if m.confirmDlgOpen {
		t.Error("rename must not open the generic confirm dialog")
	}
	if m.pendingOp != manipulateRename {
		t.Errorf("pendingOp = %v, want manipulateRename", m.pendingOp)
	}
	if len(m.pendingPaths) != 1 {
		t.Errorf("pendingPaths = %v, want one entry", m.pendingPaths)
	}
}

func TestRename_RenamesFileOnSave(t *testing.T) {
	root := t.TempDir()
	src := makeFile(t, root, "a.txt", "alpha")
	m := newModelIn(t, root)

	tm, _ := m.Update(manipulateMsg{op: manipulateRename})
	m = asModel(t, tm)
	if !m.renameDlgOpen {
		t.Fatal("rename dialog did not open")
	}

	// Backspace the prepopulated "a.txt" then type "b.txt".
	for i := 0; i < len("a.txt"); i++ {
		tm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
		m = asModel(t, tm)
	}
	for _, ch := range "b.txt" {
		tm, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = asModel(t, tm)
	}
	tm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = asModel(t, tm)

	if m.renameDlgOpen {
		t.Error("rename dialog should be closed after Enter on a changed name")
	}
	if m.pendingOp != manipulateNone {
		t.Errorf("pendingOp should reset; got %v", m.pendingOp)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("a.txt should be gone, stat err = %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "b.txt")); err != nil || string(got) != "alpha" {
		t.Errorf("b.txt content = %q, %v", got, err)
	}
}

func TestRename_EscLeavesFileAlone(t *testing.T) {
	root := t.TempDir()
	src := makeFile(t, root, "a.txt", "alpha")
	m := newModelIn(t, root)

	tm, _ := m.Update(manipulateMsg{op: manipulateRename})
	m = asModel(t, tm)
	tm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = asModel(t, tm)

	if m.renameDlgOpen {
		t.Error("Esc should close the rename dialog")
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("a.txt should still exist after cancel: %v", err)
	}
}

func TestRename_MultiSelectionDoesNothing(t *testing.T) {
	root := t.TempDir()
	makeFile(t, root, "a.txt", "x")
	makeFile(t, root, "b.txt", "y")
	m := newModelIn(t, root)

	// Select row 0 with Space, move down, select row 1 with Space.
	tm, _ := m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m = asModel(t, tm)
	tm, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	m = asModel(t, tm)
	tm, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	m = asModel(t, tm)
	if got := len(m.filetree.SelectionPaths()); got != 2 {
		t.Fatalf("expected 2 selected, got %d", got)
	}

	tm, _ = m.Update(manipulateMsg{op: manipulateRename})
	m = asModel(t, tm)
	if m.renameDlgOpen {
		t.Error("rename should be a no-op with 2+ entries selected")
	}
	if m.pendingOp != manipulateNone {
		t.Errorf("pendingOp = %v, want manipulateNone", m.pendingOp)
	}
}

func TestRename_EmptyDirDoesNothing(t *testing.T) {
	root := t.TempDir()
	m := newModelIn(t, root)
	tm, _ := m.Update(manipulateMsg{op: manipulateRename})
	m = asModel(t, tm)
	if m.renameDlgOpen {
		t.Error("rename should be a no-op in an empty directory")
	}
}

func TestRename_CollisionSurfacesError(t *testing.T) {
	root := t.TempDir()
	src := makeFile(t, root, "a.txt", "x")
	makeFile(t, root, "b.txt", "existing")
	m := newModelIn(t, root)

	tm, _ := m.Update(manipulateMsg{op: manipulateRename})
	m = asModel(t, tm)

	// Replace "a.txt" with "b.txt".
	for i := 0; i < len("a.txt"); i++ {
		tm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
		m = asModel(t, tm)
	}
	for _, ch := range "b.txt" {
		tm, _ = m.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
		m = asModel(t, tm)
	}
	tm, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = asModel(t, tm)

	if _, err := os.Stat(src); err != nil {
		t.Errorf("a.txt should still exist after refused rename: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "b.txt")); err != nil || string(got) != "existing" {
		t.Errorf("b.txt should be untouched: %q, %v", got, err)
	}
}

func TestF2_StartsRename(t *testing.T) {
	root := t.TempDir()
	makeFile(t, root, "a.txt", "x")
	m := newModelIn(t, root)
	tm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyF2})
	m = asModel(t, tm)
	if !m.renameDlgOpen || m.pendingOp != manipulateRename {
		t.Errorf("F2 should start a rename op; renameOpen=%v pendingOp=%v",
			m.renameDlgOpen, m.pendingOp)
	}
}

func TestAltT_OpensTreeView(t *testing.T) {
	m := newModel(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 't', Mod: tea.ModAlt})
	if cmd == nil {
		t.Fatal("Alt+T produced no command")
	}
	tm, _ := m.Update(cmd())
	m = asModel(t, tm)
	if !m.treeOpen {
		t.Error("Alt+T should open the tree view")
	}
}

func TestTreeView_EscClosesAndDoesNotQuit(t *testing.T) {
	m := newModel(t)
	tm, _ := m.Update(openTreeMsg{})
	m = asModel(t, tm)
	if !m.treeOpen {
		t.Fatal("tree did not open")
	}
	tm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	m = asModel(t, tm)
	if isQuitCmd(cmd) {
		t.Error("Esc should not quit while the tree view is open")
	}
	if m.treeOpen {
		t.Error("Esc should close the tree view")
	}
}

func TestAltH_OpensHelpPage(t *testing.T) {
	m := newModel(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'h', Mod: tea.ModAlt})
	if cmd == nil {
		t.Fatal("Alt+H produced no command")
	}
	tm, _ := m.Update(cmd())
	m = asModel(t, tm)
	if !m.helpOpen {
		t.Error("Alt+H should open the help page")
	}
}

func TestHelpPage_AnyKeyCloses(t *testing.T) {
	m := newModel(t)
	tm, _ := m.Update(openHelpMsg{})
	m = asModel(t, tm)
	if !m.helpOpen {
		t.Fatal("help did not open")
	}
	// q must not quit while the help page owns the screen — it should close
	// the page instead and keep the app alive.
	tm, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	m = asModel(t, tm)
	if isQuitCmd(cmd) {
		t.Error("q should not quit while help is open")
	}
	if m.helpOpen {
		t.Error("any key should close the help page")
	}
}

func TestAltM_OpensManipulateDropdown(t *testing.T) {
	m := newModel(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'm', Mod: tea.ModAlt})
	if cmd == nil {
		t.Fatal("Alt+M produced no command")
	}
	tm, _ := m.Update(cmd())
	m = asModel(t, tm)
	if !m.dropdownOpen {
		t.Error("Alt+M should open the Manipulate dropdown")
	}
}
