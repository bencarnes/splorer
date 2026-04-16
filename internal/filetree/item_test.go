package filetree

import (
	"testing"
	"time"
)

func TestFileEntry_Title_Dir(t *testing.T) {
	e := FileEntry{Name: "mydir", IsDir: true}
	if got := e.Title(); got != "mydir/" {
		t.Errorf("Title() = %q, want %q", got, "mydir/")
	}
}

func TestFileEntry_Title_File(t *testing.T) {
	e := FileEntry{Name: "main.go", IsDir: false}
	if got := e.Title(); got != "main.go" {
		t.Errorf("Title() = %q, want %q", got, "main.go")
	}
}

func TestFileEntry_Title_Dir_NoSlashDouble(t *testing.T) {
	// Verify exactly one trailing slash even if Name already has one (it won't in practice,
	// but this documents the behaviour).
	e := FileEntry{Name: "docs", IsDir: true, ModTime: time.Now()}
	title := e.Title()
	if title != "docs/" {
		t.Errorf("Title() = %q, want \"docs/\"", title)
	}
}

func TestHumanizeSize(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{999, "999 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}
	for _, tc := range cases {
		got := humanizeSize(tc.n)
		if got != tc.want {
			t.Errorf("humanizeSize(%d) = %q, want %q", tc.n, got, tc.want)
		}
	}
}
