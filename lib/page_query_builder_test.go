package lib

import (
	"path/filepath"
	"testing"

	"alexi.ch/pcms/model"
	"github.com/flosch/pongo2/v6"
)

// setupQueryBuilderDB creates a test DB and populates it with a page tree:
//
//	/              (root, enabled, metadata: {})
//	/blog          (child of /, enabled, metadata: {"tags":["go","tutorial"], "publish_date":"2025-03-01", "author":"alice"})
//	/blog/post-1   (child of /blog, enabled, metadata: {"tags":["go"], "publish_date":"2025-01-15", "author":"alice", "featured":"true"})
//	/blog/post-2   (child of /blog, enabled, metadata: {"tags":["rust","tutorial"], "publish_date":"2025-02-20", "author":"bob"})
//	/blog/draft    (child of /blog, DISABLED, metadata: {"tags":["go"], "publish_date":"2025-04-01", "author":"alice"})
//	/about         (child of /, enabled, metadata: {"author":"alice"})
//	/hidden        (child of /, DISABLED, metadata: {})
//	/hidden/child  (child of /hidden, enabled, metadata: {"author":"carol"})
func setupQueryBuilderDB(t *testing.T) *DBH {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "pcms-qb-test.db")
	dbh, err := OpenDBH(dbPath)
	if err != nil {
		t.Fatalf("OpenDBH() error = %v", err)
	}

	if err := dbh.BeginIndexRun(); err != nil {
		t.Fatalf("BeginIndexRun() error = %v", err)
	}

	root := "/"
	blog := "/blog"
	hidden := "/hidden"

	pages := []model.IndexedPage{
		{Route: "/", Title: "Root", IndexFile: "index.html", Enabled: true, Metadata: map[string]any{}},
		{Route: "/blog", ParentPageRoute: &root, Title: "Blog", IndexFile: "index.html", Enabled: true,
			Metadata: map[string]any{"tags": []any{"go", "tutorial"}, "publish_date": "2025-03-01", "author": "alice"}},
		{Route: "/blog/post-1", ParentPageRoute: &blog, Title: "First Post", IndexFile: "index.md", Enabled: true,
			Metadata: map[string]any{"tags": []any{"go"}, "publish_date": "2025-01-15", "author": "alice", "featured": "true"}},
		{Route: "/blog/post-2", ParentPageRoute: &blog, Title: "Second Post", IndexFile: "index.md", Enabled: true,
			Metadata: map[string]any{"tags": []any{"rust", "tutorial"}, "publish_date": "2025-02-20", "author": "bob"}},
		{Route: "/blog/draft", ParentPageRoute: &blog, Title: "Draft Post", IndexFile: "index.md", Enabled: false,
			Metadata: map[string]any{"tags": []any{"go"}, "publish_date": "2025-04-01", "author": "alice"}},
		{Route: "/about", ParentPageRoute: &root, Title: "About", IndexFile: "index.html", Enabled: true,
			Metadata: map[string]any{"author": "alice"}},
		{Route: "/hidden", ParentPageRoute: &root, Title: "Hidden Section", IndexFile: "index.html", Enabled: false,
			Metadata: map[string]any{}},
		{Route: "/hidden/child", ParentPageRoute: &hidden, Title: "Hidden Child", IndexFile: "index.html", Enabled: true,
			Metadata: map[string]any{"author": "carol"}},
	}

	for _, p := range pages {
		if err := dbh.ReplacePage(p); err != nil {
			t.Fatalf("ReplacePage(%s) error = %v", p.Route, err)
		}
	}

	if err := dbh.CommitIndexRun(); err != nil {
		t.Fatalf("CommitIndexRun() error = %v", err)
	}

	return dbh
}

func TestPageQueryBuilder_FetchAll_NoFilters(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	qb := NewPageQueryBuilder(dbh)
	pages := qb.FetchAll()

	// Should return all effectively enabled pages (6 total: /, /blog, /blog/post-1,
	// /blog/post-2, /about — but NOT /blog/draft (disabled), /hidden (disabled),
	// /hidden/child (parent disabled))
	if len(pages) != 5 {
		t.Fatalf("FetchAll() returned %d pages, want 5. Routes: %v", len(pages), pageRoutes(pages))
	}
}

