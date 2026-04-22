package bookmarks

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// CreateDialog is the "create bookmark" overlay shown when the user presses
// Ctrl+B. The caller should persist the new bookmark when IsSaved() is true.
type CreateDialog struct {
	path   string
	input  string
	cursor int
	closed bool
	saved  bool
}

// NewCreateDialog creates a CreateDialog for the given path.
func NewCreateDialog(path string) CreateDialog {
	return CreateDialog{path: path}
}

// IsClosed reports whether the dialog has been dismissed.
func (d CreateDialog) IsClosed() bool { return d.closed }

// IsSaved reports whether the dialog was closed with OK (Enter).
func (d CreateDialog) IsSaved() bool { return d.saved }

// Name returns the entered bookmark name.
func (d CreateDialog) Name() string { return d.input }

// BookmarkPath returns the path being bookmarked.
func (d CreateDialog) BookmarkPath() string { return d.path }

// Update processes a message and returns the updated dialog.
func (d CreateDialog) Update(msg tea.Msg) (CreateDialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			d.closed = true
		case "enter":
			if len([]rune(strings.TrimSpace(d.input))) >= 1 {
				d.closed = true
				d.saved = true
			}
		case "backspace":
			d.input, d.cursor = deleteRuneAt(d.input, d.cursor)
		case "left":
			if d.cursor > 0 {
				d.cursor--
			}
		case "right":
			if d.cursor < len([]rune(d.input)) {
				d.cursor++
			}
		case "ctrl+a":
			d.cursor = 0
		case "ctrl+e":
			d.cursor = len([]rune(d.input))
		default:
			if msg.Text != "" {
				d.input = insertTextAt(d.input, d.cursor, msg.Text)
				d.cursor += len([]rune(msg.Text))
			}
		}
	}
	return d, nil
}

// Render produces the dialog string for the given terminal dimensions.
func (d CreateDialog) Render(width, height int) string {
	boldStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)
	cursorStyle := lipgloss.NewStyle().Reverse(true)

	// Box width: min(60, width-4), never less than 34.
	bw := width - 4
	if bw > 60 {
		bw = 60
	}
	if bw < 34 {
		bw = 34
	}
	iw := bw - 2 // inner width (inside │ borders)

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

	// Path display (truncated from left if necessary)
	const pathLabel = " Path: "
	pathWidth := iw - len(pathLabel)
	if pathWidth < 1 {
		pathWidth = 1
	}
	pathRunes := []rune(d.path)
	displayPath := d.path
	if len(pathRunes) > pathWidth {
		displayPath = "…" + string(pathRunes[len(pathRunes)-pathWidth+1:])
	}

	// Name input field
	const nameLabel = " Name: "
	fieldWidth := iw - len(nameLabel)
	if fieldWidth < 10 {
		fieldWidth = 10
	}
	nameField := renderInputField(d.input, d.cursor, fieldWidth, cursorStyle)

	// Buttons: OK is highlighted only when there is input.
	var okRendered string
	if len([]rune(strings.TrimSpace(d.input))) >= 1 {
		okRendered = boldStyle.Reverse(true).Render(" OK ")
	} else {
		okRendered = dimStyle.Render(" OK ")
	}
	cancelRendered := dimStyle.Render(" Cancel ")
	buttons := "  " + okRendered + "  " + cancelRendered

	lines := []string{
		topBorder,
		pad(" " + boldStyle.Render("Create Bookmark")),
		pad(""),
		pad(pathLabel + displayPath),
		pad(""),
		pad(nameLabel + nameField),
		pad(""),
		divider,
		pad(buttons),
		pad(dimStyle.Render("  Enter save  Esc cancel")),
		botBorder,
	}

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

// deleteRuneAt removes the rune immediately before pos.
func deleteRuneAt(s string, pos int) (string, int) {
	if pos <= 0 {
		return s, 0
	}
	runes := []rune(s)
	return string(runes[:pos-1]) + string(runes[pos:]), pos - 1
}

// insertTextAt inserts text into s at rune position pos.
func insertTextAt(s string, pos int, text string) string {
	runes := []rune(s)
	return string(runes[:pos]) + text + string(runes[pos:])
}

// renderInputField renders a fixed-width text input with a cursor at cursorPos.
func renderInputField(value string, cursorPos int, fieldWidth int, cursorSt lipgloss.Style) string {
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
		if i == relCursor {
			sb.WriteString(cursorSt.Render(ch))
		} else {
			sb.WriteString(ch)
		}
	}
	return sb.String()
}
