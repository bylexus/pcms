# PCMS Rewrite Planning

This document collects design notes and implementation plans for future rewrites and new features in pcms.

---

## DB Index / File System Sync on Serve Start

### Background

pcms currently rebuilds the entire site on every `serve` start and on every detected file change (via `fsnotify`).
A planned improvement (see TODO.md: *indexing using sqlite*) is to introduce a persistent **SQLite-backed route index** that tracks every known source file together with its route, processed output path, and the timestamp of the last successful indexing run.

Once such an index exists, a full rebuild on every `serve` start is no longer necessary. Instead, the index and the file system should be **synced asynchronously** so that the server can begin serving immediately while the sync runs in the background.

---

### Implementation Plan

#### 1. Data Model – Index Entry

Each entry in the SQLite index should store at least:

| Column | Type | Description |
|---|---|---|
| `id` | INTEGER PK | Auto-increment primary key |
| `route` | TEXT UNIQUE | URL route, e.g. `/blog/hello-world` |
| `source_path` | TEXT | Absolute path to the source file |
| `output_path` | TEXT | Absolute path to the built output file |
| `indexed_at` | DATETIME | Timestamp of the last successful index/build for this entry |
| `file_mtime` | DATETIME | `mtime` of the source file at indexing time |

#### 2. Sync Algorithm

On every `serve` start, after the server socket is bound and the server is ready to accept connections, run the following three-phase sync **as a goroutine** (non-blocking, asynchronous):

```
goroutine syncIndexWithFilesystem(config, db, logger):
    phase1: AddNewEntries(config, db, logger)
    phase2: ReIndexStaleEntries(config, db, logger)
    phase3: RemoveDeletedEntries(config, db, logger)
```

##### Phase 1 – Add New Entries

Walk the entire `SourcePath` directory tree.
For each file:
- Determine the route and the expected output path (same logic as the existing `ProcessSourceFile` pipeline).
- Query the index for an entry with a matching `source_path`.
- If **no entry exists**: process / build the file, then insert a new index record.

##### Phase 2 – Re-index Stale Entries

Walk the entire `SourcePath` directory tree again (can be merged with Phase 1 for efficiency).
For each file that **already has** an index entry:
- Stat the source file to get its current `mtime`.
- Compare `mtime` with `file_mtime` stored in the index.
- If the source file is **newer** than the stored `file_mtime`: rebuild the file and update `indexed_at` and `file_mtime` in the index.

##### Phase 3 – Remove Deleted Entries

Iterate over **all rows** in the index.
For each row:
- Check whether `source_path` still exists on disk (`os.Stat`).
- If the file **no longer exists**: delete the index row (and optionally remove the stale output file from `DestPath`).

#### 3. Integration with `RunServeCmd`

```
func RunServeCmd(config):
    db = openOrCreateIndex(config.IndexDBPath)   // new
    startServer(config, db)                       // existing, slightly refactored
    go syncIndexWithFilesystem(config, db, logger) // new, non-blocking
```

The server **must not block** on the sync. Requests arriving during an ongoing sync are served from the already-built output in `DestPath` (which may be partially stale until the sync completes – this is acceptable).

#### 4. Index DB Location

The SQLite database file should live next to `pcms-config.yaml` (or in a configurable path), e.g.:

```yaml
# pcms-config.yaml
index:
  db: ".pcms-index.db"   # default; relative to config file dir
```

#### 5. Concurrency & Safety

- The sync goroutine and the `fsnotify` file watcher can both trigger builds concurrently.
  Use a **mutex** (or a buffered rebuild channel with a single worker) to serialize individual file builds and index writes.
- The sync goroutine should be **cancellable** via a `context.Context` so it can be gracefully stopped on server shutdown.

#### 6. Logging

- Log the start and end of each sync phase at `INFO` level.
- Log individual file actions (add / re-index / remove) at `DEBUG` level.
- Log any per-file errors at `ERROR` level without aborting the entire sync.

---

### Open Questions / Future Work

- Should a full rebuild still be triggered on first run (when the index is empty / missing)?
- How should template changes (which affect all pages) interact with the incremental sync?
  → Possibly invalidate the entire index and fall back to a full rebuild when a template file changes.
- Should the index also track template dependencies per page for finer-grained invalidation?
