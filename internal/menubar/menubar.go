// Package menubar provides a horizontal menu bar that sits at the top of the
// terminal screen. Items are activated by clicking on them or pressing their
// designated keyboard hotkey.
//
// An item may optionally declare SubItems, in which case activating it emits
// OpenDropdownMsg instead of its own Msg — the owning app then shows a
// dropdown component below the menu bar.
package menubar

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// dropdownIndicator is appended to an item's label when it has sub-items.
const dropdownIndicator = " ▾"

// SubItem is a single entry inside a dropdown spawned by an Item.
type SubItem struct {
	// Label is the visible text of the sub-item.
	Label string
	// Key is a single rune hotkey that activates this sub-item while the
	// dropdown is open (e.g. 'n' for "By Name"). Zero means no letter hotkey.
	Key rune
	// Msg is emitted when the sub-item is activated.
	Msg tea.Msg
}

// Item is a single entry in the menu bar.
type Item struct {
	// Label is the visible text rendered in the bar.
	Label string
	// Hotkey is a bubbletea key string (e.g. "alt+o") that activates this
	// item from the keyboard. Empty means no hotkey.
	Hotkey string
	// Msg is emitted as a tea.Cmd result when the item is activated and has
	// no sub-items.
	Msg tea.Msg
	// SubItems, if non-empty, converts this item into a dropdown trigger.
	// Activating the item emits OpenDropdownMsg{Index} instead of Msg.
	SubItems []SubItem
}

// OpenDropdownMsg is emitted when an Item with sub-items is activated.
// The owning app uses Index to look up the sub-items for the dropdown.
type OpenDropdownMsg struct {
	Index int
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

// HandleKey returns a tea.Cmd that emits the matching Item's activation
// message when msg matches an item's Hotkey, or nil if no item matches.
// Items with sub-items emit OpenDropdownMsg instead of their own Msg.
func (b MenuBar) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	for i, item := range b.Items {
		if item.Hotkey != "" && msg.String() == item.Hotkey {
			return b.activationCmd(i)
		}
	}
	return nil
}

// HandleClick returns a tea.Cmd that emits the clicked Item's activation
// message when x falls within that item's rendered column range, or nil if
// no item was hit. Items with sub-items emit OpenDropdownMsg.
func (b MenuBar) HandleClick(x int) tea.Cmd {
	for i, r := range b.itemRanges() {
		if x >= r[0] && x < r[1] {
			return b.activationCmd(i)
		}
	}
	return nil
}

// activationCmd returns the right activation command for the item at index:
// OpenDropdownMsg for items with sub-items, the item's Msg otherwise.
func (b MenuBar) activationCmd(index int) tea.Cmd {
	item := b.Items[index]
	if len(item.SubItems) > 0 {
		idx := index // capture
		return func() tea.Msg { return OpenDropdownMsg{Index: idx} }
	}
	m := item.Msg
	return func() tea.Msg { return m }
}

// ItemRanges returns the [startX, endX) column ranges for each item as
// rendered by Render. Exported so callers can do their own hit-testing.
func (b MenuBar) ItemRanges() [][2]int {
	return b.itemRanges()
}

// itemRanges computes [startX, endX) for each item label in the rendered bar.
// Layout: one leading space, then items separated by a single space each.
// Labels of items with sub-items include the ▾ dropdown indicator.
func (b MenuBar) itemRanges() [][2]int {
	ranges := make([][2]int, len(b.Items))
	x := 1 // leading space
	for i, item := range b.Items {
		label := " " + renderedLabel(item) + " "
		w := lipgloss.Width(label)
		ranges[i] = [2]int{x, x + w}
		x += w + 1 // gap between items
	}
	return ranges
}

// renderedLabel returns the item's label with the dropdown indicator appended
// if the item has sub-items.
func renderedLabel(item Item) string {
	if len(item.SubItems) > 0 {
		return item.Label + dropdownIndicator
	}
	return item.Label
}

// Render returns the full-width menu bar string for the current Width.
func (b MenuBar) Render() string {
	barStyle := lipgloss.NewStyle().Reverse(true)
	itemStyle := lipgloss.NewStyle().Reverse(true).Bold(true)

	var sb strings.Builder
	sb.WriteString(barStyle.Render(" ")) // leading space

	for i, item := range b.Items {
		label := " " + renderedLabel(item) + " "
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
