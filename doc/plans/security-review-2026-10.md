# Security Review Findings — 2026-10-24

Source: OWASP-focused audit of the complete pcms codebase (not a diff).
Findings were verified against a local test instance (copy of `site-template`) unless marked "not tested".

Items marked ⚑ depend on the deployment (reverse proxy in front of pcms or not).

Work on the items one by one. Mark each item when done: `[ ]` → `[x]`.

## Status overview

| #   | Severity | Finding                                              | Status |
| --- | -------- | ---------------------------------------------------- | ------ |
| 1   | High     | Unbounded image-resizer cache (disk / CPU)           | [x]    |
| 2   | Medium   | Open redirect via encoded `//host` path              | [ ]    |
| 3   | Medium   | `-listen` flag default overrides config ⚑            | [ ]    |
| 4   | Medium   | No HTTP server timeouts (Slowloris) ⚑                | [ ]    |
| 5   | Medium   | Outdated `golang.org/x/image` (12 reachable vulns)   | [ ]    |
| 6   | Medium   | Disabled pages / files stay public after frontmatter change | [ ] |
| 7   | Low      | No security headers ⚑                                | [ ]    |
| 8   | Low      | Internal error details sent to client                | [ ]    |
| 9   | Low      | Fallback error response has no explicit Content-Type | [ ]   |
| 10  | Low      | Ineffective exclude patterns in `doc/pcms-config.yaml` | [ ]  |
| 11  | Low      | World-writable cache files                           | [ ]    |
| 12  | Low      | Log injection via Basic-Auth username                | [ ]    |
| 13  | Low      | Symlinks in source dir are followed                  | [ ]    |
| 14  | Info     | Template trust boundary (`ssi`, absolute template paths) | [ ] |
| 15  | Low      | `regexp.MustCompile` per file / per pattern          | [ ]    |
| 16  | Low      | All HTTP methods accepted                            | [ ]    |

---

## 1. Unbounded image-resizer cache (disk / CPU exhaustion) — High

- **Location:** `webserver/image_resizer.go:362-415`
- **Problem:** Any visitor can request unlimited parameter combinations
  (width × height × fit × fillColor (16.7M values) × format × jpgQuality × webpQuality).
  Each new combination does a full decode + Lanczos resize and writes a new cache file
  (plus a `.ct` file). Nothing deletes old cache entries.
- **Verification:** 5 requests with different `fillColor` values → 5 new cache entries (10 files).
- **Fix options:**
  - Allow only named size presets from the config (e.g. `/_imageResizer/thumb/<path>`), or
  - Sign resizer URLs with an HMAC (key in config), and/or
  - Limit the total cache size (LRU eviction) and add a rate limit for uncached resizes.
- **Tests:** unknown preset / invalid signature → 4xx; no cache file written for rejected requests.
- **Status:** [x]
- **Done:** (2026-10-24, details in `doc/plans/security-01-image-cache-limits.md`)
  - Configurable max image size (`server.image_resizer.max_width` / `max_height`, default 1920):
    larger params → 400; the output is always fitted into this box.
  - Concurrency limit for uncached resizes (`max_concurrent`, `wait_timeout`) → 503 + `Retry-After`.
  - Background cache cleanup by last use (`server.cache_max_age`, `server.cache_cleanup_interval`);
    `.ct` sidecar files removed.
  - Atomic cache writes; on any cache write error (e.g. disk full) the content is served on the fly.
  - Presets / HMAC-signed URLs / LRU were **not** implemented. Instead: size-limited cache volume
    (deployment) + cleanup + concurrency limit.

## 2. Open redirect — Medium

- **Location:** `webserver/handler.go:77-78`
- **Problem:** The redirect target is the raw `req.URL.Path`. The DB lookup uses the cleaned route.
  `%2F` is decoded to `/`, so the redirect target can start with `//host`.
- **Verification:**
  - `GET /%2Fevil.example%2F..` → `301 Location: //evil.example/../`
  - `GET /%2Fevil.example%2F..%2Fhtml-page` → `301 Location: //evil.example/../html-page/`
