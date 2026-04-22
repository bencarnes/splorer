package bookmarks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bjcarnes/splorer/internal/filetree"
)

// keyPress builds a KeyPressMsg for test input.
func keyPress(key string) tea.KeyPressMsg {
	switch key {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEsc}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "delete":
		return tea.KeyPressMsg{Code: tea.KeyDelete}
	default:
		runes := []rune(key)
		if len(runes) == 1 {
			return tea.KeyPressMsg{Code: runes[0], Text: key}
		}
		return tea.KeyPressMsg{}
	}
}

// sampleBookmarks returns a set of bookmarks with fake paths for key-handling tests.
func sampleBookmarks() []Bookmark {
	return []Bookmark{
		{Name: "Alpha", Path: "/fake/alpha"},
		{Name: "Beta", Path: "/fake/beta"},
		{Name: "Gamma", Path: "/fake/gamma"},
	}
}

// ── Initial state ─────────────────────────────────────────────────────────────

func TestPage_InitialState(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	if p.IsClosed() {
		t.Error("new page should not be closed")
	}
	if p.state != stateList {
		t.Errorf("initial state = %v, want stateList", p.state)
	}
	if p.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", p.cursor)
	}
	if len(p.bookmarks) != 3 {
		t.Errorf("bookmark count = %d, want 3", len(p.bookmarks))
	}
}

func TestPage_NewPage_CopiesBookmarks(t *testing.T) {
	original := sampleBookmarks()
	p := NewPage(original, 80, 24)
	// Mutating the source slice should not affect the page.
	original[0].Name = "CHANGED"
	if p.bookmarks[0].Name == "CHANGED" {
		t.Error("NewPage should copy the bookmark slice, not reference it")
	}
}

// ── Close ─────────────────────────────────────────────────────────────────────

func TestPage_EscCloses(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	p, _ = p.Update(keyPress("esc"))
	if !p.IsClosed() {
		t.Error("esc should close page")
	}
}

func TestPage_BackspaceCloses(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	p, _ = p.Update(keyPress("backspace"))
	if !p.IsClosed() {
		t.Error("backspace should close page")
	}
}

// ── Cursor navigation ─────────────────────────────────────────────────────────

func TestPage_CursorNavigation(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)

	p, _ = p.Update(keyPress("down"))
	if p.cursor != 1 {
		t.Errorf("down: cursor = %d, want 1", p.cursor)
	}

	p, _ = p.Update(keyPress("down"))
	if p.cursor != 2 {
		t.Errorf("down: cursor = %d, want 2", p.cursor)
	}

	// Past the last entry: stays at last.
	p, _ = p.Update(keyPress("down"))
	if p.cursor != 2 {
		t.Errorf("down past end: cursor = %d, want 2", p.cursor)
	}

	p, _ = p.Update(keyPress("up"))
	if p.cursor != 1 {
		t.Errorf("up: cursor = %d, want 1", p.cursor)
	}

	// Past the first entry: stays at 0.
	p, _ = p.Update(keyPress("up"))
	p, _ = p.Update(keyPress("up"))
	if p.cursor != 0 {
		t.Errorf("up past start: cursor = %d, want 0", p.cursor)
	}
}

func TestPage_JKNavigation(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)

	p, _ = p.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if p.cursor != 1 {
		t.Errorf("j: cursor = %d, want 1", p.cursor)
	}

	p, _ = p.Update(tea.KeyPressMsg{Code: 'k', Text: "k"})
	if p.cursor != 0 {
		t.Errorf("k: cursor = %d, want 0", p.cursor)
	}
}

// ── Delete dialog ─────────────────────────────────────────────────────────────

func TestPage_DeleteKey_TransitionsToDeleteState(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	p, _ = p.Update(keyPress("delete"))
	if p.state != stateDelete {
		t.Errorf("delete key: state = %v, want stateDelete", p.state)
	}
}

func TestPage_DeleteKey_NoBookmarks_NoTransition(t *testing.T) {
	p := NewPage(nil, 80, 24)
	p, _ = p.Update(keyPress("delete"))
	if p.state != stateList {
		t.Errorf("delete on empty list: state = %v, want stateList", p.state)
	}
}

func TestPage_DeleteConfirm_Y(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	p, _ = p.Update(keyPress("delete")) // enter delete dialog
	if p.state != stateDelete {
		t.Fatalf("expected stateDelete")
	}
	p, _ = p.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if p.state != stateList {
		t.Errorf("after y: state = %v, want stateList", p.state)
	}
	if len(p.bookmarks) != 2 {
		t.Errorf("bookmark count = %d, want 2", len(p.bookmarks))
	}
	// Alpha should be gone (cursor was at 0).
	for _, bm := range p.bookmarks {
		if bm.Name == "Alpha" {
			t.Error("Alpha should have been deleted")
		}
	}
}

