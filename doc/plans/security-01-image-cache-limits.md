# Image resizer cache limits (security finding 1)

**Ticket/Source:** `doc/plans/security-review-2026-10.md`, finding 1 ("Unbounded image-resizer cache (disk / CPU)")
**Branch:** none
**Status:** done

## Goal

The image resizer endpoint (`/_imageResizer/<params>/<image-path>`) lets any visitor request
unlimited parameter combinations. Each new combination does a full decode + Lanczos resize
(CPU) and writes a new cache file (disk). Nothing ever deletes cache files.

This feature limits the damage:

- configurable maximum image size (request parameters **and** output size),
- a concurrency limit for uncached resizes (CPU protection),
- no failure when the cache disk is full: render on the fly instead,
- a background job that deletes old cache files.

In production the cache directory will be a separate, size-limited volume. So the disk can
fill up, but it cannot affect the rest of the system.

## Scope

**In scope:**

- New config keys (see *Decision log*), defaults, validation.
- Configurable max width/height (default 1920x1920) for request params and output.
- Image cache file naming with extension (removes the `.ct` sidecar file).
- Touch image cache files on cache hit (last-use age), serve with source mtime as `Last-Modified`.
- Atomic cache writes and on-the-fly fallback on any write error, for image cache **and** page cache.
- Concurrency limit (semaphore) with wait timeout and 503.
- Background cache cleanup goroutine for the full `cache_dir`.
- Docs: `doc/site/reference/index.md` and `doc/site/backend-services/image-resizer/index.md`.
- Mark finding 1 as done in `doc/plans/security-review-2026-10.md`.
- Unit tests.

**Out of scope:**

- Named size presets or HMAC-signed resizer URLs (fix options listed in the security review; not chosen).
- Rate limiting per client IP.
- Total cache size limit / LRU eviction (the size-limited volume + fallback replaces this).
- Source pixel limit via `image.DecodeConfig` before decode and upgrading `golang.org/x/image` → security finding 5.
- Cache file permissions `0o777`/`0o666` → security finding 11 (do not change the modes in this feature, except that the temp-file + rename must keep the current effective mode `0o666`).
- Disabled-page handling in the resizer → security finding 6.
- Generic error messages for 5xx → security finding 8.
- httptest-based handler tests (no test DB / test FS setup exists yet).

## Rough Architecture / Concept

- **Config** (`model/config.go`): new keys at `server` level for the cache cleanup and a new
  `server.image_resizer` section. Defaults + validation in a separate, testable function,
  called from `NewConfig`. Invalid values → `log.Fatal` (existing pattern in `NewConfig`).
- **Size limit** (`webserver/image_resizer.go`): the const `maxResizeDimension` is replaced by
  config values. Params above the max → 400. `computeTargetDimensions` gets the max box and
  returns target dimensions that fit into it, so there is exactly one resample (no second resize).
- **Cache entry**: one file `<cache_dir>/_imageResizer/<key>.<jpg|png|webp>`. The extension gives
  the Content-Type. On cache hit: `os.Chtimes` to now (last-use age for cleanup), then
  `http.ServeContent` with `Last-Modified` = source file mtime.
- **Writes** (`webserver/handler.go` `writeCacheFile`): temp file in the same dir + `os.Rename`
  (atomic, no partial files). On any write error, image and page handlers log a warning and
  write the rendered bytes from memory with 200.
- **Concurrency**: buffered channel semaphore on `RequestHandler`, used only for uncached image
  renders (decode + resize + encode). Wait for a slot, or `wait_timeout`, or client disconnect.
  Timeout → 503 + `Retry-After: 5`.
- **Cleanup** (new `webserver/cache_cleanup.go`): one goroutine started in `commands/serve.go`
  (`serve` and `serve-doc`). Runs once at startup, then every `cache_cleanup_interval`. Removes
  files with mtime older than `cache_max_age`, then empty sub-directories (never `cache_dir` itself).

**Why:** no new dependencies, no platform-specific code (release builds are cross-platform via
`make build-release`), follows the existing structure (config in `model`, HTTP in `webserver`,
startup in `commands/serve.go`). Small, self-contained changes per step.

## Context for a new agent

- **Sources read:**
  - `doc/plans/security-review-2026-10.md`, finding 1: location `webserver/image_resizer.go:362-415`.
    Any visitor can request unlimited combinations (width × height × fit × fillColor (16.7M) ×
    format × jpgQuality × webpQuality). Each new combination = full decode + Lanczos resize + new
    cache file (+ `.ct` file). Nothing deletes old entries. Verified: 5 requests with different
    `fillColor` → 5 new entries (10 files). The review's suggested fixes (presets / HMAC / LRU +
    rate limit) were **not** chosen; see decision log.
  - Related findings that are **not** part of this feature: 5 (outdated `golang.org/x/image`,
    pixel limit before decode), 6 (disabled pages stay public), 8 (error details), 11 (cache file modes).
