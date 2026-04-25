// Package search provides a full-screen file-search view. The user types a
// filename or wildcard pattern; a background goroutine walks the directory
// tree and streams results back via a channel. Results are navigable with the
// same keys as the file tree.
package search

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/bjcarnes/splorer/internal/filetree"
)

// headerHeight is the number of fixed lines at the top of the view:
//
//	line 0 – "Find in: <rootDir>"
//	line 1 – "Pattern: <input field>"
//	line 2 – separator
const headerHeight = 3

// footerHeight is the number of fixed lines at the bottom:
//
//	line n-1 – separator
//	line n   – status / help bar
const footerHeight = 2

// searchSessionCounter is atomically incremented for each new search run so
// that stale resultBatchMsg messages from cancelled searches can be discarded.
var searchSessionCounter uint64

// Result is a single search hit.
type Result struct {
	RelPath  string // path relative to rootDir
	FullPath string // absolute path
	IsDir    bool
}

// NavigateDirMsg is emitted when the user activates a directory result.
// The parent model should navigate its file tree to this path and close the
// search view.
type NavigateDirMsg struct{ Path string }

// resultBatchMsg is sent from the background search goroutine to the Tea
// event loop. Each message carries the session ID of the search that produced
// it so stale messages can be safely ignored.
type resultBatchMsg struct {
	sessionID uint64
	results   []Result
	done      bool
}

// viewState tracks what the search view is currently showing.
type viewState int

const (
	stateInput     viewState = iota // user is typing a pattern
	stateSearching                  // background search in progress
	stateDone                       // search finished, showing final results
)

// Model is the search view component. It does not implement tea.Model
// directly; the parent app.Model owns it and calls Update / Render.
type Model struct {
	rootDir    string
	state      viewState
	closed     bool
	ignoreCase bool

	// text input
	input    string
	inputCur int // cursor position (rune index)

	// results list
	results    []Result
	listCursor int
	offset     int // first visible result index

	// terminal dimensions
	width  int
	height int

	// background search
	sessionID uint64
	cancel    context.CancelFunc
	resultsCh chan resultBatchMsg

	// double-click detection
	lastClick  time.Time
	lastClickY int // result index of last click
}

// New creates a Model ready to accept a search pattern for rootDir.
// Case-insensitive matching is on by default.
func New(rootDir string, width, height int) Model {
	return Model{
		rootDir:    rootDir,
		state:      stateInput,
		ignoreCase: true,
		width:      width,
		height:     height,
	}
}

// IsClosed reports whether the search view should be dismissed.
func (m Model) IsClosed() bool { return m.closed }

// waitForBatch returns a Cmd that blocks until the next batch arrives on ch.
// When ch is closed (goroutine finished or cancelled) it returns a done msg.
func waitForBatch(ch <-chan resultBatchMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			// Channel closed — synthesise a done signal with no session ID so
			// the model can discard it safely via the session ID check.
			return resultBatchMsg{done: true}
		}
		return msg
	}
}

