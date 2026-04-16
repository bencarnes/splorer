package associations

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// focusArea identifies which part of the dialog has keyboard focus.
type focusArea int

const (
	focusList focusArea = iota
	focusExt
	focusProg
)

// Dialog is the Openers file-association manager overlay.
// It operates on a working copy of the associations map; the caller is
// responsible for persisting the result when IsClosed() returns true.
type Dialog struct {
	assocs     map[string]string
	keys       []string // sorted slice of assocs keys; rebuilt after every mutation
	cursor     int      // selected row in the list
	listOffset int      // scroll offset for the list

	extInput   string
	extCursor  int
	progInput  string
	progCursor int

	focus  focusArea
	closed bool
}

// NewDialog creates a Dialog initialised with a deep copy of assocs.
func NewDialog(assocs map[string]string) Dialog {
	d := Dialog{
		assocs: make(map[string]string, len(assocs)),
	}
	for k, v := range assocs {
		d.assocs[k] = v
	}
	d = d.rebuildKeys()
	return d
}

// IsClosed reports whether the dialog has been dismissed.
func (d Dialog) IsClosed() bool { return d.closed }

// Assocs returns a copy of the current associations map.
func (d Dialog) Assocs() map[string]string {
	result := make(map[string]string, len(d.assocs))
	for k, v := range d.assocs {
		result[k] = v
	}
	return result
}

// rebuildKeys rebuilds the sorted keys slice from assocs and clamps the cursor.
func (d Dialog) rebuildKeys() Dialog {
	d.keys = make([]string, 0, len(d.assocs))
	for k := range d.assocs {
		d.keys = append(d.keys, k)
	}
	sort.Strings(d.keys)
	if len(d.keys) == 0 {
		d.cursor = 0
		d.listOffset = 0
	} else if d.cursor >= len(d.keys) {
		d.cursor = len(d.keys) - 1
	}
	return d
}

// Update processes a Bubble Tea message and returns the updated Dialog.
func (d Dialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch d.focus {
		case focusList:
			return d.updateList(msg)
		case focusExt:
			return d.updateExt(msg)
		case focusProg:
			return d.updateProg(msg)
		}
	}
	return d, nil
}

func (d Dialog) updateList(msg tea.KeyPressMsg) (Dialog, tea.Cmd) {
	switch msg.String() {
	case "esc":
		d.closed = true
	case "up", "k":
		if d.cursor > 0 {
			d.cursor--
		}
	case "down", "j":
		if d.cursor < len(d.keys)-1 {
			d.cursor++
		}
	case "d", "delete":
		if len(d.keys) > 0 {
			delete(d.assocs, d.keys[d.cursor])
			d = d.rebuildKeys()
		}
	case "tab":
		d.focus = focusExt
	case "shift+tab":
		d.focus = focusProg
	}
	return d, nil
}

func (d Dialog) updateExt(msg tea.KeyPressMsg) (Dialog, tea.Cmd) {
	switch msg.String() {
	case "esc":
		d.closed = true
	case "tab":
		d.focus = focusProg
	case "shift+tab":
		d.focus = focusList
	case "backspace":
		d.extInput, d.extCursor = deleteRuneAt(d.extInput, d.extCursor)
	case "left":
		if d.extCursor > 0 {
			d.extCursor--
		}
	case "right":
		if d.extCursor < len([]rune(d.extInput)) {
			d.extCursor++
		}
	default:
		if msg.Text != "" {
			d.extInput = insertTextAt(d.extInput, d.extCursor, msg.Text)
			d.extCursor += len([]rune(msg.Text))
		}
	}
	return d, nil
}

func (d Dialog) updateProg(msg tea.KeyPressMsg) (Dialog, tea.Cmd) {
	switch msg.String() {
	case "esc":
		d.closed = true
	case "tab":
		d.focus = focusList
	case "shift+tab":
		d.focus = focusExt
	case "enter":
		ext := strings.TrimSpace(d.extInput)
		prog := strings.TrimSpace(d.progInput)
		if ext != "" && prog != "" {
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			d.assocs[ext] = prog
			d = d.rebuildKeys()
			// position cursor on the newly added entry
			for i, k := range d.keys {
				if k == ext {
					d.cursor = i
					break
				}
			}
			d.extInput = ""
			d.progInput = ""
			d.extCursor = 0
			d.progCursor = 0
			d.focus = focusList
		}
	case "backspace":
		d.progInput, d.progCursor = deleteRuneAt(d.progInput, d.progCursor)
	case "left":
		if d.progCursor > 0 {
			d.progCursor--
		}
	case "right":
		if d.progCursor < len([]rune(d.progInput)) {
			d.progCursor++
		}
	default:
		if msg.Text != "" {
			d.progInput = insertTextAt(d.progInput, d.progCursor, msg.Text)
			d.progCursor += len([]rune(msg.Text))
		}
	}
	return d, nil
}

