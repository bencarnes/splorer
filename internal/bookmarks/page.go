package bookmarks

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"

	"github.com/bjcarnes/splorer/internal/filetree"
)

// NavigateDirMsg is emitted when the user activates a directory bookmark.
// The parent model should navigate its file tree to this path and close the
// bookmarks page.
type NavigateDirMsg struct{ Path string }

type pageState int

const (
	stateList   pageState = iota
	stateDelete           // delete confirmation dialog
)

const (
	pageHeaderHeight = 2 // title + separator
	pageFooterHeight = 2 // separator + help bar
)

// Page is the bookmarks list view component.
type Page struct {
	bookmarks []Bookmark
	cursor    int
	offset    int
	state     pageState
	closed    bool

	width  int
	height int

	lastClick  time.Time
	lastClickY int
}

// NewPage creates a bookmarks Page initialised with a copy of bmarks.
func NewPage(bmarks []Bookmark, width, height int) Page {
	cp := make([]Bookmark, len(bmarks))
	copy(cp, bmarks)
	return Page{
		bookmarks: cp,
		width:     width,
		height:    height,
	}
}

// IsClosed reports whether the page should be dismissed.
func (p Page) IsClosed() bool { return p.closed }

// Bookmarks returns the current (possibly mutated) bookmark list.
func (p Page) Bookmarks() []Bookmark {
	result := make([]Bookmark, len(p.bookmarks))
	copy(result, p.bookmarks)
	return result
}

// Update processes a Bubble Tea message.
func (p Page) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
	case tea.KeyPressMsg:
		switch p.state {
		case stateList:
			return p.updateList(msg)
		case stateDelete:
			return p.updateDelete(msg)
		}
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft && p.state == stateList {
			return p.handleClick(int(msg.X), int(msg.Y))
		}
	case tea.MouseWheelMsg:
		if p.state == stateList {
			switch msg.Button {
			case tea.MouseWheelUp:
				p = p.moveCursor(-1)
			case tea.MouseWheelDown:
				p = p.moveCursor(1)
			}
		}
	}
	return p, nil
}

func (p Page) updateList(msg tea.KeyPressMsg) (Page, tea.Cmd) {
	switch msg.String() {
	case "esc", "backspace":
		p.closed = true
	case "up", "k":
		p = p.moveCursor(-1)
	case "down", "j":
		p = p.moveCursor(1)
	case "pgup":
		p = p.moveCursor(-p.listHeight())
	case "pgdown":
		p = p.moveCursor(p.listHeight())
	case "enter", "right", "l":
		return p.activate()
	case "delete":
		if len(p.bookmarks) > 0 {
			p.state = stateDelete
		}
	}
	return p, nil
}

func (p Page) updateDelete(msg tea.KeyPressMsg) (Page, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if p.cursor < len(p.bookmarks) {
			p.bookmarks = append(p.bookmarks[:p.cursor], p.bookmarks[p.cursor+1:]...)
			if p.cursor >= len(p.bookmarks) && p.cursor > 0 {
				p.cursor--
			}
			if p.offset >= len(p.bookmarks) && p.offset > 0 {
				p.offset = len(p.bookmarks) - 1
			}
		}
		p.state = stateList
	case "n", "N", "esc":
		p.state = stateList
	}
	return p, nil
}

func (p Page) activate() (Page, tea.Cmd) {
	if len(p.bookmarks) == 0 {
		return p, nil
	}
	bm := p.bookmarks[p.cursor]
	path := bm.Path

	info, err := os.Stat(path)
	if err != nil {
		return p, nil
	}

	if info.IsDir() {
		p.closed = true
		return p, func() tea.Msg { return NavigateDirMsg{Path: path} }
	}
	return p, func() tea.Msg { return filetree.OpenFileMsg{Path: path} }
}

func (p Page) handleClick(x, y int) (Page, tea.Cmd) {
	_ = x
	idx := y - pageHeaderHeight + p.offset
	now := time.Now()
	isDouble := idx == p.lastClickY && now.Sub(p.lastClick) < 500*time.Millisecond
	p.lastClick = now
	if idx >= 0 && idx < len(p.bookmarks) {
		p.lastClickY = idx
		p.cursor = idx
		if isDouble {
			return p.activate()
		}
	}
	return p, nil
}

func (p Page) moveCursor(delta int) Page {
	p.cursor += delta
	n := len(p.bookmarks)
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
	lh := p.height - pageHeaderHeight - pageFooterHeight
	if lh < 1 {
		return 1
	}
	return lh
}

// Render produces the full-screen content for the bookmarks page.
func (p Page) Render() string {
	if p.width == 0 || p.height == 0 {
		return "Loading…"
	}
	if p.state == stateDelete {
		return p.renderDelete()
	}
	return p.renderList()
}