// runSearch walks rootDir looking for entries whose name matches pattern and
// streams them to ch in batches of up to 100. It always closes ch on exit.
func runSearch(ctx context.Context, rootDir, pattern string, ignoreCase bool, sessionID uint64, ch chan<- resultBatchMsg) {
	defer close(ch)

	matchPattern := pattern
	if ignoreCase {
		matchPattern = strings.ToLower(pattern)
	}

	var batch []Result

	filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error { //nolint:errcheck
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil || path == rootDir {
			return nil
		}

		name := d.Name()
		checkName := name
		if ignoreCase {
			checkName = strings.ToLower(name)
		}
		matched, matchErr := filepath.Match(matchPattern, checkName)
		if matchErr != nil {
			// Invalid wildcard syntax — fall back to exact name comparison.
			if ignoreCase {
				matched = strings.EqualFold(name, pattern)
			} else {
				matched = name == pattern
			}
		}

		if matched {
			rel, _ := filepath.Rel(rootDir, path)
			batch = append(batch, Result{
				RelPath:  rel,
				FullPath: path,
				IsDir:    d.IsDir(),
			})
			if len(batch) >= 100 {
				select {
				case ch <- resultBatchMsg{sessionID: sessionID, results: batch}:
					batch = nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
		return nil
	})

	// Send remaining results as the final message.
	select {
	case ch <- resultBatchMsg{sessionID: sessionID, results: batch, done: true}:
	case <-ctx.Done():
		// Cancelled — defer close(ch) will unblock any pending waitForBatch.
	}
}

// Update processes a Bubble Tea message and returns the updated model and an
// optional command.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case resultBatchMsg:
		// Discard messages from a different (older) search session, or messages
		// that arrive after the search has already completed or been cancelled.
		if msg.sessionID != m.sessionID || m.state != stateSearching {
			return m, nil
		}
		m.results = append(m.results, msg.results...)
		if msg.done {
			sort.Slice(m.results, func(i, j int) bool {
				return m.results[i].FullPath < m.results[j].FullPath
			})
			m.state = stateDone
			m.cancel = nil
			m.resultsCh = nil
			return m, nil
		}
		return m, waitForBatch(m.resultsCh)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch m.state {
		case stateInput:
			return m.updateInput(msg)
		case stateSearching:
			return m.updateSearching(msg)
		case stateDone:
			return m.updateDone(msg)
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			return m.handleClick(int(msg.X), int(msg.Y))
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m = m.moveListCursor(-1)
		case tea.MouseWheelDown:
			m = m.moveListCursor(1)
		}
	}

	return m, nil
}

func (m Model) updateInput(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "alt+i":
		m.ignoreCase = !m.ignoreCase
		return m, nil
	case "enter":
		if strings.TrimSpace(m.input) == "" {
			return m, nil
		}
		return m.startSearch()
	case "esc":
		m.closed = true
		return m, nil
	case "backspace":
		if m.input == "" {
			// Empty input — treat backspace as "go back".
			m.closed = true
			return m, nil
		}
		m.input, m.inputCur = deleteRuneAt(m.input, m.inputCur)
	case "left":
		if m.inputCur > 0 {
			m.inputCur--
		}
	case "right":
		if m.inputCur < len([]rune(m.input)) {
			m.inputCur++
		}
	case "ctrl+a":
		m.inputCur = 0
	case "ctrl+e":
		m.inputCur = len([]rune(m.input))
	default:
		if msg.Text != "" {
			m.input = insertTextAt(m.input, m.inputCur, msg.Text)
			m.inputCur += len([]rune(msg.Text))
		}
	}
	return m, nil
}

func (m Model) updateSearching(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		// Cancel the running search and dismiss the view.
		if m.cancel != nil {
			m.cancel()
		}
		m.closed = true
		return m, nil
	case "up", "k":
		m = m.moveListCursor(-1)
	case "down", "j":
		m = m.moveListCursor(1)
	}
	return m, nil
}

func (m Model) updateDone(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		m.closed = true
		return m, nil
	case "up", "k":
		m = m.moveListCursor(-1)
	case "down", "j":
		m = m.moveListCursor(1)
	case "pgup":
		m = m.moveListCursor(-m.listHeight())
	case "pgdown":
		m = m.moveListCursor(m.listHeight())
	case "enter", "right", "l":
		return m.activateResult()
	}
	return m, nil
}

// startSearch launches a background goroutine and transitions to stateSearching.
func (m Model) startSearch() (Model, tea.Cmd) {
	if m.cancel != nil {
		m.cancel() // cancel any previously running search
	}

	id := atomic.AddUint64(&searchSessionCounter, 1)
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan resultBatchMsg)

	m.sessionID = id
	m.state = stateSearching
	m.results = nil
	m.listCursor = 0
	m.offset = 0
	m.cancel = cancel
	m.resultsCh = ch

	pattern := strings.TrimSpace(m.input)
	rootDir := m.rootDir
	ignoreCase := m.ignoreCase

	go runSearch(ctx, rootDir, pattern, ignoreCase, id, ch)

	return m, waitForBatch(ch)
}

