// Package contentsearch provides a full-screen view for searching file
// contents under a given root directory. Compared to internal/search (which
// matches filenames with wildcards), this view:
//
//   - scans lines inside files, not filenames;
//   - supports both exact-substring and stdlib-regex modes;
//   - can be case-sensitive or case-insensitive (Alt+I toggle);
//   - can be restricted to files with a given set of extensions;
//   - skips symlinks, files over 10 MB, and files that look binary (NUL byte
//     in the first 8 KB);
//   - streams results in batches from a cancellable background goroutine,
//     using the same session-ID guard as the name search.
package contentsearch

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/bjcarnes/splorer/internal/filetree"
)

// headerHeight is the number of fixed lines at the top of the view:
//
//	line 0 – "Find content in: <rootDir>"
//	line 1 – "Pattern:  [field]            Alt+R Regex: … · Alt+I Case: …"
//	line 2 – "Ext:      [field]            (comma-separated, empty = all)"
//	line 3 – separator
const headerHeight = 4

// footerHeight is the number of fixed lines at the bottom (separator +
// status/help row).
const footerHeight = 2

// sessionCounter is atomically incremented for each new search run so stale
// resultBatchMsgs from cancelled searches can be discarded.
var sessionCounter uint64

// Result is a single matched line in some file.
type Result struct {
	RelPath  string // path relative to rootDir
	FullPath string // absolute path
	LineNum  int    // 1-based line number within the file
	LineText string // the matched line's text
}

// resultBatchMsg carries a batch of results from the background walker back
// to the Tea event loop. The sessionID guards against stale messages from a
// cancelled previous search.
type resultBatchMsg struct {
	sessionID uint64
	results   []Result
	done      bool
}

// viewState tracks what the view is showing.
type viewState int

const (
	stateInput     viewState = iota // user is typing pattern / ext
	stateSearching                  // walker goroutine running
	stateDone                       // walker finished
)

// focusField is which text field currently has the input cursor.
type focusField int

const (
	focusPattern focusField = iota
	focusExtensions
)

// Model is the content-search view. Same shape as internal/search.Model but
// with two text fields and two boolean toggles.
type Model struct {
	rootDir string
	state   viewState
	closed  bool

	// inputs
	pattern    string
	patternCur int
	extensions string
	extCur     int
	focus      focusField

	// options
	mode       Mode
	ignoreCase bool

	// error surface (e.g. invalid regex), displayed in the status bar
	// while in stateInput. Cleared on the next edit.
	errMsg string

	// results
	results    []Result
	listCursor int
	offset     int

	// terminal
	width  int
	height int

	// background search
	sessionID uint64
	cancel    context.CancelFunc
	resultsCh chan resultBatchMsg

	// double-click detection on result rows
	lastClick  time.Time
	lastClickY int
}

// New creates a Model ready for user input. The view starts in stateInput
// focused on the pattern field.
func New(rootDir string, width, height int) Model {
	return Model{
		rootDir: rootDir,
		state:   stateInput,
		focus:   focusPattern,
		width:   width,
		height:  height,
	}
}

// IsClosed reports whether the view should be dismissed.
func (m Model) IsClosed() bool { return m.closed }

// waitForBatch returns a Cmd that blocks until the next batch arrives on ch.
func waitForBatch(ch <-chan resultBatchMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return resultBatchMsg{done: true}
		}
		return msg
	}
}

// Update processes a Bubble Tea message.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case resultBatchMsg:
		if msg.sessionID != m.sessionID || m.state != stateSearching {
			return m, nil
		}
		m.results = append(m.results, msg.results...)
		if msg.done {
			m.state = stateDone
			m.cancel = nil
			m.resultsCh = nil
			return m, nil
		}
		return m, waitForBatch(m.resultsCh)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.PasteMsg:
		// Bracketed paste: insert the whole payload into the focused text
		// field at the current cursor. Pastes outside stateInput are
		// ignored — there's no editable field to receive them.
		if m.state != stateInput {
			return m, nil
		}
		return m.insertPasted(msg.Content), nil

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
	// Always-active toggles and navigation, regardless of which field has focus.
	switch msg.String() {
	case "alt+r":
		if m.mode == ModeExact {
			m.mode = ModeRegex
		} else {
			m.mode = ModeExact
		}
		m.errMsg = ""
		return m, nil
	case "alt+i":
		m.ignoreCase = !m.ignoreCase
		m.errMsg = ""
		return m, nil
	case "tab":
		m.focus = nextFocus(m.focus)
		return m, nil
	case "shift+tab":
		m.focus = prevFocus(m.focus)
		return m, nil
	case "enter":
		if strings.TrimSpace(m.pattern) == "" {
			return m, nil
		}
		return m.startSearch()
	case "esc":
		m.closed = true
		return m, nil
	}

	// Field-local editing. Each field owns its text and cursor position.
	switch m.focus {
	case focusPattern:
		m.pattern, m.patternCur = editField(m.pattern, m.patternCur, msg)
		// Empty-pattern backspace: close the view, matching the name-search convention.
		if msg.String() == "backspace" && m.pattern == "" && m.patternCur == 0 {
			m.closed = true
		}
	case focusExtensions:
		m.extensions, m.extCur = editField(m.extensions, m.extCur, msg)
	}
	m.errMsg = ""
	return m, nil
}

