package treeview

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bjcarnes/splorer/internal/filetree"
)

// makeTree creates the given files (relative to root) and any parent dirs.
func makeTree(t *testing.T, root string, files []string) {
	t.Helper()
	for _, rel := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdirall: %v", err)
		}
		if err := os.WriteFile(full, []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
	}
}

func TestNew_FlattensRecursively(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, []string{
		"a.txt",
		"sub/b.txt",
		"sub/inner/c.txt",
	})

	p := New(root, 80, 24)

	// Expect: sub/, sub/inner/, sub/inner/c.txt, sub/b.txt, a.txt
	// (dirs first at every level, case-insensitive sort).
	want := []string{"sub", "inner", "c.txt", "b.txt", "a.txt"}
	if len(p.rows) != len(want) {
		t.Fatalf("rows = %d, want %d (got %v)", len(p.rows), len(want), rowNames(p.rows))
	}
	for i, w := range want {
		if p.rows[i].name != w {
			t.Errorf("rows[%d] = %q, want %q (all: %v)", i, p.rows[i].name, w, rowNames(p.rows))
		}
	}
	if p.truncated {
		t.Error("truncated should be false on a small tree")
	}
}

func TestNew_TruncatesAtMaxEntries(t *testing.T) {
	root := t.TempDir()
	// MaxEntries+50 flat files is more than enough to trip the cap.
	files := make([]string, MaxEntries+50)
	for i := range files {
		files[i] = fmt.Sprintf("f%05d.txt", i)
	}
	makeTree(t, root, files)

	p := New(root, 80, 24)
	if !p.truncated {
		t.Error("expected truncated = true")
	}
	if len(p.rows) != MaxEntries {
		t.Errorf("rows = %d, want %d", len(p.rows), MaxEntries)
	}
}

func TestRender_ShowsTruncationWarning(t *testing.T) {
	root := t.TempDir()
	files := make([]string, MaxEntries+5)
	for i := range files {
		files[i] = fmt.Sprintf("f%05d.txt", i)
	}
	makeTree(t, root, files)

	p := New(root, 80, 24)
	out := p.Render()
	if !strings.Contains(out, "truncated") {
		t.Errorf("expected truncation warning in render; got:\n%s", out)
	}
}

func TestActivate_OpensFile(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, []string{"a.txt"})
	p := New(root, 80, 24)

	p, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter on a file should produce a command")
	}
	msg := cmd()
	open, ok := msg.(filetree.OpenFileMsg)
	if !ok {
		t.Fatalf("expected OpenFileMsg, got %T", msg)
	}
	if open.Path != filepath.Join(root, "a.txt") {
		t.Errorf("OpenFileMsg.Path = %q, want %q", open.Path, filepath.Join(root, "a.txt"))
	}
}

func TestActivate_OnDirIsNoOp(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, []string{"sub/x.txt"})
	p := New(root, 80, 24)

	// First row is the "sub" directory.
	if p.rows[0].name != "sub" || !p.rows[0].isDir {
		t.Fatalf("precondition: first row should be sub/, got %+v", p.rows[0])
	}
	_, cmd := p.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Errorf("Enter on a directory should be a no-op (no nav); got cmd %T", cmd())
	}
}

func TestEsc_Closes(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, []string{"a.txt"})
	p := New(root, 80, 24)
	p, _ = p.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !p.IsClosed() {
		t.Error("Esc should close the tree view")
	}
}

func TestQ_Closes(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, []string{"a.txt"})
	p := New(root, 80, 24)
	p, _ = p.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if !p.IsClosed() {
		t.Error("q should close the tree view")
	}
}

func TestRows_LastSiblingFlagged(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, []string{"a.txt", "b.txt", "c.txt"})
	p := New(root, 80, 24)

	if len(p.rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(p.rows))
	}
	if p.rows[0].isLast || p.rows[1].isLast {
		t.Errorf("only the last row should have isLast=true; got %+v", rowFlags(p.rows))
	}
	if !p.rows[2].isLast {
		t.Errorf("c.txt should be marked last; got %+v", rowFlags(p.rows))
	}
}

func TestRows_AncestorLastTracksParents(t *testing.T) {
	root := t.TempDir()
	// sub/ is the last entry; deep.txt is its only child.
	makeTree(t, root, []string{"a.txt", "sub/deep.txt"})
	p := New(root, 80, 24)

	// Order: sub/, deep.txt, a.txt
	wantNames := []string{"sub", "deep.txt", "a.txt"}
	for i, w := range wantNames {
		if p.rows[i].name != w {
			t.Fatalf("rows[%d] = %q, want %q (all: %v)", i, p.rows[i].name, w, rowNames(p.rows))
		}
	}
	// `deep.txt` lives one level inside `sub`. `sub` is NOT the last sibling
	// at depth 0 (a.txt comes after), so deep.txt's only ancestor flag
	// should be false → its indent column draws "│ ".
	if got := p.rows[1].ancestorLast; len(got) != 1 || got[0] {
		t.Errorf("deep.txt ancestorLast = %v, want [false]", got)
	}
}

func TestRender_DrawsTreeLinesAndIcons(t *testing.T) {
	root := t.TempDir()
	makeTree(t, root, []string{"sub/inner.txt", "z.txt"})
	p := New(root, 80, 24)
	out := p.Render()

	// Branch glyphs for non-root rows.
	if !strings.Contains(out, "├─") && !strings.Contains(out, "└─") {
		t.Errorf("expected tree branch glyphs in output:\n%s", out)
	}
	// The folder icon (📁) should render for the directory row.
	if !strings.Contains(out, "📁") {
		t.Errorf("expected folder icon in output:\n%s", out)
	}
	// A .txt file maps to the text icon (📝) per filetree.FileEntry.Icon.
	if !strings.Contains(out, "📝") {
		t.Errorf("expected text-file icon in output:\n%s", out)
	}
}

func rowFlags(rows []row) []bool {
	out := make([]bool, len(rows))
	for i, r := range rows {
		out[i] = r.isLast
	}
	return out
}

func rowNames(rows []row) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.name
	}
	return out
}
