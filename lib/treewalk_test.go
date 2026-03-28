package lib

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"alexi.ch/pcms/model"
)

func TestBuildIndexSnapshot(t *testing.T) {
	srcFS := fstest.MapFS{
		"index.md":            &fstest.MapFile{Data: []byte("---\ntitle: Root Page\n---\n# root")},
		"assets/logo.png":     &fstest.MapFile{Data: []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}},
		"blog/index.html":     &fstest.MapFile{Data: []byte("---\ntitle: Blog HTML\n---\n<html></html>")},
		"blog/index.md":       &fstest.MapFile{Data: []byte("---\ntitle: Blog MD\n---\n# blog")},
		"blog/post/index.md":  &fstest.MapFile{Data: []byte("---\ntitle: Post\n---\n# post")},
		"blog/post/extra.txt": &fstest.MapFile{Data: []byte("extra")},
		"skip/index.md":       &fstest.MapFile{Data: []byte("---\ntitle: Skip\n---\n# skip")},
	}

	snapshot, err := BuildIndexSnapshot(srcFS, []string{`^/skip($|/)`})
	if err != nil {
		t.Fatalf("BuildIndexSnapshot() error = %v", err)
	}

	pagesByRoute := make(map[string]model.IndexedPage)
	for _, page := range snapshot.Pages {
		pagesByRoute[page.Route] = page
	}

	if _, ok := pagesByRoute["/"]; !ok {
		t.Fatalf("root page missing")
	}
	if pagesByRoute["/"].Title != "Root Page" {
		t.Fatalf("root title = %q, want %q", pagesByRoute["/"].Title, "Root Page")
	}
	if pagesByRoute["/"].IndexFile != "index.md" {
		t.Fatalf("root index_file = %q, want %q", pagesByRoute["/"].IndexFile, "index.md")
	}

	if _, ok := pagesByRoute["/blog"]; !ok {
		t.Fatalf("/blog page missing")
	}
	if pagesByRoute["/blog"].IndexFile != "index.html" {
		t.Fatalf("/blog index_file = %q, want %q", pagesByRoute["/blog"].IndexFile, "index.html")
	}
	if pagesByRoute["/blog"].Title != "Blog HTML" {
		t.Fatalf("/blog title = %q, want %q", pagesByRoute["/blog"].Title, "Blog HTML")
	}

	if _, ok := pagesByRoute["/blog/post"]; !ok {
		t.Fatalf("/blog/post page missing")
	}
	if pagesByRoute["/blog/post"].Title != "Post" {
		t.Fatalf("/blog/post title = %q, want %q", pagesByRoute["/blog/post"].Title, "Post")
	}
	if pagesByRoute["/blog/post"].ParentPageRoute == nil || *pagesByRoute["/blog/post"].ParentPageRoute != "/blog" {
		t.Fatalf("/blog/post parent = %v, want %q", pagesByRoute["/blog/post"].ParentPageRoute, "/blog")
	}

	if _, ok := pagesByRoute["/skip"]; ok {
		t.Fatalf("excluded page /skip should not be indexed")
	}
	if _, ok := pagesByRoute["/assets"]; ok {
		t.Fatalf("container folder /assets should not be indexed as page")
	}

	filesByRoute := make(map[string]model.IndexedFile)
	for _, file := range snapshot.Files {
		filesByRoute[file.Route] = file
	}

	if _, ok := filesByRoute["/assets/logo.png"]; !ok {
		t.Fatalf("/assets/logo.png file missing")
	}
	if filesByRoute["/assets/logo.png"].ParentPageRoute != "/" {
		t.Fatalf("/assets/logo.png parent = %q, want %q", filesByRoute["/assets/logo.png"].ParentPageRoute, "/")
	}
	if filesByRoute["/assets/logo.png"].MimeType != "image/png" {
		t.Fatalf("/assets/logo.png mime = %q, want %q", filesByRoute["/assets/logo.png"].MimeType, "image/png")
	}

	if _, ok := filesByRoute["/blog/post/extra.txt"]; !ok {
		t.Fatalf("/blog/post/extra.txt file missing")
	}
	if !strings.HasPrefix(filesByRoute["/blog/post/extra.txt"].MimeType, "text/plain") {
		t.Fatalf("/blog/post/extra.txt mime = %q, want text/plain*", filesByRoute["/blog/post/extra.txt"].MimeType)
	}

	if _, ok := filesByRoute["/skip/index.md"]; ok {
		t.Fatalf("excluded file /skip/index.md should not be indexed")
	}

	if _, ok := filesByRoute["/assets/logo.png"]; !ok {
		t.Fatalf("/assets/logo.png should be associated to nearest ancestor page")
	}
}

func TestBuildIndexSnapshotRootFallbackTitle(t *testing.T) {
	srcFS := fstest.MapFS{
		"a/index.md": &fstest.MapFile{Data: []byte("# page without frontmatter")},
	}

	snapshot, err := BuildIndexSnapshot(srcFS, nil)
	if err != nil {
		t.Fatalf("BuildIndexSnapshot() error = %v", err)
	}

	pagesByRoute := make(map[string]model.IndexedPage)
	for _, page := range snapshot.Pages {
		pagesByRoute[page.Route] = page
	}

	if pagesByRoute["/a"].Title != "a" {
		t.Fatalf("/a title = %q, want %q", pagesByRoute["/a"].Title, "a")
	}
	if _, ok := pagesByRoute["/"]; ok {
		t.Fatalf("/ should not be indexed as page without index file")
	}

	if _, err := fs.Stat(srcFS, "a/index.md"); err != nil {
		t.Fatalf("fixture validation failed: %v", err)
	}
}