- **State of the code before this feature (verified):**
  - `webserver/image_resizer.go:24`: `const maxResizeDimension = 1920`, checked in
    `parseResizeParams` for `width`, `height`, `maxWidth`, `maxHeight` → error → 400.
  - Requests without a resize (`format:webp` only, `maxWidth` no-op, empty params) return the
    full source size.
  - `serveResizedImage` (`image_resizer.go:322`): cache key = `cacheKey(params, fileRoute)`
    (sha256, 32 hex chars); cache path `<CacheDir>/_imageResizer/<key>`; content type in
    `<key>.ct`; validity via `isPageCacheValid(cachePath, sourceModTime)` (cache mtime >= source mtime);
    cache hit served with `http.ServeFile` (→ `Last-Modified` = cache file mtime).
    Cache write error → 500.
  - `writeCacheFile` (`handler.go:281`): `os.MkdirAll(dir, 0o777)` + `os.WriteFile(…, 0o666)`.
    On a full disk, it can leave a partial file with a new mtime that is then served as valid.
  - `servePage` (`handler.go:132`): page cache `<CacheDir>/<route>/index.html`, same
    `writeCacheFile`, write error → 500, served with `http.ServeFile`.
  - No automatic cache cleanup. Only `pcms cache-clear` (`commands/cache_clear.go`, deletes all)
    and `os.RemoveAll(CacheDir)` at `serve-doc` startup (`commands/serve.go:43`).
  - `image.Decode` uses decoders registered by `disintegration/imaging` (jpeg, png, gif, tiff, bmp)
    and `golang.org/x/image/webp`. `encodeImage` maps the source format: `jpeg` → jpg, `webp` → webp,
    everything else → png (when `format` is not set).
- **Key files and directories:**
  - `model/config.go` — `Config` struct, `NewConfig` (yaml load, defaults, `log.Fatal` on error).
  - `webserver/image_resizer.go` — param parsing, `cacheKey`, `computeTargetDimensions`,
    `applyResize`, `encodeImage`, `cachedContentType` (to remove), `serveResizedImage`.
  - `webserver/handler.go` — `RequestHandler`, `NewRequestHandler`, `servePage`,
    `isPageCacheValid`, `writeCacheFile`, `errorHandler`.
  - `webserver/image_resizer_test.go` — existing tests (`TestParseResizeParams`, `TestCacheKey`,
    `TestComputeTargetDimensions`, `makeTestImage`).
  - `commands/serve.go` — `RunServeCmd` for `serve` and `serve-doc`; creates loggers, handler, `http.Server`.
  - `doc/site/reference/index.md` — config reference (yaml block under `## pcms-config.yaml`).
  - `doc/site/backend-services/image-resizer/index.md` — resizer user doc
    (says "All dimension values must be between 1 and 1920"; "Caching" section).
  - `doc/plans/security-review-2026-10.md` — status table + finding 1 status.
- **Existing patterns to follow:**
  - Config defaults: `if config.Server.X == 0 { … }` block in `NewConfig` (`model/config.go:118-130`).
  - Logging inside the handler: `h.ErrorLogger.Info/Error(...)` with nil check (`handler.go:222`, `295`).
  - Table-driven tests: `webserver/image_resizer_test.go`.
- **Commands** (from repo root `/Users/alex/dev/pcms`):
  - Build: `make build` (→ `bin/pcms`), `go build ./...`
  - Tests: `go test ./...`, single: `go test ./webserver -run '^TestComputeTargetDimensions$' -count=1`
  - Run: `./bin/pcms -c <path>/pcms-config.yaml serve`, `./bin/pcms serve-doc`
  - Test site: `./bin/pcms init <scratch-dir>` creates a project from `site-template`
    (contains e.g. `html-page/sunset.webp`).
- **Environment / prerequisites:** Go 1.25 (`go.mod`). No services. `main.go` embeds the full
  `doc/` folder (`//go:embed doc`), so files under `doc/` (also this plan) go into the binary.
- **Domain terms:**
  - *page cache*: rendered HTML in `<cache_dir>/<route>/index.html`.
  - *image cache*: resized images in `<cache_dir>/_imageResizer/`.
  - *max box*: `max_width` × `max_height` from `server.image_resizer`.
- **External references:**
  - `gopkg.in/yaml.v3 v3.0.1` `decode.go:623-653`: a `time.Duration` field is decoded only from a
    string via `time.ParseDuration` (`"24h"`, `"0s"`). A bare yaml integer (e.g. `0`) is rejected
    with an unmarshal error.

## Decision log

- **CPU protection (not in the original request):** add a concurrency limit for uncached renders.
  — *Rationale:* the disk measures alone leave the CPU attack open; on a full cache disk every
  request renders on the fly, which makes it worse. *Rejected:* restricting params (combination
  space stays huge), no protection.
