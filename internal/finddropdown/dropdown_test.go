package finddropdown

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bjcarnes/splorer/internal/menubar"
)

type fireMsg struct{ id string }

func twoItemModel() Model {
	return New([]menubar.SubItem{
		{Label: "By Name", Key: 'n', Msg: fireMsg{id: "name"}},
		{Label: "By Content", Key: 'c', Msg: fireMsg{id: "content"}},
	}, 5)
}

// An item-key press (e.g. 'n') must activate that sub-item directly, regardless
// of where the cursor currently is.
func TestLetterHotkey_ActivatesSubItem(t *testing.T) {
	m := twoItemModel()
	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	if !m2.IsClosed() {
		t.Fatal("dropdown should close after activation")
	}
	if cmd == nil {
		t.Fatal("activation must return a command")
	}
	msg, ok := cmd().(fireMsg)
	if !ok || msg.id != "content" {
		t.Errorf("want fireMsg{content}, got %T %v", cmd(), cmd())
	}
}

// Letter matching must be case-insensitive ("N" matches 'n').
func TestLetterHotkey_CaseInsensitive(t *testing.T) {
	m := twoItemModel()
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'N', Text: "N"})
	if cmd == nil {
		t.Fatal("uppercase N should match 'n' sub-item")
	}
	if msg, _ := cmd().(fireMsg); msg.id != "name" {
		t.Errorf("uppercase N activated wrong item: %v", msg)
	}
}

func TestArrowKeys_MoveCursor(t *testing.T) {
	m := twoItemModel()
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("after Down: cursor = %d, want 1", m.cursor)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m.cursor != 1 {
		t.Errorf("Down past end: cursor = %d, want 1 (clamped)", m.cursor)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("after Up: cursor = %d, want 0", m.cursor)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m.cursor != 0 {
		t.Errorf("Up past top: cursor = %d, want 0 (clamped)", m.cursor)
	}
}

func TestEnter_ActivatesHighlightedItem(t *testing.T) {
	m := twoItemModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m2, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !m2.IsClosed() {
		t.Fatal("Enter should close the dropdown")
	}
	if msg, _ := cmd().(fireMsg); msg.id != "content" {
		t.Errorf("Enter activated wrong item: %v", msg)
	}
}

func TestEsc_ClosesWithoutActivating(t *testing.T) {
	m := twoItemModel()
	m2, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !m2.IsClosed() {
		t.Error("Esc should close the dropdown")
	}
	if cmd != nil {
		t.Error("Esc must not emit an activation command")
	}
}

// Clicks on a content row select and activate that item.
func TestClick_OnItemActivates(t *testing.T) {
	m := twoItemModel()
	// Box layout: y=1 is top border; y=2 is item 0; y=3 is item 1.
	click := tea.MouseClickMsg{
		X: m.x + 3, Y: m.y + 2, // second item
		Button: tea.MouseLeft,
	}
	m2, cmd := m.Update(click)
	if !m2.IsClosed() {
		t.Error("click on item should activate and close")
	}
	if msg, _ := cmd().(fireMsg); msg.id != "content" {
		t.Errorf("click activated wrong item: %v %T", cmd(), cmd())
	}
}

// Clicks outside the dropdown's bounds are not this component's concern —
// the owning app closes the dropdown on outside clicks. Inside-the-box clicks
// on a border row are absorbed without activation.
func TestClick_OnBorderAbsorbedNoActivation(t *testing.T) {
	m := twoItemModel()
	click := tea.MouseClickMsg{
		X: m.x + 3, Y: m.y, // top border row
		Button: tea.MouseLeft,
	}
	m2, cmd := m.Update(click)
	if m2.IsClosed() {
		t.Error("click on border should not close the dropdown")
	}
	if cmd != nil {
		t.Error("click on border must not emit an activation command")
	}
}

func TestContains_Bounds(t *testing.T) {
	m := twoItemModel()
	w, h := m.Width(), m.Height()

	// Inside corners and center.
	if !m.Contains(m.x, m.y) {
		t.Error("top-left corner should be inside")
	}
	if !m.Contains(m.x+w-1, m.y+h-1) {
		t.Error("bottom-right corner should be inside")
	}
	// Just outside on each side.
	if m.Contains(m.x-1, m.y) {
		t.Error("one cell left of box must not be inside")
	}
	if m.Contains(m.x+w, m.y) {
		t.Error("one cell right of box must not be inside")
	}
	if m.Contains(m.x, m.y-1) {
		t.Error("one row above box must not be inside")
	}
	if m.Contains(m.x, m.y+h) {
		t.Error("one row below box must not be inside")
	}
}

func TestRender_ContainsLabels(t *testing.T) {
	m := twoItemModel()
	out := m.Render()
	for _, want := range []string{"By Name", "By Content"} {
		if !containsSubstring(out, want) {
			t.Errorf("render missing %q", want)
		}
	}
}

func containsSubstring(s, sub string) bool {
	if len(sub) == 0 || len(s) < len(sub) {
		return len(sub) == 0
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