func (p Page) renderList() string {
	headerStyle := lipgloss.NewStyle().Bold(true)
	sepStyle := lipgloss.NewStyle().Faint(true)
	dimStyle := lipgloss.NewStyle().Faint(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	dirStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Blue)

	sep := sepStyle.Render(strings.Repeat("─", p.width))

	var b strings.Builder

	b.WriteString(headerStyle.Render(" Bookmarks"))
	b.WriteRune('\n')
	b.WriteString(sep)
	b.WriteRune('\n')

	lh := p.listHeight()
	end := p.offset + lh
	if end > len(p.bookmarks) {
		end = len(p.bookmarks)
	}

	if len(p.bookmarks) == 0 {
		b.WriteString(dimStyle.Render("  (no bookmarks)"))
		b.WriteRune('\n')
		for i := 1; i < lh; i++ {
			b.WriteRune('\n')
		}
	} else {
		// Name column: ~1/3 of width; path gets the rest.
		const prefix = 2 // "▶ " or "  "
		const sep2 = 2   // gap between name and path
		nameColWidth := p.width / 3
		if nameColWidth < 12 {
			nameColWidth = 12
		}
		pathColWidth := p.width - prefix - nameColWidth - sep2
		if pathColWidth < 10 {
			pathColWidth = 10
		}

		for i := p.offset; i < end; i++ {
			bm := p.bookmarks[i]
			selected := i == p.cursor

			cursorStr := "  "
			if selected {
				cursorStr = "▶ "
			}

			info, err := os.Stat(bm.Path)
			isDir := err == nil && info.IsDir()

			nameRunes := []rune(bm.Name)
			displayName := bm.Name
			if len(nameRunes) > nameColWidth {
				displayName = string(nameRunes[:nameColWidth-1]) + "…"
			}
			namePadded := displayName + strings.Repeat(" ", nameColWidth-lipgloss.Width(displayName))

			pathRunes := []rune(bm.Path)
			displayPath := bm.Path
			if len(pathRunes) > pathColWidth {
				displayPath = "…" + string(pathRunes[len(pathRunes)-pathColWidth+1:])
			}

			var line string
			if selected {
				line = selectedStyle.Render(cursorStr + namePadded + "  " + displayPath)
			} else if isDir {
				line = cursorStr + dirStyle.Render(namePadded) + dimStyle.Render("  "+displayPath)
			} else {
				line = cursorStr + namePadded + dimStyle.Render("  "+displayPath)
			}

			b.WriteString(line)
			b.WriteRune('\n')
		}
		rendered := end - p.offset
		for i := rendered; i < lh; i++ {
			b.WriteRune('\n')
		}
	}

	b.WriteString(sep)
	b.WriteRune('\n')

	leftStr := headerStyle.Render(fmt.Sprintf(" %d bookmark(s)", len(p.bookmarks)))
	rightStr := dimStyle.Render("↑↓ navigate  Enter open  Del delete  Esc/Backspace close")
	gap := p.width - lipgloss.Width(leftStr) - lipgloss.Width(rightStr)
	if gap < 1 {
		gap = 1
	}
	b.WriteString(leftStr + strings.Repeat(" ", gap) + rightStr)

	return b.String()
}

func (p Page) renderDelete() string {
	if p.cursor >= len(p.bookmarks) {
		return p.renderList()
	}
	bm := p.bookmarks[p.cursor]

	boldStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)
	redStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Red)

	// Box width: min(56, width-4), never less than 34.
	bw := p.width - 4
	if bw > 56 {
		bw = 56
	}
	if bw < 34 {
		bw = 34
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

	// Truncate name to fit inside the dialog.
	// The question is: Are you sure you want to delete "NAME"?
	// Max name width = iw - len(` "…"?`) - len(` Are you sure you want to delete `)
	// Simplest: truncate name to iw-4 and let it wrap naturally.
	nameMaxWidth := iw - 4
	if nameMaxWidth < 4 {
		nameMaxWidth = 4
	}
	nameRunes := []rune(bm.Name)
	displayName := bm.Name
	if len(nameRunes) > nameMaxWidth {
		displayName = string(nameRunes[:nameMaxWidth-1]) + "…"
	}

	line1 := ` Are you sure you want to delete`
	line2 := fmt.Sprintf(` "%s"?`, displayName)

	buttons := fmt.Sprintf("  %s  %s",
		boldStyle.Render("[Yes]"),
		dimStyle.Render("[No]"),
	)

	lines := []string{
		topBorder,
		pad(" " + redStyle.Render("Delete Bookmark")),
		pad(""),
		pad(line1),
		pad(line2),
		pad(""),
		pad(buttons),
		pad(dimStyle.Render("  y yes  n no  esc cancel")),
		botBorder,
	}

	leftPad := (p.width - bw) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	prefix := strings.Repeat(" ", leftPad)
	for i, l := range lines {
		lines[i] = prefix + l
	}

	topPad := (p.height - len(lines)) / 2
	if topPad < 0 {
		topPad = 0
	}

	var b strings.Builder
	for i := 0; i < topPad; i++ {
		b.WriteRune('\n')
	}
	b.WriteString(strings.Join(lines, "\n"))
	rendered := topPad + len(lines)
	for i := rendered; i < p.height; i++ {
		b.WriteRune('\n')
	}
	return b.String()
}
