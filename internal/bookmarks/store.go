// Package bookmarks manages named bookmarks to files and directories.
// Bookmarks are persisted as JSON at ~/.config/splorer/bookmarks.json.
package bookmarks

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Bookmark is a named pointer to a file or directory path.
type Bookmark struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// configPath returns the absolute path to the bookmarks config file.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "splorer", "bookmarks.json"), nil
}

// Load reads saved bookmarks from disk. Returns an empty slice when the file
// does not exist yet (first run).
func Load() ([]Bookmark, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []Bookmark{}, nil
	}
	if err != nil {
		return nil, err
	}
	var bmarks []Bookmark
	if err := json.Unmarshal(data, &bmarks); err != nil {
		return nil, err
	}
	return bmarks, nil
}

// Save writes bookmarks to disk, creating the config directory if needed.
func Save(bmarks []Bookmark) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(bmarks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
