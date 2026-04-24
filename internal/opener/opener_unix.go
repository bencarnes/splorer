//go:build !windows

package opener

// OpenFile launches the system default application for the file at path.
// It uses xdg-open which works across GNOME, KDE, XFCE, and other desktops.
func OpenFile(path string) error {
	return OpenFileWith(path, "xdg-open")
}