- **Busy slots:** wait with timeout; after timeout 503 + `Retry-After: 5`; also stop waiting on
  client disconnect (`req.Context().Done()`). — *Rationale:* normal traffic peaks (page with many
  thumbnails on first load) still work. *Rejected:* immediate 503 (breaks first page load),
  wait without timeout (goroutine pile-up).
- **Concurrency config:** both configurable: `max_concurrent` (0/unset = `runtime.NumCPU()`) and
  `wait_timeout` (default `"10s"`). Cache hits do not use a slot.
- **Max image size scope:** params above the max → 400 (as before, but with config values);
  **and** the output is always fitted into the max box, also for requests without resize
  (format-only conversion, `maxWidth` no-op, empty params). Requested by the developer.
- **Max size implementation:** cap target dimensions in `computeTargetDimensions` before
  resampling, preserving aspect ratio (e.g. `width:1920` on a tall image → fit into
  `1920 × max_height`). No second resize pass.
- **`height` + `maxWidth`:** mirror of `width` + `maxHeight` (implemented in Step 2). Scale to
  `height` keeping aspect ratio; if the projected width would exceed `maxWidth`, distort to
  `maxWidth` × `height`. — *Rationale:* symmetric to the documented case; before this step the
  combination was silently a noop.
- **Config layout:**
  ```yaml
  server:
    cache_dir: ".pcms-cache"
    cache_max_age: "24h"            # "0s" disables the cleanup
    cache_cleanup_interval: "1h"
    image_resizer:
      max_width: 1920
      max_height: 1920
      max_concurrent: 0             # 0 = number of CPUs
      wait_timeout: "10s"
  ```
  — *Rationale:* cleanup applies to the full `cache_dir`, so it is a server-level cache setting.
  *Rejected:* flat `image_*` keys under `server`; cleanup keys inside `image_resizer`.
- **Duration fields:** `time.Duration`; `cache_max_age` is `*time.Duration` so "not set"
  (nil → 24h) differs from `"0s"` (disabled). Unset/zero `cache_cleanup_interval` → 1h,
  unset/zero `wait_timeout` → 10s, unset/zero `max_width`/`max_height` → 1920.
  Negative values → `log.Fatal` at config load.
- **Cleanup scope:** full `cache_dir` (page cache + image cache). Deleted page cache files are
  rendered again on the next request.
- **Age basis:** last use. Image cache hit → `os.Chtimes(path, now, now)`. Safe for the validity
  check, because only valid files (cache mtime >= source mtime) are touched.
  *Rejected:* creation time (popular images re-rendered daily), OS atime (unreliable, noatime).
- **Last-Modified side effect:** touching changes the file mtime; `http.ServeFile` would send a new
  `Last-Modified` on every hit and break 304 revalidation. Decision: touch **image** cache files
  only; serve them with `http.ServeContent(w, req, name, sourceModTime, file)`. Page cache files
  are **not** touched (they expire `cache_max_age` after render; page count is limited by the index).
  *Rejected:* touch pages too with source mtime (template changes would keep old HTML via 304);
  accept changing `Last-Modified`.
- **Disk full detection:** fallback on **any** cache write error (ENOSPC, EDQUOT, read-only,
  permission). Write = temp file in the same dir (`os.CreateTemp`) + `os.Rename`; on error remove
  the temp file. Log a warning; serve the rendered bytes from memory with 200.
  *Rejected:* only ENOSPC/EDQUOT (EDQUOT needs build tags), pre-check free space via `syscall.Statfs`
  (platform-specific, race between check and write).
- **Page cache fallback:** yes, `servePage` gets the same fallback (same `cache_dir`).
- **`.ct` sidecar:** removed. Image cache file = `<key>.<jpg|png|webp>`; Content-Type from the
  extension. Output format before decode: `params.Format` if set, else `image.DecodeConfig` on the
  source (header only) mapped like `encodeImage` (`jpeg` → jpg, `webp` → webp, else png).
  Old cache files (no extension, `.ct`) become orphans and the cleanup removes them.
  *Rejected:* keep `.ct` and handle file pairs (sync problems).
- **Cleanup behaviour:** runs once at startup, then every interval; in `serve` and `serve-doc`;
  removes files with mtime older than `cache_max_age` (also crashed temp files), then empty
  sub-dirs, never `cache_dir` itself; errors logged, never fatal. *Rejected:* first run only after
  one interval; no disable option; keep empty dirs.
- **Testing:** unit tests + manual checks per step.
- **Docs:** update `doc/site/reference/index.md` (all new keys) and
  `doc/site/backend-services/image-resizer/index.md` (max size from config, output cap, 503,
  cleanup, disk-full fallback).
- **Security review file:** at the end, set finding 1 to `[x]` in
  `doc/plans/security-review-2026-10.md` with a short note on what was done.
