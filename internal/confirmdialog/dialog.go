// Package confirmdialog provides a generic OK/Cancel confirmation overlay
// used to gate destructive or otherwise irreversible operations.
package confirmdialog

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
)

// Dialog is a small modal asking the user to confirm an action described by
// its summary. The caller checks IsConfirmed after IsClosed returns true to
// know whether to proceed.
type Dialog struct {
	title   string
	summary string
	cursor  int // 0 = OK, 1 = Cancel
	closed  bool
	confirmed bool
}

// New returns a confirmation dialog with the given title (e.g. "Delete") and
// summary (e.g. `delete 3 items`). The confirmation prompt is rendered as
// `Are you sure you want to <summary>?`. Cursor starts on Cancel for safety.
func New(title, summary string) Dialog {
	return Dialog{title: title, summary: summary, cursor: 1}
}

// IsClosed reports whether the dialog has been dismissed.
func (d Dialog) IsClosed() bool { return d.closed }

// IsConfirmed reports whether the user pressed OK.
func (d Dialog) IsConfirmed() bool { return d.confirmed }

// Update processes a Bubble Tea message.
func (d Dialog) Update(msg tea.Msg) (Dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			d.closed = true
		case "enter":
			d.confirmed = d.cursor == 0
			d.closed = true
		case "left", "h":
			d.cursor = 0
		case "right", "l":
			d.cursor = 1
		case "tab":
			d.cursor = (d.cursor + 1) % 2
		case "y", "Y":
			d.confirmed = true
			d.closed = true
		case "n", "N":
			d.confirmed = false
			d.closed = true
		}
	}
	return d, nil
}

// Render produces the dialog string for the given terminal dimensions.
func (d Dialog) Render(width, height int) string {
	boldStyle := lipgloss.NewStyle().Bold(true)
	dimStyle := lipgloss.NewStyle().Faint(true)
	highlightStyle := lipgloss.NewStyle().Bold(true).Reverse(true)
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

	// Wrap the question across two lines if it overflows the inner width.
	question := "Are you sure you want to " + d.summary + "?"
	questionLines := wrapText(question, iw-2)

	var okBtn, cancelBtn string
	if d.cursor == 0 {
		okBtn = highlightStyle.Render(" OK ")
		cancelBtn = dimStyle.Render(" Cancel ")
	} else {
		okBtn = boldStyle.Render(" OK ")
		cancelBtn = highlightStyle.Render(" Cancel ")
	}
	buttons := "  " + okBtn + "  " + cancelBtn

	lines := []string{
		topBorder,
		pad(" " + titleStyle.Render(d.title)),
		pad(""),
	}
	for _, ql := range questionLines {
		lines = append(lines, pad(" "+ql))
	}
	lines = append(lines,
		pad(""),
		divider,
		pad(buttons),
		pad(dimStyle.Render("  ←/→ select  Enter confirm  y yes  n/Esc cancel")),
		botBorder,
	)

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

// wrapText breaks s into lines of at most width display cells, splitting on
// spaces. Words longer than width are emitted on their own line and may
// exceed it; padding is the caller's responsibility.
func wrapText(s string, width int) []string {
	if width < 1 {
		width = 1
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if lipgloss.Width(cur)+1+lipgloss.Width(w) <= width {
			cur += " " + w
		} else {
			lines = append(lines, cur)
			cur = w
		}
	}
	lines = append(lines, cur)
	return lines
}
