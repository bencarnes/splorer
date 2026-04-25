// Package fileops implements the small filesystem manipulation primitives
// used by the Manipulate menu: bulk delete, recursive copy, and move. These
// are intentionally dumb — they perform the operation requested and return
// any error encountered. Confirmation, clipboard, and selection state live
// in the app layer.
package fileops

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DeleteAll removes every path in paths recursively. Returns the first error
// encountered, but continues attempting the remaining paths regardless so a
// partial failure doesn't strand the user with a half-removed selection.
func DeleteAll(paths []string) error {
	var firstErr error
	for _, p := range paths {
		if err := os.RemoveAll(p); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("delete %s: %w", p, err)
		}
	}
	return firstErr
}

// CopyAll copies every source path into destDir. Each source's basename is
// used as the destination name. Existing entries at the destination are not
// overwritten — the first collision aborts the operation.
func CopyAll(sources []string, destDir string) error {
	for _, src := range sources {
		dst := filepath.Join(destDir, filepath.Base(src))
		if err := guardCollision(src, dst); err != nil {
			return err
		}
		if err := copyPath(src, dst); err != nil {
			return fmt.Errorf("copy %s: %w", src, err)
		}
	}
	return nil
}

// MoveAll moves every source path into destDir. os.Rename is tried first; on
// failure (typically EXDEV across mount points) the entry is copied and then
// removed. As with CopyAll, name collisions abort.
func MoveAll(sources []string, destDir string) error {
	for _, src := range sources {
		dst := filepath.Join(destDir, filepath.Base(src))
		if err := guardCollision(src, dst); err != nil {
			return err
		}
		if err := os.Rename(src, dst); err == nil {
			continue
		}
		// Rename failed (likely cross-device). Copy then remove.
		if err := copyPath(src, dst); err != nil {
			return fmt.Errorf("move %s: %w", src, err)
		}
		if err := os.RemoveAll(src); err != nil {
			return fmt.Errorf("move %s: copied but cannot remove source: %w", src, err)
		}
	}
	return nil
}

// guardCollision returns an error if dst already exists, or if dst would be
// the same path as src (a no-op move/copy onto itself). Same-name copy back
// into the same directory is rejected to avoid silently doing nothing or
// corrupting the source.
func guardCollision(src, dst string) error {
	srcAbs, _ := filepath.Abs(src)
	dstAbs, _ := filepath.Abs(dst)
	if srcAbs == dstAbs {
		return fmt.Errorf("source and destination are the same: %s", src)
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", dst, err)
	}
	return nil
}

// copyPath copies src to dst, recursing into directories. Symlinks are
// re-created (not followed). Permission bits are preserved on a best-effort
// basis.
func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case info.IsDir():
		return copyDir(src, dst, info.Mode().Perm())
	default:
		return copyFile(src, dst, info.Mode().Perm())
	}
}

func copyDir(src, dst string, perm os.FileMode) error {
	if err := os.MkdirAll(dst, perm); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		s := filepath.Join(src, e.Name())
		d := filepath.Join(dst, e.Name())
		if err := copyPath(s, d); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()         //nolint:errcheck
		os.Remove(dst)      //nolint:errcheck
		return err
	}
	return out.Close()
}
