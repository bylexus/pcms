# Rewrite implementation

As mentioned in the readme, I want to rewrite pcms to use a db index.

## DB architecture

- sqlite-based, using modernc.org/sqlite (no external C lib dependencies)

## Decision summary

- Primary in-process DB: SQLite
- Driver: `modernc.org/sqlite` (pure Go, no separate C library required)
- Keep schema minimal for first implementation step: `pages`, `files`, `app_settings`
- Keep metadata as JSON blobs in table columns (no dedicated metadata table)

## Minimal schema plan

### `pages`

- Stores one page object per route
- `route` is the primary key (must be unique)
- `parent_page_route` is optional and references `pages.route`
- Keep content fields aligned with planned `Page` model:
  - `title`
  - `index_file`
  - `metadata_json` (JSON blob, schema-less)
- Include timestamps (`created_at`, `updated_at`) for index and cache workflows
- Add index on `parent_page_route` for child-page lookups

### `files`

- Stores non-page files addressable by route
- `route` is the primary key (must be unique)
- `parent_page_route` references owning page (`pages.route`)
- Keep fields aligned with planned `File` model:
  - `file_name`
  - `mime_type`
  - `file_size`
  - `metadata_json` (JSON blob, schema-less)
- Include timestamps (`created_at`, `updated_at`)
- Add index on `parent_page_route` for per-page file listing

### `app_settings`

- Single table for arbitrary app-level settings
- Store settings as one JSON entry (`settings_json`)
- Intended for values like:
  - last indexing time
  - index/build info
  - internal rewrite/runtime flags
- Use singleton-row pattern (fixed `id = 1`)

### Constraints and behavior notes

- Enable SQLite foreign keys (`PRAGMA foreign_keys = ON`)
- Use JSON validity checks on JSON columns (`json_valid(...)`)
- Prefer WAL mode for serving workflow (`PRAGMA journal_mode = WAL`)
- Keep route identity canonical and stable, because route is the DB key for both pages and files

## Implementation step 1: Base DB architecture [DONE]

A centralized db handler should be created that collects all the needed functionality to access the DB and manages the schema.

- `lib/db.go` should define a package which contain all the code
- It defines a struct, `DBH`, that contains:
  - a sqlite connection to a local file, `pcms.db`
  - all functionality as struct function
- a function `GetDBH` to get the (single) DBH instance - it should create one instance of the DBH and return it to the caller - we want to get
  access to the single instance through the program
- On instantiation, the DB handler must make sure that its db schema is on the correct schema version:
  - create needed tables / columns if this is not the case
  - store the actual db version (the db version itself is fixed in the code) in the app_settings table (`db_version` in the `settings_json`)

The program now can easily get the singleton DB handler instance:

```golang
import (
	"alexi.ch/pcms/lib"
)
	dbh, err := lib.GetDBH()
```

### Status

Implemented in `lib/db.go`:
- `DBH` struct with singleton via `sync.Once` pattern (`GetDBH()`, `GetDBHForConfig()`)
- Schema: 3 tables (`pages`, `files`, `app_settings`) with WAL mode, foreign keys, JSON validity checks
- Full index lifecycle: `BeginIndexRun`, `CommitIndexRun`, `RollbackIndexRun`, `ReplacePage`, `ReplaceFile`, `CleanIndex`
- Query APIs: `GetPageByRoute()`, `GetFileByRoute()`, `CountPages()`, `CountFiles()`
- Uses `modernc.org/sqlite` (pure Go)

## Implementation step 2: Index command [DONE]

The index command creates the page and file index based on the given file system (`fs.FS`). The file system is either a directory on the disk (os.DirFs), or an embedded
file system as defined in the main.go file (go:embed). This fs is the root route ('/').

The index command now traverses this tree and creates page and file entries in the index db. The base logic can already be found in the commands/build.go file, but this was
implemented for the old static site builder structure. This needs to be refactored now.

The goal is to have the full page tree in the sqlite db.

The `pcms index` command now creates the full page index based on the configured fs.FS. It creates entries in the `pages` and `files` tables in the sqlite db
by walking the file tree.

### Status

Implemented in `commands/index.go` and `lib/treewalk.go`:
- `RunIndexCmd()` orchestrates: get source FS, build snapshot, persist to DB transactionally
- `BuildIndexSnapshot()` walks `fs.FS`, creates `IndexedPageRecord` / `IndexedFileRecord` entries
- Frontmatter metadata extraction (YAML), title fallback logic, parent page tracking
- File association via foreign key to nearest ancestor page
- Regex-based path exclusion, MIME type detection via `gabriel-vasile/mimetype`
- Supports both filesystem and embedded doc sources
- `commands/build.go` removed

