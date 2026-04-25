// Package helppage renders a single full-screen help overlay. Today it only
// documents multi-selection — every other binding in splorer is discoverable
// from the per-view footer hints.
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
		"  " + sectionStyle.Render("Multi-selection"),
		"",
		"  Most file-manipulation actions (Delete, Copy, Cut) operate on the",
		"  set of multi-selected entries — or on the cursor's row if there is",
		"  no explicit selection. Selected rows are marked with a yellow ●.",
		"",
		"  " + sectionStyle.Render("Mouse"),
		row("Click", "select that row only (resets the multi-selection)"),
		row("Shift+Click", "extend the selection from the anchor to the click"),
		row("Ctrl+Click", "toggle that row in/out of the selection"),
		"",
		"  " + dimStyle.Render("Note: most terminals (xterm, gnome-terminal, iTerm2,"),
		"  " + dimStyle.Render("Windows Terminal, …) reserve Shift+Click and Ctrl+Click"),
		"  " + dimStyle.Render("for their own text-selection and never forward them to"),
		"  " + dimStyle.Render("splorer. If those don't work in your terminal, use the"),
		"  " + dimStyle.Render("keyboard fallbacks below — they always work."),
		"",
		"  " + sectionStyle.Render("Keyboard"),
		row("Space", "toggle the cursor's row in/out of the selection"),
		row("Shift+↑ / Shift+↓", "move cursor and extend selection from the anchor"),
		row("Shift+PgUp / PgDn", "page-extend the selection from the anchor"),
		"",
		"  " + dimStyle.Render("The \"anchor\" is whichever row you most recently single-"),
		"  " + dimStyle.Render("clicked or toggled with Space. A plain click or navigating"),
		"  " + dimStyle.Render("into another directory clears the anchor and selection."),
		"",
		"  " + sectionStyle.Render("Manipulating the selection"),
		row("Delete", "delete the selection (with confirmation)"),
		row("Ctrl+C", "copy the selection to the clipboard"),
		row("Ctrl+X", "cut the selection to the clipboard"),
		row("Ctrl+V", "paste the clipboard into the current directory"),
		"",
		"  The same four operations are available from the Manipulate menu",
		"  (" + keyStyle.Render("Alt+M") + ") if you prefer to drive them from the menu bar.",
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