func (m Model) updateSearching(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
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

// startSearch validates inputs, launches the walker, and transitions to
// stateSearching. On validation failure it stays in stateInput and records
// an error message.
func (m Model) startSearch() (Model, tea.Cmd) {
	if m.cancel != nil {
		m.cancel()
	}

	opts := Options{
		Pattern:    m.pattern,
		Mode:       m.mode,
		IgnoreCase: m.ignoreCase,
		Extensions: m.extensions,
	}
	matcher, err := buildMatcher(opts)
	if err != nil {
		m.errMsg = err.Error()
		return m, nil
	}

	id := atomic.AddUint64(&sessionCounter, 1)
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan resultBatchMsg)

	m.sessionID = id
	m.state = stateSearching
	m.results = nil
	m.listCursor = 0
	m.offset = 0
	m.cancel = cancel
	m.resultsCh = ch
	m.errMsg = ""

	rootDir := m.rootDir
	go runContentSearch(ctx, rootDir, opts, matcher, id, ch)

	return m, waitForBatch(ch)
}

func (m Model) activateResult() (Model, tea.Cmd) {
	if len(m.results) == 0 {
		return m, nil
	}
	r := m.results[m.listCursor]
	path := r.FullPath
	return m, func() tea.Msg { return filetree.OpenFileMsg{Path: path} }
}

func (m Model) handleClick(x, y int) (Model, tea.Cmd) {
	if m.state == stateInput {
		// Simple field-targeting: clicks on row 1 focus the pattern field,
		// clicks on row 2 focus the extensions field.
		_ = x
		switch y {
		case 1:
			m.focus = focusPattern
		case 2:
			m.focus = focusExtensions
		}
		return m, nil
	}

	idx := y - headerHeight + m.offset
	now := time.Now()
	isDouble := idx == m.lastClickY && now.Sub(m.lastClick) < 500*time.Millisecond
	m.lastClick = now
	if idx >= 0 && idx < len(m.results) {
		m.lastClickY = idx
		m.listCursor = idx
		if isDouble {
			return m.activateResult()
		}
	}
	return m, nil
}

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

func (m Model) listHeight() int {
	lh := m.height - headerHeight - footerHeight
	if lh < 1 {
		return 1
	}
	return lh
}

// Render produces the full string for the view.
func (m Model) Render() string {
	if m.width == 0 || m.height == 0 {
		return "Loading…"
	}

	headerStyle := lipgloss.NewStyle().Bold(true)
	sepStyle := lipgloss.NewStyle().Faint(true)
	dimStyle := lipgloss.NewStyle().Faint(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	cursorStyle := lipgloss.NewStyle().Reverse(true)
	errorStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Red)

	sep := sepStyle.Render(strings.Repeat("─", m.width))

	var b strings.Builder

	// Line 0: "Find content in: <rootDir>"
	b.WriteString(headerStyle.Render(fmt.Sprintf(" Find content in: %s", m.rootDir)))
	b.WriteRune('\n')

	// Line 1: "Pattern: [field]    Alt+R Regex: … · Alt+I Case: …"
	const patternLabel = " Pattern: "
	const extLabel = " Ext:     "

	toggleStr := fmt.Sprintf(" Alt+R Regex: %s  ·  Alt+I Case: %s",
		modeLabel(m.mode), caseLabel(m.ignoreCase))
	toggleRendered := dimStyle.Render(toggleStr)

	patternFieldW := m.width - lipgloss.Width(patternLabel) - lipgloss.Width(toggleStr) - 2
	if patternFieldW < 10 {
		patternFieldW = 10
	}
	showPatternCursor := m.state == stateInput && m.focus == focusPattern
	patternField := renderInputField(m.pattern, m.patternCur, patternFieldW, showPatternCursor, cursorStyle)
	b.WriteString(patternLabel + patternField + " " + toggleRendered)
	b.WriteRune('\n')

	// Line 2: "Ext: [field]   (hint)"
	const extHint = " (comma-separated, empty = all)"
	extFieldW := m.width - lipgloss.Width(extLabel) - lipgloss.Width(extHint) - 2
	if extFieldW < 10 {
		extFieldW = 10
	}
	showExtCursor := m.state == stateInput && m.focus == focusExtensions
	extField := renderInputField(m.extensions, m.extCur, extFieldW, showExtCursor, cursorStyle)
	b.WriteString(extLabel + extField + dimStyle.Render(extHint))
	b.WriteRune('\n')

	// Line 3: separator
	b.WriteString(sep)
	b.WriteRune('\n')

	// Result rows
	lh := m.listHeight()
	end := m.offset + lh
	if end > len(m.results) {
		end = len(m.results)
	}
	const cursorPrefixWidth = 2
	rowWidth := m.width - cursorPrefixWidth
	if rowWidth < 8 {
		rowWidth = 8
	}

	for i := m.offset; i < end; i++ {
		r := m.results[i]
		selected := i == m.listCursor && m.state != stateInput

		prefix := "  "
		if selected {
			prefix = "▶ "
		}

		head := fmt.Sprintf("%s:%d: ", r.RelPath, r.LineNum)
		tail := strings.TrimSpace(r.LineText)
		rendered := head + tail
		runes := []rune(rendered)
		if len(runes) > rowWidth {
			rendered = string(runes[:rowWidth-1]) + "…"
		}

		var line string
		if selected {
			line = selectedStyle.Render(prefix + rendered)
		} else {
			// Show path:line in default style, matched text dimmed so the
			// eye can scan file locations without being dragged into match
			// content at a glance.
			headRunes := []rune(head)
			if len(runes) > rowWidth {
				// When we truncated above, just render the whole thing in
				// default style to keep the ellipsis visible.
				line = prefix + rendered
			} else if len(headRunes) < len(runes) {
				line = prefix + head + dimStyle.Render(tail)
			} else {
				line = prefix + rendered
			}
		}
		b.WriteString(line)
		b.WriteRune('\n')
	}

	rendered := end - m.offset
	for i := rendered; i < lh; i++ {
		b.WriteRune('\n')
	}

	// Bottom separator + status/help row.
	b.WriteString(sep)
	b.WriteRune('\n')

	var leftStr, rightStr string
	switch m.state {
	case stateInput:
		if m.errMsg != "" {
			leftStr = errorStyle.Render(" " + m.errMsg)
		} else {
			leftStr = headerStyle.Render(" Enter search  ·  Tab switch field  ·  Alt+R/Alt+I toggle")
		}
		rightStr = dimStyle.Render("Esc close")
	case stateSearching:
		leftStr = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Yellow).Render(
			fmt.Sprintf(" Searching… (%d matches so far)", len(m.results)))
		rightStr = dimStyle.Render("Esc/Backspace cancel")
	case stateDone:
		leftStr = headerStyle.Render(fmt.Sprintf(" %d match(es)", len(m.results)))
		rightStr = dimStyle.Render("↑↓/jk navigate  Enter open  Esc close")
	}

	gap := m.width - lipgloss.Width(leftStr) - lipgloss.Width(rightStr)
	if gap < 1 {
		gap = 1
	}
	b.WriteString(leftStr + strings.Repeat(" ", gap) + rightStr)

	return b.String()
}