func TestPageQueryBuilder_WhereParentRoute(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	pages := NewPageQueryBuilder(dbh).WhereParentRoute("/blog").FetchAll()

	// /blog has 3 children: post-1, post-2, draft. Draft is disabled, so 2.
	if len(pages) != 2 {
		t.Fatalf("WhereParentRoute(/blog).FetchAll() returned %d pages, want 2. Routes: %v", len(pages), pageRoutes(pages))
	}
	routes := pageRoutes(pages)
	assertContains(t, routes, "/blog/post-1")
	assertContains(t, routes, "/blog/post-2")
}

func TestPageQueryBuilder_WhereRoute_Exact(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	pages := NewPageQueryBuilder(dbh).WhereRoute("/blog/post-1").FetchAll()

	if len(pages) != 1 {
		t.Fatalf("WhereRoute(/blog/post-1) returned %d pages, want 1. Routes: %v", len(pages), pageRoutes(pages))
	}
	if pages[0].Route != "/blog/post-1" {
		t.Fatalf("expected /blog/post-1, got %s", pages[0].Route)
	}
}

func TestPageQueryBuilder_WhereRoute_Exact_NoMatch(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	pages := NewPageQueryBuilder(dbh).WhereRoute("/nonexistent").FetchAll()

	if len(pages) != 0 {
		t.Fatalf("WhereRoute(/nonexistent) returned %d pages, want 0", len(pages))
	}
}

func TestPageQueryBuilder_WhereRoute_Wildcard(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// /blog/* should match /blog itself plus all children under /blog/
	pages := NewPageQueryBuilder(dbh).WhereRoute("/blog/*").OrderBy("route", "asc").FetchAll()

	// Effectively enabled: /blog, /blog/post-1, /blog/post-2
	// NOT: /blog/draft (disabled)
	if len(pages) != 3 {
		t.Fatalf("WhereRoute(/blog/*) returned %d pages, want 3. Routes: %v", len(pages), pageRoutes(pages))
	}
	routes := pageRoutes(pages)
	assertContains(t, routes, "/blog")
	assertContains(t, routes, "/blog/post-1")
	assertContains(t, routes, "/blog/post-2")
}

func TestPageQueryBuilder_WhereRoute_Wildcard_DisabledParent(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// /hidden/* should match /hidden and /hidden/child at SQL level,
	// but both are filtered out (hidden=disabled, hidden/child=parent disabled)
	pages := NewPageQueryBuilder(dbh).WhereRoute("/hidden/*").FetchAll()

	if len(pages) != 0 {
		t.Fatalf("WhereRoute(/hidden/*) returned %d pages, want 0. Routes: %v", len(pages), pageRoutes(pages))
	}
}

func TestPageQueryBuilder_WhereRoute_First(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	page := NewPageQueryBuilder(dbh).WhereRoute("/about").First()
	if page == nil {
		t.Fatal("WhereRoute(/about).First() returned nil, want a page")
	}
	if page.Route != "/about" {
		t.Fatalf("expected /about, got %s", page.Route)
	}
}

func TestPageQueryBuilder_WhereMetadataEquals(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	pages := NewPageQueryBuilder(dbh).WhereMetadataEquals([]string{"author"}, "alice").FetchAll()

	// alice is author of: /blog, /blog/post-1, /about (but NOT /blog/draft which is disabled)
	if len(pages) != 3 {
		t.Fatalf("WhereMetadataEquals(author=alice) returned %d, want 3. Routes: %v", len(pages), pageRoutes(pages))
	}
}

func TestPageQueryBuilder_WhereMetadataEquals_MultipleFields(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// Search for "true" in fields "featured" OR "author" — featured=true is on post-1
	pages := NewPageQueryBuilder(dbh).WhereMetadataEquals([]string{"featured", "author"}, "true").FetchAll()

	if len(pages) != 1 {
		t.Fatalf("WhereMetadataEquals(featured|author=true) returned %d, want 1. Routes: %v", len(pages), pageRoutes(pages))
	}
	if pages[0].Route != "/blog/post-1" {
		t.Fatalf("expected /blog/post-1, got %s", pages[0].Route)
	}
}

