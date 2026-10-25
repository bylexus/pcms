package lib

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"alexi.ch/pcms/model"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	defaultDBPath   = "pcms.db"
	currentDBSchema = 2
)

type DBH struct {
	db   *sql.DB
	path string
}

// dbExecutor is the common part of *sql.DB and *sql.Tx used by the write helpers.
type dbExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// IndexRun is one index run in its own transaction. Other users of the DBH
// (e.g. web requests) do not see its changes until Commit.
// An IndexRun must only be used by one goroutine.
type IndexRun struct {
	tx *sql.Tx
}

var (
	dbhInstance *DBH
	dbhErr      error
	dbhOnce     sync.Once
	dbhPath     = defaultDBPath
)

// SetDBPath configures the database path used by GetDBH.
// Must be called before the first call to GetDBH.
func SetDBPath(path string) {
	dbhPath = path
}

func GetDBH() (*DBH, error) {
	dbhOnce.Do(func() {
		h, err := OpenDBH(dbhPath)
		if err != nil {
			dbhErr = err
			return
		}
		dbhInstance = h
	})

	if dbhErr != nil {
		return nil, dbhErr
	}

	return dbhInstance, nil
}

func OpenDBH(dbPath string) (*DBH, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// In-memory databases are per-connection. Restrict the pool to a single
	// connection so that all operations share the same database.
	if dbPath == ":memory:" {
		db.SetMaxOpenConns(1)
	}

	h := &DBH{db: db, path: dbPath}
	if err := h.ensureSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return h, nil
}

func (h *DBH) Path() string {
	return h.path
}

func (h *DBH) SchemaVersion() int {
	return currentDBSchema
}

func (h *DBH) Close() error {
	if h.db == nil {
		return nil
	}
	return h.db.Close()
}

// BeginIndexRun starts a new index run in its own transaction.
// Call Commit to make the changes visible, or Rollback to discard them.
func (h *DBH) BeginIndexRun() (*IndexRun, error) {
	tx, err := h.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin index transaction: %w", err)
	}
	if _, err := tx.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = tx.Rollback()
		return nil, fmt.Errorf("enable foreign keys for index transaction: %w", err)
	}

	return &IndexRun{tx: tx}, nil
}

func (r *IndexRun) Commit() error {
	if err := r.tx.Commit(); err != nil {
		return fmt.Errorf("commit index transaction: %w", err)
	}
	return nil
}

// Rollback discards the changes of the run. It is a no-op after Commit.
func (r *IndexRun) Rollback() error {
	err := r.tx.Rollback()
	if err != nil && !errors.Is(err, sql.ErrTxDone) {
		return fmt.Errorf("rollback index transaction: %w", err)
	}
	return nil
}

func (r *IndexRun) CleanIndex() error {
	if _, err := r.tx.Exec("DELETE FROM files"); err != nil {
		return fmt.Errorf("clean files index: %w", err)
	}

	if _, err := r.tx.Exec("DELETE FROM pages"); err != nil {
		return fmt.Errorf("clean pages index: %w", err)
	}

	return nil
}

func (r *IndexRun) ReplacePage(record model.IndexedPage) error {
	return replacePage(r.tx, record)
}

func (r *IndexRun) ReplaceFile(record model.IndexedFile) error {
	return replaceFile(r.tx, record)
}

func (r *IndexRun) SetLastIndexInfo(source string, pageCount int, fileCount int) error {
	stmt := `
		UPDATE app_settings
		SET
			settings_json = json_set(
				coalesce(settings_json, '{}'),
				'$.last_indexed_at', ?,
				'$.last_index_source', ?,
				'$.last_index_page_count', ?,
				'$.last_index_file_count', ?
			),
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = 1
	`

	if _, err := r.tx.Exec(stmt, time.Now().UTC().Format(time.RFC3339Nano), source, pageCount, fileCount); err != nil {
		return fmt.Errorf("update index info setting: %w", err)
	}

	return nil
}

// ReplacePage inserts or updates a single page outside of an index run.
func (h *DBH) ReplacePage(record model.IndexedPage) error {
	return replacePage(h.db, record)
}

// ReplaceFile inserts or updates a single file outside of an index run.
func (h *DBH) ReplaceFile(record model.IndexedFile) error {
	return replaceFile(h.db, record)
}

func replacePage(ex dbExecutor, record model.IndexedPage) error {
	stmt := `
		INSERT INTO pages (route, parent_page_route, title, index_file, enabled, metadata_json)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(route) DO UPDATE SET
			parent_page_route = excluded.parent_page_route,
			title = excluded.title,
			index_file = excluded.index_file,
			enabled = excluded.enabled,
			metadata_json = excluded.metadata_json,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
	`

	metadataJSON, err := marshalMetadata(record.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata for page %s: %w", record.Route, err)
	}

	if _, err := ex.Exec(stmt, record.Route, record.ParentPageRoute, record.Title, record.IndexFile, record.Enabled, metadataJSON); err != nil {
		return fmt.Errorf("replace page %s: %w", record.Route, err)
	}

	return nil
}

