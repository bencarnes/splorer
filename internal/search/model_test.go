package search

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// setupDir creates a temporary directory tree for tests.
// dirs and files are paths relative to the returned root.
func setupDir(t *testing.T, dirs []string, files []string) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	for _, f := range files {
		full := filepath.Join(root, f)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir parent of %s: %v", f, err)
		}
		if err := os.WriteFile(full, []byte("x"), 0644); err != nil {
			t.Fatalf("write %s: %v", f, err)
		}
	}
	return root
}

// drainSearch starts a search on m and drains all result messages until done,
// returning the final model.
func drainSearch(t *testing.T, m Model) Model {
	t.Helper()
	m, cmd := m.startSearch()
	for cmd != nil {
		msg := cmd()
		if msg == nil {
			break
		}
		m, cmd = m.Update(msg)
	}
	return m
}

// ── New / initial state ──────────────────────────────────────────────────────

func TestNew_InitialState(t *testing.T) {
	m := New("/some/dir", 80, 24)
	if m.state != stateInput {
		t.Errorf("initial state = %v, want stateInput", m.state)
	}
	if m.closed {
		t.Error("new model should not be closed")
	}
	if m.input != "" {
		t.Errorf("initial input = %q, want empty", m.input)
	}
	if m.rootDir != "/some/dir" {
		t.Errorf("rootDir = %q", m.rootDir)
	}
}

// ── Input state ─────────────────────────────────────────────────────────────

func TestInput_Typing(t *testing.T) {
	m := New("/d", 80, 24)
	m2, _ := m.Update(tea.KeyPressMsg{Text: "f"})
	m2, _ = m2.Update(tea.KeyPressMsg{Text: "o"})
	m2, _ = m2.Update(tea.KeyPressMsg{Text: "o"})
	if m2.input != "foo" {
		t.Errorf("input = %q, want %q", m2.input, "foo")
	}
	if m2.inputCur != 3 {
		t.Errorf("cursor = %d, want 3", m2.inputCur)
	}
}

func TestInput_BackspaceDeletesChar(t *testing.T) {
	m := New("/d", 80, 24)
	m.input = "foo"
	m.inputCur = 3
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace, Text: ""})
	if m2.input != "fo" {
		t.Errorf("after backspace input = %q, want %q", m2.input, "fo")
	}
	if m2.closed {
		t.Error("backspace on non-empty input should not close")
	}
}

func TestInput_BackspaceOnEmptyCloses(t *testing.T) {
	m := New("/d", 80, 24)
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if !m2.closed {
		t.Error("backspace on empty input should close")
	}
}

func TestInput_EscCloses(t *testing.T) {
	m := New("/d", 80, 24)
	m.input = "something"
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !m2.closed {
		t.Error("esc should close")
	}
}

func TestInput_EnterEmptyDoesNothing(t *testing.T) {
	m := New("/d", 80, 24)
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m2.state != stateInput {
		t.Errorf("enter on empty input changed state to %v", m2.state)
	}
}

func TestInput_EnterStartsSearch(t *testing.T) {
	root := setupDir(t, nil, []string{"main.go"})
	m := New(root, 80, 24)
	m.input = "main.go"
	m2, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m2.state != stateSearching {
		t.Errorf("state = %v, want stateSearching", m2.state)
	}
	if cmd == nil {
		t.Error("expected a command to be returned")
	}
}

func TestInput_CursorMovement(t *testing.T) {
	m := New("/d", 80, 24)
	m.input = "abc"
	m.inputCur = 3
	// Move left
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m2.inputCur != 2 {
		t.Errorf("after left cursor = %d, want 2", m2.inputCur)
	}
	// Move right back
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m3.inputCur != 3 {
		t.Errorf("after right cursor = %d, want 3", m3.inputCur)
	}
	// Left at position 0 stays at 0.
	m4 := m
	m4.inputCur = 0
	m5, _ := m4.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if m5.inputCur != 0 {
		t.Errorf("left at 0: cursor = %d, want 0", m5.inputCur)
	}
	// Right at end stays at end.
	m6, _ := m3.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if m6.inputCur != 3 {
		t.Errorf("right at end: cursor = %d, want 3", m6.inputCur)
	}
}