- **Plan file path:** `doc/plans/security-01-image-cache-limits.md` (developer choice).
- **`max_body_size` moved and renamed (post-step change, 2026-10-24):** `server.max_body_size` →
  `server.image_resizer.max_source_size` (Go: `ImageResizerConfig.MaxSourceSize int64`, default
  `DefaultImageMaxSourceSize` = 32 MB, negative → config error, set in `applyCacheAndResizerDefaults`).
  — *Rationale:* it limits the source image file size of the resizer (413), not an HTTP body.
  Hard break: the old key is silently ignored (yaml.v3 ignores unknown keys) and the default applies.
  *Rejected:* fail at config load when the old key is set; fallback to the old key with a deprecation
  warning; keep the name `max_body_size`.

## Testing strategy

Go unit tests, run with `go test ./...`; all existing tests must stay green. Per step:

- Step 1: config defaults and validation function (unset → defaults, explicit values kept,
  `"0s"` max age → disabled, negative → error).
- Step 2: `parseResizeParams` with configurable max; `computeTargetDimensions` with max box
  (noop on big source → fit; `width` only on tall image → capped; small source → unchanged).
- Step 3: cache filename extension mapping; touch on hit updates mtime.
- Step 4: `writeCacheFile` atomic write (no temp file left), error on read-only dir.
- Step 5: semaphore acquire: success, timeout, context cancelled.
- Step 6: cleanup with a temp dir: old files removed, new files kept, empty dirs removed,
  root kept, disabled when max age is 0.

