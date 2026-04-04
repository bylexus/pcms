package lib

import (
	"path/filepath"
	"testing"

	"alexi.ch/pcms/model"
)

// setupEnableTestDB builds the following hierarchy:
//
//	/            (enabled)
//	/section     (enabled)
//	/section/a   (enabled)
//	/section/b   (enabled)
//	/section/a/deep (enabled)
//	/other       (enabled)
//
// Files:
//
//	/section/img.png      (parent=/section, enabled)
//	/section/a/logo.png   (parent=/section/a, enabled)
//	/other/doc.pdf        (parent=/other, enabled)
func setupEnableTestDB(t *testing.T) *DBH {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "pcms-enable-test.db")
	dbh, err := OpenDBH(dbPath)
	if err != nil {
		t.Fatalf("OpenDBH: %v", err)
	}

	if err := dbh.BeginIndexRun(); err != nil {
		t.Fatalf("BeginIndexRun: %v", err)
	}

	root := "/"
	section := "/section"
	sectionA := "/section/a"

	pages := []model.IndexedPage{
		{Route: "/", Title: "Root", IndexFile: "index.html", Enabled: true, Metadata: map[string]any{}},
		{Route: "/section", ParentPageRoute: &root, Title: "Section", IndexFile: "index.html", Enabled: true, Metadata: map[string]any{}},
		{Route: "/section/a", ParentPageRoute: &section, Title: "Section A", IndexFile: "index.html", Enabled: true, Metadata: map[string]any{}},
		{Route: "/section/b", ParentPageRoute: &section, Title: "Section B", IndexFile: "index.html", Enabled: true, Metadata: map[string]any{}},
		{Route: "/section/a/deep", ParentPageRoute: &sectionA, Title: "Deep", IndexFile: "index.html", Enabled: true, Metadata: map[string]any{}},
		{Route: "/other", ParentPageRoute: &root, Title: "Other", IndexFile: "index.html", Enabled: true, Metadata: map[string]any{}},
	}
	for _, p := range pages {
		if err := dbh.ReplacePage(p); err != nil {
			t.Fatalf("ReplacePage(%s): %v", p.Route, err)
		}
	}

	files := []model.IndexedFile{
		{Route: "/section/img.png", ParentPageRoute: "/section", FileName: "img.png", MimeType: "image/png", FileSize: 100, Enabled: true},
		{Route: "/section/a/logo.png", ParentPageRoute: "/section/a", FileName: "logo.png", MimeType: "image/png", FileSize: 200, Enabled: true},
		{Route: "/other/doc.pdf", ParentPageRoute: "/other", FileName: "doc.pdf", MimeType: "application/pdf", FileSize: 300, Enabled: true},
	}
	for _, f := range files {
		if err := dbh.ReplaceFile(f); err != nil {
			t.Fatalf("ReplaceFile(%s): %v", f.Route, err)
		}
	}

	if err := dbh.CommitIndexRun(); err != nil {
		t.Fatalf("CommitIndexRun: %v", err)
	}

	return dbh
}

func pageEnabled(t *testing.T, dbh *DBH, route string) bool {
	t.Helper()
	p, found, err := dbh.GetPageByRoute(route)
	if err != nil {
		t.Fatalf("GetPageByRoute(%s): %v", route, err)
	}
	if !found {
		t.Fatalf("page not found: %s", route)
	}
	return p.Enabled
}

func fileEnabled(t *testing.T, dbh *DBH, route string) bool {
	t.Helper()
	f, found, err := dbh.GetFileByRoute(route)
	if err != nil {
		t.Fatalf("GetFileByRoute(%s): %v", route, err)
	}
	if !found {
		t.Fatalf("file not found: %s", route)
	}
	return f.Enabled
}

// TestSetPageEnabled_PageNotFound verifies that an error is returned for unknown routes.
func TestSetPageEnabled_PageNotFound(t *testing.T) {
	dbh := setupEnableTestDB(t)
	defer dbh.Close()

	err := dbh.SetPageEnabled("/nonexistent", false, false)
	if err == nil {
		t.Fatal("expected error for missing page, got nil")
	}
}

// TestSetPageEnabled_DisableSingleLeaf disables a leaf page that has no children.
func TestSetPageEnabled_DisableSingleLeaf(t *testing.T) {
	dbh := setupEnableTestDB(t)
	defer dbh.Close()

	if err := dbh.SetPageEnabled("/section/b", false, false); err != nil {
		t.Fatalf("SetPageEnabled: %v", err)
	}

	if pageEnabled(t, dbh, "/section/b") {
		t.Error("/section/b should be disabled")
	}
	// Siblings and parent must be unaffected.
	if !pageEnabled(t, dbh, "/section") {
		t.Error("/section should still be enabled")
	}
	if !pageEnabled(t, dbh, "/section/a") {
		t.Error("/section/a should still be enabled")
	}
}

