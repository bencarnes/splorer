package filetree

import (
	"path/filepath"
	"sort"
	"strings"
)

// SortOrder controls how entries are ordered within each group (directories
// always appear before files regardless of sort order).
type SortOrder int

const (
	SortByName SortOrder = iota // alphabetical by name, case-insensitive (default)
	SortByTime                  // modification time, newest first
	SortBySize                  // file size, largest first; ties broken by name
	SortByType                  // file extension, then name within extension
)

// AllSortOrders lists every valid sort order in display order.
var AllSortOrders = []SortOrder{SortByName, SortByTime, SortBySize, SortByType}

// Label returns a short human-readable name for the sort order.
func (so SortOrder) Label() string {
	switch so {
	case SortByName:
		return "Name"
	case SortByTime:
		return "Timestamp"
	case SortBySize:
		return "Size"
	case SortByType:
		return "Type"
	default:
		return "Name"
	}
}

// sortGroup sorts entries in place according to so.
func sortGroup(entries []FileEntry, so SortOrder) {
	switch so {
	case SortByTime:
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].ModTime.After(entries[j].ModTime)
		})
	case SortBySize:
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Size != entries[j].Size {
				return entries[i].Size > entries[j].Size
			}
			return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
		})
	case SortByType:
		sort.SliceStable(entries, func(i, j int) bool {
			extI := strings.ToLower(filepath.Ext(entries[i].Name))
			extJ := strings.ToLower(filepath.Ext(entries[j].Name))
			// Entries without an extension sort last within their group.
			if extI == "" && extJ != "" {
				return false
			}
			if extI != "" && extJ == "" {
				return true
			}
			if extI != extJ {
				return extI < extJ
			}
			return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
		})
	default: // SortByName
		sort.SliceStable(entries, func(i, j int) bool {
			return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
		})
	}
}
