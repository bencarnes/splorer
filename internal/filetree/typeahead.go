package filetree

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// typeaheadTimeout is how long a typed prefix stays alive after the last
// keystroke. Windows Explorer uses roughly a second: long enough to type a
// multi-character name at a normal pace, short enough that a prefix you
// abandoned doesn't swallow the next thing you type.
const typeaheadTimeout = time.Second

// chordMods are the two modifiers splorer builds bindings from (Ctrl+C,
// Alt+M, …), either of which on its own makes a keystroke a binding rather
// than text. Both at once is AltGr, which is how non-US layouts type
// characters like @ and € — no splorer binding uses Ctrl+Alt together, so
// that combination stays text. See typeaheadText.
const chordMods = tea.ModCtrl | tea.ModAlt

// desktopMods are the window-manager-level modifiers. Nothing in splorer binds
// them, but a keystroke carrying one was meant for something else.
const desktopMods = tea.ModMeta | tea.ModHyper | tea.ModSuper

// typeaheadTickMsg is emitted one timeout after a typeahead keystroke purely
// to force a repaint, so the prefix indicator in the status bar disappears
// when the prefix goes stale instead of lingering until the next event.
// Expiry itself is decided by the clock (see typeahead.active), so dropping
// this message — an open overlay swallows it, for instance — costs nothing
// but the repaint.
type typeaheadTickMsg struct{}

// typeahead is the "start typing a name to jump to it" state: the prefix
// typed so far, and when the last character of it was typed.
type typeahead struct {
	buf string
	at  time.Time
}

// active reports whether the prefix is still accepting characters.
func (t typeahead) active() bool {
	return t.buf != "" && time.Since(t.at) < typeaheadTimeout
}

// typeaheadText reports whether msg is a plain printable keystroke — the kind
// that feeds the typeahead prefix rather than triggering a binding. Text is
// only ever non-empty for keys that produce characters, so Enter, Tab, and
// the arrow/Home/End family are excluded by that alone. Shift is not
// disqualifying (it's how you type a capital), nor are the lock states, which
// say nothing about whether this particular key was chorded.
func typeaheadText(msg tea.KeyPressMsg) (string, bool) {
	mod := msg.Mod
	chords := mod & chordMods
	if msg.Text == "" || mod&desktopMods != 0 || (chords != 0 && chords != chordMods) {
		return "", false
	}
	return msg.Text, true
}

// matchIndex returns the index of the first entry at or after start — wrapping
// once through the whole list — whose name begins with prefix, compared
// case-insensitively. Returns -1 when nothing matches.
func matchIndex(entries []FileEntry, prefix string, start int) int {
	n := len(entries)
	if n == 0 || prefix == "" {
		return -1
	}
	lower := strings.ToLower(prefix)
	start = ((start % n) + n) % n
	for i := 0; i < n; i++ {
		idx := (start + i) % n
		if strings.HasPrefix(strings.ToLower(entries[idx].Name), lower) {
			return idx
		}
	}
	return -1
}

// typeaheadPress folds one printable keystroke into the typeahead prefix and
// moves the cursor to the matching entry. Three cases, matching Windows
// Explorer:
//
//   - a fresh prefix (nothing buffered, or the buffer has expired) searches
//     from the row after the cursor, so the keystroke moves off a row that
//     already matches;
//   - repeating the single character already buffered cycles to the next entry
//     starting with it, rather than searching for a doubled prefix;
//   - anything else extends the prefix and searches from the cursor, since the
//     current row may still match the longer prefix.
//
// A keystroke that matches nothing is swallowed: the prefix and the cursor
// both stay put and only the timeout is refreshed, so a typo can't strand the
// buffer on a prefix no entry can ever match.
func (m Model) typeaheadPress(text string) Model {
	if len(m.entries) == 0 {
		return m
	}

	prefix, start := text, m.cursor+1
	if m.typeahead.active() && m.typeahead.buf != text {
		prefix, start = m.typeahead.buf+text, m.cursor
	}

	m.typeahead.at = time.Now()
	idx := matchIndex(m.entries, prefix, start)
	if idx < 0 {
		return m
	}
	m.typeahead.buf = prefix
	m = m.ClearSelection()
	return m.moveCursor(idx - m.cursor)
}

// typeaheadTickCmd schedules the repaint that retires a stale prefix indicator.
func typeaheadTickCmd() tea.Cmd {
	return tea.Tick(typeaheadTimeout, func(time.Time) tea.Msg { return typeaheadTickMsg{} })
}

// clearTypeahead drops any prefix in flight.
func (m Model) clearTypeahead() Model {
	m.typeahead = typeahead{}
	return m
}

// TypeaheadActive reports whether a typed prefix is currently in flight. The
// app consults this so Esc can cancel the prefix instead of quitting.
func (m Model) TypeaheadActive() bool { return m.typeahead.active() }
