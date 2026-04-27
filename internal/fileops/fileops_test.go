package fileops

import (
	"os"
	"path/filepath"
	"testing"
)

// makeFile creates a regular file with the given content under root and
// returns its absolute path.
func makeFile(t *testing.T, root, name, content string) string {
	t.Helper()
	p := filepath.Join(root, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func TestDeleteAll_RemovesFilesAndDirs(t *testing.T) {
	root := t.TempDir()
	f := makeFile(t, root, "x.txt", "hi")
	d := filepath.Join(root, "subdir")
	if err := os.MkdirAll(filepath.Join(d, "nested"), 0755); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}
	makeFile(t, d, "y.txt", "hello")

	if err := DeleteAll([]string{f, d}); err != nil {
		t.Fatalf("DeleteAll: %v", err)
	}
	for _, p := range []string{f, d} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s still exists", p)
		}
	}
}

func TestDeleteAll_ContinuesAfterError(t *testing.T) {
	root := t.TempDir()
	good := makeFile(t, root, "g.txt", "ok")
	missing := filepath.Join(root, "does-not-exist") // os.RemoveAll treats this as success
	other := makeFile(t, root, "o.txt", "ok")

	if err := DeleteAll([]string{good, missing, other}); err != nil {
		t.Errorf("DeleteAll on missing path should succeed (RemoveAll semantics); got %v", err)
	}
	for _, p := range []string{good, other} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s should be gone", p)
		}
	}
}

func TestCopyAll_FileAndDir(t *testing.T) {
	root := t.TempDir()
	dest := t.TempDir()

	srcFile := makeFile(t, root, "a.txt", "alpha")
	srcDir := filepath.Join(root, "tree")
	if err := os.MkdirAll(filepath.Join(srcDir, "sub"), 0755); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}
	makeFile(t, srcDir, "b.txt", "beta")
	makeFile(t, filepath.Join(srcDir, "sub"), "c.txt", "gamma")

	if err := CopyAll([]string{srcFile, srcDir}, dest); err != nil {
		t.Fatalf("CopyAll: %v", err)
	}

	// File copied.
	if got, err := os.ReadFile(filepath.Join(dest, "a.txt")); err != nil || string(got) != "alpha" {
		t.Errorf("a.txt copy = %q, %v", got, err)
	}
	// Tree copied recursively.
	if got, err := os.ReadFile(filepath.Join(dest, "tree", "b.txt")); err != nil || string(got) != "beta" {
		t.Errorf("tree/b.txt copy = %q, %v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(dest, "tree", "sub", "c.txt")); err != nil || string(got) != "gamma" {
		t.Errorf("tree/sub/c.txt copy = %q, %v", got, err)
	}
	// Source must still exist.
	if _, err := os.Stat(srcFile); err != nil {
		t.Errorf("source file removed: %v", err)
	}
}

func TestCopyAll_RefusesCollision(t *testing.T) {
	root := t.TempDir()
	dest := t.TempDir()

	src := makeFile(t, root, "a.txt", "x")
	makeFile(t, dest, "a.txt", "existing") // collision

	if err := CopyAll([]string{src}, dest); err == nil {
		t.Errorf("expected error when destination exists")
	}
	// Existing destination must not be clobbered.
	if got, err := os.ReadFile(filepath.Join(dest, "a.txt")); err != nil || string(got) != "existing" {
		t.Errorf("existing file changed: %q, %v", got, err)
	}
}

func TestCopyAll_RefusesSelfCopy(t *testing.T) {
	root := t.TempDir()
	src := makeFile(t, root, "a.txt", "x")
	if err := CopyAll([]string{src}, root); err == nil {
		t.Errorf("expected error when copying onto itself")
	}
}

func TestMoveAll_RenamesFiles(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "destdir")
	if err := os.Mkdir(dest, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := makeFile(t, root, "a.txt", "alpha")

	if err := MoveAll([]string{src}, dest); err != nil {
		t.Fatalf("MoveAll: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source should be gone after move, stat err = %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(dest, "a.txt")); err != nil || string(got) != "alpha" {
		t.Errorf("dest content = %q, %v", got, err)
	}
}

func TestRename_File(t *testing.T) {
	root := t.TempDir()
	src := makeFile(t, root, "old.txt", "alpha")

	if err := Rename(src, "new.txt"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("old name should be gone, stat err = %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "new.txt")); err != nil || string(got) != "alpha" {
		t.Errorf("new file = %q, %v", got, err)
	}
}

func TestRename_Directory(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "old")
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0755); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}
	makeFile(t, src, "x.txt", "hi")

	if err := Rename(src, "new"); err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("old dir should be gone, stat err = %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "new", "x.txt")); err != nil || string(got) != "hi" {
		t.Errorf("renamed dir contents = %q, %v", got, err)
	}
}

func TestRename_RefusesCollision(t *testing.T) {
	root := t.TempDir()
	src := makeFile(t, root, "a.txt", "x")
	makeFile(t, root, "b.txt", "existing")

	if err := Rename(src, "b.txt"); err == nil {
		t.Errorf("expected error when renaming onto an existing entry")
	}
	// Source should still exist; existing target must not be clobbered.
	if got, err := os.ReadFile(src); err != nil || string(got) != "x" {
		t.Errorf("source changed after refused rename: %q, %v", got, err)
	}
	if got, err := os.ReadFile(filepath.Join(root, "b.txt")); err != nil || string(got) != "existing" {
		t.Errorf("existing target changed: %q, %v", got, err)
	}
}

func TestRename_RefusesSameName(t *testing.T) {
	root := t.TempDir()
	src := makeFile(t, root, "a.txt", "x")
	if err := Rename(src, "a.txt"); err == nil {
		t.Errorf("expected error when renaming to the same name")
	}
}

func TestRename_RejectsPathSeparator(t *testing.T) {
	root := t.TempDir()
	src := makeFile(t, root, "a.txt", "x")
	if err := Rename(src, "sub/b.txt"); err == nil {
		t.Errorf("expected error when newName contains '/'")
	}
	if err := Rename(src, `sub\b.txt`); err == nil {
		t.Errorf("expected error when newName contains backslash")
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("source missing after rejected rename: %v", err)
	}
}

func TestRename_RejectsEmptyAndDotNames(t *testing.T) {
	root := t.TempDir()
	src := makeFile(t, root, "a.txt", "x")
	for _, bad := range []string{"", ".", ".."} {
		if err := Rename(src, bad); err == nil {
			t.Errorf("Rename(%q) should fail", bad)
		}
	}
}

func TestMoveAll_RefusesCollision(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dst")
	if err := os.Mkdir(dest, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	src := makeFile(t, root, "a.txt", "x")
	makeFile(t, dest, "a.txt", "existing")

	if err := MoveAll([]string{src}, dest); err == nil {
		t.Errorf("expected error on collision")
	}
	// Source should still exist.
	if _, err := os.Stat(src); err != nil {
		t.Errorf("source removed despite collision: %v", err)
	}
}
