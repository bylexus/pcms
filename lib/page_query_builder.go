package lib

import (
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"alexi.ch/pcms/model"
)

// sqlFilter holds a parameterized WHERE clause fragment and its bind values.
type sqlFilter struct {
	clause string
	args   []any
}

// sqlOrder holds a single ORDER BY expression and direction.
type sqlOrder struct {
	expr      string
	direction string
	args      []any
}

// validJSONPath matches safe JSON path segments like "foo", "bar.baz", "items[0].name".
var validJSONPath = regexp.MustCompile(`^[a-zA-Z0-9_]+(\.[a-zA-Z0-9_]+|\[[0-9]+\])*$`)

// PageQueryBuilder provides a chainable, SQL-injection-safe query builder for
// indexed pages. It is designed to be used both from Go code and from pongo2
// templates.
//
// Every filter/ordering/paging method returns a shallow copy so calls can be
// chained without mutating the original builder.
//
// # Template usage
//
// The builder is exposed as the "PageQuery" factory function in pongo2 templates.
// Slice arguments (multiple JSON field paths) are created with the "List" helper.
//
//	{% for child in PageQuery().WhereParentRoute(page.Route).OrderBy("title","asc").FetchAll() %}
//	    {{ child.Title }}
//	{% endfor %}
//
//	{{ PageQuery().WhereMetadataEquals(List("tags"), "go").Count() }}
//
//	{{ PageQuery().WhereMetadataContains(List("tags","categories"), "tutorial").PageSize(5).Page(2).FetchAll() }}
type PageQueryBuilder struct {
	dbh      *DBH
	filters  []sqlFilter
	orders   []sqlOrder
	pageSize int // 0 = no limit
	page     int // 1-based, default 1
}

// NewPageQueryBuilder creates a new PageQueryBuilder using the given DBH instance.
func NewPageQueryBuilder(dbh *DBH) *PageQueryBuilder {
	return &PageQueryBuilder{
		dbh:  dbh,
		page: 1,
	}
}

// copy returns a shallow copy of the builder with independent filter/order slices.
func (b *PageQueryBuilder) copy() *PageQueryBuilder {
	c := *b
	c.filters = append([]sqlFilter{}, b.filters...)
	c.orders = append([]sqlOrder{}, b.orders...)
	return &c
}

// ---------- filter methods ----------

// WhereRoute adds a filter that matches pages by their route. Supports exact
// match (e.g. "/foo/bar") or prefix match with a trailing wildcard
// (e.g. "/foo/bar/*" matches all routes starting with "/foo/bar/").
//
// Template examples:
//
//	PageQuery().WhereRoute("/blog/post-1").First()
//	PageQuery().WhereRoute("/blog/*").FetchAll()
func (b *PageQueryBuilder) WhereRoute(route string) *PageQueryBuilder {
	c := b.copy()
	if strings.HasSuffix(route, "/*") {
		prefix := strings.TrimSuffix(route, "*")
		c.filters = append(c.filters, sqlFilter{
			clause: "(route = ? OR route LIKE ?)",
			args:   []any{strings.TrimSuffix(prefix, "/"), prefix + "%"},
		})
	} else {
		c.filters = append(c.filters, sqlFilter{
			clause: "route = ?",
			args:   []any{route},
		})
	}
	return c
}

// WhereParentRoute adds a filter that matches pages whose parent_page_route
// equals the given route.
//
// Template example:
//
//	PageQuery().WhereParentRoute("/blog").FetchAll()
func (b *PageQueryBuilder) WhereParentRoute(route string) *PageQueryBuilder {
	c := b.copy()
	c.filters = append(c.filters, sqlFilter{
		clause: "parent_page_route = ?",
		args:   []any{route},
	})
	return c
}

// WhereMetadataEquals adds a filter that matches pages where at least one of the
// given JSON field paths has the exact value. Multiple paths are ORed.
//
// Template example (single field):
//
//	PageQuery().WhereMetadataEquals(List("author"), "alice").FetchAll()
//
// Template example (multiple fields, ORed):
//
//	PageQuery().WhereMetadataEquals(List("author", "editor"), "alice").FetchAll()
func (b *PageQueryBuilder) WhereMetadataEquals(fields []string, value string) *PageQueryBuilder {
	c := b.copy()
	orParts, args := metadataOrClauses(fields, "= ?", value)
	if len(orParts) > 0 {
		c.filters = append(c.filters, sqlFilter{
			clause: "(" + strings.Join(orParts, " OR ") + ")",
			args:   args,
		})
	}
	return c
}