func replaceFile(ex dbExecutor, record model.IndexedFile) error {
	stmt := `
		INSERT INTO files (route, parent_page_route, file_name, mime_type, file_size, enabled)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(route) DO UPDATE SET
			parent_page_route = excluded.parent_page_route,
			file_name = excluded.file_name,
			mime_type = excluded.mime_type,
			file_size = excluded.file_size,
			enabled = excluded.enabled,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
	`

	enabled := 1
	if !record.Enabled {
		enabled = 0
	}
	if _, err := ex.Exec(stmt, record.Route, record.ParentPageRoute, record.FileName, record.MimeType, record.FileSize, enabled); err != nil {
		return fmt.Errorf("replace file %s: %w", record.Route, err)
	}

	return nil
}

// IsDBBusy reports whether err is an SQLite "database is busy / locked" error,
// e.g. because an index run holds the write lock.
func IsDBBusy(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	// Code() can be an extended result code: the primary code is the low byte.
	code := sqliteErr.Code() & 0xff
	return code == sqlite3.SQLITE_BUSY || code == sqlite3.SQLITE_LOCKED
}

func (h *DBH) CountPages() (int, error) {
	row := h.db.QueryRow("SELECT COUNT(1) FROM pages")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count pages: %w", err)
	}
	return count, nil
}

func (h *DBH) CountFiles() (int, error) {
	row := h.db.QueryRow("SELECT COUNT(1) FROM files")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count files: %w", err)
	}
	return count, nil
}

func (h *DBH) GetPageByRoute(route string) (model.IndexedPage, bool, error) {
	stmt := `
		SELECT route, parent_page_route, title, index_file, enabled, metadata_json, updated_at
		FROM pages
		WHERE route = ?
	`

	var record model.IndexedPage
	var parentRoute sql.NullString
	var metadataJSON string
	var updatedAtStr string
	var enabledInt int
	err := h.db.QueryRow(stmt, route).Scan(
		&record.Route,
		&parentRoute,
		&record.Title,
		&record.IndexFile,
		&enabledInt,
		&metadataJSON,
		&updatedAtStr,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.IndexedPage{}, false, nil
		}
		return model.IndexedPage{}, false, fmt.Errorf("query page by route %s: %w", route, err)
	}

	if parentRoute.Valid {
		r := parentRoute.String
		record.ParentPageRoute = &r
	}
	record.Enabled = enabledInt != 0

	record.Metadata, err = unmarshalMetadata(metadataJSON)
	if err != nil {
		return model.IndexedPage{}, false, fmt.Errorf("unmarshal metadata for page %s: %w", route, err)
	}

	record.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAtStr)
	if err != nil {
		return model.IndexedPage{}, false, fmt.Errorf("parse updated_at for page %s: %w", route, err)
	}

	return record, true, nil
}

func (h *DBH) GetFileByRoute(route string) (model.IndexedFile, bool, error) {
	stmt := `
		SELECT route, parent_page_route, file_name, mime_type, file_size, enabled
		FROM files
		WHERE route = ?
	`

	var record model.IndexedFile
	var enabledInt int
	err := h.db.QueryRow(stmt, route).Scan(
		&record.Route,
		&record.ParentPageRoute,
		&record.FileName,
		&record.MimeType,
		&record.FileSize,
		&enabledInt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.IndexedFile{}, false, nil
		}
		return model.IndexedFile{}, false, fmt.Errorf("query file by route %s: %w", route, err)
	}
	record.Enabled = enabledInt != 0

	return record, true, nil
}

