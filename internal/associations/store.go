// Package associations manages file-extension-to-program mappings that
// override the default xdg-open handler when opening files in splorer.
// Associations are persisted as JSON at ~/.config/splorer/openers.json.
package associations

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// configPath returns the absolute path to the openers config file.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "splorer", "openers.json"), nil
}

// Load reads saved associations from disk. Returns an empty map when the file
// does not exist yet (first run). Returns an error only for genuine I/O or
// parse failures.
func Load() (map[string]string, error) {
	path, err := configPath()
	if err != nil {
		return map[string]string{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return map[string]string{}, err
	}
	var assocs map[string]string
	if err := json.Unmarshal(data, &assocs); err != nil {
		return map[string]string{}, err
	}
	return assocs, nil
}

// Save writes associations to disk, creating the config directory if needed.
func Save(assocs map[string]string) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(assocs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