- **Fix:** Build the redirect target only from the cleaned route,
  e.g. `path.Join("/", config.Server.Prefix, route) + "/"`, or return 404 when `rawRoutePath != route`.
  Check the prefix handling: `http.StripPrefix` removes the prefix from `req.URL.Path`.
- **Tests:** the two requests above must not produce a `Location` header with a foreign host.
- **Status:** [ ]

## 3. `-listen` flag default overrides `server.listen` from config ⚑ — Medium

- **Location:** `model/config.go:114-116`, `main.go:70`
- **Problem:** `FlagSet.Lookup("listen")` always returns the flag, so the default `:3000`
  always replaces the config value.
- **Verification:** config `listen: "127.0.0.1:3917"` → server bound to `:3000` (all interfaces).
  A server configured for localhost behind a proxy is reachable directly.
- **Fix:** Use the flag value only when the flag was set (detect with `flag.Visit`).
- **Tests:** config value is used when no `-listen` flag is given; flag wins when given.
- **Status:** [ ]

## 4. No HTTP server timeouts (Slowloris) ⚑ — Medium

- **Location:** `commands/serve.go:100-103`
- **Problem:** `http.Server` has no `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`,
  `IdleTimeout`, `MaxHeaderBytes`. Not tested.
- **Fix:** Set all of them, e.g. `ReadHeaderTimeout: 10s`, `ReadTimeout: 30s`,
  `WriteTimeout: 60s`, `IdleTimeout: 120s`, `MaxHeaderBytes: 1 << 16`. Optionally make them configurable.
- **Status:** [ ]

## 5. Outdated `golang.org/x/image` — Medium

- **Location:** `go.mod` (`golang.org/x/image v0.0.0-20211028202545-6944b10bf410`)
- **Problem:** govulncheck reports 12 reachable vulnerabilities. Trace for all:
  `webserver/image_resizer.go:396` `image.Decode` → TIFF / BMP / WebP decoders
  (OOM, panic, CPU DoS; e.g. GO-2026-6222, GO-2026-5062, GO-2023-1572).
  Only images from the site source reach the decoder, so the real risk is lower.
  `disintegration/imaging` is archived.
- **Fix:**
  - `go get golang.org/x/image@latest && go mod tidy`
  - Before `image.Decode`, call `image.DecodeConfig` and reject images above a pixel limit.
  - Optional: update Go 1.27.2 → 1.27.3 (GO-2026-6809, crypto/x509, not called).
- **Tests:** `govulncheck ./...` has no reachable findings; oversized image → 413.
- **Status:** [ ]

## 6. Disabled pages / files stay public after frontmatter change — Medium

- **Location:** `webserver/handler.go:149-189`, `lib/treewalk.go:198-216` (`ReindexSinglePage`)
- **Problem / verification:** set `enabled: false` in `html-page/index.html`:
  - 1st request `/html-page/` → **200** (re-index happens, but `Enabled` is not checked again; page is rendered and cached).
  - 2nd request → 404.
  - `/html-page/sunset.webp` and `/_imageResizer/.../html-page/sunset.webp` → **200** until `index` runs again.
  - Child pages are not updated either.
- **Fix:**
  - Check `page.Enabled` after `reindexPageIfStale`; return 404 when disabled.
  - When the re-index changes the enabled state, apply the same cascade as `disable-page` / `enable-page`
    to child pages and files (`lib/db.go` subtree update).
- **Tests:** handler test for the three requests above after a frontmatter change.
- **Status:** [ ]

## 7. No security headers ⚑ — Low

- **Location:** `webserver/handler.go` (all responses)
- **Problem:** No `X-Content-Type-Options`, `Content-Security-Policy`, `Referrer-Policy`,
  `X-Frame-Options` / `frame-ancestors`, HSTS.
- **Fix:** Header middleware (configurable), or set them in the reverse proxy.
- **Status:** [ ]

## 8. Internal error details sent to client — Low

- **Location:** `webserver/handler.go:304`
- **Problem:** Internal error strings are shown to the client
  (e.g. `image: unknown format`, `indexed page source missing: …`, pongo2 template errors with file names).
- **Fix:** For 5xx, show a generic message; log the details only.
- **Status:** [ ]

