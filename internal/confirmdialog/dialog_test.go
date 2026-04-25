package confirmdialog

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// drive feeds a sequence of key presses (described as strings handled by
// Update's switch) through the dialog and returns the final state.
func drive(t *testing.T, d Dialog, keys ...tea.KeyPressMsg) Dialog {
	t.Helper()
	for _, k := range keys {
		d, _ = d.Update(k)
	}
	return d
}

func TestNew_DefaultsToCancel(t *testing.T) {
	d := New("Delete", "delete 3 items")
	if d.IsClosed() {
		t.Error("dialog should not start closed")
	}
	if d.IsConfirmed() {
		t.Error("dialog should not start confirmed")
	}
	if d.cursor != 1 {
		t.Errorf("default cursor = %d, want 1 (Cancel)", d.cursor)
	}
}

func TestEsc_CancelsWithoutConfirming(t *testing.T) {
	d := drive(t, New("Delete", "delete 1 item"), tea.KeyPressMsg{Code: tea.KeyEsc})
	if !d.IsClosed() {
		t.Error("Esc should close the dialog")
	}
	if d.IsConfirmed() {
		t.Error("Esc should not confirm")
	}
}

func TestN_CancelsImmediately(t *testing.T) {
	d := drive(t, New("Delete", "x"), tea.KeyPressMsg{Code: 'n', Text: "n"})
	if !d.IsClosed() || d.IsConfirmed() {
		t.Errorf("n should cancel without confirming; closed=%v confirmed=%v", d.IsClosed(), d.IsConfirmed())
	}
}

func TestY_ConfirmsImmediately(t *testing.T) {
	d := drive(t, New("Delete", "x"), tea.KeyPressMsg{Code: 'y', Text: "y"})
	if !d.IsClosed() || !d.IsConfirmed() {
		t.Errorf("y should confirm; closed=%v confirmed=%v", d.IsClosed(), d.IsConfirmed())
	}
}

func TestEnter_OnCancelDoesNotConfirm(t *testing.T) {
	// New starts on Cancel, so plain Enter should cancel.
	d := drive(t, New("Delete", "x"), tea.KeyPressMsg{Code: tea.KeyEnter})
	if !d.IsClosed() {
		t.Error("Enter should close the dialog")
	}
	if d.IsConfirmed() {
		t.Error("Enter on Cancel button must not confirm")
	}
}

func TestLeftArrow_MovesToOK_ThenEnterConfirms(t *testing.T) {
	d := drive(t,
		New("Delete", "x"),
		tea.KeyPressMsg{Code: tea.KeyLeft},
		tea.KeyPressMsg{Code: tea.KeyEnter},
	)
	if !d.IsConfirmed() {
		t.Error("Left + Enter should confirm")
	}
}

func TestRender_IncludesSummary(t *testing.T) {
	d := New("Delete", "delete 3 items")
	out := d.Render(80, 24)
	if !strings.Contains(out, "delete 3 items") {
		t.Errorf("rendered dialog missing summary; got:\n%s", out)
	}
	if !strings.Contains(out, "Are you sure you want to") {
		t.Errorf("rendered dialog missing prompt prefix; got:\n%s", out)
	}
}