// ── Searching state ──────────────────────────────────────────────────────────

func TestSearching_EscCancelsAndCloses(t *testing.T) {
	root := setupDir(t, nil, []string{"a.go", "b.go"})
	m := New(root, 80, 24)
	m.input = "*.go"
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.state != stateSearching {
		t.Fatalf("expected stateSearching, got %v", m.state)
	}
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !m2.closed {
		t.Error("esc while searching should close")
	}
}

func TestSearching_BackspaceCancelsAndCloses(t *testing.T) {
	root := setupDir(t, nil, []string{"x.txt"})
	m := New(root, 80, 24)
	m.input = "x.txt"
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if !m2.closed {
		t.Error("backspace while searching should close")
	}
}

func TestSearching_CursorNavigation(t *testing.T) {
	root := setupDir(t, nil, nil)
	m := New(root, 80, 24)
	m.input = "*.go"
	m.state = stateSearching
	// Inject some results so there is something to navigate.
	m.results = []Result{
		{RelPath: "a.go", FullPath: filepath.Join(root, "a.go")},
		{RelPath: "b.go", FullPath: filepath.Join(root, "b.go")},
	}

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m2.listCursor != 1 {
		t.Errorf("down: cursor = %d, want 1", m2.listCursor)
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m3.listCursor != 0 {
		t.Errorf("up: cursor = %d, want 0", m3.listCursor)
	}
}

// ── Result batch messages ────────────────────────────────────────────────────

func TestResultBatch_AppendResults(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateSearching
	m.sessionID = 1
	m.resultsCh = make(chan resultBatchMsg)

	batch := []Result{
		{RelPath: "a.txt", FullPath: "/d/a.txt"},
		{RelPath: "b.txt", FullPath: "/d/b.txt"},
	}
	m2, cmd := m.Update(resultBatchMsg{sessionID: 1, results: batch, done: false})
	if len(m2.results) != 2 {
		t.Errorf("results count = %d, want 2", len(m2.results))
	}
	if m2.state != stateSearching {
		t.Errorf("state = %v, want stateSearching", m2.state)
	}
	if cmd == nil {
		t.Error("expected another waitForBatch command")
	}
}

func TestResultBatch_DoneTransitionsSortsResults(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateSearching
	m.sessionID = 1

	// Send an unsorted batch.
	batch := []Result{
		{RelPath: "z.txt", FullPath: "/d/z.txt"},
		{RelPath: "a.txt", FullPath: "/d/a.txt"},
		{RelPath: "m.txt", FullPath: "/d/m.txt"},
	}
	m2, cmd := m.Update(resultBatchMsg{sessionID: 1, results: batch, done: true})
	if m2.state != stateDone {
		t.Errorf("state = %v, want stateDone", m2.state)
	}
	if cmd != nil {
		t.Error("done batch should return nil command")
	}
	// Results should be sorted by FullPath.
	want := []string{"/d/a.txt", "/d/m.txt", "/d/z.txt"}
	for i, r := range m2.results {
		if r.FullPath != want[i] {
			t.Errorf("results[%d].FullPath = %q, want %q", i, r.FullPath, want[i])
		}
	}
}

func TestResultBatch_StaleSessionIDIgnored(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateSearching
	m.sessionID = 2

	m2, _ := m.Update(resultBatchMsg{sessionID: 1, results: []Result{{RelPath: "x.txt"}}, done: false})
	if len(m2.results) != 0 {
		t.Error("stale session batch should be ignored")
	}
}

func TestResultBatch_IgnoredWhenDone(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateDone
	m.sessionID = 1

	m2, _ := m.Update(resultBatchMsg{sessionID: 1, results: []Result{{RelPath: "x.txt"}}, done: false})
	if len(m2.results) != 0 {
		t.Error("batch received in stateDone should be ignored")
	}
}

// ── Done state ───────────────────────────────────────────────────────────────

func TestDone_EscCloses(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateDone
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !m2.closed {
		t.Error("esc in done state should close")
	}
}

func TestDone_BackspaceCloses(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateDone
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if !m2.closed {
		t.Error("backspace in done state should close")
	}
}

func TestDone_CursorNavigation(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateDone
	m.height = 24
	m.results = []Result{
		{RelPath: "a.txt", FullPath: "/d/a.txt"},
		{RelPath: "b.txt", FullPath: "/d/b.txt"},
		{RelPath: "c.txt", FullPath: "/d/c.txt"},
	}

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m2.listCursor != 1 {
		t.Errorf("down: cursor = %d, want 1", m2.listCursor)
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m3.listCursor != 0 {
		t.Errorf("up: cursor = %d, want 0", m3.listCursor)
	}
}

func TestDone_CursorBounds(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateDone
	m.results = []Result{
		{RelPath: "a.txt", FullPath: "/d/a.txt"},
	}
	m.listCursor = 0

	// Up at top: stays at 0.
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m2.listCursor != 0 {
		t.Errorf("cursor above top = %d, want 0", m2.listCursor)
	}
	// Down at bottom: stays at last.
	m3, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m3.listCursor != 0 {
		t.Errorf("cursor below bottom = %d, want 0", m3.listCursor)
	}
}

func TestDone_ActivateFile(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateDone
	m.results = []Result{
		{RelPath: "a.txt", FullPath: "/d/a.txt", IsDir: false},
	}
	m.listCursor = 0

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command for file activation")
	}
	msg := cmd()
	// Should emit filetree.OpenFileMsg — we can only check the type string.
	if msg == nil {
		t.Fatal("command returned nil message")
	}
}

func TestDone_ActivateDir(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateDone
	m.results = []Result{
		{RelPath: "sub", FullPath: "/d/sub", IsDir: true},
	}
	m.listCursor = 0

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command for directory activation")
	}
	msg := cmd()
	navMsg, ok := msg.(NavigateDirMsg)
	if !ok {
		t.Fatalf("expected NavigateDirMsg, got %T", msg)
	}
	if navMsg.Path != "/d/sub" {
		t.Errorf("NavigateDirMsg.Path = %q, want %q", navMsg.Path, "/d/sub")
	}
}