func (h *DBH) GetChildPages(route string) ([]model.IndexedPage, error) {
	stmt := `
		SELECT route, parent_page_route, title, index_file, enabled, metadata_json
		FROM pages
		WHERE parent_page_route = ?
		  AND enabled = 1
		ORDER BY route
	`

	rows, err := h.db.Query(stmt, route)
	if err != nil {
		return nil, fmt.Errorf("query child pages for %s: %w", route, err)
	}
	defer rows.Close()

	var pages []model.IndexedPage
	for rows.Next() {
		var record model.IndexedPage
		var parentRoute sql.NullString
		var metadataJSON string
		var enabledInt int
		if err := rows.Scan(
			&record.Route,
			&parentRoute,
			&record.Title,
			&record.IndexFile,
			&enabledInt,
			&metadataJSON,
		); err != nil {
			return nil, fmt.Errorf("scan child page for %s: %w", route, err)
		}
		if parentRoute.Valid {
			r := parentRoute.String
			record.ParentPageRoute = &r
		}
		record.Enabled = enabledInt != 0
		var err error
		record.Metadata, err = unmarshalMetadata(metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("unmarshal metadata for child page %s: %w", record.Route, err)
		}
		pages = append(pages, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate child pages for %s: %w", route, err)
	}

	return pages, nil
}

func (h *DBH) GetChildFiles(route string) ([]model.IndexedFile, error) {
	stmt := `
		SELECT route, parent_page_route, file_name, mime_type, file_size, enabled
		FROM files
		WHERE parent_page_route = ?
		  AND enabled = 1
		ORDER BY route
	`

	rows, err := h.db.Query(stmt, route)
	if err != nil {
		return nil, fmt.Errorf("query child files for %s: %w", route, err)
	}
	defer rows.Close()

	var files []model.IndexedFile
	for rows.Next() {
		var record model.IndexedFile
		var enabledInt int
		if err := rows.Scan(
			&record.Route,
			&record.ParentPageRoute,
			&record.FileName,
			&record.MimeType,
			&record.FileSize,
			&enabledInt,
		); err != nil {
			return nil, fmt.Errorf("scan child file for %s: %w", route, err)
		}
		record.Enabled = enabledInt != 0
		files = append(files, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate child files for %s: %w", route, err)
	}

	return files, nil
}

// SetPageEnabled updates the enabled flag for the given page and all its direct
// files. When enabled is false, all descendant pages and their files are also
// disabled (always recursive for disable). When enabled is true, descendants are
// only updated when recursive is true; without it only the target page and its
// direct files are updated.
//
// Returns an error if the page does not exist.
func (h *DBH) SetPageEnabled(route string, enabled bool, recursive bool) error {
	_, found, err := h.GetPageByRoute(route)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("page not found: %s", route)
	}

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	// Disable always cascades to all descendants; enable only cascades with -r.
	if !enabled || recursive {
		pageStmt := `
			WITH RECURSIVE subtree(route) AS (
				SELECT route FROM pages WHERE route = ?
				UNION ALL
				SELECT p.route FROM pages p
				INNER JOIN subtree s ON p.parent_page_route = s.route
			)
			UPDATE pages
			SET enabled = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
			WHERE route IN (SELECT route FROM subtree)
		`
		if _, err := h.db.Exec(pageStmt, route, enabledInt); err != nil {
			return fmt.Errorf("update pages enabled in subtree of %s: %w", route, err)
		}

		fileStmt := `
			WITH RECURSIVE subtree(route) AS (
				SELECT route FROM pages WHERE route = ?
				UNION ALL
				SELECT p.route FROM pages p
				INNER JOIN subtree s ON p.parent_page_route = s.route
			)
			UPDATE files
			SET enabled = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
			WHERE parent_page_route IN (SELECT route FROM subtree)
		`
		if _, err := h.db.Exec(fileStmt, route, enabledInt); err != nil {
			return fmt.Errorf("update files enabled in subtree of %s: %w", route, err)
		}
	} else {
		// Enable non-recursive: update only this page and its direct files.
		if _, err := h.db.Exec(
			"UPDATE pages SET enabled = 1, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE route = ?",
			route,
		); err != nil {
			return fmt.Errorf("update page enabled for %s: %w", route, err)
		}
		if _, err := h.db.Exec(
			"UPDATE files SET enabled = 1, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE parent_page_route = ?",
			route,
		); err != nil {
			return fmt.Errorf("update files enabled for %s: %w", route, err)
		}
	}

	return nil
}

func (h *DBH) ensureSchema() error {
	if _, err := h.db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := h.db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return fmt.Errorf("enable wal mode: %w", err)
	}

	if _, err := h.db.Exec("PRAGMA synchronous = NORMAL"); err != nil {
		return fmt.Errorf("set synchronous mode: %w", err)
	}

	if err := h.ensurePagesTable(); err != nil {
		return err
	}

	if err := h.ensureFilesTable(); err != nil {
		return err
	}

	if err := h.ensureAppSettingsTable(); err != nil {
		return err
	}

	if err := h.ensureDBVersionSetting(); err != nil {
		return err
	}

	return nil
}

