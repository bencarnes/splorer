package contentsearch

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// maxFileSize is the upper bound on files this search will read. Beyond
// ~10 MB the payoff in real-world code search is minimal and the latency
// cost is large.
const maxFileSize int64 = 10 * 1024 * 1024

// binarySampleSize is the prefix we read before line-scanning; if it
// contains a NUL byte we treat the file as binary and skip it. This is the
// same heuristic git/grep/ripgrep use.
const binarySampleSize = 8192

// maxLineSize is the largest single line bufio.Scanner will tolerate before
// returning bufio.ErrTooLong. We skip oversized-line files silently.
const maxLineSize = 1 * 1024 * 1024

// runContentSearch walks rootDir, filters files by the given options, and
// streams matching lines back on ch in batches. It always closes ch on exit.
// Symlinks (to files or directories) are skipped — fs.WalkDir does not
// follow symlinks to directories by default, and we additionally skip
// symlink file entries.
func runContentSearch(
	ctx context.Context,
	rootDir string,
	opts Options,
	m matcher,
	sessionID uint64,
	ch chan<- resultBatchMsg,
) {
	defer close(ch)

	extFilter := parseExtensions(opts.Extensions)
	var batch []Result

	filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error { //nolint:errcheck
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil || path == rootDir {
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			// Skip symlinks. For a symlinked directory, returning nil lets
			// WalkDir continue past it without descending; for a symlinked
			// file, returning nil simply moves on.
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}

		if extFilter != nil {
			ext := strings.ToLower(filepath.Ext(d.Name()))
			if !extFilter[ext] {
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > maxFileSize {
			return nil
		}

		hits, err := scanFile(ctx, path, m)
		if err != nil || len(hits) == 0 {
			return nil
		}
		rel, _ := filepath.Rel(rootDir, path)
		for i := range hits {
			hits[i].RelPath = rel
			hits[i].FullPath = path
		}
		batch = append(batch, hits...)
		if len(batch) >= 100 {
			select {
			case ch <- resultBatchMsg{sessionID: sessionID, results: batch}:
				batch = nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	})

	select {
	case ch <- resultBatchMsg{sessionID: sessionID, results: batch, done: true}:
	case <-ctx.Done():
	}
}

// scanFile reads path, rejects binaries, and returns all matching lines.
// RelPath and FullPath are left for the caller to fill in.
func scanFile(ctx context.Context, path string, m matcher) ([]Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read a sample prefix to detect binary content. If there's a NUL byte
	// anywhere in the first 8 KB, treat the file as binary and skip.
	sample := make([]byte, binarySampleSize)
	n, _ := io.ReadFull(f, sample)
	sample = sample[:n]
	if bytes.IndexByte(sample, 0) >= 0 {
		return nil, nil
	}

	// Continue scanning from where ReadFull stopped, prepending the sample
	// so line boundaries inside the sample are honored.
	reader := io.MultiReader(bytes.NewReader(sample), f)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), maxLineSize)

	var hits []Result
	lineNum := 0
	for scanner.Scan() {
		if ctx.Err() != nil {
			return hits, ctx.Err()
		}
		lineNum++
		line := scanner.Text()
		if m.Match(line) {
			hits = append(hits, Result{
				LineNum:  lineNum,
				LineText: line,
			})
		}
	}
	// scanner.Err() is intentionally ignored: files with lines longer than
	// maxLineSize will simply return whatever we matched up to the error.
	return hits, nil
}
