package webserver

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// writeFileWithMtime creates the file (and its parent dirs) and sets its mtime.
func writeFileWithMtime(t *testing.T, path string, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
}

func pathExists(t *testing.T, path string) bool {
	t.Helper()
	_, err := os.Lstat(path)
	if err == nil {
		return true
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	t.Fatalf("lstat %s: %v", path, err)
	return false
}

func TestCleanupCacheDir(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	maxAge := time.Hour
	old := now.Add(-2 * time.Hour)
	fresh := now.Add(-10 * time.Minute)

	oldFiles := []string{
		"old-page/index.html",
		"nested/a/b/index.html",
		"_imageResizer/oldkey.webp",
		"_imageResizer/.tmp-oldkey.webp-abc123",
		"_imageResizer/orphan",    // old format without extension
		"_imageResizer/orphan.ct", // old content type sidecar
		"keep-dir/old.html",
	}
	freshFiles := []string{
		"index.html",
		"keep-dir/index.html",
		"_imageResizer/freshkey.jpg",
		"_imageResizer/.tmp-freshkey.jpg-xyz",
	}
	for _, f := range oldFiles {
		writeFileWithMtime(t, filepath.Join(root, f), old)
	}
	for _, f := range freshFiles {
		writeFileWithMtime(t, filepath.Join(root, f), fresh)
	}
	// already empty dir:
	if err := os.MkdirAll(filepath.Join(root, "empty/deeper"), 0o755); err != nil {
		t.Fatal(err)
	}

	res := cleanupCacheDir(root, maxAge, now)

	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if res.FilesRemoved != len(oldFiles) {
		t.Errorf("FilesRemoved = %d, want %d", res.FilesRemoved, len(oldFiles))
	}
	// removed dirs: old-page, nested, nested/a, nested/a/b, empty, empty/deeper
	if res.DirsRemoved != 6 {
		t.Errorf("DirsRemoved = %d, want 6", res.DirsRemoved)
	}
	for _, f := range oldFiles {
		if pathExists(t, filepath.Join(root, f)) {
			t.Errorf("old file %s still exists", f)
		}
	}
	for _, f := range freshFiles {
		if !pathExists(t, filepath.Join(root, f)) {
			t.Errorf("fresh file %s was removed", f)
		}
	}
	for _, d := range []string{"old-page", "nested", "empty"} {
		if pathExists(t, filepath.Join(root, d)) {
			t.Errorf("empty dir %s still exists", d)
		}
	}
	for _, d := range []string{"keep-dir", "_imageResizer"} {
		if !pathExists(t, filepath.Join(root, d)) {
			t.Errorf("non-empty dir %s was removed", d)
		}
	}
}

func TestCleanupCacheDirFileExactlyAtCutoffKept(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	f := filepath.Join(root, "index.html")
	writeFileWithMtime(t, f, now.Add(-time.Hour))

	res := cleanupCacheDir(root, time.Hour, now)
	if res.FilesRemoved != 0 || !pathExists(t, f) {
		t.Errorf("file at cutoff removed (FilesRemoved = %d)", res.FilesRemoved)
	}
}

func TestCleanupCacheDirRootKept(t *testing.T) {
	root := filepath.Join(t.TempDir(), "cache")
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	writeFileWithMtime(t, filepath.Join(root, "sub/index.html"), now.Add(-48*time.Hour))

	res := cleanupCacheDir(root, time.Hour, now)
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if res.FilesRemoved != 1 || res.DirsRemoved != 1 {
		t.Errorf("removed files=%d dirs=%d, want 1/1", res.FilesRemoved, res.DirsRemoved)
	}
	if !pathExists(t, root) {
		t.Fatal("cache root was removed")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("cache root not empty: %d entries", len(entries))
	}

	// second pass on the empty root: nothing to do, root still there
	res = cleanupCacheDir(root, time.Hour, now)
	if len(res.Errors) != 0 || res.FilesRemoved != 0 || res.DirsRemoved != 0 {
		t.Errorf("second pass: %+v", res)
	}
	if !pathExists(t, root) {
		t.Fatal("cache root was removed on second pass")
	}
}

func TestCleanupCacheDirNotExisting(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")
	res := cleanupCacheDir(root, time.Hour, time.Now())
	if len(res.Errors) != 0 || res.FilesRemoved != 0 || res.DirsRemoved != 0 {
		t.Errorf("non-existing cache dir: %+v", res)
	}
	if pathExists(t, root) {
		t.Error("cache dir was created")
	}

	res = cleanupCacheDir("", time.Hour, time.Now())
	if len(res.Errors) != 0 || res.FilesRemoved != 0 || res.DirsRemoved != 0 {
		t.Errorf("empty cache dir: %+v", res)
	}
}

func TestCleanupCacheDirSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need extra privileges on windows")
	}
	base := t.TempDir()
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	old := now.Add(-48 * time.Hour)

	// outside of the cache dir: must never be touched
	outside := filepath.Join(base, "outside")
	outsideFile := filepath.Join(outside, "old.html")
	writeFileWithMtime(t, outsideFile, old)

	cache := filepath.Join(base, "cache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(cache, "link-dir")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, filepath.Join(cache, "link-file")); err != nil {
		t.Fatal(err)
	}
	writeFileWithMtime(t, filepath.Join(cache, "old.html"), old)

	res := cleanupCacheDir(cache, time.Hour, now)
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if res.FilesRemoved != 1 {
		t.Errorf("FilesRemoved = %d, want 1", res.FilesRemoved)
	}
	if !pathExists(t, outsideFile) {
		t.Error("file outside the cache dir was removed")
	}
	if !pathExists(t, filepath.Join(cache, "link-dir")) || !pathExists(t, filepath.Join(cache, "link-file")) {
		t.Error("symlink inside the cache dir was removed")
	}

	// cache dir itself is a symlink (e.g. to a separate volume): cleaned
	volume := filepath.Join(base, "volume")
	writeFileWithMtime(t, filepath.Join(volume, "sub/old.html"), old)
	linkedCache := filepath.Join(base, "linked-cache")
	if err := os.Symlink(volume, linkedCache); err != nil {
		t.Fatal(err)
	}
	res = cleanupCacheDir(linkedCache, time.Hour, now)
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}
	if res.FilesRemoved != 1 || res.DirsRemoved != 1 {
		t.Errorf("symlinked root: removed files=%d dirs=%d, want 1/1", res.FilesRemoved, res.DirsRemoved)
	}
	if !pathExists(t, volume) || !pathExists(t, linkedCache) {
		t.Error("symlinked cache root or its target was removed")
	}
}

func TestStartCacheCleanupDisabled(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "index.html")
	writeFileWithMtime(t, f, time.Now().Add(-48*time.Hour))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	StartCacheCleanup(ctx, root, 0, time.Millisecond, nil)
	time.Sleep(20 * time.Millisecond)
	if !pathExists(t, f) {
		t.Error("file removed although cleanup is disabled")
	}
}

func TestStartCacheCleanupRunsAtStartup(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "sub/index.html")
	writeFileWithMtime(t, f, time.Now().Add(-48*time.Hour))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// long interval: only the startup pass can remove the file
	StartCacheCleanup(ctx, root, time.Hour, time.Hour, nil)

	deadline := time.Now().Add(5 * time.Second)
	for pathExists(t, filepath.Join(root, "sub")) {
		if time.Now().After(deadline) {
			t.Fatal("startup cleanup pass did not remove the old file and dir")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !pathExists(t, root) {
		t.Fatal("cache root was removed")
	}
}
