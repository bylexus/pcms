package lib

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"alexi.ch/pcms/model"
	_ "modernc.org/sqlite"
)

const (
	defaultDBPath   = "pcms.db"
	currentDBSchema = 2
)

type DBH struct {
	db      *sql.DB
	path    string
	indexTx *sql.Tx
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
	if h.indexTx != nil {
		if err := h.RollbackIndexRun(); err != nil {
			return err
		}
	}
	if h.db == nil {
		return nil
	}
	return h.db.Close()
}

func (h *DBH) BeginIndexRun() error {
	if h.indexTx != nil {
		return fmt.Errorf("index transaction already active")
	}

	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("begin index transaction: %w", err)
	}
	if _, err := tx.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("enable foreign keys for index transaction: %w", err)
	}

	h.indexTx = tx
	return nil
}

func (h *DBH) CommitIndexRun() error {
	if h.indexTx == nil {
		return fmt.Errorf("no active index transaction")
	}

	err := h.indexTx.Commit()
	h.indexTx = nil
	if err != nil {
		return fmt.Errorf("commit index transaction: %w", err)
	}

	return nil
}

func (h *DBH) RollbackIndexRun() error {
	if h.indexTx == nil {
		return nil
	}

	err := h.indexTx.Rollback()
	h.indexTx = nil
	if err != nil && err != sql.ErrTxDone {
		return fmt.Errorf("rollback index transaction: %w", err)
	}

	return nil
}

func (h *DBH) CleanIndex() error {
	if _, err := h.execIndex("DELETE FROM files"); err != nil {
		return fmt.Errorf("clean files index: %w", err)
	}

	if _, err := h.execIndex("DELETE FROM pages"); err != nil {
		return fmt.Errorf("clean pages index: %w", err)
	}

	return nil
}

func (h *DBH) ReplacePage(record model.IndexedPage) error {
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

	if _, err := h.execIndex(stmt, record.Route, record.ParentPageRoute, record.Title, record.IndexFile, record.Enabled, metadataJSON); err != nil {
		return fmt.Errorf("replace page %s: %w", record.Route, err)
	}

	return nil
}

func (h *DBH) ReplaceFile(record model.IndexedFile) error {
	stmt := `
		INSERT INTO files (route, parent_page_route, file_name, mime_type, file_size)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(route) DO UPDATE SET
			parent_page_route = excluded.parent_page_route,
			file_name = excluded.file_name,
			mime_type = excluded.mime_type,
			file_size = excluded.file_size,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
	`

	if _, err := h.execIndex(stmt, record.Route, record.ParentPageRoute, record.FileName, record.MimeType, record.FileSize); err != nil {
		return fmt.Errorf("replace file %s: %w", record.Route, err)
	}

	return nil
}

func (h *DBH) SetLastIndexInfo(source string, pageCount int, fileCount int) error {
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

	if _, err := h.execIndex(stmt, time.Now().UTC().Format(time.RFC3339Nano), source, pageCount, fileCount); err != nil {
		return fmt.Errorf("update index info setting: %w", err)
	}

	return nil
}

func (h *DBH) CountPages() (int, error) {
	row := h.queryRowIndex("SELECT COUNT(1) FROM pages")
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count pages: %w", err)
	}
	return count, nil
}

func (h *DBH) CountFiles() (int, error) {
	row := h.queryRowIndex("SELECT COUNT(1) FROM files")
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
	err := h.queryRowIndex(stmt, route).Scan(
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
		SELECT route, parent_page_route, file_name, mime_type, file_size
		FROM files
		WHERE route = ?
	`

	var record model.IndexedFile
	err := h.queryRowIndex(stmt, route).Scan(
		&record.Route,
		&record.ParentPageRoute,
		&record.FileName,
		&record.MimeType,
		&record.FileSize,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.IndexedFile{}, false, nil
		}
		return model.IndexedFile{}, false, fmt.Errorf("query file by route %s: %w", route, err)
	}

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

	rows, err := h.queryIndex(stmt, route)
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

// IsPageEffectivelyEnabled checks whether the page itself and all its ancestor
// pages are enabled. Returns false if the page or any parent is disabled.
func (h *DBH) IsPageEffectivelyEnabled(page model.IndexedPage) (bool, error) {
	if !page.Enabled {
		return false, nil
	}
	parentRoute := page.ParentPageRoute
	for parentRoute != nil {
		parent, found, err := h.GetPageByRoute(*parentRoute)
		if err != nil {
			return false, err
		}
		if !found {
			break
		}
		if !parent.Enabled {
			return false, nil
		}
		parentRoute = parent.ParentPageRoute
	}
	return true, nil
}

func (h *DBH) GetChildFiles(route string) ([]model.IndexedFile, error) {
	stmt := `
		SELECT route, parent_page_route, file_name, mime_type, file_size
		FROM files
		WHERE parent_page_route = ?
		ORDER BY route
	`

	rows, err := h.queryIndex(stmt, route)
	if err != nil {
		return nil, fmt.Errorf("query child files for %s: %w", route, err)
	}
	defer rows.Close()

	var files []model.IndexedFile
	for rows.Next() {
		var record model.IndexedFile
		if err := rows.Scan(
			&record.Route,
			&record.ParentPageRoute,
			&record.FileName,
			&record.MimeType,
			&record.FileSize,
		); err != nil {
			return nil, fmt.Errorf("scan child file for %s: %w", route, err)
		}
		files = append(files, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate child files for %s: %w", route, err)
	}

	return files, nil
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

func (h *DBH) execIndex(query string, args ...any) (sql.Result, error) {
	if h.indexTx != nil {
		return h.indexTx.Exec(query, args...)
	}

	return h.db.Exec(query, args...)
}

func (h *DBH) queryRowIndex(query string, args ...any) *sql.Row {
	if h.indexTx != nil {
		return h.indexTx.QueryRow(query, args...)
	}

	return h.db.QueryRow(query, args...)
}

func (h *DBH) queryIndex(query string, args ...any) (*sql.Rows, error) {
	if h.indexTx != nil {
		return h.indexTx.Query(query, args...)
	}

	return h.db.Query(query, args...)
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