func (h *DBH) ensurePagesTable() error {
	stmt := `
		CREATE TABLE IF NOT EXISTS pages (
			route             TEXT PRIMARY KEY,
			parent_page_route TEXT NULL REFERENCES pages(route)
				ON UPDATE CASCADE
				ON DELETE SET NULL,
			title             TEXT NOT NULL DEFAULT '',
			index_file        TEXT NOT NULL DEFAULT '',
			enabled           INTEGER NOT NULL DEFAULT 1,
			metadata_json     TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(metadata_json)),
			created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)
	`

	if _, err := h.db.Exec(stmt); err != nil {
		return fmt.Errorf("create pages table: %w", err)
	}

	if err := h.ensureTableColumn("pages", "parent_page_route", "TEXT NULL REFERENCES pages(route) ON UPDATE CASCADE ON DELETE SET NULL"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("pages", "title", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("pages", "index_file", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("pages", "enabled", "INTEGER NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("pages", "metadata_json", "TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(metadata_json))"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("pages", "created_at", "TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("pages", "updated_at", "TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))"); err != nil {
		return err
	}

	if _, err := h.db.Exec("CREATE INDEX IF NOT EXISTS idx_pages_parent_page_route ON pages(parent_page_route)"); err != nil {
		return fmt.Errorf("create pages parent index: %w", err)
	}

	return nil
}

func (h *DBH) ensureFilesTable() error {
	stmt := `
		CREATE TABLE IF NOT EXISTS files (
			route             TEXT PRIMARY KEY,
			parent_page_route TEXT NOT NULL REFERENCES pages(route)
				ON UPDATE CASCADE
				ON DELETE CASCADE,
			file_name         TEXT NOT NULL,
			mime_type         TEXT NOT NULL DEFAULT 'application/octet-stream',
			file_size         INTEGER NOT NULL DEFAULT 0 CHECK (file_size >= 0),
			enabled           INTEGER NOT NULL DEFAULT 1,
			created_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
			updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)
	`

	if _, err := h.db.Exec(stmt); err != nil {
		return fmt.Errorf("create files table: %w", err)
	}

	if err := h.ensureTableColumn("files", "parent_page_route", "TEXT NOT NULL REFERENCES pages(route) ON UPDATE CASCADE ON DELETE CASCADE"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("files", "file_name", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("files", "mime_type", "TEXT NOT NULL DEFAULT 'application/octet-stream'"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("files", "file_size", "INTEGER NOT NULL DEFAULT 0 CHECK (file_size >= 0)"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("files", "enabled", "INTEGER NOT NULL DEFAULT 1"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("files", "created_at", "TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("files", "updated_at", "TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))"); err != nil {
		return err
	}

	if _, err := h.db.Exec("CREATE INDEX IF NOT EXISTS idx_files_parent_page_route ON files(parent_page_route)"); err != nil {
		return fmt.Errorf("create files parent index: %w", err)
	}

	return nil
}

func (h *DBH) ensureAppSettingsTable() error {
	stmt := `
		CREATE TABLE IF NOT EXISTS app_settings (
			id            INTEGER PRIMARY KEY CHECK (id = 1),
			settings_json TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(settings_json)),
			updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
		)
	`

	if _, err := h.db.Exec(stmt); err != nil {
		return fmt.Errorf("create app_settings table: %w", err)
	}

	if err := h.ensureTableColumn("app_settings", "settings_json", "TEXT NOT NULL DEFAULT '{}' CHECK (json_valid(settings_json))"); err != nil {
		return err
	}
	if err := h.ensureTableColumn("app_settings", "updated_at", "TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))"); err != nil {
		return err
	}

	if _, err := h.db.Exec("INSERT INTO app_settings (id, settings_json) VALUES (1, '{}') ON CONFLICT(id) DO NOTHING"); err != nil {
		return fmt.Errorf("ensure app_settings singleton row: %w", err)
	}

	return nil
}

func (h *DBH) ensureDBVersionSetting() error {
	stmt := `
		UPDATE app_settings
		SET
			settings_json = json_set(coalesce(settings_json, '{}'), '$.db_version', ?),
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id = 1
	`

	if _, err := h.db.Exec(stmt, currentDBSchema); err != nil {
		return fmt.Errorf("update db_version setting: %w", err)
	}

	return nil
}

func (h *DBH) ensureTableColumn(tableName string, columnName string, columnDefinition string) error {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := h.db.Query(query)
	if err != nil {
		return fmt.Errorf("read table info for %s: %w", tableName, err)
	}
	defer rows.Close()

	hasColumn := false
	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("scan table info for %s: %w", tableName, err)
		}

		if name == columnName {
			hasColumn = true
			break
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table info for %s: %w", tableName, err)
	}

	if hasColumn {
		return nil
	}

	alterStmt := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, columnDefinition)
	if _, err := h.db.Exec(alterStmt); err != nil {
		return fmt.Errorf("add missing column %s.%s: %w", tableName, columnName, err)
	}

	return nil
}

func marshalMetadata(m map[string]any) (string, error) {
	if m == nil {
		return "{}", nil
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func unmarshalMetadata(s string) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func GetDBHForConfig(config model.Config) (*DBH, bool, error) {
	dbh, err := GetDBH()
	if err != nil {
		return nil, false, err
	}

	return dbh, false, nil
}
