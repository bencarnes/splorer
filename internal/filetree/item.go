package filetree

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
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

// Icon returns an emoji representing the file type.
func (f FileEntry) Icon() string {
	if f.IsDir {
		return "📁"
	}
	switch strings.ToLower(filepath.Ext(f.Name)) {
	case ".go", ".py", ".js", ".ts", ".jsx", ".tsx", ".java", ".c", ".cpp", ".h",
		".rs", ".rb", ".php", ".swift", ".kt", ".cs", ".sh", ".bash", ".zsh", ".fish",
		".lua", ".vim", ".el", ".clj", ".hs", ".ml", ".ex", ".exs":
		return "💻"
	case ".txt", ".md", ".rst", ".doc", ".docx", ".odt", ".rtf":
		return "📝"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".bmp", ".ico", ".webp", ".tiff", ".tif":
		return "🖼"
	case ".mp3", ".flac", ".ogg", ".wav", ".aac", ".m4a", ".opus":
		return "🎵"
	case ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v":
		return "🎬"
	case ".zip", ".tar", ".gz", ".bz2", ".xz", ".7z", ".rar", ".zst":
		return "📦"
	case ".json", ".yaml", ".yml", ".toml", ".ini", ".conf", ".cfg", ".env":
		return "🔧"
	case ".pdf":
		return "📑"
	case ".csv", ".sql", ".db", ".sqlite", ".sqlite3":
		return "📊"
	default:
		return "📄"
	}
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