func modeLabel(m Mode) string {
	if m == ModeRegex {
		return "on "
	}
	return "off"
}

func caseLabel(ignoreCase bool) string {
	if ignoreCase {
		return "insensitive"
	}
	return "sensitive  "
}

func nextFocus(f focusField) focusField {
	if f == focusPattern {
		return focusExtensions
	}
	return focusPattern
}

func prevFocus(f focusField) focusField {
	return nextFocus(f) // only two fields: next == prev
}

// insertPasted inserts pasted content into the currently focused text field
// at the current cursor position. CR and LF are stripped so that multi-line
// clipboard payloads collapse into the single-line field they're pasted
// into (a pattern or extension list wouldn't meaningfully contain them).
func (m Model) insertPasted(content string) Model {
	cleaned := strings.ReplaceAll(content, "\r", "")
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	if cleaned == "" {
		return m
	}
	switch m.focus {
	case focusPattern:
		m.pattern = insertTextAt(m.pattern, m.patternCur, cleaned)
		m.patternCur += len([]rune(cleaned))
	case focusExtensions:
		m.extensions = insertTextAt(m.extensions, m.extCur, cleaned)
		m.extCur += len([]rune(cleaned))
	}
	m.errMsg = ""
	return m
}

// editField applies a single key event to a text input value and returns
// the updated value and cursor position. Non-edit keys are no-ops.
func editField(value string, cur int, msg tea.KeyPressMsg) (string, int) {
	switch msg.String() {
	case "backspace":
		return deleteRuneAt(value, cur)
	case "left":
		if cur > 0 {
			return value, cur - 1
		}
	case "right":
		if cur < len([]rune(value)) {
			return value, cur + 1
		}
	case "ctrl+a":
		return value, 0
	case "ctrl+e":
		return value, len([]rune(value))
	default:
		if msg.Text != "" {
			return insertTextAt(value, cur, msg.Text), cur + len([]rune(msg.Text))
		}
	}
	return value, cur
}

// renderInputField renders a fixed-width text input, scrolling the visible
// window so the cursor stays in view. The cursor glyph is only drawn when
// showCursor is true — un-focused fields show their text but no caret.
func renderInputField(value string, cursorPos, fieldWidth int, showCursor bool, cursorSt lipgloss.Style) string {
	runes := []rune(value)

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
		if showCursor && i == relCursor {
			sb.WriteString(cursorSt.Render(ch))
		} else {
			sb.WriteString(ch)
		}
	}
	return sb.String()
}

func insertTextAt(s string, pos int, text string) string {
	runes := []rune(s)
	return string(runes[:pos]) + text + string(runes[pos:])
}

func deleteRuneAt(s string, pos int) (string, int) {
	if pos <= 0 {
		return s, 0
	}
	runes := []rune(s)
	return string(runes[:pos-1]) + string(runes[pos:]), pos - 1
}
