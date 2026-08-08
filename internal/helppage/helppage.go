// Package helppage renders a single full-screen help overlay. It documents
// typeahead and multi-selection — every other binding in splorer is
// discoverable from the per-view footer hints.
package helppage

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// Page is the help overlay model. It has no internal state beyond a closed
// flag; pressing any key dismisses it.
type Page struct {
	closed bool
}

// New returns a fresh help page.
func New() Page { return Page{} }

// IsClosed reports whether the page has been dismissed.
func (p Page) IsClosed() bool { return p.closed }

// Update closes the page on any key press or mouse click.
func (p Page) Update(msg tea.Msg) (Page, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyPressMsg, tea.MouseClickMsg:
		p.closed = true
	}
	return p, nil
}

// Render produces the full-screen help body for the given dimensions.
func (p Page) Render(width, height int) string {
	headerStyle := lipgloss.NewStyle().Bold(true)
	sectionStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Yellow)
	dimStyle := lipgloss.NewStyle().Faint(true)
	sepStyle := lipgloss.NewStyle().Faint(true)

	sep := sepStyle.Render(strings.Repeat("─", width))

	row := func(key, desc string) string {
		const keyCol = 22
		k := keyStyle.Render(key)
		pad := keyCol - lipgloss.Width(k)
		if pad < 1 {
			pad = 1
		}
		return "  " + k + strings.Repeat(" ", pad) + desc
	}

	lines := []string{
		headerStyle.Render(" Help"),
		sep,
		"",
		"  " + sectionStyle.Render("Typeahead — jump to a file by name"),
		"",
		"  Start typing a name and the cursor jumps to the first entry that",
		"  starts with it (case-insensitive). The prefix so far shows in the",
		"  bottom-left corner and expires a second after your last keystroke.",
		"",
		row("the same letter again", "cycle to the next entry starting with it"),
		row("Esc", "cancel the prefix — Esc again quits splorer"),
		"",
		"  " + dimStyle.Render("Every printable key types, so the file tree has no single-letter"),
		"  " + dimStyle.Render("bindings: navigate with ↑↓, PgUp/PgDn and Home/End, open with"),
		"  " + dimStyle.Render("Enter/→, go up with Backspace/←, and jump home with Alt+Home."),
		"",
		"  " + sectionStyle.Render("Multi-selection"),
		"",
		"  Delete, Copy and Cut act on the rows marked ● — or on the cursor's",
		"  row when nothing is marked.",
		"",
		row("Click", "select that row only (resets the selection)"),
		row("Shift+Click", "extend the selection from the anchor to the click"),
		row("Ctrl+Click", "toggle that row in/out of the selection"),
		row("Space", "toggle the cursor's row (types a space mid-prefix)"),
		row("Shift+↑ / Shift+↓", "move cursor and extend selection from the anchor"),
		row("Shift+PgUp / PgDn", "page-extend the selection from the anchor"),
		"",
		"  " + dimStyle.Render("Most terminals keep Shift+Click and Ctrl+Click for their own text"),
		"  " + dimStyle.Render("selection and never forward them; the Space and Shift+arrow fallbacks"),
		"  " + dimStyle.Render("always work. The \"anchor\" is the row you last clicked or toggled."),
		"",
		row("Delete", "delete the selection (with confirmation)"),
		row("Ctrl+C / Ctrl+X", "copy / cut the selection to the clipboard"),
		row("Ctrl+V", "paste the clipboard into the current directory"),
		row("F2", "rename (only with a single entry selected)"),
		"",
		"  The same operations are on the Manipulate menu (" + keyStyle.Render("Alt+M") + ").",
	}

	// Pad up to the available height minus a footer line.
	bodyHeight := height - 2
	if bodyHeight < len(lines) {
		bodyHeight = len(lines)
	}

	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l)
		b.WriteRune('\n')
	}
	for i := len(lines); i < bodyHeight; i++ {
		b.WriteRune('\n')
	}
	b.WriteString(sep)
	b.WriteRune('\n')
	footer := dimStyle.Render(" Press any key to close")
	b.WriteString(footer)

	return b.String()
}