func TestPage_DeleteConfirm_YUppercase(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	p, _ = p.Update(keyPress("delete"))
	p, _ = p.Update(tea.KeyPressMsg{Code: 'Y', Text: "Y"})
	if len(p.bookmarks) != 2 {
		t.Errorf("Y should also confirm deletion, count = %d", len(p.bookmarks))
	}
}

func TestPage_DeleteConfirm_N_Cancels(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	p, _ = p.Update(keyPress("delete"))
	p, _ = p.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if p.state != stateList {
		t.Errorf("after n: state = %v, want stateList", p.state)
	}
	if len(p.bookmarks) != 3 {
		t.Errorf("n should cancel; bookmark count = %d, want 3", len(p.bookmarks))
	}
}

func TestPage_DeleteConfirm_Esc_Cancels(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	p, _ = p.Update(keyPress("delete"))
	p, _ = p.Update(keyPress("esc"))
	if p.state != stateList {
		t.Errorf("after esc: state = %v, want stateList", p.state)
	}
	if len(p.bookmarks) != 3 {
		t.Errorf("esc should cancel; bookmark count = %d, want 3", len(p.bookmarks))
	}
	// The page itself should NOT be closed (esc only cancels the dialog).
	if p.IsClosed() {
		t.Error("esc in delete state should cancel dialog, not close page")
	}
}

func TestPage_DeleteLast_CursorClamped(t *testing.T) {
	p := NewPage([]Bookmark{{Name: "Only", Path: "/only"}}, 80, 24)
	p, _ = p.Update(keyPress("delete"))
	p, _ = p.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if len(p.bookmarks) != 0 {
		t.Errorf("expected 0 bookmarks after deleting only one, got %d", len(p.bookmarks))
	}
	if p.cursor != 0 {
		t.Errorf("cursor after deleting last = %d, want 0", p.cursor)
	}
}

func TestPage_DeleteMiddle_CursorAdjusted(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	// Navigate to last entry (index 2), then delete.
	p, _ = p.Update(keyPress("down"))
	p, _ = p.Update(keyPress("down"))
	if p.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", p.cursor)
	}
	p, _ = p.Update(keyPress("delete"))
	p, _ = p.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if len(p.bookmarks) != 2 {
		t.Fatalf("expected 2 bookmarks, got %d", len(p.bookmarks))
	}
	// Cursor should clamp to the new last index.
	if p.cursor != 1 {
		t.Errorf("cursor after deleting last = %d, want 1", p.cursor)
	}
}

// ── Activation ────────────────────────────────────────────────────────────────

func TestPage_ActivateDir_EmitsNavigateDirMsg(t *testing.T) {
	dir := t.TempDir()
	p := NewPage([]Bookmark{{Name: "Temp", Path: dir}}, 80, 24)

	_, cmd := p.Update(keyPress("enter"))
	if cmd == nil {
		t.Fatal("enter on directory bookmark should emit a command")
	}
	msg := cmd()
	navMsg, ok := msg.(NavigateDirMsg)
	if !ok {
		t.Fatalf("expected NavigateDirMsg, got %T", msg)
	}
	if navMsg.Path != dir {
		t.Errorf("NavigateDirMsg.Path = %q, want %q", navMsg.Path, dir)
	}
	if !p.IsClosed() {
		// IsClosed is evaluated on the returned p, not the original.
		// The page's closed flag is set inside activate(). Re-check.
	}
}

func TestPage_ActivateDir_ClosesPage(t *testing.T) {
	dir := t.TempDir()
	p := NewPage([]Bookmark{{Name: "Temp", Path: dir}}, 80, 24)
	p, _ = p.Update(keyPress("enter"))
	if !p.IsClosed() {
		t.Error("activating a directory bookmark should close the page")
	}
}

func TestPage_ActivateFile_EmitsOpenFileMsg(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	p := NewPage([]Bookmark{{Name: "TestFile", Path: filePath}}, 80, 24)
	_, cmd := p.Update(keyPress("enter"))
	if cmd == nil {
		t.Fatal("enter on file bookmark should emit a command")
	}
	msg := cmd()
	openMsg, ok := msg.(filetree.OpenFileMsg)
	if !ok {
		t.Fatalf("expected filetree.OpenFileMsg, got %T", msg)
	}
	if openMsg.Path != filePath {
		t.Errorf("OpenFileMsg.Path = %q, want %q", openMsg.Path, filePath)
	}
}

