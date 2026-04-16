package opener

import "os/exec"

// OpenFile launches the system default application for the file at path.
// It uses xdg-open which works across GNOME, KDE, XFCE, and other desktops.
// cmd.Start is used (not Run) so the TUI remains responsive.
func OpenFile(path string) error {
	return OpenFileWith(path, "xdg-open")
}

// OpenFileWith launches program with path as its sole argument, non-blocking.
// The caller is responsible for supplying a valid executable name or path.
func OpenFileWith(path, program string) error {
	cmd := exec.Command(program, path)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}
