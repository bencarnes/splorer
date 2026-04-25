package filetree

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// OpenFileMsg is emitted as a tea.Cmd result when the user activates a file.
// The parent model is responsible for resolving the correct opener program.
type OpenFileMsg struct{ Path string }

// headerHeight is the number of lines consumed by the path + separator at the
// top of the view (used to map mouse Y coordinates to entry indices).
const headerHeight = 2

// footerHeight is the number of lines consumed by the separator + status bar.
const footerHeight = 2

// Model is the file tree component. It does not implement tea.Model directly;
// the parent app.Model owns it and calls Update/Render.
type Model struct {
	cwd        string
	entries    []FileEntry
	cursor     int
	offset     int // scroll: index of first visible entry
	sortOrder  SortOrder
	width      int
	height     int
	lastClick  time.Time
	lastClickY int // entry index of last click (for double-click detection)
	err        string
}

// New creates a Model starting in cwd with sensible defaults.
func New(cwd string) Model {
	m := Model{width: 80, height: 24}
	m, _ = m.navigateTo(cwd)
	return m
}

// CWD returns the directory the model is currently showing.
func (m Model) CWD() string { return m.cwd }

// CurrentSortOrder returns the active sort order.
func (m Model) CurrentSortOrder() SortOrder { return m.sortOrder }

// SetSortOrder applies a new sort order, re-sorts the current directory, and
// returns a fresh watch command bound to the new order so stale watcher ticks
// from the previous order are discarded.
func (m Model) SetSortOrder(so SortOrder) (Model, tea.Cmd) {
	m.sortOrder = so
	entries, err := loadDir(m.cwd, so)
	if err == nil {
		m.entries = entries
		m.cursor = 0
		m.offset = 0
	}
	return m, m.WatchCmd()
}

// SelectedPath returns the path of the currently highlighted entry.
// Returns CWD if the directory is empty.
func (m Model) SelectedPath() string {
	if len(m.entries) == 0 {
		return m.cwd
	}
	return m.entries[m.cursor].Path
}

// NavigateTo navigates the model to path, returning the updated model. Returns
// an error if the directory cannot be read; in that case the model is unchanged.
func (m Model) NavigateTo(path string) (Model, error) {
	return m.navigateTo(path)
}

// navigateTo loads the directory at path into the model. On error the error
// message is stored and the model is returned unchanged.
func (m Model) navigateTo(path string) (Model, error) {
	entries, err := loadDir(path, m.sortOrder)
	if err != nil {
		m.err = fmt.Sprintf("cannot read %s: %s", path, err)
		return m, err
	}
	m.cwd = path
	m.entries = entries
	m.cursor = 0
	m.offset = 0
	m.err = ""
	return m, nil
}

// loadDir reads the directory at path and returns sorted entries. Directories
// always appear before files; within each group entries are ordered by so.
func loadDir(path string, so SortOrder) ([]FileEntry, error) {
	des, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var dirs, files []FileEntry
	for _, de := range des {
		info, err := de.Info()
		if err != nil {
			continue
		}
		e := FileEntry{
			Name:    de.Name(),
			Path:    filepath.Join(path, de.Name()),
			IsDir:   de.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
		}
		if de.IsDir() {
			dirs = append(dirs, e)
		} else {
			files = append(files, e)
		}
	}
	sortGroup(dirs, so)
	sortGroup(files, so)
	return append(dirs, files...), nil
}

// Update processes a bubbletea message and returns the updated model and an
// optional command.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		m.err = "" // clear stale errors on any keypress
		switch msg.String() {
		case "up", "k":
			m = m.moveCursor(-1)
		case "down", "j":
			m = m.moveCursor(1)
		case "pgup":
			m = m.moveCursor(-m.listHeight())
		case "pgdown":
			m = m.moveCursor(m.listHeight())
		case "enter", "right", "l":
			return m.activate()
		case "backspace", "left", "h":
			return m.goUp()
		case "~":
			if home, err := os.UserHomeDir(); err == nil {
				if newM, err2 := m.navigateTo(home); err2 == nil {
					return newM, newM.WatchCmd()
				}
			}
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			idx := int(msg.Y) - headerHeight + m.offset
			now := time.Now()
			isDouble := idx == m.lastClickY && now.Sub(m.lastClick) < 500*time.Millisecond
			m.lastClick = now
			if idx >= 0 && idx < len(m.entries) {
				m.lastClickY = idx
				m.cursor = idx
				if isDouble {
					return m.activate()
				}
			}
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m = m.moveCursor(-1)
		case tea.MouseWheelDown:
			m = m.moveCursor(1)
		}

	case DirChangedMsg:
		// Discard ticks from a previous directory or a superseded sort order.
		if msg.Dir != m.cwd || msg.SortOrder != m.sortOrder {
			return m, nil
		}
		if msg.Entries != nil && !entriesEqual(m.entries, msg.Entries) {
			m = m.applyEntryRefresh(msg.Entries)
		}
		return m, m.WatchCmd()

	case DirGoneMsg:
		if msg.Dir != m.cwd {
			return m, nil
		}
		ancestor := nearestExistingAncestor(m.cwd)
		newM, err := m.navigateTo(ancestor)
		if err != nil {
			m.err = "directory removed"
			return m, nil
		}
		return newM, newM.WatchCmd()
	}

	return m, nil
}

