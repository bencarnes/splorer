// Package renamedialog provides the modal overlay used by the Manipulate →
// Rename operation. The dialog pre-populates its input with the entry's
// current basename and reports the new name back to the app on save.
package renamedialog

import (
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// Dialog is the rename overlay. The caller observes IsClosed/IsSaved and
// reads NewName when IsSaved is true.
type Dialog struct {
	path   string
	input  string
	cursor int
	closed bool
	saved  bool
}

// New returns a Dialog targeting path. The input is pre-populated with the
// entry's current basename and the cursor is placed at the end so the user
// can immediately edit or type to overwrite (the input field has no built-in
// "select all", so this matches what most users expect from rename).
func New(path string) Dialog {
	name := filepath.Base(path)
	return Dialog{
		path:   path,
		input:  name,
		cursor: len([]rune(name)),
	}
}

// IsClosed reports whether the dialog has been dismissed.
func (d Dialog) IsClosed() bool { return d.closed }

// IsSaved reports whether the dialog was closed via OK (Enter) with a name
// that differs from the current basename.
func (d Dialog) IsSaved() bool { return d.saved }

// NewName returns the trimmed user-entered name.
func (d Dialog) NewName() string { return strings.TrimSpace(d.input) }

// Path returns the path being renamed.
func (d Dialog) Path() string { return d.path }

// Update processes a Bubble Tea message.
func (d Dialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			d.closed = true
		case "enter":
			trimmed := strings.TrimSpace(d.input)
			if trimmed == "" || trimmed == filepath.Base(d.path) {
				return d, nil
			}
			d.closed = true
			d.saved = true
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
func (d Dialog) Render(width, height int) string {
	boldStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)
	cursorStyle := lipgloss.NewStyle().Reverse(true)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Red)

	bw := width - 4
	if bw > 60 {
		bw = 60
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
	divider := "├" + strings.Repeat("─", iw) + "┤"

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

	const nameLabel = " Name: "
	fieldWidth := iw - len(nameLabel)
	if fieldWidth < 10 {
		fieldWidth = 10
	}
	nameField := renderInputField(d.input, d.cursor, fieldWidth, cursorStyle)

	trimmed := strings.TrimSpace(d.input)
	canSave := trimmed != "" && trimmed != filepath.Base(d.path)

	var okRendered string
	if canSave {
		okRendered = boldStyle.Reverse(true).Render(" OK ")
	} else {
		okRendered = dimStyle.Render(" OK ")
	}
	cancelRendered := dimStyle.Render(" Cancel ")
	buttons := "  " + okRendered + "  " + cancelRendered

	lines := []string{
		topBorder,
		pad(" " + titleStyle.Render("Rename")),
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

	leftPad := (width - bw) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	prefix := strings.Repeat(" ", leftPad)
	for i, l := range lines {
		lines[i] = prefix + l
	}

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

func deleteRuneAt(s string, pos int) (string, int) {
	if pos <= 0 {
		return s, 0
	}
	runes := []rune(s)
	return string(runes[:pos-1]) + string(runes[pos:]), pos - 1
}

func insertTextAt(s string, pos int, text string) string {
	runes := []rune(s)
	return string(runes[:pos]) + text + string(runes[pos:])
}

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