// WhereMetadataContains adds a filter that matches pages where at least one of
// the given JSON field paths contains the value. Works for both string values
// (substring match) and JSON array values (element match). Multiple paths are ORed.
//
// Template example:
//
//	PageQuery().WhereMetadataContains(List("tags", "categories"), "go").FetchAll()
func (b *PageQueryBuilder) WhereMetadataContains(fields []string, value string) *PageQueryBuilder {
	c := b.copy()
	var orParts []string
	var args []any
	for _, f := range fields {
		if !validJSONPath.MatchString(f) {
			continue
		}
		extract := fmt.Sprintf("json_extract(metadata_json, '$.%s')", f)
		// string contains: LIKE '%%value%%'
		// array contains: use two-argument json_each(doc, path) which handles
		// both scalar and array values without error on non-array fields.
		part := fmt.Sprintf(
			"(%s LIKE ? OR EXISTS(SELECT 1 FROM json_each(metadata_json, '$.%s') WHERE value = ?))",
			extract, f,
		)
		orParts = append(orParts, part)
		args = append(args, "%"+value+"%", value)
	}
	if len(orParts) > 0 {
		c.filters = append(c.filters, sqlFilter{
			clause: "(" + strings.Join(orParts, " OR ") + ")",
			args:   args,
		})
	}
	return c
}

// WhereMetadataIsOneOf adds a filter that matches pages where at least one of
// the given JSON field paths matches at least one of the given values. Works for
// both string and JSON array metadata values. Multiple paths and values are ORed.
//
// Template example:
//
//	PageQuery().WhereMetadataIsOneOf(List("tags"), List("go", "rust")).FetchAll()
func (b *PageQueryBuilder) WhereMetadataIsOneOf(fields []string, values []string) *PageQueryBuilder {
	c := b.copy()
	var orParts []string
	var args []any
	for _, f := range fields {
		if !validJSONPath.MatchString(f) {
			continue
		}
		extract := fmt.Sprintf("json_extract(metadata_json, '$.%s')", f)
		for _, v := range values {
			// exact string match OR array element match (two-argument json_each
			// handles both scalar and array values safely)
			part := fmt.Sprintf(
				"(%s = ? OR EXISTS(SELECT 1 FROM json_each(metadata_json, '$.%s') WHERE value = ?))",
				extract, f,
			)
			orParts = append(orParts, part)
			args = append(args, v, v)
		}
	}
	if len(orParts) > 0 {
		c.filters = append(c.filters, sqlFilter{
			clause: "(" + strings.Join(orParts, " OR ") + ")",
			args:   args,
		})
	}
	return c
}

// WhereMetadataLT adds a filter: metadata field < value (string comparison).
//
// Template example:
//
//	PageQuery().WhereMetadataLT(List("publish_date"), "2025-06-01").FetchAll()
func (b *PageQueryBuilder) WhereMetadataLT(fields []string, value string) *PageQueryBuilder {
	return b.whereMetadataCompare(fields, "< ?", value)
}

// WhereMetadataLTE adds a filter: metadata field <= value (string comparison).
func (b *PageQueryBuilder) WhereMetadataLTE(fields []string, value string) *PageQueryBuilder {
	return b.whereMetadataCompare(fields, "<= ?", value)
}

// WhereMetadataGT adds a filter: metadata field > value (string comparison).
func (b *PageQueryBuilder) WhereMetadataGT(fields []string, value string) *PageQueryBuilder {
	return b.whereMetadataCompare(fields, "> ?", value)
}

// WhereMetadataGTE adds a filter: metadata field >= value (string comparison).
//
// Template example:
//
//	PageQuery().WhereMetadataGTE(List("publish_date"), "2025-01-01").FetchAll()
func (b *PageQueryBuilder) WhereMetadataGTE(fields []string, value string) *PageQueryBuilder {
	return b.whereMetadataCompare(fields, ">= ?", value)
}

