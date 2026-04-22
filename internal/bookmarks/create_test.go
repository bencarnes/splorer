package bookmarks

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestCreateDialog_InitialState(t *testing.T) {
	d := NewCreateDialog("/some/path")
	if d.IsClosed() {
		t.Error("new dialog should not be closed")
	}
	if d.IsSaved() {
		t.Error("new dialog should not be saved")
	}
	if d.Name() != "" {
		t.Errorf("initial name = %q, want empty", d.Name())
	}
	if d.BookmarkPath() != "/some/path" {
		t.Errorf("path = %q, want /some/path", d.BookmarkPath())
	}
}

func TestCreateDialog_EscClosesWithoutSaving(t *testing.T) {
	d := NewCreateDialog("/tmp")
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !d.IsClosed() {
		t.Error("esc should close dialog")
	}
	if d.IsSaved() {
		t.Error("esc should not save")
	}
}

func TestCreateDialog_EnterWithEmptyInputDoesNothing(t *testing.T) {
	d := NewCreateDialog("/tmp")
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if d.IsClosed() {
		t.Error("enter on empty input should not close")
	}
	if d.IsSaved() {
		t.Error("enter on empty input should not save")
	}
}

func TestCreateDialog_EnterWithWhitespaceOnlyDoesNothing(t *testing.T) {
	d := NewCreateDialog("/tmp")
	d.input = "   "
	d.cursor = 3
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if d.IsClosed() {
		t.Error("enter on whitespace-only input should not close")
	}
}

func TestCreateDialog_EnterWithNameSaves(t *testing.T) {
	d := NewCreateDialog("/home/user")
	// Type "My Bookmark"
	for _, ch := range "My Bookmark" {
		d, _ = d.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	if d.Name() != "My Bookmark" {
		t.Fatalf("name = %q, want %q", d.Name(), "My Bookmark")
	}
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !d.IsClosed() {
		t.Error("enter with name should close dialog")
	}
	if !d.IsSaved() {
		t.Error("enter with name should save")
	}
	if d.Name() != "My Bookmark" {
		t.Errorf("name after save = %q, want %q", d.Name(), "My Bookmark")
	}
}

func TestCreateDialog_SingleCharSaves(t *testing.T) {
	d := NewCreateDialog("/tmp")
	d, _ = d.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !d.IsSaved() {
		t.Error("single character name should be accepted")
	}
}

func TestCreateDialog_Backspace(t *testing.T) {
	d := NewCreateDialog("/tmp")
	d, _ = d.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	d, _ = d.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	d, _ = d.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	if d.Name() != "abc" {
		t.Fatalf("name = %q, want abc", d.Name())
	}
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if d.Name() != "ab" {
		t.Errorf("after backspace name = %q, want ab", d.Name())
	}
	if d.cursor != 2 {
		t.Errorf("cursor = %d, want 2", d.cursor)
	}
}

func TestCreateDialog_CursorMovement(t *testing.T) {
	d := NewCreateDialog("/tmp")
	for _, ch := range "abc" {
		d, _ = d.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
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

	// Left at 0 stays at 0.
	d2 := NewCreateDialog("/tmp")
	d2, _ = d2.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if d2.cursor != 0 {
		t.Errorf("left at 0: cursor = %d, want 0", d2.cursor)
	}

	// Right at end stays at end.
	d, _ = d.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if d.cursor != 3 {
		t.Errorf("right at end: cursor = %d, want 3", d.cursor)
	}
}

func TestCreateDialog_CtrlA_CtrlE(t *testing.T) {
	d := NewCreateDialog("/tmp")
	for _, ch := range "hello" {
		d, _ = d.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	d, _ = d.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl})
	if d.cursor != 0 {
		t.Errorf("ctrl+a: cursor = %d, want 0", d.cursor)
	}
	d, _ = d.Update(tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl})
	if d.cursor != 5 {
		t.Errorf("ctrl+e: cursor = %d, want 5", d.cursor)
	}
}

func TestCreateDialog_Render_DoesNotPanic(t *testing.T) {
	d := NewCreateDialog("/home/user/documents")
	_ = d.Render(80, 24)

	// With name typed
	for _, ch := range "My Docs" {
		d, _ = d.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	_ = d.Render(80, 24)
}

func TestCreateDialog_Render_SmallDimensions(t *testing.T) {
	d := NewCreateDialog("/tmp")
	// Should not panic even at very small sizes.
	_ = d.Render(20, 5)
}

func TestCreateDialog_Render_ContainsPath(t *testing.T) {
	d := NewCreateDialog("/home/user/docs")
	out := d.Render(80, 24)
	// The path should appear somewhere in the rendered output.
	if !strings.Contains(out, "/home/user/docs") {
		t.Error("render should display the path")
	}
}