// Render produces the full dialog string for the given outer dimensions.
// width and height are the available terminal area (excluding any rows above,
// such as the menu bar).
func (d Dialog) Render(width, height int) string {
	// Box inner width (subtract the two │ border characters).
	iw := width - 2
	if iw < 20 {
		iw = 20
	}

	dimStyle := lipgloss.NewStyle().Faint(true)
	boldStyle := lipgloss.NewStyle().Bold(true)
	selectedStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
	cursorStyle := lipgloss.NewStyle().Reverse(true)

	topBorder := "╭" + strings.Repeat("─", iw) + "╮"
	botBorder := "╰" + strings.Repeat("─", iw) + "╯"
	divider := "├" + strings.Repeat("─", iw) + "┤"

	// row pads s to iw and wraps it in │ characters.
	row := func(s string) string {
		w := lipgloss.Width(s)
		if w < iw {
			s += strings.Repeat(" ", iw-w)
		}
		return "│" + s + "│"
	}

	// Fixed chrome line counts: top + title + div + colhdr + empty + div +
	// add-label + form + empty + div + help + bottom = 12 lines.
	const fixedLines = 12
	listRows := height - fixedLines
	if listRows < 1 {
		listRows = 1
	}

	// Adjust list scroll offset so cursor is visible.
	if d.cursor < d.listOffset {
		d.listOffset = d.cursor
	}
	if d.cursor >= d.listOffset+listRows {
		d.listOffset = d.cursor - listRows + 1
	}

	var lines []string
	lines = append(lines, topBorder)
	lines = append(lines, row(" "+boldStyle.Render("Openers")+" — file associations"))
	lines = append(lines, divider)

	// Column headers
	hdr := fmt.Sprintf("  %-14s %s", "Extension", "Program")
	lines = append(lines, row(dimStyle.Render(hdr)))

	// Association list
	if len(d.keys) == 0 {
		lines = append(lines, row(dimStyle.Render("  (no associations)")))
		for i := 1; i < listRows; i++ {
			lines = append(lines, row(""))
		}
	} else {
		end := d.listOffset + listRows
		if end > len(d.keys) {
			end = len(d.keys)
		}
		for i := d.listOffset; i < end; i++ {
			k := d.keys[i]
			indicator := "  "
			if i == d.cursor {
				indicator = "▶ "
			}
			content := fmt.Sprintf("%s%-14s %s", indicator, k, d.assocs[k])
			if i == d.cursor && d.focus == focusList {
				lines = append(lines, row(selectedStyle.Render(content)))
			} else {
				lines = append(lines, row(content))
			}
		}
		// fill remaining list rows with blanks
		rendered := end - d.listOffset
		for i := rendered; i < listRows; i++ {
			lines = append(lines, row(""))
		}
	}

	lines = append(lines, row(""))
	lines = append(lines, divider)

	// Add-association form
	lines = append(lines, row("  "+boldStyle.Render("Add association")))

	extFieldWidth := 14
	progFieldWidth := iw - 14 - 16 // 14 for ext field+label, 16 for prog label
	if progFieldWidth < 10 {
		progFieldWidth = 10
	}
	extField := renderInputField(d.extInput, d.extCursor, d.focus == focusExt, extFieldWidth, cursorStyle)
	progField := renderInputField(d.progInput, d.progCursor, d.focus == focusProg, progFieldWidth, cursorStyle)
	formLine := fmt.Sprintf("  Ext: %s  Program: %s", extField, progField)
	lines = append(lines, row(formLine))
	lines = append(lines, row(""))
	lines = append(lines, divider)

	// Context-sensitive help
	var help string
	switch d.focus {
	case focusList:
		help = "  ↑↓ navigate  d delete  tab → add form  esc close"
	case focusExt:
		help = "  type extension (e.g. .pdf)  tab → program  shift+tab back  esc close"
	case focusProg:
		help = "  type program name  enter add  tab back  shift+tab → ext  esc close"
	}
	lines = append(lines, row(dimStyle.Render(help)))
	lines = append(lines, botBorder)

	result := strings.Join(lines, "\n")

	// Pad with blank lines if the dialog is shorter than the available height.
	rendered := len(lines)
	for i := rendered; i < height; i++ {
		result += "\n"
	}
	return result
}

// renderInputField renders a fixed-width text input field as "[content]".
// The character at cursorPos is highlighted when focused.
func renderInputField(value string, cursorPos int, focused bool, fieldWidth int, cursorSt lipgloss.Style) string {
	runes := []rune(value)

	// Build a window of fieldWidth cells centred around the cursor.
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
		if focused && i == relCursor {
			sb.WriteString(cursorSt.Render(ch))
		} else {
			sb.WriteString(ch)
		}
	}
	return "[" + sb.String() + "]"
}

// insertTextAt inserts text into s at rune position pos.
func insertTextAt(s string, pos int, text string) string {
	runes := []rune(s)
	return string(runes[:pos]) + text + string(runes[pos:])
}

// deleteRuneAt removes the rune immediately before pos and returns the new
// string and adjusted cursor position.
func deleteRuneAt(s string, pos int) (string, int) {
	if pos <= 0 {
		return s, 0
	}
	runes := []rune(s)
	return string(runes[:pos-1]) + string(runes[pos:]), pos - 1
}