// ── Mouse handling ───────────────────────────────────────────────────────────

func TestMouseClick_SelectsResult(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateDone
	m.results = []Result{
		{RelPath: "a.txt", FullPath: "/d/a.txt"},
		{RelPath: "b.txt", FullPath: "/d/b.txt"},
	}

	// Click on the second result row (headerHeight + 1).
	m2, _ := m.Update(tea.MouseClickMsg{
		X:      0,
		Y:      headerHeight + 1,
		Button: tea.MouseLeft,
	})
	if m2.listCursor != 1 {
		t.Errorf("click: cursor = %d, want 1", m2.listCursor)
	}
}

func TestMouseDoubleClick_ActivatesResult(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateDone
	m.results = []Result{
		{RelPath: "a.txt", FullPath: "/d/a.txt", IsDir: false},
	}

	clickY := headerHeight + 0
	// First click: selects.
	m.lastClick = time.Time{}
	m2, _ := m.Update(tea.MouseClickMsg{X: 0, Y: clickY, Button: tea.MouseLeft})
	if m2.listCursor != 0 {
		t.Errorf("first click cursor = %d, want 0", m2.listCursor)
	}

	// Second click within 500ms: activates.
	m2.lastClick = time.Now().Add(-100 * time.Millisecond)
	_, cmd := m2.Update(tea.MouseClickMsg{X: 0, Y: clickY, Button: tea.MouseLeft})
	if cmd == nil {
		t.Error("double-click should emit a command")
	}
}