Plus manual checks by the developer per step (see each step's acceptance criteria).

## Steps

### Step 1 — Config keys, defaults, validation
- **Status:** done (reviewed 2026-10-24)
- **What:** add `CacheMaxAge *time.Duration` (`cache_max_age`), `CacheCleanupInterval time.Duration`
  (`cache_cleanup_interval`) and `ImageResizer` struct (`max_width`, `max_height`, `max_concurrent`,
  `wait_timeout`) to `Config.Server` in `model/config.go`. Add a testable function for defaults +
  validation (returns error), called from `NewConfig` (`log.Fatal` on error). Unit tests in
  `model/config_test.go`. No behaviour change yet.
- **Acceptance criteria:** `go test ./...` green; `./bin/pcms -c <cfg> serve` starts with no new
  keys; a negative value (e.g. `max_width: -1`) stops the start with a clear error; a bare
  `cache_max_age: 0` stops with a yaml error.
- **Notes:** `model/config.go`: new type `ImageResizerConfig` (fields `MaxWidth`, `MaxHeight`,
  `MaxConcurrent`, `WaitTimeout`), fields `Server.CacheMaxAge *time.Duration`,
  `Server.CacheCleanupInterval time.Duration`, `Server.ImageResizer ImageResizerConfig`; constants
  `DefaultImageMaxWidth`, `DefaultImageMaxHeight`, `DefaultImageWaitTimeout`, `DefaultCacheMaxAge`,
  `DefaultCacheCleanupInterval`; method `(*Config).applyCacheAndResizerDefaults() error`, called in
  `NewConfig` after the other defaults (`log.Fatal("invalid config: …")` on error). After `NewConfig`,
  all values are set (`CacheMaxAge` never nil, `MaxConcurrent` >= 1). Tests in `model/config_test.go`.
  Verified manually: negative `max_width` → fatal error at start.

### Step 2 — Configurable param limit + output cap
- **Status:** done (2026-10-24, not yet manually reviewed)
- **What:** remove `maxResizeDimension`; `parseResizeParams` checks against config max width
  (width, maxWidth) and max height (height, maxHeight); `computeTargetDimensions` fits all targets
  into the max box (aspect ratio kept, single resample). Update tests.
- **Acceptance criteria:** `/_imageResizer/format:webp/<big image>` → output ≤ 1920x1920;
  `/_imageResizer/width:2000/<img>` → 400; with `max_width: 800`: `width:900` → 400 and a
  format-only request → width ≤ 800.
- **Notes:** `webserver/image_resizer.go`:
  - `const maxResizeDimension` removed.
  - New signature `parseResizeParams(paramStr string, maxWidthLimit, maxHeightLimit int)`:
    `width`/`maxWidth` checked against `maxWidthLimit`, `height`/`maxHeight` against
    `maxHeightLimit` (error `"<name>: must be <= <limit>, got <n>"` → existing 400 path).
  - New signature `computeTargetDimensions(src image.Image, p ResizeParams, maxWidthLimit, maxHeightLimit int)`.
    New helper `fitIntoBox(srcW, srcH, boxW, boxH int) (op string, w, h int)`: `"noop"` if the source
    fits, else `"resize-ar"` with `(boxW, 0)` or `(0, boxH)` depending on the limiting side (downscale
    only, aspect ratio kept). Behaviour per case:
    - empty / format-only / any other unmatched combination (default branch) → `fitIntoBox(src, max box)`.
    - `maxWidth` only → `fitIntoBox(src, maxWidth × maxHeightLimit)`; `maxHeight` only →
      `fitIntoBox(src, maxWidthLimit × maxHeight)`. Old results unchanged when the other side fits.
    - `width` only → `resize-ar (width, 0)`; if the exact derived height `srcH*width/srcW` >
      `maxHeightLimit` → `resize-ar (0, maxHeightLimit)` (width derived by `imaging.Resize`, always
      ≤ `width`). `height` only: symmetric with `maxWidthLimit`. Upscaling as before (only when the
      request asks for it).
    - `maxWidth`+`maxHeight` (`fit`), `width`+`maxHeight`, `width`+`height` (cover/contain/distort):
      unchanged, already inside the box because the params are validated.
    - New case `height`+`maxWidth` (mirror of `width`+`maxHeight`, see decision log): projected width
      `round(srcW*height/srcH)` <= `maxWidth` → `resize-ar (0, height)`, else
      `resize-distort (maxWidth, height)`. Inside the box because the params are validated.
    - Comparisons use `int64` products (no float rounding, no overflow).
  - `serveResizedImage` reads `h.ServerConfig.Server.ImageResizer.MaxWidth/MaxHeight` once and passes
    them to both functions. Still exactly one resample (`applyResize` unchanged).
  - Tests (`webserver/image_resizer_test.go`): `TestParseResizeParams` and `TestComputeTargetDimensions`
    got optional per-case `maxW`/`maxH` (0 = 1920). New cases: width at/above default max, all four
    params above a custom max (800×600), height/width checked against the correct limit; noop on big
    landscape/portrait source → fit, format-only on custom box (width- and height-limited), small source
    unchanged, `width` only on tall image → capped, derived height exactly at max → not capped, small
    upscale kept, `height` only on wide image → capped, `maxWidth`/`maxHeight` only with the other side
    over the box → capped, `height`+`maxWidth` with projected width <= / > `maxWidth`. `gofmt -w` also re-aligned some pre-existing, unformatted lines in the test file.
  - In this sandbox `go build ./...` needs `-buildvcs=false` (git cannot read `~/.gitconfig`); not a code issue.

### Step 3 — Cache filename with extension, touch on hit, ServeContent
- **Status:** done (2026-10-24, not yet manually reviewed)
- **What:** cache path `<key>.<ext>`; output format from `params.Format` or `image.DecodeConfig`;
  remove `.ct` write/read (`cachedContentType`); on hit `os.Chtimes` + `http.ServeContent` with
  source mtime. Tests.
- **Acceptance criteria:** cache dir contains `<key>.webp` etc., no `.ct`; `curl -I` twice →
  same `Last-Modified` (= source mtime); `curl -H 'If-Modified-Since: …'` → 304; cache file mtime
  changes on each hit.
- **Notes:** `webserver/image_resizer.go`:
  - New helpers: `resolveOutputFormat(params, srcFormat) outFormat` (single mapping: `params.Format`
    if set, else `jpeg` → jpg, `webp` → webp, everything else → png); `detectOutputFormat(fsys, fsPath,
    params) (outFormat, error)` (returns `params.Format` without opening the source; else own
    `fsys.Open` + `image.DecodeConfig` (header only) + `defer Close`, then `resolveOutputFormat`);
    `formatContentType(format)` (image/jpeg, image/png, image/webp; default png);
    `formatExtension(format)` (jpg/png/webp; default png); `imageCachePath(cacheDir, key, format)` →
    `<cacheDir>/_imageResizer/<key>.<ext>`; `touchCacheFile(path, now)` (`os.Chtimes(path, now, now)`).
  - `encodeImage` new signature `encodeImage(img, params, format outFormat) ([]byte, error)`: it gets the
    format that was resolved before the decode, so the cache extension always matches the encoded
    content. Content-Type comes from `formatContentType`.
  - `cachedContentType` and the `.ct` write removed.
  - `serveResizedImage`: order now stat source (size check, mtime) → `detectOutputFormat` (error → 500,
    `"detect image format: …"`; an undecodable source was 500 before too, from `image.Decode`) →
    cache key/path → `isPageCacheValid`. Cache hit: `touchCacheFile(now)` (failure → `ErrorLogger.Warning`
    with nil check, still served), `os.Open` cache file (error → 500), `Content-Type` set explicitly,
    `http.ServeContent(w, req, <key>.<ext>, sourceModTime, file)` → `Last-Modified` = source mtime,
    `If-Modified-Since` → 304, Range support. Fresh render: decode (fresh reader, second open as before),
    resize, encode, `writeCacheFile` (write error still → 500, unchanged for Step 4), `Content-Type` +
    bytes written directly as before.
  - Page cache (`servePage`) unchanged.
  - Tests: new file `webserver/image_cache_test.go`: `TestResolveOutputFormat`,
    `TestFormatContentTypeAndExtension` (incl. `imageCachePath`), `TestDetectOutputFormat`
    (`fstest.MapFS`: jpeg/png/gif sources, content wins over file extension, param wins, param does not
    open the source, broken/missing source → error), `TestDetectOutputFormatMatchesDecode` (header
    format = full decode format), `TestTouchCacheFile` (temp dir: mtime set to now, content kept; missing
    file → error). `TestCacheKey` unchanged (key itself unchanged).
  - Smoke test (scratch site, `serve -listen 127.0.0.1:<port>`): cache dir contains `<key>.webp` /
    `<key>.jpg`, no `.ct`; hit sends `Last-Modified` = source mtime; `If-Modified-Since` → 304;
    cache file mtime updated on hit.
  - Deviations: none.

### Step 4 — Atomic write + fallback (images + pages)
- **Status:** done (2026-10-24, not yet manually reviewed)
- **What:** `writeCacheFile` with temp file + rename; `serveResizedImage` and `servePage` log a
  warning and serve from memory on write error. Tests.
- **Acceptance criteria:** make `<cache_dir>` read-only (`chmod -R a-w`) → uncached images and pages
  return 200 with correct content, warning in error log, no temp/partial files.
- **Notes:**
  - `webserver/handler.go`: `writeCacheFile` = `os.MkdirAll(dir, 0o777)` (unchanged) → new temp file in
    the same dir → write → close → `os.Rename` to the final path. On any error after the temp file is
    created: close + `os.Remove` of the temp file, error returned. Never a partial file at the final path.
  - New const `cacheTempFilePrefix = ".tmp-"`; temp file name `.tmp-<cache file base name>-<random base36>`
    (e.g. `.tmp-index.html-lnhn38u32tn5`). Crashed temp files: Step 6 cleanup removes them by age.
  - **Deviation from the plan wording (`os.CreateTemp`):** new helper `createCacheTempFile(dir, base)`
    uses `os.OpenFile(name, O_RDWR|O_CREATE|O_EXCL, 0o666)` with a random name (`math/rand/v2`, retry on
    `fs.ErrExist`, max 100 tries) instead of `os.CreateTemp`. Reason: `os.CreateTemp` creates `0o600`,
    and `f.Chmod(0o666)` ignores the umask (→ `0o666` instead of today's `0o644` with umask 022). With
    `OpenFile(…, 0o666)` the umask applies exactly as with the old `os.WriteFile(…, 0o666)`: same
    effective mode, no platform-specific code. Finding 11 stays out of scope.
  - `servePage`: write error → `ErrorLogger.Warning("write page cache …(serving from memory)")` (nil
    check), then `Content-Type: text/html; charset=utf-8`, 200, rendered bytes from memory; no
    `http.ServeFile` in the fallback. Success path unchanged (`http.ServeFile`).
  - `serveResizedImage` (`image_resizer.go`): write error → `ErrorLogger.Warning("write image cache …
    (serving from memory)")` (nil check); the existing code below then writes Content-Type + 200 + bytes.
    No 500 any more for cache write errors.
  - Tests (`webserver/handler_test.go`): `TestWriteCacheFile` (new nested dir, overwrite, content correct,
    only the cache file in the dir, effective mode = mode of a reference file written with
    `os.WriteFile(…, 0o666)`), `TestWriteCacheFileReadOnlyDir` (dir `0o555`: new file and overwrite of an
    existing file → error, no temp/partial file, existing file unchanged; skipped on windows and as root).
    No handler tests (out of scope).
  - Smoke test (scratch site, `serve -listen 127.0.0.1:<port>`): success path → cache files written,
    mode `-rw-r--r--` (as before). After `chmod -R a-w .pcms-cache`: uncached `/` → 200
    `text/html; charset=utf-8`, content identical to the cached render; uncached
    `/_imageResizer/width:120/html-page/sunset.webp` → 200 `image/webp`, valid 120x85 webp; `/markdown-page/`
    (cache sub-dir cannot be created) → 200; one `WARNING` per request in the error log; no temp/partial
    files in the cache dir.

### Step 5 — Concurrency limit + 503
- **Status:** done (2026-10-24, not yet manually reviewed)
- **What:** semaphore (buffered channel, size `max_concurrent`) in `RequestHandler`, created in
  `NewRequestHandler`; acquire around decode/resize/encode of uncached images; wait up to
  `wait_timeout` or client disconnect; timeout → 503 + `Retry-After: 5`. Tests.
- **Acceptance criteria:** with `max_concurrent: 1`, `wait_timeout: "1ms"`, run ~10 parallel
  uncached requests (different `width`) → some 503 with `Retry-After`; cache hits never 503.
- **Notes:**
  - New file `webserver/resize_semaphore.go`: type `resizeSemaphore chan struct{}`,
    `newResizeSemaphore(size)` (size < 1 → 1, defensive), `acquire(ctx, timeout) error` (non-blocking fast
    path first, so a free slot always wins; then `time.NewTimer` + `defer Stop` and select on slot / timer /
    `ctx.Done()`), `release()`. Errors: `errResizeSlotTimeout` on timeout, `ctx.Err()` on cancel.
  - `webserver/handler.go`: new private field `RequestHandler.resizeSem`, created in `NewRequestHandler`
    from `config.Server.ImageResizer.MaxConcurrent`.
  - `webserver/image_resizer.go` `serveResizedImage`: acquire after the cache-hit check fails and before the
    source is opened for decode (stat, `detectOutputFormat`/`DecodeConfig` and the cache hit path use no
    slot). Timeout → `Retry-After: 5` + 503 via `h.errorHandler` (logged as ERROR like every errorHandler
    call). Client disconnect → `ErrorLogger.Debug` (nil check) and return without response.
    Release: guarded `releaseSlot` closure, called explicitly right after a successful encode and via
    `defer` on all early-return paths. Cache write and response writing do not hold the slot. On the error
    paths between acquire and encode (open/decode/encode error) the slot is held until the small error
    response is written (deferred release); kept for simplicity.
  - Tests: new file `webserver/resize_semaphore_test.go`: size defaulting, acquire with free slot (also
    with zero timeout), timeout when all slots are taken, context cancelled (before and while waiting),
    release frees a slot, release wakes a waiting acquire. Passes with `-race`.
  - Smoke test (scratch site, `max_concurrent: 1`, `wait_timeout: "1ms"`, 20 parallel uncached requests with
    different `width`): 1×200, 19×503 with `Retry-After: 5`; then 20 parallel requests of one cached image
    → all 200. Client-disconnect path covered only by unit test.
  - Deviations: none.

### Step 6 — Background cache cleanup
- **Status:** done (2026-10-24, not yet manually reviewed)
- **What:** `webserver/cache_cleanup.go` with the cleanup walk + goroutine loop; started in
  `commands/serve.go` after config/loggers are ready; skipped when `cache_max_age` is 0. Tests.
- **Acceptance criteria:** `cache_max_age: "1m"`, `cache_cleanup_interval: "10s"` → files not
  used for >1 min and empty dirs disappear; recently hit images stay; `"0s"` → nothing removed;
  log lines for removed count / errors.
- **Notes:**
  - New file `webserver/cache_cleanup.go`:
    - `cleanupCacheDir(cacheDir, maxAge, now) cacheCleanupResult` (`FilesRemoved`, `DirsRemoved`, `Errors`):
      one pass. `filepath.WalkDir` over the full `cache_dir`; removes every **regular** file with
      mtime < `now - maxAge` (file exactly at the cutoff is kept); includes `.tmp-*` temp files and old-format
      orphans (extensionless, `.ct`). Then removes empty sub-dirs, deepest first (sorted by path length desc;
      `os.ReadDir` empty check + `os.Remove`, which fails on a non-empty dir, so a file written in between is
      safe). Never removes `cache_dir` itself. Empty or non-existing `cache_dir` → no error, nothing done.
      Per-entry errors are collected (ENOENT ignored, race with concurrent removal) and do not stop the pass;
      unreadable sub-dirs are skipped.
    - Symlinks: symlinks **inside** the cache dir (to files or dirs) are neither followed nor removed
      (only regular files are removed; `WalkDir` does not follow symlinked dirs). If `cache_dir` **itself**
      is a symlink (e.g. to a separate volume), it is resolved with `filepath.EvalSymlinks` first, because
      `WalkDir` does not descend into a symlinked root. Small addition to the plan wording; nothing outside
      the resolved cache dir is touched.
    - `StartCacheCleanup(ctx, cacheDir, maxAge, interval, logger)`: `maxAge <= 0` → one INFO line
      "Cache cleanup disabled (cache_max_age is 0)", no goroutine. Empty `cacheDir` → INFO line, no goroutine.
      `interval <= 0` → 1h (defensive; config always sets it). Else one INFO line with dir/max age/interval,
      then one goroutine: pass at startup, then every `interval` (`time.NewTicker`, stopped on `ctx.Done()`).
    - Logging per pass (`runCacheCleanupPass`, logger nil-checked): each error → WARNING; INFO
      "removed N files and M empty directories" only when something was removed, else DEBUG. Never fatal.
  - `commands/serve.go` `RunServeCmd`: after the logger `defer Close()` lines (so after the serve-doc
    `os.RemoveAll(CacheDir)` and the index run), `context.WithCancel(context.Background())` + deferred cancel,
    `webserver.StartCacheCleanup(ctx, config.Server.CacheDir, *config.Server.CacheMaxAge,
    config.Server.CacheCleanupInterval, errorLogger)`. Used by `serve` and `serve-doc`.
  - Race (accepted, see task): an image cache hit touches then serves the file; a removal between the
    cleanup's stat and remove only causes a re-render on the next request.
  - Tests: new file `webserver/cache_cleanup_test.go`: `TestCleanupCacheDir` (old page/image/`.tmp-*`/
    extensionless/`.ct` files removed, fresh files incl. fresh `.tmp-*` kept, nested and already-empty dirs
    removed, non-empty dirs kept, counts), `TestCleanupCacheDirFileExactlyAtCutoffKept`,
    `TestCleanupCacheDirRootKept` (root kept when empty, second pass no-op), `TestCleanupCacheDirNotExisting`
    (also empty path), `TestCleanupCacheDirSymlinks` (symlinks inside kept, target outside untouched,
    symlinked root cleaned; skipped on windows), `TestStartCacheCleanupDisabled`,
    `TestStartCacheCleanupRunsAtStartup` (interval 1h, polls up to 5s for the startup pass). Pass with `-race`.
  - Smoke test (scratch site, `cache_max_age: "1m"`, `cache_cleanup_interval: "10s"`,
    `serve -listen 127.0.0.1:<port>`; note: `server.listen` in the yaml is overridden by the default of the
    `-listen` flag): backdated entry present before start → removed by the startup pass (+ its empty dir).
    After requests: backdated `markdown-page/index.html`, one backdated `<key>.webp`, backdated `orphan.ct`
    and `.tmp-x-abc` → removed in the next pass (4 files, 1 empty dir); fresh `index.html` and fresh
    `<key>.webp` kept; next pass logs DEBUG "nothing to remove". With `"0s"`: INFO "Cache cleanup disabled",
    backdated file kept.
  - Deviations: none (symlinked-root resolution is an addition, see above).

### Step 7 — Docs + security review status
- **Status:** done (2026-10-24, not yet manually reviewed)
- **What:** document all new keys in the yaml block of `doc/site/reference/index.md`; update
  `doc/site/backend-services/image-resizer/index.md` (max size from config, output cap, 503,
  caching/cleanup/fallback, no `.ct`; add a `height` + `maxWidth` row to the table
  "Mixed dimension combinations"); set finding 1 to `[x]` in
  `doc/plans/security-review-2026-10.md` (table + finding status) with a short note.
- **Acceptance criteria:** `make build && ./bin/pcms serve-doc` → reference and image-resizer
  pages show the new content.
- **Notes:**
  - Docs only, no code changes.
  - `doc/site/reference/index.md`: directory tree line and bullet for `.pcms-cache/` (pages + resized
    images, automatic cleanup, on-the-fly fallback + warning when the cache cannot be written);
    yaml block: `cache_dir` comment extended, new `cache_max_age`, `cache_cleanup_interval` and
    `image_resizer` (`max_width`, `max_height`, `max_concurrent`, `wait_timeout`) with meaning, format,
    default. `cache_max_age` comment says the age counts from the last use for images and from the
    render time for pages (matches the decision log: page cache files are not touched), and that a bare
    number is rejected.
  - `doc/site/backend-services/image-resizer/index.md`: "1 and 1920" sentence replaced by a new
    `### Size limits` section (config keys, 400 above the limit, output cap; link
    `{{ Webroot("/reference/") }}#pcms-configyaml`, same style as other cross-section doc links; it renders
    as `/reference#pcms-configyaml` → 301 to `/reference/`, like the existing links). "Mixed dimension
    combinations" table: new rows "no size parameter" and `height` + `maxWidth`; `width` only / `height`
    only / `maxWidth` only / `maxHeight` only rows now mention the max-box cap (checked against
    `computeTargetDimensions`). New `### Concurrency limit` section (503 + `Retry-After: 5`). "Caching"
    rewritten (`<key>.<jpg|png|webp>`, validity by source mtime, `Last-Modified` = source mtime,
    cleanup by last use, fallback). The `<params>` usage line now notes the output cap.
  - `doc/plans/security-review-2026-10.md`: finding 1 `[x]` in the table and the `**Status:**` line, plus a
    `**Done:**` note with a reference to this plan. Other findings unchanged.
  - Verified: `go test -count=1 ./...` green; `go build -buildvcs=false` OK; `serve-doc` on a free port:
    `/reference/` and `/backend-services/image-resizer/` return 200 and show the new content; server stopped.
  - Deviations: none.

## Open questions

- None.

## Current state / Next action

Plan approved 2026-10-24. All steps done. Step 1 reviewed; steps 2–7 done (2026-10-24, not yet manually
reviewed; `go vet ./...` and `go test ./...` green). Step 4 has one deviation from the plan wording
(`os.OpenFile` with `0o666` instead of `os.CreateTemp`, to keep the effective file mode; see Step 4 notes).
The developer asked to implement steps 2–7 in sequence without a review stop between steps.
After step 7: `server.max_body_size` was moved to `server.image_resizer.max_source_size` (see decision
log). Changed: `model/config.go`, `model/config_test.go`, `webserver/image_resizer.go` (413 check),
`site-template/pcms-config.yaml`, `doc/site/reference/index.md`, `doc/site/backend-services/image-resizer/index.md`.

Next action: developer manual review of steps 2–7 (acceptance criteria per step), then commit.

Notes for the reviewer:
- The pre-existing, uncommitted `Makefile` modification is an unrelated developer change and is not part of
  this feature.
- `server.listen` from the yaml is always overridden by the default of the `-listen` flag (use
  `-listen` for manual tests). This is security finding 3, not part of this feature.