func TestPageQueryBuilder_WhereMetadataContains_Array(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// Pages where tags array contains "go": /blog (tags:[go,tutorial]), /blog/post-1 (tags:[go])
	// NOT /blog/draft (disabled, tags:[go])
	pages := NewPageQueryBuilder(dbh).WhereMetadataContains([]string{"tags"}, "go").FetchAll()

	if len(pages) != 2 {
		t.Fatalf("WhereMetadataContains(tags,go) returned %d, want 2. Routes: %v", len(pages), pageRoutes(pages))
	}
}

func TestPageQueryBuilder_WhereMetadataContains_String(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// Pages where author contains "lic" (substring of "alice"):
	// /blog, /blog/post-1, /about (not /blog/draft=disabled, not /hidden/child=parent disabled)
	pages := NewPageQueryBuilder(dbh).WhereMetadataContains([]string{"author"}, "lic").FetchAll()

	if len(pages) != 3 {
		t.Fatalf("WhereMetadataContains(author,lic) returned %d, want 3. Routes: %v", len(pages), pageRoutes(pages))
	}
}

func TestPageQueryBuilder_WhereMetadataContains_MultipleFields(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// Search "tutorial" in tags OR author fields
	pages := NewPageQueryBuilder(dbh).WhereMetadataContains([]string{"tags", "author"}, "tutorial").FetchAll()

	// tags contains "tutorial": /blog, /blog/post-2
	if len(pages) != 2 {
		t.Fatalf("WhereMetadataContains(tags|author,tutorial) returned %d, want 2. Routes: %v", len(pages), pageRoutes(pages))
	}
}

func TestPageQueryBuilder_WhereMetadataIsOneOf(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// Pages where tags contain "go" OR "rust"
	pages := NewPageQueryBuilder(dbh).WhereMetadataIsOneOf([]string{"tags"}, []string{"go", "rust"}).FetchAll()

	// /blog (tags:[go,tutorial]), /blog/post-1 (tags:[go]), /blog/post-2 (tags:[rust,tutorial])
	if len(pages) != 3 {
		t.Fatalf("WhereMetadataIsOneOf(tags,[go,rust]) returned %d, want 3. Routes: %v", len(pages), pageRoutes(pages))
	}
}

func TestPageQueryBuilder_WhereMetadataGTE(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// Pages with publish_date >= "2025-02-01"
	pages := NewPageQueryBuilder(dbh).WhereMetadataGTE([]string{"publish_date"}, "2025-02-01").FetchAll()

	// /blog (2025-03-01), /blog/post-2 (2025-02-20) — NOT /blog/draft (2025-04-01, disabled)
	if len(pages) != 2 {
		t.Fatalf("WhereMetadataGTE(publish_date>=2025-02-01) returned %d, want 2. Routes: %v", len(pages), pageRoutes(pages))
	}
}

func TestPageQueryBuilder_WhereMetadataLT(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// Pages with publish_date < "2025-02-01"
	pages := NewPageQueryBuilder(dbh).WhereMetadataLT([]string{"publish_date"}, "2025-02-01").FetchAll()

	// /blog/post-1 (2025-01-15)
	if len(pages) != 1 {
		t.Fatalf("WhereMetadataLT(publish_date<2025-02-01) returned %d, want 1. Routes: %v", len(pages), pageRoutes(pages))
	}
	if pages[0].Route != "/blog/post-1" {
		t.Fatalf("expected /blog/post-1, got %s", pages[0].Route)
	}
}