func TestMouseWheel_ScrollsResults(t *testing.T) {
	m := New("/d", 80, 24)
	m.state = stateDone
	m.results = []Result{
		{RelPath: "a.txt", FullPath: "/d/a.txt"},
		{RelPath: "b.txt", FullPath: "/d/b.txt"},
	}

	m2, _ := m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m2.listCursor != 1 {
		t.Errorf("wheel down: cursor = %d, want 1", m2.listCursor)
	}
	m3, _ := m2.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if m3.listCursor != 0 {
		t.Errorf("wheel up: cursor = %d, want 0", m3.listCursor)
	}
}

// ── End-to-end search ────────────────────────────────────────────────────────

func TestSearch_ExactName(t *testing.T) {
	root := setupDir(t, nil, []string{"main.go", "util.go", "README.md"})
	m := New(root, 80, 24)
	m.input = "main.go"
	m = drainSearch(t, m)

	if m.state != stateDone {
		t.Fatalf("state = %v, want stateDone", m.state)
	}
	if len(m.results) != 1 {
		t.Fatalf("result count = %d, want 1", len(m.results))
	}
	if m.results[0].RelPath != "main.go" {
		t.Errorf("result = %q, want %q", m.results[0].RelPath, "main.go")
	}
}

func TestSearch_WildcardPattern(t *testing.T) {
	root := setupDir(t, nil, []string{"a.go", "b.go", "c.txt"})
	m := New(root, 80, 24)
	m.input = "*.go"
	m = drainSearch(t, m)

	if m.state != stateDone {
		t.Fatalf("state = %v, want stateDone", m.state)
	}
	if len(m.results) != 2 {
		t.Fatalf("result count = %d, want 2; results: %v", len(m.results), m.results)
	}
}

func TestSearch_Recursive(t *testing.T) {
	root := setupDir(t,
		[]string{"sub"},
		[]string{"top.go", "sub/nested.go"},
	)
	m := New(root, 80, 24)
	m.input = "*.go"
	m = drainSearch(t, m)

	if len(m.results) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(m.results), m.results)
	}
}

func TestSearch_SortedByFullPath(t *testing.T) {
	root := setupDir(t, nil, []string{"z.go", "a.go", "m.go"})
	m := New(root, 80, 24)
	m.input = "*.go"
	m = drainSearch(t, m)

	for i := 1; i < len(m.results); i++ {
		if m.results[i].FullPath < m.results[i-1].FullPath {
			t.Errorf("results not sorted: [%d]=%q > [%d]=%q",
				i-1, m.results[i-1].FullPath, i, m.results[i].FullPath)
		}
	}
}

func TestSearch_NoMatch(t *testing.T) {
	root := setupDir(t, nil, []string{"readme.txt"})
	m := New(root, 80, 24)
	m.input = "*.go"
	m = drainSearch(t, m)

	if m.state != stateDone {
		t.Fatalf("state = %v, want stateDone", m.state)
	}
	if len(m.results) != 0 {
		t.Errorf("expected 0 results, got %d", len(m.results))
	}
}

func TestSearch_RelativePathsCorrect(t *testing.T) {
	root := setupDir(t,
		[]string{"pkg"},
		[]string{"pkg/foo.go"},
	)
	m := New(root, 80, 24)
	m.input = "foo.go"
	m = drainSearch(t, m)

	if len(m.results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(m.results))
	}
	want := filepath.Join("pkg", "foo.go")
	if m.results[0].RelPath != want {
		t.Errorf("RelPath = %q, want %q", m.results[0].RelPath, want)
	}
}

func TestSearch_DirectoriesIncluded(t *testing.T) {
	root := setupDir(t, []string{"mydir"}, nil)
	m := New(root, 80, 24)
	m.input = "mydir"
	m = drainSearch(t, m)

	if len(m.results) != 1 {
		t.Fatalf("expected 1 result (directory), got %d", len(m.results))
	}
	if !m.results[0].IsDir {
		t.Error("result should be marked as directory")
	}
}

