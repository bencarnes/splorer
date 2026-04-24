//go:build windows

package opener

import (
	"os/exec"
	"syscall"
)

// OpenFile launches the system default application for the file at path.
// It uses the Windows "start" shell builtin, invoked via cmd.exe. The empty
// first quoted argument to start is a placeholder window title: without it,
// start would interpret a quoted path as the title and do nothing.
// HideWindow suppresses the transient cmd.exe console that would otherwise
// briefly flash over the TUI.
func OpenFile(path string) error {
	cmd := exec.Command("cmd", "/c", "start", "", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}
