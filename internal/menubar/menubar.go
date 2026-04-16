// Package menubar provides a horizontal menu bar that sits at the top of the
// terminal screen. Items are activated by clicking on them or pressing their
// designated keyboard hotkey.
//
// Design is intentionally extensible: additional Item values can be appended
// to the slice passed to New, and future work can add dropdown support by
// adding a sub-items field to Item.
package menubar

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// Item is a single entry in the menu bar.
type Item struct {
	// Label is the visible text rendered in the bar.
	Label string
	// Hotkey is a bubbletea key string (e.g. "alt+o") that activates this
	// item from the keyboard. Empty means no hotkey.
	Hotkey string
	// Msg is emitted as a tea.Cmd result when the item is activated.
	Msg tea.Msg
}

// MenuBar is a horizontal strip of Items rendered as one terminal line.
type MenuBar struct {
	Items []Item
	Width int
}

// New creates a MenuBar containing the given items.
func New(items []Item) MenuBar {
	return MenuBar{Items: items}
}

// HandleKey returns a tea.Cmd that emits the matching Item's Msg when msg
// matches an item's Hotkey, or nil if no item matches.
func (b MenuBar) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	for _, item := range b.Items {
		if item.Hotkey != "" && msg.String() == item.Hotkey {
			m := item.Msg // capture
			return func() tea.Msg { return m }
		}
	}
	return nil
}

// HandleClick returns a tea.Cmd that emits the clicked Item's Msg when x falls
// within that item's rendered column range, or nil if no item was hit.
func (b MenuBar) HandleClick(x int) tea.Cmd {
	for i, r := range b.itemRanges() {
		if x >= r[0] && x < r[1] {
			m := b.Items[i].Msg
			return func() tea.Msg { return m }
		}
	}
	return nil
}

// ItemRanges returns the [startX, endX) column ranges for each item as
// rendered by Render. Exported so callers can do their own hit-testing.
func (b MenuBar) ItemRanges() [][2]int {
	return b.itemRanges()
}

// itemRanges computes [startX, endX) for each item label in the rendered bar.
// Layout: one leading space, then items separated by a single space each.
func (b MenuBar) itemRanges() [][2]int {
	ranges := make([][2]int, len(b.Items))
	x := 1 // leading space
	for i, item := range b.Items {
		label := " " + item.Label + " "
		w := lipgloss.Width(label)
		ranges[i] = [2]int{x, x + w}
		x += w + 1 // gap between items
	}
	return ranges
}

// Render returns the full-width menu bar string for the current Width.
func (b MenuBar) Render() string {
	barStyle := lipgloss.NewStyle().Reverse(true)
	itemStyle := lipgloss.NewStyle().Reverse(true).Bold(true)

	var sb strings.Builder
	sb.WriteString(barStyle.Render(" ")) // leading space

	for i, item := range b.Items {
		label := " " + item.Label + " "
		sb.WriteString(itemStyle.Render(label))
		if i < len(b.Items)-1 {
			sb.WriteString(barStyle.Render(" ")) // gap between items
		}
	}

	// Pad the rest of the line so the bar spans the full width.
	rendered := lipgloss.Width(sb.String())
	if b.Width > rendered {
		sb.WriteString(barStyle.Render(strings.Repeat(" ", b.Width-rendered)))
	}

	return sb.String()
}
