package filetree

import (
	"os"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// typeaheadFixture builds a directory whose listing is, in order:
//
//	0 data/  1 docs/  2 apple.txt  3 Banana.md  4 banjo.go  5 cherry.txt
func typeaheadFixture(t *testing.T) Model {
	t.Helper()
	root := setupTempDir(t,
		[]string{"data", "docs"},
		[]string{"apple.txt", "Banana.md", "banjo.go", "cherry.txt"},
	)
	m := New(root)
	if len(m.entries) != 6 {
		t.Fatalf("fixture: got %d entries, want 6", len(m.entries))
	}
	return m
}

// typeStr feeds each rune of s to the model as a printable keystroke.
func typeStr(m Model, s string) Model {
	for _, r := range s {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestTypeahead_JumpsToPrefix(t *testing.T) {
	m := typeaheadFixture(t)

	m = typeStr(m, "ban")
	if got := m.entries[m.cursor].Name; got != "Banana.md" {
		t.Errorf("after typing \"ban\": cursor on %q, want \"Banana.md\"", got)
	}
	if !m.TypeaheadActive() {
		t.Error("prefix should still be in flight right after typing")
	}
	if m.typeahead.buf != "ban" {
		t.Errorf("prefix = %q, want \"ban\"", m.typeahead.buf)
	}
}

func TestTypeahead_MatchIsCaseInsensitive(t *testing.T) {
	m := typeaheadFixture(t)

	m = typeStr(m, "BAN")
	if got := m.entries[m.cursor].Name; got != "Banana.md" {
		t.Errorf("after typing \"BAN\": cursor on %q, want \"Banana.md\"", got)
	}
}

func TestTypeahead_MatchesDirectories(t *testing.T) {
	m := typeaheadFixture(t)

	m = typeStr(m, "doc")
	if got := m.entries[m.cursor].Name; got != "docs" {
		t.Errorf("after typing \"doc\": cursor on %q, want \"docs\"", got)
	}
}

// A prefix that runs off the end of the list wraps around to the top rather
// than giving up: "da" is only reachable from row 0, below the starting cursor.
func TestTypeahead_SearchWrapsAround(t *testing.T) {
	m := typeaheadFixture(t)

	m = typeStr(m, "da")
	if got := m.entries[m.cursor].Name; got != "data" {
		t.Errorf("after typing \"da\": cursor on %q, want \"data\"", got)
	}
}

// Repeating one character cycles through every entry starting with it, in
// listing order, wrapping back to the first at the end.
func TestTypeahead_RepeatedCharCycles(t *testing.T) {
	m := typeaheadFixture(t)

	want := []string{"Banana.md", "banjo.go", "Banana.md"}
	for i, w := range want {
		m = typeStr(m, "b")
		if got := m.entries[m.cursor].Name; got != w {
			t.Errorf("press %d of \"b\": cursor on %q, want %q", i+1, got, w)
		}
		if m.typeahead.buf != "b" {
			t.Errorf("press %d of \"b\": prefix = %q, want \"b\"", i+1, m.typeahead.buf)
		}
	}
}

// A keystroke that matches nothing is swallowed: neither the cursor nor the
// prefix moves, so a typo can't strand the buffer on an impossible prefix.
func TestTypeahead_NoMatchLeavesCursorAndPrefix(t *testing.T) {
	m := typeaheadFixture(t)

	m = typeStr(m, "ba")
	cursor, prefix := m.cursor, m.typeahead.buf

	m = typeStr(m, "z")
	if m.cursor != cursor {
		t.Errorf("cursor moved to %d on a non-matching key, want %d", m.cursor, cursor)
	}
	if m.typeahead.buf != prefix {
		t.Errorf("prefix = %q after a non-matching key, want %q", m.typeahead.buf, prefix)
	}
}

func TestTypeahead_ExpiresAndStartsFresh(t *testing.T) {
	m := typeaheadFixture(t)

	m = typeStr(m, "b")
	if got := m.entries[m.cursor].Name; got != "Banana.md" {
		t.Fatalf("after typing \"b\": cursor on %q, want \"Banana.md\"", got)
	}

	// Age the prefix past its timeout; the next key must start over rather
	// than extend "b" into "ba".
	m.typeahead.at = time.Now().Add(-2 * typeaheadTimeout)
	if m.TypeaheadActive() {
		t.Fatal("prefix should have expired")
	}

	m = typeStr(m, "a")
	if got := m.entries[m.cursor].Name; got != "apple.txt" {
		t.Errorf("after the prefix expired, typing \"a\" landed on %q, want \"apple.txt\"", got)
	}
	if m.typeahead.buf != "a" {
		t.Errorf("prefix = %q, want \"a\"", m.typeahead.buf)
	}
}

// Jumping to an entry resets the multi-selection, matching what a plain click
// does: the row you typed to is what Delete/Copy/Cut will act on.
func TestTypeahead_ClearsMultiSelection(t *testing.T) {
	m := typeaheadFixture(t)

	m, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if len(m.selected) != 1 {
		t.Fatalf("Space should have selected 1 entry, got %d", len(m.selected))
	}

	m = typeStr(m, "che")
	if len(m.selected) != 0 {
		t.Errorf("typeahead should clear the multi-selection, %d entries still marked", len(m.selected))
	}
	if got := m.SelectionPaths(); len(got) != 1 || got[0] != m.entries[m.cursor].Path {
		t.Errorf("SelectionPaths() should fall back to the cursor row, got %v", got)
	}
}

func TestTypeahead_NonPrintableKeyEndsSession(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{
		{Code: tea.KeyDown},
		{Code: tea.KeyUp},
		{Code: tea.KeyEsc},
		{Code: tea.KeyHome},
	} {
		m := typeaheadFixture(t)
		m = typeStr(m, "b")
		if !m.TypeaheadActive() {
			t.Fatal("prefix should be in flight")
		}
		m, _ = m.Update(k)
		if m.TypeaheadActive() {
			t.Errorf("%s should have ended the typeahead session", k.String())
		}
	}
}

// Ctrl or Alt on its own makes a keystroke a binding, not text. Shift and the
// lock states don't. Ctrl+Alt together is AltGr — how non-US layouts type
// characters — and no splorer binding uses it, so it stays text.
func TestTypeahead_ModifiersDecideTextVsBinding(t *testing.T) {
	tests := []struct {
		name string
		mod  tea.KeyMod
		want bool
	}{
		{"unmodified", 0, true},
		{"shift", tea.ModShift, true},
		{"caps lock", tea.ModCapsLock, true},
		{"altgr (ctrl+alt)", tea.ModCtrl | tea.ModAlt, true},
		{"alt", tea.ModAlt, false},
		{"ctrl", tea.ModCtrl, false},
		{"super", tea.ModSuper, false},
	}
	for _, tc := range tests {
		msg := tea.KeyPressMsg{Code: 'b', Text: "b", Mod: tc.mod}
		if _, got := typeaheadText(msg); got != tc.want {
			t.Errorf("%s: typeaheadText ok = %v, want %v", tc.name, got, tc.want)
		}
	}

	// End to end: an Alt chord must leave the cursor alone.
	m := typeaheadFixture(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'b', Text: "b", Mod: tea.ModAlt})
	if m.TypeaheadActive() {
		t.Error("Alt+b should not start a typeahead prefix")
	}
	if m.cursor != 0 {
		t.Errorf("Alt+b moved the cursor to %d, want 0", m.cursor)
	}
}

// Enter, Tab, Backspace, and the arrow family carry no Text, so they can never
// be mistaken for typeahead characters regardless of how the terminal reports
// them.
func TestTypeahead_SpecialKeysCarryNoText(t *testing.T) {
	for _, code := range []rune{
		tea.KeyEnter, tea.KeyTab, tea.KeyBackspace, tea.KeyEsc,
		tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight, tea.KeyHome, tea.KeyEnd,
	} {
		msg := tea.KeyPressMsg{Code: code}
		if _, ok := typeaheadText(msg); ok {
			t.Errorf("%s was treated as typeahead text", msg.String())
		}
	}
}

// Space is the one printable key that keeps a binding: with no prefix in
// flight it toggles the cursor row's selection.
func TestTypeahead_SpaceTogglesWhenIdle(t *testing.T) {
	m := typeaheadFixture(t)

	m, _ = m.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if len(m.selected) != 1 {
		t.Errorf("Space with no prefix should toggle selection, %d entries marked", len(m.selected))
	}
	if m.TypeaheadActive() {
		t.Error("Space with no prefix should not start a typeahead prefix")
	}
}

// Mid-prefix, Space is just another character, so names containing spaces are
// reachable.
func TestTypeahead_SpaceTypesWhenPrefixActive(t *testing.T) {
	root := setupTempDir(t, nil, []string{"my file.txt", "myx.txt"})
	m := New(root)

	m = typeStr(m, "my file")
	if got := m.entries[m.cursor].Name; got != "my file.txt" {
		t.Errorf("after typing \"my file\": cursor on %q, want \"my file.txt\"", got)
	}
	if len(m.selected) != 0 {
		t.Errorf("Space mid-prefix should type, not toggle selection; %d entries marked", len(m.selected))
	}
}

func TestTypeahead_EmptyDirectoryIsSafe(t *testing.T) {
	m := New(t.TempDir())

	m = typeStr(m, "abc")
	if m.cursor != 0 {
		t.Errorf("cursor = %d in an empty directory, want 0", m.cursor)
	}
	if m.TypeaheadActive() {
		t.Error("no prefix should accumulate in an empty directory")
	}
}

func TestTypeahead_ClearedByNavigation(t *testing.T) {
	m := typeaheadFixture(t)

	m = typeStr(m, "do") // lands on docs/
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.TypeaheadActive() {
		t.Error("entering a directory should clear the typeahead prefix")
	}
}

func TestHomeEndJumpToFirstAndLast(t *testing.T) {
	m := typeaheadFixture(t)

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	if m.cursor != len(m.entries)-1 {
		t.Errorf("End: cursor = %d, want %d", m.cursor, len(m.entries)-1)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	if m.cursor != 0 {
		t.Errorf("Home: cursor = %d, want 0", m.cursor)
	}
}

// The home-directory jump moved off "~" (now a typeahead character) onto
// Alt+Home. Plain "~" must type instead of navigating.
func TestAltHomeJumpsToHomeDirectory(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory in this environment: %v", err)
	}
	m := typeaheadFixture(t)

	m, _ = m.Update(tea.KeyPressMsg{Code: '~', Text: "~"})
	if m.CWD() == home {
		t.Fatal("plain ~ should type, not navigate to the home directory")
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyHome, Mod: tea.ModAlt})
	if m.CWD() != home {
		t.Errorf("Alt+Home: CWD = %q, want %q", m.CWD(), home)
	}
}

func TestMatchIndex(t *testing.T) {
	entries := []FileEntry{
		{Name: "alpha"}, {Name: "Beta"}, {Name: "beacon"}, {Name: "gamma"},
	}

	tests := []struct {
		prefix string
		start  int
		want   int
	}{
		{"a", 0, 0},
		{"b", 0, 1},
		{"b", 2, 2},
		{"be", 3, 1}, // wraps past the end
		{"z", 0, -1},
		{"", 0, -1},
		{"BEA", 0, 2}, // case-insensitive
	}
	for _, tc := range tests {
		if got := matchIndex(entries, tc.prefix, tc.start); got != tc.want {
			t.Errorf("matchIndex(%q, start=%d) = %d, want %d", tc.prefix, tc.start, got, tc.want)
		}
	}
	if got := matchIndex(nil, "a", 0); got != -1 {
		t.Errorf("matchIndex(nil, …) = %d, want -1", got)
	}
}