## Implementation step 3: serve command [DONE]

The `pcms serve` command starts the web server and begins serving the pages. This involves the following steps:

1. Setup and start web server:
   1. setup logging
   2. creating middlewares (e.g. access logger middleware to log web server access)
   3. register the handler to process routes
   4. start the http.Server
2. The handler then gets requests and processes them:
   1. it extracts the URL route from the request
   2. it checks if there is a page or file entry in the index matching the request
		- if it's a page, process the page (see description below)
		- if it's a file, deliver it directly

### handle page requests

Handling a page request involves the following steps:

1. determine if the page exists in the page cache:
   - if there is a cached version, and the cache is valid (e.g. newer than the index file), deliver it, and done.
   - if there is no cached version, or if the cache needs updates, continue with step 2
2. process the page index file: 
    - determine the page type: Is it an HTML or Markdown page?
	- determine the page processor to be used (see processor package), and process the file
	- the output then is cached
3. deliver the processed file (from cache, as there IS now a cached variant) to the client


The cache is just a directory (configurable in the pcms-config.yaml, `cacheDir` in the `server` section) that keeps the rendered / processed files.

### handle file requests
   
File requests are just delivered back to the client using the correct mime type

## Implementation plan for step 3

### Goals and boundaries

- Serve content based on DB index entries from step 2 (`pages` + `files`), not by direct path probing. Only content in the index is handled, other routes get a 404.
- Keep current server bootstrap flow from `commands/serve.go` (logging, middleware, `http.Server`) and evolve the request handler logic.
- Reuse existing processing behavior where feasible, especially index-file rendering logic already implemented in `processor`.
	- If template variables change caused by the new structure, update the documentation in `doc/site/**/*.md` 
	- goal for now is to keep the available variables - we will refactor them later for the new structure.
- Add page cache support based on configured `server.cacheDir` (`cacheDir` in `pcms-config.yaml`).

### 1) Config and startup wiring [DONE]

1. Extend `model.Config`:
   - add `Server.CacheDir string \`yaml:"cacheDir"\``.
   - resolve it to absolute path in `NewConfig` for `serve` mode.
   - define a safe default when missing (e.g. `.pcms-cache` relative to config dir).
2. In `commands.RunServeCmd`:
   - keep existing logger + middleware setup.
   - open DB via `lib.GetDBH()` and pass DB handle + resolved `siteFS` + cache dir to the request handler.
   - keep existing embedded-doc/file serve mode selection.

### 2) DB read API for runtime lookup [DONE]

Add query methods to `lib.DBH` for request-time resolution:

1. `GetPageByRoute(route string) (IndexedPageRecord, bool, error)`.
2. `GetFileByRoute(route string) (IndexedFileRecord, bool, error)`.
3. (Optional helper) `ResolvePageRoute(route string)` that normalizes trailing slash behavior for page routes.

Notes:
- Keep route matching canonical (`/foo` and `/foo/` normalization in handler before lookup).
- Reuse existing record structs from `lib/treewalk.go` to avoid duplicate models.

### 3) Handler rewrite (DB-first dispatch) [DONE]

In `webserver/handler.go`, replace filesystem-first `fs.Stat` flow with DB lookup flow:

1. Normalize incoming URL path to canonical route.
2. Query DB for matching page route first.
3. If page exists: run page handling pipeline (cache check -> render when needed -> serve cache file).
4. Else query DB for file route.
5. If file exists: serve file directly from `siteFS` with DB mime type.
6. If neither exists: 404.

This preserves existing middleware and logging behavior while changing only route resolution and response generation.

### 4) Page cache design [DONE]

Cache storage uses a dedicated directory tree under `server.cacheDir`:

1. Deterministic cache path by route:
   - `/` -> `<cacheDir>/index.html`
   - `/blog` -> `<cacheDir>/blog/index.html`
2. Cache validity rule:
   - valid if cache file exists and cache mtime >= source index file mtime.
3. Source index file path reconstruction from DB page record:
   - route + `index_file` from DB (`/blog` + `index.md` => `blog/index.md`, root => `index.md`).
4. If invalid/missing cache: render page and overwrite cache atomically.

### 5) Processor package changes (required) [DONE]

Current processors are build-oriented (`ProcessFile` writes directly to `dest`). Step 3 needs render-for-serve behavior. Planned refactor:

1. Introduce render-centric API for page processors (`html`, `md`):
   - input: source file (or source bytes + logical path), config, and path context.
   - output: rendered HTML bytes/string (no forced write to `dest`).
