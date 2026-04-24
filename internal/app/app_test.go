package app

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

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
