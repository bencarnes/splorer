package contentsearch

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// newInputModel returns a fresh Model in stateInput with non-zero dimensions,
// ready to receive keyboard / paste messages.
func newInputModel() Model {
	return New("/root", 80, 24)
}

func TestPaste_InsertsIntoPatternField(t *testing.T) {
	m := newInputModel()
	m2, _ := m.Update(tea.PasteMsg{Content: "hello world"})
	m = m2

	if m.pattern != "hello world" {
		t.Errorf("pattern = %q, want %q", m.pattern, "hello world")
	}
	if m.patternCur != len([]rune("hello world")) {
		t.Errorf("patternCur = %d, want %d", m.patternCur, len([]rune("hello world")))
	}
	if m.extensions != "" {
		t.Errorf("extensions field was touched: %q", m.extensions)
	}
}

func TestPaste_InsertsIntoExtensionsFieldWhenFocused(t *testing.T) {
	m := newInputModel()
	// Tab once to focus the extensions field.
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	m = m2
	if m.focus != focusExtensions {
		t.Fatalf("precondition: expected focusExtensions, got %v", m.focus)
	}

	m3, _ := m.Update(tea.PasteMsg{Content: ".go,.md"})
	m = m3

	if m.extensions != ".go,.md" {
		t.Errorf("extensions = %q, want %q", m.extensions, ".go,.md")
	}
	if m.pattern != "" {
		t.Errorf("pattern field was touched: %q", m.pattern)
	}
}

// Paste at the current cursor — not always the end — should splice, not
// append. We type "AB", move left once, then paste "X" between them.
func TestPaste_InsertsAtCursor(t *testing.T) {
	m := newInputModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: 'A', Text: "A"})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'B', Text: "B"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m.patternCur != 1 {
		t.Fatalf("precondition: patternCur = %d, want 1", m.patternCur)
	}

	m, _ = m.Update(tea.PasteMsg{Content: "X"})
	if m.pattern != "AXB" {
		t.Errorf("pattern = %q, want %q", m.pattern, "AXB")
	}
	if m.patternCur != 2 {
		t.Errorf("patternCur = %d, want 2 (after pasted char)", m.patternCur)
	}
}

// Multi-line clipboard payloads (CR, LF, CRLF) collapse into one line —
// the Pattern and Ext fields are both single-line.
func TestPaste_StripsNewlines(t *testing.T) {
	cases := []struct {
		name, input, want string
	}{
		{"lf", "foo\nbar", "foobar"},
		{"crlf", "foo\r\nbar", "foobar"},
		{"cr", "foo\rbar", "foobar"},
		{"trailing_lf", "pattern\n", "pattern"},
		{"leading_lf", "\npattern", "pattern"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := newInputModel()
			m, _ = m.Update(tea.PasteMsg{Content: c.input})
			if m.pattern != c.want {
				t.Errorf("pattern = %q, want %q", m.pattern, c.want)
			}
		})
	}
}

// Paste of an empty (or all-newline) payload must not change the model.
func TestPaste_EmptyIsNoOp(t *testing.T) {
	cases := []string{"", "\n", "\r\n", "\r\n\r\n"}
	for _, c := range cases {
		m := newInputModel()
		m, _ = m.Update(tea.PasteMsg{Content: c})
		if m.pattern != "" || m.patternCur != 0 {
			t.Errorf("empty paste %q modified field: pattern=%q cur=%d", c, m.pattern, m.patternCur)
		}
	}
}

// While a search is running, pastes must be ignored — there's no editable
// field to receive them and accepting one could confuse the stateInput/
// stateSearching invariant.
func TestPaste_IgnoredWhileSearching(t *testing.T) {
	m := newInputModel()
	// Minimal manual transition to stateSearching — no goroutine.
	m.state = stateSearching
	before := m.pattern

	m, _ = m.Update(tea.PasteMsg{Content: "ignored"})
	if m.pattern != before {
		t.Errorf("pattern mutated during stateSearching: %q (was %q)", m.pattern, before)
	}
}

func TestPaste_IgnoredInDoneState(t *testing.T) {
	m := newInputModel()
	m.state = stateDone
	m.pattern = "prev"
	m.patternCur = 4

	m, _ = m.Update(tea.PasteMsg{Content: "more"})
	if m.pattern != "prev" || m.patternCur != 4 {
		t.Errorf("paste in stateDone mutated field: pattern=%q cur=%d", m.pattern, m.patternCur)
	}
}

// Paste must clear any prior validation error (e.g. a dangling "invalid
// regex" message from a bad Enter), so the status line doesn't stick after
// the user has started typing a new pattern.
func TestPaste_ClearsErrMsg(t *testing.T) {
	m := newInputModel()
	m.errMsg = "invalid regex: foo"
	m, _ = m.Update(tea.PasteMsg{Content: "valid"})
	if m.errMsg != "" {
		t.Errorf("errMsg should be cleared on paste, got %q", m.errMsg)
	}
}
