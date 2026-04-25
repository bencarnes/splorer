package filetree

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
)

const watchInterval = time.Second

// DirChangedMsg is sent when the watcher polls the current directory and its
// contents differ from the last known snapshot. Entries is nil on a transient
// read error; the model reschedules without updating in that case.
type DirChangedMsg struct {
	Dir       string
	SortOrder SortOrder
	Entries   []FileEntry
}

// DirGoneMsg is sent when the watched directory no longer exists on disk.
type DirGoneMsg struct{ Dir string }

// WatchCmd returns a one-shot command that sleeps watchInterval, reads the
// current directory, and sends DirChangedMsg or DirGoneMsg. Callers chain it
// from Update to keep watching continuously.
func (m Model) WatchCmd() tea.Cmd {
	return watchDirOnce(m.cwd, m.sortOrder)
}

func watchDirOnce(dir string, so SortOrder) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(watchInterval)
		entries, err := loadDir(dir, so)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return DirGoneMsg{Dir: dir}
			}
			// Transient error — reschedule without an update.
			return DirChangedMsg{Dir: dir, SortOrder: so}
		}
		return DirChangedMsg{Dir: dir, SortOrder: so, Entries: entries}
	}
}

// applyEntryRefresh replaces m.entries, preserving the cursor on the same
// entry name when possible and clamping it otherwise. Multi-selection entries
// that no longer exist are pruned silently.
func (m Model) applyEntryRefresh(newEntries []FileEntry) Model {
	var prevName string
	if m.cursor < len(m.entries) {
		prevName = m.entries[m.cursor].Name
	}
	m.entries = newEntries

	// Drop selections for entries that have disappeared.
	if len(m.selected) > 0 {
		alive := make(map[string]bool, len(m.selected))
		for _, e := range newEntries {
			if m.selected[e.Path] {
				alive[e.Path] = true
			}
		}
		if len(alive) == 0 {
			m.selected = nil
		} else {
			m.selected = alive
		}
	}

	if prevName != "" {
		for i, e := range newEntries {
			if e.Name == prevName {
				m.cursor = i
				lh := m.listHeight()
				if m.cursor < m.offset {
					m.offset = m.cursor
				} else if m.cursor >= m.offset+lh {
					m.offset = m.cursor - lh + 1
				}
				return m
			}
		}
	}
	// Previously selected entry is gone; clamp.
	if m.cursor >= len(newEntries) {
		if len(newEntries) > 0 {
			m.cursor = len(newEntries) - 1
		} else {
			m.cursor = 0
		}
	}
	if m.offset > m.cursor {
		m.offset = m.cursor
	}
	return m
}

// nearestExistingAncestor returns the closest ancestor of path that can be
// stat'd. Used when the watched directory disappears.
func nearestExistingAncestor(path string) string {
	p := filepath.Dir(path)
	for {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(p)
		if parent == p {
			return p // reached filesystem root
		}
		p = parent
	}
}

// entriesEqual reports whether two slices are identical in name, IsDir, size
// and mtime — sufficient to detect any visible change.
func entriesEqual(a, b []FileEntry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name || a[i].IsDir != b[i].IsDir ||
			a[i].Size != b[i].Size || !a[i].ModTime.Equal(b[i].ModTime) {
			return false
		}
	}
	return true
}
