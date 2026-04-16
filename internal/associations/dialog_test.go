package associations

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// keyPress builds a KeyPressMsg from a key string (e.g. "a", "tab", "esc").
func keyPress(key string) tea.KeyPressMsg {
	runes := []rune(key)
	// Single printable character.
	if len(runes) == 1 && key != "tab" && key != "esc" {
		return tea.KeyPressMsg{Code: runes[0], Text: key}
	}
	// Special / combo keys — match via Code constants.
	switch key {
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		return tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEsc}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "d":
		return tea.KeyPressMsg{Code: 'd', Text: "d"}
	}
	return tea.KeyPressMsg{}
}

func TestDialog_InitialState(t *testing.T) {
	d := NewDialog(map[string]string{".pdf": "evince"})
	if d.IsClosed() {
		t.Error("new dialog should not be closed")
	}
	if d.focus != focusList {
		t.Errorf("initial focus = %v, want focusList", d.focus)
	}
	if len(d.keys) != 1 || d.keys[0] != ".pdf" {
		t.Errorf("keys = %v, want [.pdf]", d.keys)
	}
}

func TestDialog_EscCloses(t *testing.T) {
	d := NewDialog(nil)
	d, _ = d.Update(keyPress("esc"))
	if !d.IsClosed() {
		t.Error("esc should close dialog")
	}
}

func TestDialog_TabCyclesFocus(t *testing.T) {
	d := NewDialog(nil)
	// list → ext
	d, _ = d.Update(keyPress("tab"))
	if d.focus != focusExt {
		t.Errorf("after tab from list: focus = %v, want focusExt", d.focus)
	}
	// ext → prog
	d, _ = d.Update(keyPress("tab"))
	if d.focus != focusProg {
		t.Errorf("after tab from ext: focus = %v, want focusProg", d.focus)
	}
	// prog → list
	d, _ = d.Update(keyPress("tab"))
	if d.focus != focusList {
		t.Errorf("after tab from prog: focus = %v, want focusList", d.focus)
	}
}

func TestDialog_ShiftTabReversesCycle(t *testing.T) {
	d := NewDialog(nil)
	// list → prog (backwards)
	d, _ = d.Update(keyPress("shift+tab"))
	if d.focus != focusProg {
		t.Errorf("shift+tab from list: focus = %v, want focusProg", d.focus)
	}
	// prog → ext
	d, _ = d.Update(keyPress("shift+tab"))
	if d.focus != focusExt {
		t.Errorf("shift+tab from prog: focus = %v, want focusExt", d.focus)
	}
	// ext → list
	d, _ = d.Update(keyPress("shift+tab"))
	if d.focus != focusList {
		t.Errorf("shift+tab from ext: focus = %v, want focusList", d.focus)
	}
}

func TestDialog_AddAssociation(t *testing.T) {
	d := NewDialog(nil)

	// Move to ext field and type extension
	d, _ = d.Update(keyPress("tab")) // list → ext

	for _, ch := range ".txt" {
		d, _ = d.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}

	// Move to prog field and type program
	d, _ = d.Update(keyPress("tab")) // ext → prog
	for _, ch := range "gedit" {
		d, _ = d.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}

	// Confirm with enter
	d, _ = d.Update(keyPress("enter"))

	assocs := d.Assocs()
	if assocs[".txt"] != "gedit" {
		t.Errorf("assocs[.txt] = %q, want %q", assocs[".txt"], "gedit")
	}
	if d.focus != focusList {
		t.Errorf("after add: focus = %v, want focusList", d.focus)
	}
	if d.extInput != "" || d.progInput != "" {
		t.Errorf("inputs should be cleared after add, got ext=%q prog=%q", d.extInput, d.progInput)
	}
}