## 9. Fallback error response has no explicit Content-Type — Low

- **Location:** `webserver/handler.go:315-316`
- **Problem:** At the moment the `error:` prefix makes content sniffing return `text/plain` (tested, no XSS).
  This is fragile.
- **Fix:** Set `Content-Type: text/plain; charset=utf-8` and `X-Content-Type-Options: nosniff` explicitly.
- **Status:** [ ]

## 10. Ineffective exclude patterns in `doc/pcms-config.yaml` — Low

- **Location:** `doc/pcms-config.yaml`, matched in `lib/treewalk.go:43`
- **Problem:** Patterns `"^\\..*"` and `"^page\\.json$"` are applied to routes that start with `/`, so they never match.
- **Fix:** Use `/`-anchored patterns (e.g. `"/\\..*"`, `"/page\\.json$"`). Check the alexi.ch config too.
  Optional: document in the config reference that patterns match against the `/`-prefixed route.
- **Status:** [ ]

## 11. World-writable cache files — Low

- **Location:** `webserver/handler.go:283,287`
- **Problem:** `os.MkdirAll(dir, 0o777)`, `os.WriteFile(..., 0o666)`.
- **Fix:** Use `0o750` / `0o640`.
- **Status:** [ ]

## 12. Log injection via Basic-Auth username — Low

- **Location:** `webserver/handler.go:353`
- **Problem:** The Basic-Auth username goes into the access log without newline filtering.
  pongo2 autoescape escapes HTML only. Not tested.
- **Fix:** Replace control characters (`\r`, `\n`, …) in all logged request fields.
- **Status:** [ ]

## 13. Symlinks in source dir are followed — Low

- **Location:** `commands/serve.go:63` (`os.DirFS`)
- **Problem:** A symlink inside the source dir can point outside of it.
- **Fix:** Use `os.OpenRoot(config.SourcePath)` and `root.FS()` (Go ≥ 1.24) for hard containment.
- **Status:** [ ]

## 14. Template trust boundary — Info

- **Location:** `processor/md_processor.go:84`, `processor/html_processor.go:46`
- **Problem:** Page sources are compiled as pongo2 templates. `{% ssi "/abs/path" %}` and
  `template: /abs/path` in frontmatter can read any file (pongo2 `LocalFilesystemLoader.Abs` accepts absolute paths).
  Acceptable while the author is the only trusted content source.
  Also: `template.(string)` panics if the frontmatter value is not a string.
- **Fix (optional):** `pongo2.DefaultSet.BanTag("ssi")`; reject absolute / `..` template names; checked type assertion.
- **Status:** [ ]

## 15. `regexp.MustCompile` per file / per pattern — Low

- **Location:** `lib/treewalk.go:271`
- **Problem:** Patterns are compiled again for every file (performance). An invalid pattern causes a panic.
- **Fix:** Compile the patterns once at config load; return an error for invalid patterns.
- **Status:** [ ]

## 16. All HTTP methods accepted — Low

- **Location:** `webserver/handler.go:52`
- **Problem:** POST, PUT, DELETE, … are handled like GET.
- **Fix:** Allow only `GET` and `HEAD`; return `405` with an `Allow` header for all other methods.
- **Status:** [ ]

---

## Verified as OK (no action)

- Path traversal / webroot escape: only DB-indexed routes after `path.Clean`.
  Tested `/../../../etc/passwd`, `%2e%2e%2f…`, double-encoded `..%252f` in resizer → 404.
- Excluded / raw sources: `/restricted/`, `/.keep`, `/html-page/index.html`, `/markdown-page/index.md`, `/pcms-config.yaml` → 404.
- XSS: pongo2 autoescape is on by default; reflected path in 404 page is escaped.
- Code / template injection (SSTI): no request data reaches `pongo2.FromString` / `FromFile`.
- SQL injection: parameterized queries; JSON paths validated by regex; `ORDER BY` allowlist.
- SSRF: resizer serves only indexed, enabled local files.
- CSRF / auth / sessions: no state-changing endpoints, no auth.
- Insecure deserialization: YAML only from config and author files.
