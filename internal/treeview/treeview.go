// Package treeview renders a recursively-expanded view of every directory
// and file under a starting root, capped at a fixed number of entries so the
// app stays responsive on huge trees. It supports cursor navigation and
// opening a file (emitting filetree.OpenFileMsg); it intentionally does not
// support directory navigation, multi-selection, or any manipulation.
package treeview

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/bjcarnes/splorer/internal/filetree"
)

// MaxEntries caps how many rows the tree view will materialise. Crossing
// this threshold halts the walk and surfaces a "not everything shown"
// message in the footer.
const MaxEntries = 2000

const (
	headerHeight = 2 // root path + separator
	footerHeight = 2 // separator + status/help line
)

// row is one rendered line in the tree.
type row struct {
	depth int
	name  string
	path  string
	isDir bool
	// isLast reports whether this entry is the final child of its parent
	// directory. The renderer uses it to choose between "├─" and "└─" for
	// the current row, and "│ " vs. "  " for the indentation of any
	// descendants that pass through this depth.
	isLast bool
	// ancestorLast[d] is true if the ancestor at depth d was the last child
	// of its parent. When true, the column at depth d should be blank;
	// otherwise it should draw a vertical "│ " connector.
	ancestorLast []bool
}

// Page is the tree-view component.
type Page struct {
	root       string
	rows       []row
	cursor     int
	offset     int
	closed     bool
	truncated  bool
	width      int
	height     int
	lastClick  time.Time
	lastClickY int
}

// New walks root recursively (depth-first, dirs before files at each level)
// and builds a flat list of rows up to MaxEntries. Errors reading individual
// directories are skipped silently — the tree view is read-only and best-
// effort.
func New(root string, width, height int) Page {
	rows, truncated := buildRows(root)
	return Page{
		root:      root,
		rows:      rows,
		truncated: truncated,
		width:     width,
		height:    height,
	}
}

// IsClosed reports whether the page should be dismissed.
func (p Page) IsClosed() bool { return p.closed }

// buildRows walks root and returns up to MaxEntries rows. The bool reports
// whether the walk was truncated. Each row carries the connector context
// needed to render tree lines: whether it is the last sibling at its level,
// and the same flag for every ancestor (used to decide whether each indent
// column should render a vertical bar or whitespace).
func buildRows(root string) ([]row, bool) {
	rows := make([]row, 0, 64)
	truncated := false
	var walk func(dir string, depth int, ancestorLast []bool)
	walk = func(dir string, depth int, ancestorLast []bool) {
		if truncated {
			return
		}
		des, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		// Sort: dirs first (case-insensitive), then files.
		sort.SliceStable(des, func(i, j int) bool {
			a, b := des[i], des[j]
			if a.IsDir() != b.IsDir() {
				return a.IsDir() && !b.IsDir()
			}
			return strings.ToLower(a.Name()) < strings.ToLower(b.Name())
		})
		for i, de := range des {
			if len(rows) >= MaxEntries {
				truncated = true
				return
			}
			isLast := i == len(des)-1
			// Snapshot the ancestor-flags for this row; the slice we recurse
			// with is a separate allocation so siblings don't share state.
			ancestors := make([]bool, len(ancestorLast))
			copy(ancestors, ancestorLast)
			rows = append(rows, row{
				depth:        depth,
				name:         de.Name(),
				path:         filepath.Join(dir, de.Name()),
				isDir:        de.IsDir(),
				isLast:       isLast,
				ancestorLast: ancestors,
			})
			if de.IsDir() {
				childAncestors := append(append([]bool(nil), ancestorLast...), isLast)
				walk(filepath.Join(dir, de.Name()), depth+1, childAncestors)
				if truncated {
					return
				}
			}
		}
	}
	walk(root, 0, nil)
	return rows, truncated
}

// Update processes a Bubble Tea message.
func (p Page) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc", "backspace", "q":
			p.closed = true
		case "up", "k":
			p = p.moveCursor(-1)
		case "down", "j":
			p = p.moveCursor(1)
		case "pgup":
			p = p.moveCursor(-p.listHeight())
		case "pgdown":
			p = p.moveCursor(p.listHeight())
		case "home":
			p.cursor = 0
			p.offset = 0
		case "end":
			if len(p.rows) > 0 {
				p.cursor = len(p.rows) - 1
				lh := p.listHeight()
				if p.cursor >= lh {
					p.offset = p.cursor - lh + 1
				}
			}
		case "enter", "right", "l":
			return p.activate()
		}

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			idx := int(msg.Y) - headerHeight + p.offset
			now := time.Now()
			isDouble := idx == p.lastClickY && now.Sub(p.lastClick) < 500*time.Millisecond
			p.lastClick = now
			if idx >= 0 && idx < len(p.rows) {
				p.lastClickY = idx
				p.cursor = idx
				if isDouble {
					return p.activate()
				}
			}
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			p = p.moveCursor(-1)
		case tea.MouseWheelDown:
			p = p.moveCursor(1)
		}
	}
	return p, nil
}

