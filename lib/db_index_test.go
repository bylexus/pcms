package lib

import (
	"path/filepath"
	"testing"

	"alexi.ch/pcms/model"
)

func TestDBHIndexLifecycle(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pcms-test.db")
	dbh, err := OpenDBH(dbPath)
	if err != nil {
		t.Fatalf("OpenDBH() error = %v", err)
	}
	defer dbh.Close()

	if err := dbh.BeginIndexRun(); err != nil {
		t.Fatalf("BeginIndexRun() error = %v", err)
	}

	if err := dbh.CleanIndex(); err != nil {
		t.Fatalf("CleanIndex() error = %v", err)
	}

	if err := dbh.ReplacePage(model.IndexedPage{
		Route:        "/",
		Title:        "root",
		IndexFile:    "index.md",
		Metadata: map[string]any{"title": "root"},
	}); err != nil {
		t.Fatalf("ReplacePage(root) error = %v", err)
	}

	rootRoute := "/"
	if err := dbh.ReplacePage(model.IndexedPage{
		Route:           "/blog",
		ParentPageRoute: &rootRoute,
		Title:           "blog",
		IndexFile:       "index.html",
		Metadata:        map[string]any{"title": "blog"},
	}); err != nil {
		t.Fatalf("ReplacePage(/blog) error = %v", err)
	}

	if err := dbh.ReplaceFile(model.IndexedFile{
		Route:           "/blog/image.png",
		ParentPageRoute: "/blog",
		FileName:        "image.png",
		MimeType:        "image/png",
		FileSize:        42,
	}); err != nil {
		t.Fatalf("ReplaceFile() error = %v", err)
	}

	if err := dbh.SetLastIndexInfo("test-source", 2, 1); err != nil {
		t.Fatalf("SetLastIndexInfo() error = %v", err)
	}

	if err := dbh.CommitIndexRun(); err != nil {
		t.Fatalf("CommitIndexRun() error = %v", err)
	}

	pageCount, err := dbh.CountPages()
	if err != nil {
		t.Fatalf("CountPages() error = %v", err)
	}
	if pageCount != 2 {
		t.Fatalf("CountPages() = %d, want %d", pageCount, 2)
	}

	fileCount, err := dbh.CountFiles()
	if err != nil {
		t.Fatalf("CountFiles() error = %v", err)
	}
	if fileCount != 1 {
		t.Fatalf("CountFiles() = %d, want %d", fileCount, 1)
	}
}

func TestDBHIndexForeignKeyIntegrity(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pcms-test-fk.db")
	dbh, err := OpenDBH(dbPath)
	if err != nil {
		t.Fatalf("OpenDBH() error = %v", err)
	}
	defer dbh.Close()

	if err := dbh.BeginIndexRun(); err != nil {
		t.Fatalf("BeginIndexRun() error = %v", err)
	}
	defer dbh.RollbackIndexRun()

	if err := dbh.CleanIndex(); err != nil {
		t.Fatalf("CleanIndex() error = %v", err)
	}

	err = dbh.ReplaceFile(model.IndexedFile{
		Route:           "/orphan.txt",
		ParentPageRoute: "/missing",
		FileName:        "orphan.txt",
		MimeType:        "text/plain",
		FileSize:        1,
	})
	if err == nil {
		t.Fatalf("ReplaceFile() expected foreign key error, got nil")
	}
}

func TestDBHGetByRoute(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pcms-test-query.db")
	dbh, err := OpenDBH(dbPath)
	if err != nil {
		t.Fatalf("OpenDBH() error = %v", err)
	}
	defer dbh.Close()

	if err := dbh.BeginIndexRun(); err != nil {
		t.Fatalf("BeginIndexRun() error = %v", err)
	}

	if err := dbh.CleanIndex(); err != nil {
		t.Fatalf("CleanIndex() error = %v", err)
	}

	if err := dbh.ReplacePage(model.IndexedPage{
		Route:        "/",
		Title:        "root",
		IndexFile:    "index.md",
		Metadata: map[string]any{"title": "root"},
	}); err != nil {
		t.Fatalf("ReplacePage(root) error = %v", err)
	}

	if err := dbh.ReplaceFile(model.IndexedFile{
		Route:           "/robots.txt",
		ParentPageRoute: "/",
		FileName:        "robots.txt",
		MimeType:        "text/plain",
		FileSize:        10,
	}); err != nil {
		t.Fatalf("ReplaceFile(robots) error = %v", err)
	}

	if err := dbh.CommitIndexRun(); err != nil {
		t.Fatalf("CommitIndexRun() error = %v", err)
	}

	page, found, err := dbh.GetPageByRoute("/")
	if err != nil {
		t.Fatalf("GetPageByRoute(/) error = %v", err)
	}
	if !found {
		t.Fatalf("GetPageByRoute(/) found = false, want true")
	}
	if page.IndexFile != "index.md" {
		t.Fatalf("GetPageByRoute(/).IndexFile = %q, want %q", page.IndexFile, "index.md")
	}

	_, found, err = dbh.GetPageByRoute("/missing")
	if err != nil {
		t.Fatalf("GetPageByRoute(/missing) error = %v", err)
	}
	if found {
		t.Fatalf("GetPageByRoute(/missing) found = true, want false")
	}

	file, found, err := dbh.GetFileByRoute("/robots.txt")
	if err != nil {
		t.Fatalf("GetFileByRoute(/robots.txt) error = %v", err)
	}
	if !found {
		t.Fatalf("GetFileByRoute(/robots.txt) found = false, want true")
	}
	if file.MimeType != "text/plain" {
		t.Fatalf("GetFileByRoute(/robots.txt).MimeType = %q, want %q", file.MimeType, "text/plain")
	}

	_, found, err = dbh.GetFileByRoute("/missing.txt")
	if err != nil {
		t.Fatalf("GetFileByRoute(/missing.txt) error = %v", err)
	}
	if found {
		t.Fatalf("GetFileByRoute(/missing.txt) found = true, want false")
	}
}
