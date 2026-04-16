package menubar

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// dummyMsg is a test message type used to verify activation.
type dummyMsg struct{ label string }

func makeBar(labels ...string) MenuBar {
	items := make([]Item, len(labels))
	for i, l := range labels {
		label := l // capture
		items[i] = Item{
			Label:  label,
			Hotkey: "alt+" + string(rune('a'+i)),
			Msg:    dummyMsg{label: label},
		}
	}
	return New(items)
}

func TestHandleKey_Match(t *testing.T) {
	// makeBar assigns "alt+a" to the first item (index 0).
	b := makeBar("Openers")
	cmd := b.HandleKey(tea.KeyPressMsg{Code: 'a', Mod: tea.ModAlt})
	if cmd == nil {
		t.Fatal("HandleKey returned nil for matching hotkey")
	}
	msg := cmd()
	dm, ok := msg.(dummyMsg)
	if !ok || dm.label != "Openers" {
		t.Errorf("cmd() = %v, want dummyMsg{Openers}", msg)
	}
}

func TestHandleKey_NoMatch(t *testing.T) {
	b := makeBar("Openers")
	cmd := b.HandleKey(tea.KeyPressMsg{Code: 'z'})
	if cmd != nil {
		t.Error("HandleKey should return nil for non-matching key")
	}
}

func TestHandleKey_NoHotkey(t *testing.T) {
	b := New([]Item{{Label: "NoKey", Msg: dummyMsg{label: "x"}}})
	cmd := b.HandleKey(tea.KeyPressMsg{Code: 'n'})
	if cmd != nil {
		t.Error("item with empty hotkey should never match")
	}
}

func TestHandleClick_HitsItem(t *testing.T) {
	b := makeBar("Openers")
	b.Width = 80

	ranges := b.ItemRanges()
	if len(ranges) == 0 {
		t.Fatal("no item ranges computed")
	}
	// Click in the middle of the first item's range.
	mid := (ranges[0][0] + ranges[0][1]) / 2
	cmd := b.HandleClick(mid)
	if cmd == nil {
		t.Fatalf("HandleClick(%d) returned nil; range was %v", mid, ranges[0])
	}
	msg := cmd()
	dm, ok := msg.(dummyMsg)
	if !ok || dm.label != "Openers" {
		t.Errorf("cmd() = %v, want dummyMsg{Openers}", msg)
	}
}

func TestHandleClick_MissesGap(t *testing.T) {
	b := makeBar("Alpha", "Beta")
	b.Width = 80

	ranges := b.ItemRanges()
	// The gap between items is one cell past the end of the first item.
	gap := ranges[0][1]
	cmd := b.HandleClick(gap)
	if cmd != nil {
		t.Errorf("click in gap (x=%d) should return nil, got %v", gap, cmd)
	}
}

func TestHandleClick_BeforeFirstItem(t *testing.T) {
	b := makeBar("Openers")
	b.Width = 80
	cmd := b.HandleClick(0) // leading space
	if cmd != nil {
		t.Error("click before first item should return nil")
	}
}

func TestItemRanges_MultipleItems(t *testing.T) {
	b := makeBar("Alpha", "Beta", "Gamma")
	b.Width = 80
	ranges := b.ItemRanges()

	if len(ranges) != 3 {
		t.Fatalf("expected 3 ranges, got %d", len(ranges))
	}
	// Each range must be non-empty.
	for i, r := range ranges {
		if r[1] <= r[0] {
			t.Errorf("ranges[%d] = %v is empty", i, r)
		}
	}
	// Ranges must not overlap.
	for i := 1; i < len(ranges); i++ {
		if ranges[i][0] < ranges[i-1][1] {
			t.Errorf("ranges[%d] starts before ranges[%d] ends: %v vs %v",
				i, i-1, ranges[i], ranges[i-1])
		}
	}
}

func TestRender_NonEmpty(t *testing.T) {
	b := makeBar("Openers")
	b.Width = 80
	if got := b.Render(); got == "" {
		t.Error("Render returned empty string")
	}
}

func TestRender_ContainsLabel(t *testing.T) {
	b := makeBar("Openers")
	b.Width = 80
	rendered := b.Render()
	// The label text must appear somewhere in the rendered output
	// (ANSI codes may surround it, but the rune sequence is present).
	if !containsSubstring(rendered, "Openers") {
		t.Errorf("Render output does not contain label \"Openers\": %q", rendered)
	}
}

// containsSubstring reports whether sub appears anywhere in s as a
// contiguous run of bytes (ANSI escape codes may be interspersed, so this
// is a best-effort sanity check).
func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}()
}