func (b *PageQueryBuilder) whereMetadataCompare(fields []string, op string, value string) *PageQueryBuilder {
	c := b.copy()
	orParts, args := metadataOrClauses(fields, op, value)
	if len(orParts) > 0 {
		c.filters = append(c.filters, sqlFilter{
			clause: "(" + strings.Join(orParts, " OR ") + ")",
			args:   args,
		})
	}
	return c
}

// ---------- ordering and paging ----------

// standardColumns lists page table columns that can be used directly in OrderBy
// without going through json_extract.
var standardColumns = map[string]string{
	"route":      "route",
	"title":      "title",
	"updated_at": "updated_at",
	"created_at": "created_at",
	"enabled":    "enabled",
}

// OrderBy adds a sort clause. The field can be a standard page column
// ("route", "title", "updated_at", "created_at", "enabled") or a metadata
// JSON path (e.g. "publish_date"). The direction must be "asc" or "desc".
// Multiple calls are cumulative.
//
// Template example:
//
//	PageQuery().OrderBy("title", "asc").FetchAll()
//	PageQuery().OrderBy("publish_date", "desc").FetchAll()
func (b *PageQueryBuilder) OrderBy(field string, direction string) *PageQueryBuilder {
	dir := strings.ToUpper(strings.TrimSpace(direction))
	if dir != "ASC" && dir != "DESC" {
		dir = "ASC"
	}

	c := b.copy()
	if col, ok := standardColumns[field]; ok {
		c.orders = append(c.orders, sqlOrder{expr: col, direction: dir})
	} else if validJSONPath.MatchString(field) {
		c.orders = append(c.orders, sqlOrder{
			expr:      fmt.Sprintf("json_extract(metadata_json, '$.%s')", field),
			direction: dir,
		})
	}
	return c
}

// PageSize sets the maximum number of results per page. A value <= 0 removes
// the limit. Non-cumulative: the last call wins.
//
// Template example:
//
//	PageQuery().PageSize(10).Page(2).FetchAll()
func (b *PageQueryBuilder) PageSize(size int) *PageQueryBuilder {
	c := b.copy()
	if size <= 0 {
		c.pageSize = 0
	} else {
		c.pageSize = size
	}
	return c
}

// Page sets the 1-based page number. Has no effect unless PageSize is set.
//
// Template example:
//
//	PageQuery().PageSize(10).Page(3).FetchAll()
func (b *PageQueryBuilder) Page(page int) *PageQueryBuilder {
	c := b.copy()
	if page < 1 {
		c.page = 1
	} else {
		c.page = page
	}
	return c
}

// ---------- terminal methods ----------

