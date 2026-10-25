package lib

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"alexi.ch/pcms/model"
)

// writeIndexTestSite creates a small source tree:
//
//	/index.md
//	/blog/index.md
//	/blog/image.png
func writeIndexTestSite(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"index.md":       "---\ntitle: Root\n---\nroot",
		"blog/index.md":  "---\ntitle: Blog\n---\nblog",
		"blog/image.png": "png",
	}
	for name, content := range files {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return root
}

func openIndexTestDB(t *testing.T) *DBH {
	t.Helper()
	dbh, err := OpenDBH(filepath.Join(t.TempDir(), "pcms-index-test.db"))
	if err != nil {
		t.Fatalf("OpenDBH() error = %v", err)
	}
	t.Cleanup(func() { dbh.Close() })
	return dbh
}

func TestRunIndex(t *testing.T) {
	src := writeIndexTestSite(t)
	dbh := openIndexTestDB(t)

	res, err := RunIndex(context.Background(), dbh, os.DirFS(src), src, nil)
	if err != nil {
		t.Fatalf("RunIndex() error = %v", err)
	}
	if res.Pages != 2 || res.Files != 1 {
		t.Fatalf("RunIndex() = %d pages, %d files, want 2, 1", res.Pages, res.Files)
	}
	if res.Source != src || res.DBPath != dbh.Path() {
		t.Fatalf("RunIndex() result = %+v", res)
	}
	if _, found, err := dbh.GetPageByRoute("/blog"); err != nil || !found {
		t.Fatalf("GetPageByRoute(/blog) found=%v err=%v", found, err)
	}
}

func TestRunIndexCanceledKeepsOldIndex(t *testing.T) {
	src := writeIndexTestSite(t)
	dbh := openIndexTestDB(t)

	if _, err := RunIndex(context.Background(), dbh, os.DirFS(src), src, nil); err != nil {
		t.Fatalf("RunIndex() error = %v", err)
	}
	// remove a page from the source, then run canceled:
	if err := os.RemoveAll(filepath.Join(src, "blog")); err != nil {
		t.Fatalf("remove: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := RunIndex(ctx, dbh, os.DirFS(src), src, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunIndex() error = %v, want context.Canceled", err)
	}
	count, err := dbh.CountPages()
	if err != nil {
		t.Fatalf("CountPages() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("CountPages() = %d, want 2 (old index)", count)
	}
}

// While an index run is open, other DBH users see the last committed index.
func TestIndexRunIsolation(t *testing.T) {
	src := writeIndexTestSite(t)
	dbh := openIndexTestDB(t)
	if _, err := RunIndex(context.Background(), dbh, os.DirFS(src), src, nil); err != nil {
		t.Fatalf("RunIndex() error = %v", err)
	}

	run, err := dbh.BeginIndexRun()
	if err != nil {
		t.Fatalf("BeginIndexRun() error = %v", err)
	}
	defer run.Rollback()
	if err := run.CleanIndex(); err != nil {
		t.Fatalf("CleanIndex() error = %v", err)
	}

	if _, found, err := dbh.GetPageByRoute("/blog"); err != nil || !found {
		t.Fatalf("GetPageByRoute(/blog) during run: found=%v err=%v, want found", found, err)
	}
	count, err := dbh.CountPages()
	if err != nil || count != 2 {
		t.Fatalf("CountPages() during run = %d, %v, want 2", count, err)
	}

	// a write outside of the run is rejected as busy:
	err = dbh.ReplacePage(model.IndexedPage{Route: "/", Title: "x", IndexFile: "index.md"})
	if err == nil || !IsDBBusy(err) {
		t.Fatalf("ReplacePage() during run error = %v, want busy error", err)
	}
}

func TestIsDBBusy(t *testing.T) {
	if IsDBBusy(nil) {
		t.Fatalf("IsDBBusy(nil) = true")
	}
	if IsDBBusy(errors.New("other")) {
		t.Fatalf("IsDBBusy(other) = true")
	}
}

func TestStartBackgroundIndex(t *testing.T) {
	src := writeIndexTestSite(t)
	dbh := openIndexTestDB(t)

	var config model.Config
	config.ServeMode = model.SERVE_MODE_FILES
	config.SourcePath = src

	done := StartBackgroundIndex(context.Background(), config, dbh, nil)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("background index not done after 10s")
	}

	count, err := dbh.CountPages()
	if err != nil || count != 2 {
		t.Fatalf("CountPages() = %d, %v, want 2", count, err)
	}
}

func TestStartBackgroundIndexDisabled(t *testing.T) {
	src := writeIndexTestSite(t)
	dbh := openIndexTestDB(t)

	var config model.Config
	config.ServeMode = model.SERVE_MODE_FILES
	config.SourcePath = src
	off := false
	config.Server.IndexOnStart = &off

	done := StartBackgroundIndex(context.Background(), config, dbh, nil)
	select {
	case <-done:
	default:
		t.Fatalf("done channel not closed when disabled")
	}

	count, err := dbh.CountPages()
	if err != nil || count != 0 {
		t.Fatalf("CountPages() = %d, %v, want 0", count, err)
	}
}