// ── Case sensitivity ─────────────────────────────────────────────────────────

func TestNew_DefaultCaseInsensitive(t *testing.T) {
	m := New("/d", 80, 24)
	if !m.ignoreCase {
		t.Error("new model should default to case-insensitive")
	}
}

func TestAltI_TogglesCase(t *testing.T) {
	m := New("/d", 80, 24)
	if !m.ignoreCase {
		t.Fatal("precondition: should start case-insensitive")
	}
	m2, _ := m.Update(tea.KeyPressMsg{Code: 'i', Mod: tea.ModAlt})
	if m2.ignoreCase {
		t.Error("Alt+I should switch to case-sensitive")
	}
	m3, _ := m2.Update(tea.KeyPressMsg{Code: 'i', Mod: tea.ModAlt})
	if !m3.ignoreCase {
		t.Error("Alt+I again should switch back to case-insensitive")
	}
}

func TestSearch_CaseInsensitiveMatchesUpperAndLower(t *testing.T) {
	root := setupDir(t, nil, []string{"Hello.go", "world.go"})
	m := New(root, 80, 24)
	m.input = "hello.go" // lowercase pattern, uppercase filename
	m.ignoreCase = true
	m = drainSearch(t, m)

	if len(m.results) != 1 {
		t.Fatalf("expected 1 result, got %d: %v", len(m.results), m.results)
	}
	if m.results[0].RelPath != "Hello.go" {
		t.Errorf("result = %q, want Hello.go", m.results[0].RelPath)
	}
}

func TestSearch_CaseInsensitiveWildcard(t *testing.T) {
	root := setupDir(t, nil, []string{"Main.Go", "util.go"})
	m := New(root, 80, 24)
	m.input = "*.go" // lowercase wildcard should match mixed-case filenames
	m.ignoreCase = true
	m = drainSearch(t, m)

	if len(m.results) != 2 {
		t.Fatalf("expected 2 results, got %d: %v", len(m.results), m.results)
	}
}

func TestSearch_CaseSensitiveDoesNotMatchWrongCase(t *testing.T) {
	root := setupDir(t, nil, []string{"Hello.go", "hello.go"})
	m := New(root, 80, 24)
	m.input = "hello.go"
	m.ignoreCase = false
	m = drainSearch(t, m)

	if len(m.results) != 1 {
		t.Fatalf("expected 1 result (exact case only), got %d: %v", len(m.results), m.results)
	}
	if m.results[0].RelPath != "hello.go" {
		t.Errorf("result = %q, want hello.go", m.results[0].RelPath)
	}
}

// ── Render smoke test ────────────────────────────────────────────────────────

func TestRender_DoesNotPanic(t *testing.T) {
	m := New("/d", 80, 24)
	// stateInput
	_ = m.Render()

	m.state = stateSearching
	m.input = "*.go"
	_ = m.Render()

	m.state = stateDone
	m.results = []Result{{RelPath: "a.go", FullPath: "/d/a.go"}}
	_ = m.Render()
}

func TestRender_ZeroDimensions(t *testing.T) {
	m := New("/d", 0, 0)
	out := m.Render()
	if out != "Loading…" {
		t.Errorf("zero-size render = %q, want %q", out, "Loading…")
	}
}

// ── IsClosed ─────────────────────────────────────────────────────────────────

func TestIsClosed_FalseByDefault(t *testing.T) {
	m := New("/d", 80, 24)
	if m.IsClosed() {
		t.Error("new model should not be closed")
	}
}

// ── WindowSizeMsg ────────────────────────────────────────────────────────────

func TestWindowSizeMsg_UpdatesDimensions(t *testing.T) {
	m := New("/d", 80, 24)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if m2.width != 120 || m2.height != 40 {
		t.Errorf("dimensions = %dx%d, want 120x40", m2.width, m2.height)
	}
}