// activateResult opens the currently selected result file or navigates into a
// selected result directory.
func (m Model) activateResult() (Model, tea.Cmd) {
	if len(m.results) == 0 {
		return m, nil
	}
	r := m.results[m.listCursor]
	if r.IsDir {
		path := r.FullPath
		return m, func() tea.Msg { return NavigateDirMsg{Path: path} }
	}
	path := r.FullPath
	return m, func() tea.Msg { return filetree.OpenFileMsg{Path: path} }
}

// handleClick processes a left mouse click at (x, y) in the search view's
// coordinate space (menu bar offset already applied by the parent).
func (m Model) handleClick(x, y int) (Model, tea.Cmd) {
	idx := y - headerHeight + m.offset
	now := time.Now()
	isDouble := idx == m.lastClickY && now.Sub(m.lastClick) < 500*time.Millisecond
	m.lastClick = now
	if idx >= 0 && idx < len(m.results) {
		m.lastClickY = idx
		m.listCursor = idx
		if isDouble && m.state != stateInput {
			return m.activateResult()
		}
	}
	return m, nil
}

// moveListCursor moves the result cursor by delta and adjusts the scroll offset.
func (m Model) moveListCursor(delta int) Model {
	m.listCursor += delta
	n := len(m.results)
	if m.listCursor < 0 {
		m.listCursor = 0
	}
	if n > 0 && m.listCursor >= n {
		m.listCursor = n - 1
	}
	lh := m.listHeight()
	if m.listCursor < m.offset {
		m.offset = m.listCursor
	}
	if m.listCursor >= m.offset+lh {
		m.offset = m.listCursor - lh + 1
	}
	return m
}

// listHeight returns the number of rows available for results.
func (m Model) listHeight() int {
	lh := m.height - headerHeight - footerHeight
	if lh < 1 {
		return 1
	}
	return lh
}