// FetchAll executes the query and returns all matching pages.
// The enabled flag is pre-computed during indexing, so enabled = 1 in the SQL
// filter is sufficient — no recursive ancestor check is needed at query time.
//
// Template example:
//
//	{% for p in PageQuery().WhereParentRoute("/blog").OrderBy("title","asc").FetchAll() %}
//	    {{ p.Title }}
//	{% endfor %}
func (b *PageQueryBuilder) FetchAll() []model.IndexedPage {
	query, args := b.buildSelectSQL()
	rows, err := b.dbh.queryIndex(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	return b.scanRows(rows)
}

// First executes the query and returns the first matching result, or nil if
// no result is found.
//
// Template example:
//
//	{% with featured=PageQuery().WhereMetadataEquals(List("featured"),"true").First() %}
//	    {{ featured.Title }}
//	{% endwith %}
func (b *PageQueryBuilder) First() *model.IndexedPage {
	c := b.copy()
	c.pageSize = 1
	c.page = 1

	query, args := c.buildSelectSQL()
	rows, err := c.dbh.queryIndex(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	page, ok := c.scanNextRow(rows)
	if !ok {
		return nil
	}
	return &page
}

// Count returns the total number of matching pages (ignoring PageSize/Page).
//
// Template example:
//
//	{{ PageQuery().WhereParentRoute("/blog").Count() }}
func (b *PageQueryBuilder) Count() int {
	query, args := b.buildCountSQL()
	rows, err := b.dbh.queryIndex(query, args...)
	if err != nil {
		return 0
	}
	defer rows.Close()

	if !rows.Next() {
		return 0
	}
	var count int
	if err := rows.Scan(&count); err != nil {
		return 0
	}
	return count
}

// NrOfPages returns the number of available result pages based on Count() and
// the configured PageSize. Returns 1 if PageSize is not set.
//
// Template example:
//
//	{{ PageQuery().WhereParentRoute("/blog").PageSize(10).NrOfPages() }}
func (b *PageQueryBuilder) NrOfPages() int {
	if b.pageSize <= 0 {
		return 1
	}
	total := b.Count()
	return int(math.Ceil(float64(total) / float64(b.pageSize)))
}

// ---------- SQL building ----------

func (b *PageQueryBuilder) buildWhereClause() (string, []any) {
	// Always filter for enabled = 1 at the SQL level as a first pass.
	clauses := []string{"enabled = 1"}
	var args []any

	for _, f := range b.filters {
		clauses = append(clauses, f.clause)
		args = append(args, f.args...)
	}
	return strings.Join(clauses, " AND "), args
}

func (b *PageQueryBuilder) buildOrderClause() string {
	if len(b.orders) == 0 {
		return ""
	}
	var parts []string
	for _, o := range b.orders {
		parts = append(parts, o.expr+" "+o.direction)
	}
	return " ORDER BY " + strings.Join(parts, ", ")
}

func (b *PageQueryBuilder) buildLimitOffset() (string, []any) {
	if b.pageSize <= 0 {
		return "", nil
	}
	offset := (b.page - 1) * b.pageSize
	if offset > 0 {
		return " LIMIT ? OFFSET ?", []any{b.pageSize, offset}
	}
	return " LIMIT ?", []any{b.pageSize}
}

func (b *PageQueryBuilder) buildSelectSQL() (string, []any) {
	where, args := b.buildWhereClause()
	query := "SELECT route, parent_page_route, title, index_file, enabled, metadata_json, updated_at FROM pages WHERE " + where
	query += b.buildOrderClause()
	limitSQL, limitArgs := b.buildLimitOffset()
	query += limitSQL
	args = append(args, limitArgs...)
	return query, args
}

func (b *PageQueryBuilder) buildCountSQL() (string, []any) {
	where, args := b.buildWhereClause()
	return "SELECT COUNT(1) FROM pages WHERE " + where, args
}

// ---------- row scanning ----------

func (b *PageQueryBuilder) scanRows(rows *sql.Rows) []model.IndexedPage {
	var pages []model.IndexedPage
	for rows.Next() {
		page, ok := scanPageRow(rows)
		if ok {
			pages = append(pages, page)
		}
	}
	return pages
}

func (b *PageQueryBuilder) scanNextRow(rows *sql.Rows) (model.IndexedPage, bool) {
	if !rows.Next() {
		return model.IndexedPage{}, false
	}
	return scanPageRow(rows)
}

// scanPageRow scans a single row with the standard pages column set.
func scanPageRow(rows *sql.Rows) (model.IndexedPage, bool) {
	var record model.IndexedPage
	var parentRoute sql.NullString
	var metadataJSON string
	var updatedAtStr string
	var enabledInt int

	if err := rows.Scan(
		&record.Route,
		&parentRoute,
		&record.Title,
		&record.IndexFile,
		&enabledInt,
		&metadataJSON,
		&updatedAtStr,
	); err != nil {
		return model.IndexedPage{}, false
	}

	if parentRoute.Valid {
		r := parentRoute.String
		record.ParentPageRoute = &r
	}
	record.Enabled = enabledInt != 0

	var err error
	record.Metadata, err = unmarshalMetadata(metadataJSON)
	if err != nil {
		return model.IndexedPage{}, false
	}

	record.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAtStr)
	if err != nil {
		return model.IndexedPage{}, false
	}

	return record, true
}

// ---------- helpers ----------

// metadataOrClauses builds OR-connected json_extract clauses for the given
// field paths and comparison operator.
func metadataOrClauses(fields []string, op string, value string) ([]string, []any) {
	var parts []string
	var args []any
	for _, f := range fields {
		if !validJSONPath.MatchString(f) {
			continue
		}
		part := fmt.Sprintf("json_extract(metadata_json, '$.%s') %s", f, op)
		parts = append(parts, part)
		args = append(args, value)
	}
	return parts, args
}
