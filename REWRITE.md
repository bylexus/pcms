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

## Implementation step 1: Base DB architecture

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

## Implementation step 2: Index command

The index command creates the page and file index based on the given file system (`fs.FS`). The file system is either a directory on the disk (os.DirFs), or an embedded
file system as defined in the main.go file (go:embed). This fs is the root route ('/').

The index command now traverses this tree and creates page and file entries in the index db. The base logic can already be found in the commands/build.go file, but this was
implemented for the old static site builder structure. This needs to be refactored now.

The goal is to have the full page tree in the sqlite db.

### Implementation plan (for review)

#### 1) Define command contract and source FS handling

- Keep `index` as a first-class command in `main.go` and route to `commands.RunIndexCmd(config)` (pass config, unlike current no-arg variant).
- `RunIndexCmd` should accept a source `fs.FS` abstraction and root context from config:
  - default mode: `os.DirFS(config.SourcePath)`
  - embedded-doc mode: `config.EmbeddedDocFS` (subdir-scoped if needed, same as existing serve-doc semantics)
- Keep route root canonical as `/` regardless of backing FS type.

#### 2) Add DB write API in `lib/db.go` for indexing lifecycle

- Extend `DBH` with explicit index operations (instead of ad-hoc SQL from command layer):
  - `BeginIndexRun()` / `CommitIndexRun()` / `RollbackIndexRun()` transaction helpers
  - `CleanIndex(...)` for cleaning out an existing, previous index
  - `ReplacePage(...)` (upsert by `route`)
  - `ReplaceFile(...)` (upsert by `route`)
  - `SetLastIndexInfo(...)` in `app_settings.settings_json` (timestamp, source hash/path marker, optional counters)
- Keep all writes for one index execution inside one transaction for consistency and speed.
- Preserve schema v1 for this step if possible; only add columns/settings when strictly required. If schema changes are needed, implement it as if it was for version 1: we are in 
  development of this feature right now, so we can make schema changes without being backward-compatible for the moment.

#### 3) Implement reusable tree traversal for indexing

- Introduce a new traversal unit that walks `fs.FS` from root and emits typed index records (`page`, `file`) with canonical routes.
- Route normalization rules (must be deterministic):
  - directories/pages: `/`, `/blog`, `/blog/post`
  - files: `/blog/image.png`
  - use URL-style joining/cleaning (`path` package), not OS-specific separators.
- For each directory:
  - always emit a page record for that route (folder-based routing)
  - detect `index.*` candidate for `pages.index_file` using explicit precedence rules
  - derive `title`: The `title` comes from the YAML Frontmatter in the index file (`title` property). Read the frontmatter yaml for the index files. As a fallback, use the directory name.
  - set `parent_page_route` (`NULL` for root)
- For each non-index file:
  - emit file record with `parent_page_route`, `file_name`, `route`, `mime_type`, `file_size`
  - `metadata_json`:
    - for pages, this is the parsed YAML frontmatter of the index file.
	- for files: keep it as `{}` (empty json) for now
- Apply existing exclude-pattern behavior in traversal (or a dedicated shared helper) so index/build remain compatible during transition.

#### 4) Full reindex semantics (step 2 baseline)

- First version uses full traversal + reconciliation (not incremental diff):
  0. Delete the existing index
  1. traverse FS and collect/stream current routes
  2. insert all current pages/files
- This gives deterministic state and simplifies correctness before introducing partial reindexing.

#### 5) Change `commands/build.go` tree logic into shared walker

The `build` command is no longer needed for now. Remove it, and re-use the walker logic by moving it to the `lib/` package.

- The current recursive logic in `commands/build.go` (`processInputFS`) is tied to OS paths and static-build processing. Adapt as needed.
- Replace it with a walker in the `lib` package
  - for the `index` command (DB records)
  - `build` command: legacy, no longer needed, can be removed.
- Suggested split:
  - `lib/treewalk.go` (FS walk + route derivation + exclude checks)
  - `commands/build.go` can be removed
  - `commands/index.go` uses same walker with DB-write callback
- Result: single source of truth for page/file discovery and parent-child route derivation.

#### 6) Error handling, logging, and observability

- Fail fast on DB write/traversal errors and roll back transaction.
- Emit concise run stats at end: pages indexed, files indexed, deleted stale pages/files, duration.
- Keep command output CLI-friendly and consistent with existing command style.

#### 7) Tests and validation

- Add focused tests for new traversal + index logic using fixture FS trees:
  - root page creation
  - nested routes and parent linkage
  - index-file detection and precedence
  - exclude pattern behavior
  - stale row cleanup after source deletion
- Add DB-level assertions for referential integrity (`files.parent_page_route` must exist).
- Run gate: `go test ./...` and `go build ./...`.

#### 8) Proposed implementation sequence

1. Change `RunIndexCmd` signature and wiring (`main.go` -> pass config + source FS).
2. Add DBH index lifecycle/write methods in `lib/db.go`.
3. Implement tree walker with canonical route derivation.
4. Implement `index` command end-to-end using walker + DB transaction.
5. Remove `commands/build.go`, no longer needed
6. Add tests and run full verification.

### Open questions to resolve before coding

- What is the exact `index.*` precedence when multiple files exist (e.g. `index.md` and `index.html`)?
  - There must only be one index file. The first (alphabetically) is taken. Supported are:
	- index.md (for markdown templates)
	- index.html (for html templates)
- Should excluded files/folders be fully absent from DB (recommended), or recorded with a flag?
	- they should completely be skipped
- For embedded-doc indexing, should the traversed FS root be exactly `doc/build` subtree or binary FS root with path prefix stripping?
  - The root should always be `/` so path prefixing is needed.