// activate emits a filetree.OpenFileMsg for the cursor's row when it points
// at a file. Directory rows are no-ops — this view explicitly does not
// support directory navigation.
func (p Page) activate() (Page, tea.Cmd) {
	if len(p.rows) == 0 {
		return p, nil
	}
	r := p.rows[p.cursor]
	if r.isDir {
		return p, nil
	}
	path := r.path
	return p, func() tea.Msg { return filetree.OpenFileMsg{Path: path} }
}

func (p Page) moveCursor(delta int) Page {
	p.cursor += delta
	n := len(p.rows)
	if p.cursor < 0 {
		p.cursor = 0
	}
	if n > 0 && p.cursor >= n {
		p.cursor = n - 1
	}
	lh := p.listHeight()
	if p.cursor < p.offset {
		p.offset = p.cursor
	}
	if p.cursor >= p.offset+lh {
		p.offset = p.cursor - lh + 1
	}
	return p
}

func (p Page) listHeight() int {
	lh := p.height - headerHeight - footerHeight
	if lh < 1 {
		return 1
	}
	return lh
}

// Render produces the full-screen content for the tree view.
func (p Page) Render() string {
	if p.width == 0 || p.height == 0 {
		return "Loading…"
	}

	headerStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)
	dirStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Blue)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	warnStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Yellow)
	sepStyle := lipgloss.NewStyle().Faint(true)

	sep := sepStyle.Render(strings.Repeat("─", p.width))

	var b strings.Builder

	b.WriteString(headerStyle.Render(" Tree: " + p.root))
	b.WriteRune('\n')
	b.WriteString(sep)
	b.WriteRune('\n')

	lh := p.listHeight()
	end := p.offset + lh
	if end > len(p.rows) {
		end = len(p.rows)
	}

	lineStyle := dimStyle

	for i := p.offset; i < end; i++ {
		r := p.rows[i]

		cursorStr := "  "
		if i == p.cursor {
			cursorStr = "▶ "
		}

		// Indentation columns: each ancestor either contributes a vertical
		// connector ("│ ") or blank space, depending on whether that
		// ancestor was the last sibling in its parent directory.
		var indent strings.Builder
		for _, last := range r.ancestorLast {
			if last {
				indent.WriteString("  ")
			} else {
				indent.WriteString("│ ")
			}
		}
		// Branch glyph for the current row. Drawn at every depth (including
		// the root level) so a top-level row's descendants don't appear to
		// dangle off an invisible parent line.
		var branch string
		if r.isLast {
			branch = "└─"
		} else {
			branch = "├─"
		}

		// Compose the prefix (indent + branch + space). Styled uniformly as
		// faint so the lines fade behind the names.
		prefix := indent.String() + branch + " "
		styledPrefix := lineStyle.Render(prefix)

		// Build the visible row: prefix + icon + " " + name (+ trailing slash for dirs).
		ent := filetree.FileEntry{Name: r.name, IsDir: r.isDir}
		icon := ent.Icon()
		display := r.name
		if r.isDir {
			display += "/"
		}

		// Truncate the name (only the name part) so the prefix and icon
		// stay intact when the row is too wide.
		prefixWidth := lipgloss.Width(prefix) + lipgloss.Width(icon) + 1 // +1 for space after icon
		nameMaxW := p.width - 2 - prefixWidth                            // -2 for the cursor column
		if nameMaxW < 4 {
			nameMaxW = 4
		}
		runes := []rune(display)
		if lipgloss.Width(display) > nameMaxW {
			display = string(runes[:nameMaxW-1]) + "…"
		}

		var nameRendered string
		if r.isDir {
			nameRendered = dirStyle.Render(display)
		} else {
			nameRendered = display
		}

		var line string
		if i == p.cursor {
			// Render the full row through the reverse style so the cursor
			// highlight is uniform across prefix, icon, and name.
			line = selectedStyle.Render(cursorStr + prefix + icon + " " + display)
		} else {
			line = cursorStr + styledPrefix + icon + " " + nameRendered
		}
		b.WriteString(line)
		b.WriteRune('\n')
	}

	rendered := end - p.offset
	for i := rendered; i < lh; i++ {
		b.WriteRune('\n')
	}

	b.WriteString(sep)
	b.WriteRune('\n')

	var leftStr string
	if p.truncated {
		leftStr = warnStyle.Render(
			fmt.Sprintf(" Showing first %d entries — tree truncated.", MaxEntries))
	} else {
		leftStr = headerStyle.Render(fmt.Sprintf(" %d entries", len(p.rows)))
	}
	rightStr := dimStyle.Render("↑↓/jk navigate  Enter open file  Esc/q close")
	gap := p.width - lipgloss.Width(leftStr) - lipgloss.Width(rightStr)
	if gap < 1 {
		gap = 1
	}
	b.WriteString(leftStr + strings.Repeat(" ", gap) + rightStr)

	return b.String()
}