// moveCursor moves the cursor by delta rows and adjusts the scroll offset.
func (m Model) moveCursor(delta int) Model {
	m.cursor += delta
	n := len(m.entries)
	if m.cursor < 0 {
		m.cursor = 0
	}
	if n > 0 && m.cursor >= n {
		m.cursor = n - 1
	}
	// Adjust scroll offset so the cursor is visible.
	lh := m.listHeight()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+lh {
		m.offset = m.cursor - lh + 1
	}
	return m
}

// activate opens or navigates into the currently selected entry.
func (m Model) activate() (Model, tea.Cmd) {
	if len(m.entries) == 0 {
		return m, nil
	}
	entry := m.entries[m.cursor]
	if entry.IsDir {
		newM, err := m.navigateTo(entry.Path)
		if err != nil {
			m.err = fmt.Sprintf("cannot open: %s", err)
			return m, nil
		}
		return newM, newM.WatchCmd()
	}
	path := entry.Path
	return m, func() tea.Msg { return OpenFileMsg{Path: path} }
}

// goUp navigates to the parent directory, restoring the cursor to the
// subdirectory we came from.
func (m Model) goUp() (Model, tea.Cmd) {
	parent := filepath.Dir(m.cwd)
	if parent == m.cwd {
		return m, nil // already at root
	}
	prevName := filepath.Base(m.cwd)
	newM, err := m.navigateTo(parent)
	if err != nil {
		m.err = fmt.Sprintf("cannot open: %s", err)
		return m, nil
	}
	for i, e := range newM.entries {
		if e.Name == prevName {
			newM.cursor = i
			lh := newM.listHeight()
			if i >= lh {
				newM.offset = i - lh + 1
			}
			break
		}
	}
	return newM, newM.WatchCmd()
}

// listHeight is the number of visible entry rows (total height minus header and footer).
func (m Model) listHeight() int {
	lh := m.height - headerHeight - footerHeight
	if lh < 1 {
		return 1
	}
	return lh
}

// Render produces the full-screen string content for this view.
func (m Model) Render() string {
	if m.width == 0 || m.height == 0 {
		return "Loading…"
	}

	dirStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Blue)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	dimStyle := lipgloss.NewStyle().Faint(true)
	headerStyle := lipgloss.NewStyle().Bold(true)
	sepStyle := lipgloss.NewStyle().Faint(true)

	sep := sepStyle.Render(strings.Repeat("─", m.width))

	var b strings.Builder

	// Line 0: current path + sort label on the right
	sortLabel := dimStyle.Render("[" + m.sortOrder.Label() + "]")
	pathStr := headerStyle.Render(" " + m.cwd)
	gap := m.width - lipgloss.Width(pathStr) - lipgloss.Width(sortLabel)
	if gap < 1 {
		gap = 1
	}
	b.WriteString(pathStr + strings.Repeat(" ", gap) + sortLabel)
	b.WriteRune('\n')
	// Line 1: separator
	b.WriteString(sep)
	b.WriteRune('\n')

	// Entry rows
	lh := m.listHeight()
	end := m.offset + lh
	if end > len(m.entries) {
		end = len(m.entries)
	}

	// Column widths: name takes the bulk; right column is "XXXXXX YYYY-MM-DD"
	// = 6 (size) + 1 (space) + 10 (date) = 17, plus 2 for the cursor prefix,
	// plus 3 for the icon column (2-cell emoji + 1 space).
	const cursorWidth = 2
	const iconColWidth = 3
	const rightWidth = 17
	nameWidth := m.width - cursorWidth - iconColWidth - rightWidth - 1
	if nameWidth < 8 {
		nameWidth = 8
	}

	for i := m.offset; i < end; i++ {
		e := m.entries[i]

		// Cursor indicator
		cursorStr := "  "
		if i == m.cursor {
			cursorStr = "▶ "
		}

		icon := e.Icon()

		// Name (truncated to fit)
		name := e.Title()
		runes := []rune(name)
		if len(runes) > nameWidth {
			name = string(runes[:nameWidth-1]) + "…"
		}
		// Pad name to nameWidth
		namePadded := name + strings.Repeat(" ", nameWidth-lipgloss.Width(name))

		// Right column: size + date
		var sizeStr string
		if e.IsDir {
			sizeStr = "dir   "
		} else {
			sizeStr = fmt.Sprintf("%-6s", humanizeSize(e.Size))
		}
		dateStr := e.ModTime.Format("2006-01-02")
		right := sizeStr + " " + dateStr

		var line string
		if i == m.cursor {
			// Highlight entire row
			line = selectedStyle.Render(cursorStr + icon + " " + namePadded + right)
		} else if e.IsDir {
			line = cursorStr + dirStyle.Render(icon+" "+namePadded) + dimStyle.Render(right)
		} else {
			line = cursorStr + icon + " " + namePadded + dimStyle.Render(right)
		}

		b.WriteString(line)
		b.WriteRune('\n')
	}

	// Fill remaining rows so the footer stays at the bottom
	rendered := end - m.offset
	for i := rendered; i < lh; i++ {
		b.WriteRune('\n')
	}

	// Bottom separator
	b.WriteString(sep)
	b.WriteRune('\n')

	// Status bar
	var leftStr string
	if m.err != "" {
		leftStr = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Red).Render(" " + m.err)
	} else {
		leftStr = lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf(" %d items", len(m.entries)))
	}
	rightStr := dimStyle.Render("q quit  ↑↓/jk navigate  enter open  ←/h go up  ~ home")
	footerGap := m.width - lipgloss.Width(leftStr) - lipgloss.Width(rightStr)
	if footerGap < 1 {
		footerGap = 1
	}
	b.WriteString(leftStr + strings.Repeat(" ", footerGap) + rightStr)

	return b.String()
}
