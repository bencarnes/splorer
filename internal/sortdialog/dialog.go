// Package sortdialog provides a small picker overlay for choosing the file-tree
// sort order.
package sortdialog

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/bjcarnes/splorer/internal/filetree"
)

// Dialog is the sort-order picker overlay.
type Dialog struct {
	cursor int
	closed bool
	saved  bool
	chosen filetree.SortOrder
}

// New creates a Dialog with the cursor pre-positioned on current.
func New(current filetree.SortOrder) Dialog {
	cursor := 0
	for i, o := range filetree.AllSortOrders {
		if o == current {
			cursor = i
			break
		}
	}
	return Dialog{cursor: cursor, chosen: current}
}

// IsClosed reports whether the dialog has been dismissed.
func (d Dialog) IsClosed() bool { return d.closed }

// IsSaved reports whether the user confirmed a selection.
func (d Dialog) IsSaved() bool { return d.saved }

// Chosen returns the selected sort order.
func (d Dialog) Chosen() filetree.SortOrder { return d.chosen }

// Update processes a Bubble Tea message.
func (d Dialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			d.closed = true
		case "enter":
			d.chosen = filetree.AllSortOrders[d.cursor]
			d.saved = true
			d.closed = true
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case "down", "j":
			if d.cursor < len(filetree.AllSortOrders)-1 {
				d.cursor++
			}
		}
	}
	return d, nil
}

// Render produces the dialog string for the given terminal dimensions.
func (d Dialog) Render(width, height int) string {
	boldStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)

	// Box width: wide enough for the longest option label plus chrome.
	bw := 32
	if width-4 < bw {
		bw = width - 4
	}
	if bw < 24 {
		bw = 24
	}
	iw := bw - 2

	pad := func(s string) string {
		w := lipgloss.Width(s)
		if w < iw {
			s += strings.Repeat(" ", iw-w)
		}
		return "│" + s + "│"
	}

	topBorder := "╭" + strings.Repeat("─", iw) + "╮"
	botBorder := "╰" + strings.Repeat("─", iw) + "╯"
	divider := "├" + strings.Repeat("─", iw) + "┤"

	lines := []string{
		topBorder,
		pad(" " + boldStyle.Render("Sort Order")),
		pad(""),
	}

	for i, o := range filetree.AllSortOrders {
		prefix := "  "
		if i == d.cursor {
			prefix = "▶ "
		}
		label := prefix + o.Label()
		if i == d.cursor {
			lines = append(lines, pad(selectedStyle.Render(label)))
		} else {
			lines = append(lines, pad(label))
		}
	}

	lines = append(lines, pad(""))
	lines = append(lines, divider)
	lines = append(lines, pad(dimStyle.Render("  ↑↓ navigate  Enter select  Esc cancel")))
	lines = append(lines, botBorder)

	// Center horizontally.
	leftPad := (width - bw) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	prefix := strings.Repeat(" ", leftPad)
	for i, l := range lines {
		lines[i] = prefix + l
	}

	// Center vertically.
	topPad := (height - len(lines)) / 2
	if topPad < 0 {
		topPad = 0
	}

	var b strings.Builder
	for i := 0; i < topPad; i++ {
		b.WriteRune('\n')
	}
	b.WriteString(strings.Join(lines, "\n"))
	rendered := topPad + len(lines)
	for i := rendered; i < height; i++ {
		b.WriteRune('\n')
	}
	return b.String()
}