func TestPageQueryBuilder_ChainedFilters(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// Combine: parent=/blog AND author=alice
	pages := NewPageQueryBuilder(dbh).
		WhereParentRoute("/blog").
		WhereMetadataEquals([]string{"author"}, "alice").
		FetchAll()

	// /blog/post-1 (parent=/blog, author=alice) — NOT /blog/draft (disabled)
	if len(pages) != 1 {
		t.Fatalf("chained filters returned %d, want 1. Routes: %v", len(pages), pageRoutes(pages))
	}
	if pages[0].Route != "/blog/post-1" {
		t.Fatalf("expected /blog/post-1, got %s", pages[0].Route)
	}
}

func TestPageQueryBuilder_OrderBy_Title(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	pages := NewPageQueryBuilder(dbh).WhereParentRoute("/blog").OrderBy("title", "asc").FetchAll()

	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	if pages[0].Title != "First Post" {
		t.Fatalf("first page title = %q, want 'First Post'", pages[0].Title)
	}
	if pages[1].Title != "Second Post" {
		t.Fatalf("second page title = %q, want 'Second Post'", pages[1].Title)
	}
}

func TestPageQueryBuilder_OrderBy_MetadataField(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	pages := NewPageQueryBuilder(dbh).WhereParentRoute("/blog").OrderBy("publish_date", "desc").FetchAll()

	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	// desc: post-2 (2025-02-20) before post-1 (2025-01-15)
	if pages[0].Route != "/blog/post-2" {
		t.Fatalf("first page = %s, want /blog/post-2", pages[0].Route)
	}
	if pages[1].Route != "/blog/post-1" {
		t.Fatalf("second page = %s, want /blog/post-1", pages[1].Route)
	}
}

func TestPageQueryBuilder_OrderBy_InvalidDirection(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// Invalid direction should default to ASC
	pages := NewPageQueryBuilder(dbh).WhereParentRoute("/blog").OrderBy("title", "invalid").FetchAll()
	if len(pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(pages))
	}
	if pages[0].Title != "First Post" {
		t.Fatalf("first page title = %q, want 'First Post' (ASC default)", pages[0].Title)
	}
}

func TestPageQueryBuilder_PageSize_And_Page(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// All enabled pages under /blog ordered by title: First Post, Second Post
	qb := NewPageQueryBuilder(dbh).WhereParentRoute("/blog").OrderBy("title", "asc").PageSize(1)

	page1 := qb.Page(1).FetchAll()
	if len(page1) != 1 {
		t.Fatalf("Page(1) returned %d pages, want 1", len(page1))
	}
	if page1[0].Title != "First Post" {
		t.Fatalf("Page(1) title = %q, want 'First Post'", page1[0].Title)
	}

	page2 := qb.Page(2).FetchAll()
	if len(page2) != 1 {
		t.Fatalf("Page(2) returned %d pages, want 1", len(page2))
	}
	if page2[0].Title != "Second Post" {
		t.Fatalf("Page(2) title = %q, want 'Second Post'", page2[0].Title)
	}

	page3 := qb.Page(3).FetchAll()
	if len(page3) != 0 {
		t.Fatalf("Page(3) returned %d pages, want 0", len(page3))
	}
}

func TestPageQueryBuilder_First(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	page := NewPageQueryBuilder(dbh).WhereMetadataEquals([]string{"featured"}, "true").First()
	if page == nil {
		t.Fatal("First() returned nil, want a page")
	}
	if page.Route != "/blog/post-1" {
		t.Fatalf("First() route = %s, want /blog/post-1", page.Route)
	}
}

func TestPageQueryBuilder_First_NoResult(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	page := NewPageQueryBuilder(dbh).WhereMetadataEquals([]string{"featured"}, "nonexistent").First()
	if page != nil {
		t.Fatalf("First() returned %v, want nil", page.Route)
	}
}

func TestPageQueryBuilder_Count(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	count := NewPageQueryBuilder(dbh).WhereParentRoute("/blog").Count()
	if count != 2 {
		t.Fatalf("Count() = %d, want 2", count)
	}
}

func TestPageQueryBuilder_Count_All(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	count := NewPageQueryBuilder(dbh).Count()
	// 5 effectively enabled pages
	if count != 5 {
		t.Fatalf("Count() = %d, want 5", count)
	}
}