func TestPage_ActivateFile_DoesNotClosePage(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	p := NewPage([]Bookmark{{Name: "TestFile", Path: filePath}}, 80, 24)
	p, _ = p.Update(keyPress("enter"))
	if p.IsClosed() {
		t.Error("activating a file bookmark should not close the page")
	}
}

func TestPage_ActivateNonExistentPath_NoCommand(t *testing.T) {
	p := NewPage([]Bookmark{{Name: "Gone", Path: "/does/not/exist/ever"}}, 80, 24)
	_, cmd := p.Update(keyPress("enter"))
	if cmd != nil {
		t.Error("activating a non-existent path should not emit a command")
	}
}

func TestPage_ActivateEmpty_NoCommand(t *testing.T) {
	p := NewPage(nil, 80, 24)
	_, cmd := p.Update(keyPress("enter"))
	if cmd != nil {
		t.Error("enter on empty list should not emit a command")
	}
}

// ── Mouse ─────────────────────────────────────────────────────────────────────

func TestPage_MouseClick_SelectsRow(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)

	// Click on the second result row (headerHeight + 1).
	p, _ = p.Update(tea.MouseClickMsg{
		X:      0,
		Y:      pageHeaderHeight + 1,
		Button: tea.MouseLeft,
	})
	if p.cursor != 1 {
		t.Errorf("click: cursor = %d, want 1", p.cursor)
	}
}

func TestPage_MouseDoubleClick_Activates(t *testing.T) {
	dir := t.TempDir()
	p := NewPage([]Bookmark{{Name: "Temp", Path: dir}}, 80, 24)

	clickY := pageHeaderHeight + 0
	// First click: selects.
	p.lastClick = time.Time{}
	p, _ = p.Update(tea.MouseClickMsg{X: 0, Y: clickY, Button: tea.MouseLeft})
	if p.cursor != 0 {
		t.Fatalf("first click cursor = %d, want 0", p.cursor)
	}

	// Second click within 500ms: activates.
	p.lastClick = time.Now().Add(-100 * time.Millisecond)
	_, cmd := p.Update(tea.MouseClickMsg{X: 0, Y: clickY, Button: tea.MouseLeft})
	if cmd == nil {
		t.Error("double-click should emit a command")
	}
}

func TestPage_MouseWheel_Scrolls(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)

	p, _ = p.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if p.cursor != 1 {
		t.Errorf("wheel down: cursor = %d, want 1", p.cursor)
	}
	p, _ = p.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if p.cursor != 0 {
		t.Errorf("wheel up: cursor = %d, want 0", p.cursor)
	}
}

// ── Bookmarks() accessor ──────────────────────────────────────────────────────

func TestPage_Bookmarks_ReturnsCopy(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	got := p.Bookmarks()
	got[0].Name = "CHANGED"
	if p.bookmarks[0].Name == "CHANGED" {
		t.Error("Bookmarks() should return a copy, not a reference")
	}
}

// ── WindowSizeMsg ─────────────────────────────────────────────────────────────

func TestPage_WindowSizeMsg_UpdatesDimensions(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	p, _ = p.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if p.width != 120 || p.height != 40 {
		t.Errorf("dimensions = %dx%d, want 120x40", p.width, p.height)
	}
}

// ── Render ────────────────────────────────────────────────────────────────────

func TestPage_Render_DoesNotPanic(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	_ = p.Render()

	// Delete state
	p2 := NewPage(sampleBookmarks(), 80, 24)
	p2, _ = p2.Update(keyPress("delete"))
	_ = p2.Render()

	// Empty list
	p3 := NewPage(nil, 80, 24)
	_ = p3.Render()
}

func TestPage_Render_ZeroDimensions(t *testing.T) {
	p := NewPage(sampleBookmarks(), 0, 0)
	out := p.Render()
	if out != "Loading…" {
		t.Errorf("zero-size render = %q, want %q", out, "Loading…")
	}
}

func TestPage_Render_ShowsBookmarkNames(t *testing.T) {
	p := NewPage(sampleBookmarks(), 80, 24)
	out := p.Render()
	for _, bm := range sampleBookmarks() {
		if !strings.Contains(out, bm.Name) {
			t.Errorf("render should contain bookmark name %q", bm.Name)
		}
	}
}

func TestPage_Render_DeleteDialog_ShowsBookmarkName(t *testing.T) {
	bm := Bookmark{Name: "My Important Dir", Path: "/some/path"}
	p := NewPage([]Bookmark{bm}, 80, 24)
	p, _ = p.Update(keyPress("delete"))
	out := p.Render()
	if !strings.Contains(out, "My Important Dir") {
		t.Error("delete dialog render should show the bookmark name being deleted")
	}
}
