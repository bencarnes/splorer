package opener

import "os/exec"

// OpenFileWith launches program with path as its sole argument, non-blocking.
// The caller is responsible for supplying a valid executable name or path.
// cmd.Start is used (not Run) so the TUI remains responsive.
func OpenFileWith(path, program string) error {
	cmd := exec.Command(program, path)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}
