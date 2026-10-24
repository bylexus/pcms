package webserver

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNormalizeFileLookupRoute(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantRoute string
		wantOK    bool
	}{
		{name: "root", in: "/", wantRoute: "/", wantOK: true},
		{name: "file", in: "/robots.txt", wantRoute: "/robots.txt", wantOK: true},
		{name: "strict trailing slash", in: "/robots.txt/", wantRoute: "", wantOK: false},
		{name: "double slash trailing", in: "//robots.txt//", wantRoute: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRoute, gotOK := normalizeFileLookupRoute(tt.in)
			if gotOK != tt.wantOK {
				t.Fatalf("normalizeFileLookupRoute(%q) ok = %v, want %v", tt.in, gotOK, tt.wantOK)
			}
			if gotRoute != tt.wantRoute {
				t.Fatalf("normalizeFileLookupRoute(%q) route = %q, want %q", tt.in, gotRoute, tt.wantRoute)
			}
		})
	}
}

// dirEntryNames returns the names of all entries in dir.
func dirEntryNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestWriteCacheFile(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "sub", "page")
	cacheFile := filepath.Join(dir, "index.html")

	// new file in a new directory:
	if err := writeCacheFile(cacheFile, []byte("first")); err != nil {
		t.Fatalf("writeCacheFile: %v", err)
	}
	got, err := os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("read cache file: %v", err)
	}
	if !bytes.Equal(got, []byte("first")) {
		t.Fatalf("content = %q, want %q", got, "first")
	}

	// overwrite an existing file:
	if err := writeCacheFile(cacheFile, []byte("second")); err != nil {
		t.Fatalf("writeCacheFile (overwrite): %v", err)
	}
	got, err = os.ReadFile(cacheFile)
	if err != nil {
		t.Fatalf("read cache file: %v", err)
	}
	if !bytes.Equal(got, []byte("second")) {
		t.Fatalf("content = %q, want %q", got, "second")
	}

	// no temp file left, only the cache file:
	names := dirEntryNames(t, dir)
	if len(names) != 1 || names[0] != "index.html" {
		t.Fatalf("dir entries = %v, want [index.html]", names)
	}

	// the effective file mode is the same as with os.WriteFile(…, 0o666):
	refFile := filepath.Join(root, "reference")
	if err := os.WriteFile(refFile, []byte("ref"), 0o666); err != nil {
		t.Fatalf("write reference file: %v", err)
	}
	refInfo, err := os.Stat(refFile)
	if err != nil {
		t.Fatalf("stat reference file: %v", err)
	}
	info, err := os.Stat(cacheFile)
	if err != nil {
		t.Fatalf("stat cache file: %v", err)
	}
	if info.Mode().Perm() != refInfo.Mode().Perm() {
		t.Fatalf("cache file mode = %v, want %v (as os.WriteFile 0o666)", info.Mode().Perm(), refInfo.Mode().Perm())
	}
}

func TestWriteCacheFileReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only directories are not supported on windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can write to read-only directories")
	}

	dir := t.TempDir()
	existing := filepath.Join(dir, "existing.png")
	if err := os.WriteFile(existing, []byte("old"), 0o666); err != nil {
		t.Fatalf("write existing file: %v", err)
	}
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	// new file:
	if err := writeCacheFile(filepath.Join(dir, "new.png"), []byte("new")); err == nil {
		t.Fatalf("writeCacheFile in read-only dir: want error, got nil")
	}
	// existing file:
	if err := writeCacheFile(existing, []byte("changed")); err == nil {
		t.Fatalf("writeCacheFile over existing file in read-only dir: want error, got nil")
	}

	// no temp or partial file, existing file unchanged:
	names := dirEntryNames(t, dir)
	if len(names) != 1 || names[0] != "existing.png" {
		t.Fatalf("dir entries = %v, want [existing.png]", names)
	}
	got, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("read existing file: %v", err)
	}
	if !bytes.Equal(got, []byte("old")) {
		t.Fatalf("existing content = %q, want %q", got, "old")
	}
}
