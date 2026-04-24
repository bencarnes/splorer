// Package finddropdown renders a small overlay dropdown beneath a menu bar
// item and handles its keyboard and mouse interactions. The app composes the
// dropdown over the top of its body rows using ANSI-safe column splicing;
// this package only knows about its own contents and its own bounding box.
package finddropdown

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/bjcarnes/splorer/internal/menubar"
)

// Model is the dropdown component. Construct one, feed it Update messages
// until IsClosed reports true, then discard.
type Model struct {
	items  []menubar.SubItem
	cursor int
	closed bool
	// x is the column where the dropdown's leftmost cell lives. Set by New
	// so the owning app can anchor the dropdown under its menu item.
	x int
	// y is the row where the dropdown's topmost cell lives. Typically 1
	// (just beneath the one-row menu bar).
	y int
}

// New returns a Model positioned at (x, y=1) in terminal coordinates.
// The app is expected to pass the same x value it computed from the menu
// bar's ItemRanges so the dropdown visually attaches to its trigger label.
func New(items []menubar.SubItem, x int) Model {
	return Model{items: items, x: x, y: 1}
}

// IsClosed reports whether the dropdown should be dismissed. The owning app
// should drop its reference once IsClosed becomes true.
func (m Model) IsClosed() bool { return m.closed }

// X returns the anchor column.
func (m Model) X() int { return m.x }

// Y returns the anchor row.
func (m Model) Y() int { return m.y }

// Width is the visual column width of the rendered dropdown (borders included).
func (m Model) Width() int {
	return lipgloss.Width(m.Render())
}

// Height is the number of rows occupied by the rendered dropdown.
func (m Model) Height() int {
	return strings.Count(m.Render(), "\n") + 1
}

// Contains reports whether the terminal coordinate (col, row) lies inside the
// dropdown's bounding box. Used by the app to route clicks: inside → let the
// dropdown handle it; outside → close the dropdown.
func (m Model) Contains(col, row int) bool {
	return row >= m.y && row < m.y+m.Height() &&
		col >= m.x && col < m.x+m.Width()
}

// Update handles keyboard and mouse input while the dropdown is open.
// Activating a sub-item returns its Msg as a tea.Cmd and closes the dropdown.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			m.closed = true
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", "right", "l":
			return m.activate()
		default:
			// Letter hotkeys: match msg.Text case-insensitively against the
			// Key rune of any sub-item. Activating both moves the cursor to
			// that item and closes the dropdown with the item's Msg.
			if msg.Text != "" {
				for i, it := range m.items {
					if it.Key != 0 && strings.EqualFold(msg.Text, string(it.Key)) {
						m.cursor = i
						return m.activate()
					}
				}
			}
		}

	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		return m.handleClick(int(msg.X), int(msg.Y))
	}

	return m, nil
}

// activate fires the currently highlighted sub-item's Msg and closes.
func (m Model) activate() (Model, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return m, nil
	}
	out := m.items[m.cursor].Msg
	m.closed = true
	return m, func() tea.Msg { return out }
}

// handleClick processes a left click at absolute terminal coordinates.
// Clicks on a content row select and activate that item; other clicks
// inside the box are absorbed but do nothing.
func (m Model) handleClick(x, y int) (Model, tea.Cmd) {
	if !m.Contains(x, y) {
		return m, nil
	}
	// Content rows: the box has a 1-cell top border, then one row per item,
	// then a 1-cell bottom border. Item i lives at row (m.y + 1 + i).
	idx := y - m.y - 1
	if idx < 0 || idx >= len(m.items) {
		return m, nil // clicked a border row
	}
	m.cursor = idx
	return m.activate()
}

// Render returns the boxed dropdown as a multi-line string. The app is
// responsible for overlaying this onto its body rows at column m.x.
func (m Model) Render() string {
	// Width of the item region: widest "key  Label" + surrounding padding.
	// " n  By Name " ← one leading space, then key, then two spaces, then
	// label, then one trailing space.
	maxLabel := 0
	for _, it := range m.items {
		if lipgloss.Width(it.Label) > maxLabel {
			maxLabel = lipgloss.Width(it.Label)
		}
	}
	// Row layout: "▶ " (2) + "k " (2) + " " (1) + label + trailing space
	contentWidth := 2 + 2 + 1 + maxLabel + 1
	if contentWidth < 14 {
		contentWidth = 14
	}

	sel := lipgloss.NewStyle().Reverse(true).Bold(true)
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Yellow)

	var rows []string
	for i, it := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = "▶ "
		}
		key := "  "
		if it.Key != 0 {
			key = keyStyle.Render(string(it.Key)) + " "
		}
		label := it.Label
		row := cursor + key + " " + label
		pad := contentWidth - lipgloss.Width(row)
		if pad > 0 {
			row += strings.Repeat(" ", pad)
		}
		if i == m.cursor {
			row = sel.Render(row)
		}
		rows = append(rows, row)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Render(strings.Join(rows, "\n"))
	return box
}