func TestPageQueryBuilder_NrOfPages(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// 5 enabled pages total, page size 2 => 3 result pages
	nr := NewPageQueryBuilder(dbh).PageSize(2).NrOfPages()
	if nr != 3 {
		t.Fatalf("NrOfPages() = %d, want 3", nr)
	}
}

func TestPageQueryBuilder_NrOfPages_NoPageSize(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	nr := NewPageQueryBuilder(dbh).NrOfPages()
	if nr != 1 {
		t.Fatalf("NrOfPages() without PageSize = %d, want 1", nr)
	}
}

func TestPageQueryBuilder_DisabledPage_NotReturned(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// /blog/draft is directly disabled — should not appear
	pages := NewPageQueryBuilder(dbh).FetchAll()
	for _, p := range pages {
		if p.Route == "/blog/draft" {
			t.Fatal("disabled page /blog/draft should not be in results")
		}
	}
}

func TestPageQueryBuilder_DisabledParent_ChildNotReturned(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// /hidden/child has enabled=true but parent /hidden is disabled
	pages := NewPageQueryBuilder(dbh).FetchAll()
	for _, p := range pages {
		if p.Route == "/hidden/child" {
			t.Fatal("/hidden/child should not be in results (parent is disabled)")
		}
		if p.Route == "/hidden" {
			t.Fatal("/hidden should not be in results (disabled)")
		}
	}
}

func TestPageQueryBuilder_Immutability(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	base := NewPageQueryBuilder(dbh).WhereParentRoute("/")

	// Adding more filters to a derived builder must not affect base
	_ = base.WhereMetadataEquals([]string{"author"}, "alice").FetchAll()

	pages := base.FetchAll()
	// Children of /: /blog, /about (not /hidden which is disabled)
	if len(pages) != 2 {
		t.Fatalf("base builder was mutated: got %d pages, want 2. Routes: %v", len(pages), pageRoutes(pages))
	}
}

func TestPageQueryBuilder_InvalidJSONPath_Ignored(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// SQL injection attempt via field path: should be silently ignored
	pages := NewPageQueryBuilder(dbh).WhereMetadataEquals([]string{"'; DROP TABLE pages; --"}, "value").FetchAll()

	// Invalid path is skipped, no filter applied besides enabled=1, so all 5 enabled pages returned
	if len(pages) != 5 {
		t.Fatalf("invalid path filter returned %d pages, want 5", len(pages))
	}
}

func TestPageQueryBuilder_EmptyFieldsList(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	// Empty fields list: no filter added, returns all enabled
	pages := NewPageQueryBuilder(dbh).WhereMetadataEquals([]string{}, "value").FetchAll()
	if len(pages) != 5 {
		t.Fatalf("empty fields list returned %d pages, want 5", len(pages))
	}
}

// --- pongo2 integration tests ---

func TestPageQueryBuilder_Pongo2_WhereRoute_Wildcard(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	tpl, err := pongo2.FromString(`{% for p in PageQuery().WhereRoute("/blog/*").OrderBy("title","asc").FetchAll() %}[{{p.Title}}]{% endfor %}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	out, err := tpl.Execute(pongo2.Context{
		"PageQuery": func() *PageQueryBuilder {
			return NewPageQueryBuilder(dbh)
		},
	})
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}

	expected := "[Blog][First Post][Second Post]"
	if out != expected {
		t.Fatalf("output = %q, want %q", out, expected)
	}
}

func TestPageQueryBuilder_Pongo2_WhereRoute_Exact(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	tpl, err := pongo2.FromString(`{% with p=PageQuery().WhereRoute("/about").First() %}{{p.Title}}{% endwith %}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	out, err := tpl.Execute(pongo2.Context{
		"PageQuery": func() *PageQueryBuilder {
			return NewPageQueryBuilder(dbh)
		},
	})
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}

	if out != "About" {
		t.Fatalf("output = %q, want 'About'", out)
	}
}

