package renamedialog

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNew_PrepopulatesWithBasename(t *testing.T) {
	d := New("/some/dir/old.txt")
	if d.NewName() != "old.txt" {
		t.Errorf("NewName() = %q, want old.txt", d.NewName())
	}
	if d.cursor != len("old.txt") {
		t.Errorf("cursor = %d, want %d", d.cursor, len("old.txt"))
	}
	if d.IsClosed() || d.IsSaved() {
		t.Error("new dialog should not be closed or saved")
	}
	if d.Path() != "/some/dir/old.txt" {
		t.Errorf("Path() = %q, want /some/dir/old.txt", d.Path())
	}
}

func TestEscClosesWithoutSaving(t *testing.T) {
	d := New("/tmp/a.txt")
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !d.IsClosed() {
		t.Error("esc should close dialog")
	}
	if d.IsSaved() {
		t.Error("esc should not save")
	}
}

func TestEnterWithUnchangedNameDoesNothing(t *testing.T) {
	d := New("/tmp/a.txt")
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if d.IsClosed() {
		t.Error("enter on unchanged name should not close")
	}
	if d.IsSaved() {
		t.Error("enter on unchanged name should not save")
	}
}

func TestEnterWithEmptyInputDoesNothing(t *testing.T) {
	d := New("/tmp/a.txt")
	// Backspace the whole name out.
	for i := 0; i < len("a.txt"); i++ {
		d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	if d.NewName() != "" {
		t.Fatalf("NewName() after clearing = %q, want empty", d.NewName())
	}
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if d.IsClosed() {
		t.Error("enter on empty input should not close")
	}
	if d.IsSaved() {
		t.Error("enter on empty input should not save")
	}
}

func TestEnterWithChangedNameSaves(t *testing.T) {
	d := New("/tmp/a.txt")
	// Backspace then type a new name.
	for i := 0; i < len("a.txt"); i++ {
		d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	for _, ch := range "b.txt" {
		d, _ = d.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	if d.NewName() != "b.txt" {
		t.Fatalf("NewName() = %q, want b.txt", d.NewName())
	}
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !d.IsClosed() {
		t.Error("enter with new name should close")
	}
	if !d.IsSaved() {
		t.Error("enter with new name should save")
	}
}

func TestBackspaceDeletesFromEnd(t *testing.T) {
	d := New("/tmp/abc")
	if d.NewName() != "abc" {
		t.Fatalf("NewName = %q, want abc", d.NewName())
	}
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if d.NewName() != "ab" {
		t.Errorf("after backspace NewName = %q, want ab", d.NewName())
	}
	if d.cursor != 2 {
		t.Errorf("cursor = %d, want 2", d.cursor)
	}
}

func TestCursorMovement(t *testing.T) {
	d := New("/tmp/abc")
	if d.cursor != 3 {
		t.Fatalf("initial cursor = %d, want 3", d.cursor)
	}
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if d.cursor != 2 {
		t.Errorf("left: cursor = %d, want 2", d.cursor)
	}
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if d.cursor != 3 {
		t.Errorf("right: cursor = %d, want 3", d.cursor)
	}
	// Right at end stays at end.
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if d.cursor != 3 {
		t.Errorf("right at end: cursor = %d, want 3", d.cursor)
	}
}

func TestCtrlA_CtrlE(t *testing.T) {
	d := New("/tmp/hello")
	d, _ = d.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	if d.cursor != 0 {
		t.Errorf("ctrl+a: cursor = %d, want 0", d.cursor)
	}
	d, _ = d.Update(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	if d.cursor != 5 {
		t.Errorf("ctrl+e: cursor = %d, want 5", d.cursor)
	}
}

func TestNewName_TrimsWhitespace(t *testing.T) {
	d := New("/tmp/x")
	for _, ch := range "  " {
		d, _ = d.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	// Input is now "x  " — NewName should trim to "x".
	if d.NewName() != "x" {
		t.Errorf("NewName() = %q, want x", d.NewName())
	}
	// And Enter should reject it (same as basename after trim).
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if d.IsSaved() {
		t.Error("trimmed name equal to basename should not save")
	}
}

func TestRender_DoesNotPanic(t *testing.T) {
	d := New("/home/user/documents/report.txt")
	_ = d.Render(80, 24)
	_ = d.Render(20, 5)
}

func TestRender_ContainsPath(t *testing.T) {
	d := New("/home/user/docs/file.go")
	out := d.Render(80, 24)
	if !strings.Contains(out, "file.go") {
		t.Error("render should display the entry name")
	}
}