// Render produces the full string content for the search view.
func (m Model) Render() string {
	if m.width == 0 || m.height == 0 {
		return "Loading…"
	}

	headerStyle := lipgloss.NewStyle().Bold(true)
	sepStyle := lipgloss.NewStyle().Faint(true)
	dimStyle := lipgloss.NewStyle().Faint(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	cursorStyle := lipgloss.NewStyle().Reverse(true)
	dirStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Blue)

	sep := sepStyle.Render(strings.Repeat("─", m.width))

	var b strings.Builder

	// Line 0: root directory being searched.
	b.WriteString(headerStyle.Render(fmt.Sprintf(" Find in: %s", m.rootDir)))
	b.WriteRune('\n')

	// Line 1: pattern input + case-toggle indicator.
	const inputLabel = " Pattern: "
	toggleStr := fmt.Sprintf("  Alt+I Case: %s", caseLabel(m.ignoreCase))
	toggleRendered := dimStyle.Render(toggleStr)
	if m.state == stateInput {
		fieldWidth := m.width - lipgloss.Width(inputLabel) - lipgloss.Width(toggleStr) - 1
		if fieldWidth < 10 {
			fieldWidth = 10
		}
		field := renderInputField(m.input, m.inputCur, fieldWidth, cursorStyle)
		b.WriteString(inputLabel + field + " " + toggleRendered)
	} else {
		frozen := fmt.Sprintf("%s%s", inputLabel, m.input)
		gap := m.width - lipgloss.Width(frozen) - lipgloss.Width(toggleStr)
		if gap < 1 {
			gap = 1
		}
		b.WriteString(dimStyle.Render(frozen) + strings.Repeat(" ", gap) + toggleRendered)
	}
	b.WriteRune('\n')

	// Line 2: separator.
	b.WriteString(sep)
	b.WriteRune('\n')

	// Result rows.
	lh := m.listHeight()
	end := m.offset + lh
	if end > len(m.results) {
		end = len(m.results)
	}

	const cursorPrefixWidth = 2 // "▶ " or "  "
	nameWidth := m.width - cursorPrefixWidth
	if nameWidth < 8 {
		nameWidth = 8
	}

	for i := m.offset; i < end; i++ {
		r := m.results[i]
		selected := i == m.listCursor && m.state != stateInput

		prefix := "  "
		if selected {
			prefix = "▶ "
		}

		name := r.RelPath
		if r.IsDir {
			name += "/"
		}
		// Truncate long paths from the left so the filename stays visible.
		runes := []rune(name)
		if len(runes) > nameWidth {
			name = "…" + string(runes[len(runes)-nameWidth+1:])
		}

		var line string
		if selected {
			line = selectedStyle.Render(prefix + name)
		} else if r.IsDir {
			line = prefix + dirStyle.Render(name)
		} else {
			line = prefix + name
		}

		b.WriteString(line)
		b.WriteRune('\n')
	}

	// Fill any remaining rows so the footer stays at the bottom.
	rendered := end - m.offset
	for i := rendered; i < lh; i++ {
		b.WriteRune('\n')
	}

	// Bottom separator.
	b.WriteString(sep)
	b.WriteRune('\n')

	// Status / help bar.
	var leftStr, rightStr string
	switch m.state {
	case stateInput:
		leftStr = headerStyle.Render(" Type a filename or wildcard pattern, then Enter")
		rightStr = dimStyle.Render("Alt+I toggle case  Enter search  Esc/Backspace close")
	case stateSearching:
		leftStr = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Yellow).Render(
			fmt.Sprintf(" Searching… (%d found so far)", len(m.results)))
		rightStr = dimStyle.Render("Esc/Backspace cancel")
	case stateDone:
		leftStr = headerStyle.Render(fmt.Sprintf(" %d result(s)", len(m.results)))
		rightStr = dimStyle.Render("↑↓/jk navigate  Enter open  Esc/Backspace close")
	}

	gap := m.width - lipgloss.Width(leftStr) - lipgloss.Width(rightStr)
	if gap < 1 {
		gap = 1
	}
	b.WriteString(leftStr + strings.Repeat(" ", gap) + rightStr)

	return b.String()
}

// renderInputField renders a fixed-width text-input area with a cursor
// highlight at cursorPos.
func renderInputField(value string, cursorPos int, fieldWidth int, cursorSt lipgloss.Style) string {
	runes := []rune(value)

	// Scroll the window so the cursor is always visible.
	start := 0
	if cursorPos >= fieldWidth {
		start = cursorPos - fieldWidth + 1
	}
	end := start + fieldWidth
	if end > len(runes) {
		end = len(runes)
	}
	window := runes[start:end]
	relCursor := cursorPos - start

	var sb strings.Builder
	for i := 0; i < fieldWidth; i++ {
		var ch string
		if i < len(window) {
			ch = string(window[i])
		} else {
			ch = " "
		}
		if i == relCursor {
			sb.WriteString(cursorSt.Render(ch))
		} else {
			sb.WriteString(ch)
		}
	}
	return sb.String()
}

// insertTextAt inserts text into s at rune position pos.
func insertTextAt(s string, pos int, text string) string {
	runes := []rune(s)
	return string(runes[:pos]) + text + string(runes[pos:])
}

func caseLabel(ignoreCase bool) string {
	if ignoreCase {
		return "insensitive"
	}
	return "sensitive  "
}

// deleteRuneAt removes the rune immediately before pos and returns the updated
// string and adjusted cursor position.
func deleteRuneAt(s string, pos int) (string, int) {
	if pos <= 0 {
		return s, 0
	}
	runes := []rune(s)
	return string(runes[:pos-1]) + string(runes[pos:]), pos - 1
}
