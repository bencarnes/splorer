package contentsearch

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// collect runs the walker against rootDir with the given options and returns
// all results produced, regardless of batch boundaries. Fails the test on
// matcher-build errors.
func collect(t *testing.T, rootDir string, opts Options) []Result {
	t.Helper()
	m, err := buildMatcher(opts)
	if err != nil {
		t.Fatalf("buildMatcher: %v", err)
	}
	ch := make(chan resultBatchMsg, 64)
	done := make(chan struct{})
	go func() {
		runContentSearch(context.Background(), rootDir, opts, m, 1, ch)
		close(done)
	}()
	var all []Result
	for batch := range ch {
		all = append(all, batch.results...)
	}
	<-done
	return all
}

func TestWalker_FindsSubstringMatch(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "hello world\nanother line\n")
	writeFile(t, dir, "b.txt", "nothing relevant\n")

	results := collect(t, dir, Options{Pattern: "hello", Mode: ModeExact})
	if len(results) != 1 {
		t.Fatalf("expected 1 match, got %d: %+v", len(results), results)
	}
	if results[0].LineNum != 1 || !strings.Contains(results[0].LineText, "hello") {
		t.Errorf("unexpected result: %+v", results[0])
	}
}

func TestWalker_IgnoresCase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "HELLO world\n")

	sensitive := collect(t, dir, Options{Pattern: "hello", Mode: ModeExact})
	if len(sensitive) != 0 {
		t.Errorf("case-sensitive mode should not match HELLO, got %v", sensitive)
	}

	insensitive := collect(t, dir, Options{Pattern: "hello", Mode: ModeExact, IgnoreCase: true})
	if len(insensitive) != 1 {
		t.Errorf("case-insensitive mode should match HELLO, got %v", insensitive)
	}
}

func TestWalker_RegexMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "func Foo() {}\nfunc Bar(x int) {}\ntype T struct{}\n")

	results := collect(t, dir, Options{Pattern: `^func\s+\w+\(`, Mode: ModeRegex})
	if len(results) != 2 {
		t.Fatalf("expected 2 regex matches, got %d: %+v", len(results), results)
	}
}

// Files with a NUL byte in the first 8 KB must be treated as binary and
// skipped, so searching "the" in a binary that happens to contain that word
// must produce zero matches.
func TestWalker_SkipsBinaryFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "binary.dat", "the\x00quick brown fox\n")
	writeFile(t, dir, "text.txt", "the quick brown fox\n")

	results := collect(t, dir, Options{Pattern: "the", Mode: ModeExact})
	if len(results) != 1 {
		t.Fatalf("expected 1 match (text only), got %d: %+v", len(results), results)
	}
	if !strings.HasSuffix(results[0].RelPath, "text.txt") {
		t.Errorf("match came from binary file: %+v", results[0])
	}
}

// Files larger than maxFileSize must be skipped even if they match.
func TestWalker_SkipsOversizeFiles(t *testing.T) {
	dir := t.TempDir()
	big := bytes.Repeat([]byte("a"), int(maxFileSize)+1)
	writeFile(t, dir, "huge.txt", string(big))
	writeFile(t, dir, "small.txt", "aaa\n")

	results := collect(t, dir, Options{Pattern: "aaa", Mode: ModeExact})
	if len(results) != 1 {
		t.Fatalf("expected 1 match (small.txt only), got %d", len(results))
	}
	if !strings.HasSuffix(results[0].RelPath, "small.txt") {
		t.Errorf("match came from oversize file: %+v", results[0])
	}
}

// With an extension filter set, files that don't match the filter must be
// skipped entirely — matches inside them are invisible to the search.
func TestWalker_ExtensionFilter(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.go", "target\n")
	writeFile(t, dir, "b.md", "target\n")
	writeFile(t, dir, "c.txt", "target\n")

	results := collect(t, dir, Options{
		Pattern:    "target",
		Mode:       ModeExact,
		Extensions: ".go,.md",
	})
	if len(results) != 2 {
		t.Fatalf("expected 2 matches (.go and .md), got %d: %+v", len(results), results)
	}
	for _, r := range results {
		ext := filepath.Ext(r.RelPath)
		if ext != ".go" && ext != ".md" {
			t.Errorf("unexpected result extension %s: %+v", ext, r)
		}
	}
}

// The walker must not follow symlinks. On Windows, mklink requires admin
// privileges in many setups, so we skip there.
func TestWalker_SkipsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation on Windows typically requires privileges we " +
			"don't assume in a test environment")
	}

	dir := t.TempDir()
	// Real file that matches.
	writeFile(t, dir, "real.txt", "target\n")
	// A symlink pointing at the real file.
	if err := os.Symlink(filepath.Join(dir, "real.txt"), filepath.Join(dir, "link.txt")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	results := collect(t, dir, Options{Pattern: "target", Mode: ModeExact})
	// Only real.txt should produce a match; the symlink must be skipped
	// instead of giving a duplicate result.
	if len(results) != 1 {
		t.Fatalf("expected 1 match (real only), got %d: %+v", len(results), results)
	}
	if !strings.HasSuffix(results[0].RelPath, "real.txt") {
		t.Errorf("unexpected match: %+v", results[0])
	}
}

// Helpers.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
