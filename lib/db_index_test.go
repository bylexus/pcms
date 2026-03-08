package lib

import (
	"path/filepath"
	"testing"
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

	if err := dbh.ReplacePage(IndexedPageRecord{
		Route:        "/",
		Title:        "root",
		IndexFile:    "index.md",
		MetadataJSON: `{"title":"root"}`,
	}); err != nil {
		t.Fatalf("ReplacePage(root) error = %v", err)
	}

	rootRoute := "/"
	if err := dbh.ReplacePage(IndexedPageRecord{
		Route:           "/blog",
		ParentPageRoute: &rootRoute,
		Title:           "blog",
		IndexFile:       "index.html",
		MetadataJSON:    `{"title":"blog"}`,
	}); err != nil {
		t.Fatalf("ReplacePage(/blog) error = %v", err)
	}

	if err := dbh.ReplaceFile(IndexedFileRecord{
		Route:           "/blog/image.png",
		ParentPageRoute: "/blog",
		FileName:        "image.png",
		MimeType:        "image/png",
		FileSize:        42,
		MetadataJSON:    `{}`,
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

	err = dbh.ReplaceFile(IndexedFileRecord{
		Route:           "/orphan.txt",
		ParentPageRoute: "/missing",
		FileName:        "orphan.txt",
		MimeType:        "text/plain",
		FileSize:        1,
		MetadataJSON:    `{}`,
	})
	if err == nil {
		t.Fatalf("ReplaceFile() expected foreign key error, got nil")
	}
}