func TestDialog_AddAssociation_AutoDot(t *testing.T) {
	d := NewDialog(nil)
	d, _ = d.Update(keyPress("tab")) // → ext

	// type without leading dot
	for _, ch := range "pdf" {
		d, _ = d.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	d, _ = d.Update(keyPress("tab")) // → prog
	for _, ch := range "evince" {
		d, _ = d.Update(tea.KeyPressMsg{Code: ch, Text: string(ch)})
	}
	d, _ = d.Update(keyPress("enter"))

	assocs := d.Assocs()
	if assocs[".pdf"] != "evince" {
		t.Errorf("expected auto-dot; assocs[.pdf] = %q, want evince", assocs[".pdf"])
	}
}

func TestDialog_AddAssociation_EmptyFieldsIgnored(t *testing.T) {
	d := NewDialog(nil)
	d, _ = d.Update(keyPress("tab")) // → ext (leave empty)
	d, _ = d.Update(keyPress("tab")) // → prog (leave empty)
	d, _ = d.Update(keyPress("enter"))

	if len(d.Assocs()) != 0 {
		t.Errorf("empty fields should not add association, got %v", d.Assocs())
	}
	if d.IsClosed() {
		t.Error("dialog should stay open when add is a no-op")
	}
}

func TestDialog_DeleteAssociation(t *testing.T) {
	d := NewDialog(map[string]string{
		".pdf": "evince",
		".mp3": "vlc",
	})
	// list is focused, cursor is on first key (.mp3, sorted)
	if d.keys[0] != ".mp3" {
		t.Fatalf("expected first key .mp3, got %q", d.keys[0])
	}

	d, _ = d.Update(keyPress("d"))

	assocs := d.Assocs()
	if _, ok := assocs[".mp3"]; ok {
		t.Error(".mp3 should have been deleted")
	}
	if assocs[".pdf"] != "evince" {
		t.Error(".pdf should still be present")
	}
}

func TestDialog_ListNavigation(t *testing.T) {
	d := NewDialog(map[string]string{
		".a": "prog1",
		".b": "prog2",
		".c": "prog3",
	})
	if d.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", d.cursor)
	}

	d, _ = d.Update(keyPress("down"))
	if d.cursor != 1 {
		t.Errorf("cursor after down = %d, want 1", d.cursor)
	}

	d, _ = d.Update(keyPress("down"))
	d, _ = d.Update(keyPress("down")) // should stay at 2
	if d.cursor != 2 {
		t.Errorf("cursor after past-end down = %d, want 2", d.cursor)
	}

	d, _ = d.Update(keyPress("up"))
	if d.cursor != 1 {
		t.Errorf("cursor after up = %d, want 1", d.cursor)
	}
}

func TestDialog_Backspace(t *testing.T) {
	d := NewDialog(nil)
	d, _ = d.Update(keyPress("tab")) // → ext

	// type "ab"
	d, _ = d.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
	d, _ = d.Update(tea.KeyPressMsg{Code: 'b', Text: "b"})
	if d.extInput != "ab" {
		t.Fatalf("extInput = %q, want \"ab\"", d.extInput)
	}

	d, _ = d.Update(keyPress("backspace"))
	if d.extInput != "a" {
		t.Errorf("after backspace: extInput = %q, want \"a\"", d.extInput)
	}
	if d.extCursor != 1 {
		t.Errorf("extCursor = %d, want 1", d.extCursor)
	}
}

func TestDialog_AssocsReturnsCopy(t *testing.T) {
	d := NewDialog(map[string]string{".pdf": "evince"})
	a := d.Assocs()
	a[".pdf"] = "CHANGED"

	// Original dialog should be unaffected
	if d.Assocs()[".pdf"] != "evince" {
		t.Error("Assocs() should return a copy, not a reference")
	}
}

func TestInsertTextAt(t *testing.T) {
	cases := []struct {
		s    string
		pos  int
		text string
		want string
	}{
		{"hello", 0, "X", "Xhello"},
		{"hello", 5, "!", "hello!"},
		{"hello", 2, "XY", "heXYllo"},
		{"", 0, "abc", "abc"},
	}
	for _, tc := range cases {
		got := insertTextAt(tc.s, tc.pos, tc.text)
		if got != tc.want {
			t.Errorf("insertTextAt(%q, %d, %q) = %q, want %q", tc.s, tc.pos, tc.text, got, tc.want)
		}
	}
}

func TestDeleteRuneAt(t *testing.T) {
	cases := []struct {
		s       string
		pos     int
		wantS   string
		wantPos int
	}{
		{"hello", 5, "hell", 4},
		{"hello", 1, "ello", 0},
		{"hello", 0, "hello", 0}, // nothing to delete
		{"a", 1, "", 0},
	}
	for _, tc := range cases {
		gotS, gotPos := deleteRuneAt(tc.s, tc.pos)
		if gotS != tc.wantS || gotPos != tc.wantPos {
			t.Errorf("deleteRuneAt(%q, %d) = (%q, %d), want (%q, %d)",
				tc.s, tc.pos, gotS, gotPos, tc.wantS, tc.wantPos)
		}
	}
}