// TestSetPageEnabled_DisableCascadesToChildren disables a subtree recursively.
func TestSetPageEnabled_DisableCascadesToChildren(t *testing.T) {
	dbh := setupEnableTestDB(t)
	defer dbh.Close()

	if err := dbh.SetPageEnabled("/section", false, false); err != nil {
		t.Fatalf("SetPageEnabled: %v", err)
	}

	for _, route := range []string{"/section", "/section/a", "/section/b", "/section/a/deep"} {
		if pageEnabled(t, dbh, route) {
			t.Errorf("%s should be disabled", route)
		}
	}
	// Files under the disabled subtree must also be disabled.
	if fileEnabled(t, dbh, "/section/img.png") {
		t.Error("/section/img.png should be disabled")
	}
	if fileEnabled(t, dbh, "/section/a/logo.png") {
		t.Error("/section/a/logo.png should be disabled")
	}
	// Unrelated pages and files must be unaffected.
	if !pageEnabled(t, dbh, "/other") {
		t.Error("/other should still be enabled")
	}
	if !fileEnabled(t, dbh, "/other/doc.pdf") {
		t.Error("/other/doc.pdf should still be enabled")
	}
	if !pageEnabled(t, dbh, "/") {
		t.Error("/ should still be enabled")
	}
}

// TestSetPageEnabled_EnableNonRecursive enables only the target page and its
// direct files, leaving child pages untouched.
func TestSetPageEnabled_EnableNonRecursive(t *testing.T) {
	dbh := setupEnableTestDB(t)
	defer dbh.Close()

	// First disable the whole subtree.
	if err := dbh.SetPageEnabled("/section", false, false); err != nil {
		t.Fatalf("disable SetPageEnabled: %v", err)
	}

	// Enable only /section (non-recursive).
	if err := dbh.SetPageEnabled("/section", true, false); err != nil {
		t.Fatalf("enable SetPageEnabled: %v", err)
	}

	if !pageEnabled(t, dbh, "/section") {
		t.Error("/section should now be enabled")
	}
	// Direct files of /section must be re-enabled.
	if !fileEnabled(t, dbh, "/section/img.png") {
		t.Error("/section/img.png should be enabled (direct file of re-enabled page)")
	}
	// Child pages must remain disabled.
	if pageEnabled(t, dbh, "/section/a") {
		t.Error("/section/a should still be disabled (non-recursive enable)")
	}
	if pageEnabled(t, dbh, "/section/b") {
		t.Error("/section/b should still be disabled (non-recursive enable)")
	}
	if pageEnabled(t, dbh, "/section/a/deep") {
		t.Error("/section/a/deep should still be disabled (non-recursive enable)")
	}
	// Files of child pages must remain disabled.
	if fileEnabled(t, dbh, "/section/a/logo.png") {
		t.Error("/section/a/logo.png should still be disabled")
	}
}

// TestSetPageEnabled_EnableRecursive re-enables a full subtree.
func TestSetPageEnabled_EnableRecursive(t *testing.T) {
	dbh := setupEnableTestDB(t)
	defer dbh.Close()

	// Disable the whole subtree first.
	if err := dbh.SetPageEnabled("/section", false, false); err != nil {
		t.Fatalf("disable SetPageEnabled: %v", err)
	}

	// Enable recursively.
	if err := dbh.SetPageEnabled("/section", true, true); err != nil {
		t.Fatalf("enable recursive SetPageEnabled: %v", err)
	}

	for _, route := range []string{"/section", "/section/a", "/section/b", "/section/a/deep"} {
		if !pageEnabled(t, dbh, route) {
			t.Errorf("%s should be enabled", route)
		}
	}
	if !fileEnabled(t, dbh, "/section/img.png") {
		t.Error("/section/img.png should be enabled")
	}
	if !fileEnabled(t, dbh, "/section/a/logo.png") {
		t.Error("/section/a/logo.png should be enabled")
	}
}

// TestSetPageEnabled_DisableRootCascadesAll disables every page and file.
func TestSetPageEnabled_DisableRootCascadesAll(t *testing.T) {
	dbh := setupEnableTestDB(t)
	defer dbh.Close()

	if err := dbh.SetPageEnabled("/", false, false); err != nil {
		t.Fatalf("SetPageEnabled: %v", err)
	}

	for _, route := range []string{"/", "/section", "/section/a", "/section/b", "/section/a/deep", "/other"} {
		if pageEnabled(t, dbh, route) {
			t.Errorf("%s should be disabled", route)
		}
	}
	for _, route := range []string{"/section/img.png", "/section/a/logo.png", "/other/doc.pdf"} {
		if fileEnabled(t, dbh, route) {
			t.Errorf("file %s should be disabled", route)
		}
	}
}

// TestSetPageEnabled_EnableLeafOnly enables a leaf that has no children.
func TestSetPageEnabled_EnableLeafOnly(t *testing.T) {
	dbh := setupEnableTestDB(t)
	defer dbh.Close()

	if err := dbh.SetPageEnabled("/section/b", false, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if err := dbh.SetPageEnabled("/section/b", true, false); err != nil {
		t.Fatalf("enable: %v", err)
	}

	if !pageEnabled(t, dbh, "/section/b") {
		t.Error("/section/b should be enabled")
	}
}

// TestSetPageEnabled_RecursiveOnLeafHasNoEffect enables a leaf recursively
// (no descendants to cascade to).
func TestSetPageEnabled_RecursiveOnLeafHasNoEffect(t *testing.T) {
	dbh := setupEnableTestDB(t)
	defer dbh.Close()

	if err := dbh.SetPageEnabled("/section/b", false, false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if err := dbh.SetPageEnabled("/section/b", true, true); err != nil {
		t.Fatalf("enable recursive: %v", err)
	}

	if !pageEnabled(t, dbh, "/section/b") {
		t.Error("/section/b should be enabled")
	}
	// Siblings must be unaffected.
	if !pageEnabled(t, dbh, "/section/a") {
		t.Error("/section/a should remain enabled")
	}
}
