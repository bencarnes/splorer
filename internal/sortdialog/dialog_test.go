package sortdialog

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bjcarnes/splorer/internal/filetree"
)

func kp(key string) tea.KeyPressMsg {
	switch key {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEsc}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	default:
		runes := []rune(key)
		if len(runes) == 1 {
			return tea.KeyPressMsg{Code: runes[0], Text: key}
		}
		return tea.KeyPressMsg{}
	}
}

func TestNew_CursorOnCurrent(t *testing.T) {
	for i, so := range filetree.AllSortOrders {
		d := New(so)
		if d.cursor != i {
			t.Errorf("New(%v): cursor = %d, want %d", so, d.cursor, i)
		}
		if d.IsClosed() {
			t.Errorf("New(%v): should not be closed", so)
		}
		if d.IsSaved() {
			t.Errorf("New(%v): should not be saved", so)
		}
	}
}

func TestNew_DefaultsToName(t *testing.T) {
	d := New(filetree.SortByName)
	if d.cursor != 0 {
		t.Errorf("SortByName cursor = %d, want 0", d.cursor)
	}
}

func TestEsc_Closes(t *testing.T) {
	d := New(filetree.SortByName)
	d, _ = d.Update(kp("esc"))
	if !d.IsClosed() {
		t.Error("esc should close dialog")
	}
	if d.IsSaved() {
		t.Error("esc should not save")
	}
}

func TestEnter_SavesAndCloses(t *testing.T) {
	d := New(filetree.SortByName)
	d, _ = d.Update(kp("down")) // move to SortByTime
	d, _ = d.Update(kp("enter"))
	if !d.IsClosed() {
		t.Error("enter should close")
	}
	if !d.IsSaved() {
		t.Error("enter should save")
	}
	if d.Chosen() != filetree.SortByTime {
		t.Errorf("chosen = %v, want SortByTime", d.Chosen())
	}
}

func TestEnter_NoMove_KeepsOriginal(t *testing.T) {
	d := New(filetree.SortBySize)
	d, _ = d.Update(kp("enter"))
	if d.Chosen() != filetree.SortBySize {
		t.Errorf("chosen = %v, want SortBySize", d.Chosen())
	}
}

func TestNavigation_Down(t *testing.T) {
	d := New(filetree.SortByName)
	for i := 1; i < len(filetree.AllSortOrders); i++ {
		d, _ = d.Update(kp("down"))
		if d.cursor != i {
			t.Errorf("after %d downs: cursor = %d, want %d", i, d.cursor, i)
		}
	}
	// Past last: stays at last.
	last := len(filetree.AllSortOrders) - 1
	d, _ = d.Update(kp("down"))
	if d.cursor != last {
		t.Errorf("past last: cursor = %d, want %d", d.cursor, last)
	}
}

func TestNavigation_Up(t *testing.T) {
	d := New(filetree.SortByType) // last option
	d, _ = d.Update(kp("up"))
	if d.cursor != 2 { // SortBySize index
		t.Errorf("up from last: cursor = %d, want 2", d.cursor)
	}
	// Up past first: stays at 0.
	d2 := New(filetree.SortByName)
	d2, _ = d2.Update(kp("up"))
	if d2.cursor != 0 {
		t.Errorf("up from first: cursor = %d, want 0", d2.cursor)
	}
}

func TestNavigation_JK(t *testing.T) {
	d := New(filetree.SortByName)
	d, _ = d.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if d.cursor != 1 {
		t.Errorf("j: cursor = %d, want 1", d.cursor)
	}
	d, _ = d.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if d.cursor != 0 {
		t.Errorf("k: cursor = %d, want 0", d.cursor)
	}
}

func TestRender_DoesNotPanic(t *testing.T) {
	for _, so := range filetree.AllSortOrders {
		d := New(so)
		_ = d.Render(80, 24)
	}
}

func TestRender_ZeroDimensions(t *testing.T) {
	d := New(filetree.SortByName)
	// Should not panic on very small sizes.
	_ = d.Render(10, 5)
}

func TestRender_ContainsAllLabels(t *testing.T) {
	d := New(filetree.SortByName)
	out := d.Render(80, 24)
	for _, so := range filetree.AllSortOrders {
		if !strings.Contains(out, so.Label()) {
			t.Errorf("render missing label %q", so.Label())
		}
	}
}

func TestChosen_DefaultIsInput(t *testing.T) {
	d := New(filetree.SortByTime)
	if d.Chosen() != filetree.SortByTime {
		t.Errorf("Chosen() = %v before any input, want SortByTime", d.Chosen())
	}
}
