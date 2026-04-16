package filetree

import (
	"fmt"
	"io/fs"
	"time"
)

// FileEntry represents a single file or directory in the listing.
type FileEntry struct {
	Name    string
	Path    string // absolute path
	IsDir   bool
	Size    int64
	ModTime time.Time
	Mode    fs.FileMode
}

// Title returns the display name: directories get a trailing slash.
func (f FileEntry) Title() string {
	if f.IsDir {
		return f.Name + "/"
	}
	return f.Name
}

// humanizeSize formats a byte count as a human-readable string.
func humanizeSize(n int64) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.1f GB", float64(n)/(1024*1024*1024))
	}
}