2. Keep existing `ProcessFile` for build compatibility, but implement it via the new render API + file write wrapper.
3. Add helper(s) to resolve source from `fs.FS` for serve mode:
   - read index file from `siteFS`.
   - still support current template context variables (`variables`, `paths`, `webroot`, helpers).
4. Add a shared selector for page processor choice by index file extension (`index.html` vs `index.md`), reusing existing selection logic from `processor.GetProcessor` where feasible.

This keeps processor behavior consistent across `build` and `serve`, while avoiding duplicate render logic.

The 'ScssProcessor' can be removed: we do no longer support SCSS building.

### 6) Reuse from `commands/build.go` [DONE]

**Note:** The build command is ONLY used for reference! It would / should not run / build anymore. After the implementation of the serve command, it can be removed.

The old build flow still contains reusable pieces:

1. Reuse file exclusion and processor-selection patterns (`processor.IsFileExcluded`, extension-based processor selection).
2. Reuse the separation of concerns:
   - command layer orchestrates,
   - processor layer renders/transforms,
   - handler layer serves.
3. Avoid reusing recursive filesystem traversal from build for request handling (serve must use DB lookup), but keep traversal logic in indexing (step 2) as already refactored into `lib/treewalk.go`.

### 7) File serving details [DONE]

For DB-matched files:

1. Open source from `siteFS` by route-derived fs path.
2. Set `Content-Type` from DB `mime_type`.
3. Serve via `http.ServeContent` / stream to response.
4. Return 404 if DB entry exists but source file is missing (stale index case), with error log entry.

### 8) Tests to add/update [PARTIAL]

1. `lib/db` tests: [DONE]
   - query methods by route (found/not found behavior).
   - `TestDBHIndexLifecycle`, `TestDBHIndexForeignKeyIntegrity`, `TestDBHGetByRoute`
2. `webserver` handler tests: [PARTIAL]
   - route normalization tests exist (`handler_test.go`)
   - TODO: page request hits cache when valid.
   - TODO: page request re-renders when cache stale/missing.
   - TODO: file request returns DB mime type.
   - TODO: unknown route returns 404.
3. `processor` tests: [DONE]
   - render API parity with existing output behavior for md/html.
   - index-file extension dispatch.
4. Integration test (serve path): [TODO]
   - run index + serve handler against fixture FS and assert page/file responses.

### 9) Incremental execution order

1. [x] Add config + DB query APIs.
2. [x] Refactor processors to shared render API (keep build backward compatible).
3. [x] Rewrite request handler to DB-first routing and cache pipeline.
4. [x] Implement direct file serving from `siteFS` using DB metadata.
5. [x] remove unnecessary code like build.go
6. [ ] Add/update tests, then run `go test ./...` and `go build ./...`.
   - DB and treewalk tests: done
   - Processor tests: done
   - Handler tests: partial (route normalization only, missing cache/render/404 tests)
   - Integration test (full serve path): TODO


## Remove variables.yaml [DONE]

The `variables.yaml` file support has been removed:
- Removed `collectPageVariables()`, `mergeStringMaps()` from `processor/processor.go`
- Removed `variables` template context variable (use `page.Metadata` instead)
- Removed `variables.yaml` skip from `lib/treewalk.go` indexing exclusion
- Removed `variables.yaml` files from `site-template/`
- Updated documentation (reference, architecture, quickstart)

## add enabled flag

Each page should be able to define an `enabled` property in its YAML front matter. This boolean property defines whether the page is active.

If a page is not active, it must not be served on requests and should return `404`.

When loading a page object from the database, the `enabled` property must be resolved recursively through its parent pages, with override logic applied. If any parent page is already disabled, child pages must also be treated as disabled and must not be served.

The `enabled` property must also be stored as a separate database column. During indexing, read it from front matter (analogous to `title`) and persist it in the `pages` table.

Implement the required code changes and database/schema updates so this new flag is fully supported. The flag must be included when creating/populating the index database.

If a page/front matter does not define `enabled`, it must default to active (`true`).

## create page query builder


## TODO

- de/serialize the page metadata json when loading from the  db
- Remove the Metadata column from the File objects
- remove unused template vars, as it can be fetched from the Page object
- add demo page with all available template objects and functions
- migrate `site-template/site/variables-page/` to show available template vars and functions instead of the removed variables.yaml feature
- migrate templates (`doc/templates/base.html`, `site-template/templates/base.html`) from `variables.xxx` to `page.Metadata.xxx` / `page.Title`
- remove `variables` config key from `pcms-config.yaml` once doc/template migration is complete
- use in-memory index+db for internal doc
- ~~re-index single route when page newer index entry~~ [DONE]
- pcms config in template context, too. It also should contain all non-defined props in the yaml, too