func TestPageQueryBuilder_Pongo2_FetchAllInForLoop(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	tpl, err := pongo2.FromString(`{% for p in PageQuery().WhereParentRoute("/blog").OrderBy("title","asc").FetchAll() %}[{{p.Title}}]{% endfor %}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	out, err := tpl.Execute(pongo2.Context{
		"PageQuery": func() *PageQueryBuilder {
			return NewPageQueryBuilder(dbh)
		},
	})
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}

	expected := "[First Post][Second Post]"
	if out != expected {
		t.Fatalf("output = %q, want %q", out, expected)
	}
}

func TestPageQueryBuilder_Pongo2_WithVariableArg(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	tpl, err := pongo2.FromString(`{% for p in PageQuery().WhereParentRoute(page.Route).FetchAll() %}[{{p.Title}}]{% endfor %}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	out, err := tpl.Execute(pongo2.Context{
		"PageQuery": func() *PageQueryBuilder {
			return NewPageQueryBuilder(dbh)
		},
		"page": map[string]any{"Route": "/"},
	})
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}

	// Children of / that are enabled: /about, /blog (alphabetical by route from DB)
	if out != "[About][Blog]" && out != "[Blog][About]" {
		t.Fatalf("output = %q, want [About][Blog] or [Blog][About]", out)
	}
}

func TestPageQueryBuilder_Pongo2_ListHelper(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	tpl, err := pongo2.FromString(`{{ PageQuery().WhereMetadataEquals(List("author"), "bob").Count() }}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	out, err := tpl.Execute(pongo2.Context{
		"PageQuery": func() *PageQueryBuilder {
			return NewPageQueryBuilder(dbh)
		},
		"List": func(items ...string) []string {
			return items
		},
	})
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}

	// bob is author of /blog/post-2 only
	if out != "1" {
		t.Fatalf("output = %q, want '1'", out)
	}
}

func TestPageQueryBuilder_Pongo2_ListHelper_IsOneOf(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	tpl, err := pongo2.FromString(`{{ PageQuery().WhereMetadataIsOneOf(List("tags"), List("go","rust")).Count() }}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	out, err := tpl.Execute(pongo2.Context{
		"PageQuery": func() *PageQueryBuilder {
			return NewPageQueryBuilder(dbh)
		},
		"List": func(items ...string) []string {
			return items
		},
	})
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}

	// go or rust in tags: /blog, /blog/post-1, /blog/post-2
	if out != "3" {
		t.Fatalf("output = %q, want '3'", out)
	}
}

func TestPageQueryBuilder_Pongo2_First(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	tpl, err := pongo2.FromString(`{% with p=PageQuery().WhereMetadataEquals(List("featured"),"true").First() %}{{p.Title}}{% endwith %}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	out, err := tpl.Execute(pongo2.Context{
		"PageQuery": func() *PageQueryBuilder {
			return NewPageQueryBuilder(dbh)
		},
		"List": func(items ...string) []string {
			return items
		},
	})
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}

	if out != "First Post" {
		t.Fatalf("output = %q, want 'First Post'", out)
	}
}

func TestPageQueryBuilder_Pongo2_NrOfPages(t *testing.T) {
	dbh := setupQueryBuilderDB(t)
	defer dbh.Close()

	tpl, err := pongo2.FromString(`{{ PageQuery().PageSize(2).NrOfPages() }}`)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	out, err := tpl.Execute(pongo2.Context{
		"PageQuery": func() *PageQueryBuilder {
			return NewPageQueryBuilder(dbh)
		},
	})
	if err != nil {
		t.Fatalf("exec error: %v", err)
	}

	// 5 enabled pages / pageSize 2 = 3 pages
	if out != "3" {
		t.Fatalf("output = %q, want '3'", out)
	}
}

// --- helpers ---

func pageRoutes(pages []model.IndexedPage) []string {
	var routes []string
	for _, p := range pages {
		routes = append(routes, p.Route)
	}
	return routes
}

func assertContains(t *testing.T, haystack []string, needle string) {
	t.Helper()
	for _, s := range haystack {
		if s == needle {
			return
		}
	}
	t.Fatalf("expected %v to contain %q", haystack, needle)
}
