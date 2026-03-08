# AGENTS.md

Guidance for autonomous coding agents working in `pcms`.

## Scope and intent

- This repository is a Go project (`module alexi.ch/pcms`).
- Main binary entrypoint: `main.go`.
- Core behavior: static site build + serve, with template/markdown/scss processing.
- Use this file as the default operating guide for edits, tests, and reviews.

## Repository facts

- Language/toolchain: Go 1.26 (`go.mod`).
- Build orchestrator: `Makefile`.
- Existing tests are in `processor/*_test.go`.
- No dedicated lint config file was found (`.golangci*` absent).

## Cursor/Copilot rule files

I checked for repository-specific IDE agent rules:

- `.cursor/rules/**`: not present
- `.cursorrules`: not present
- `.github/copilot-instructions.md`: not present

If these files are added later, update this document and treat them as higher-priority behavioral constraints.

## Quick command reference

Run commands from repo root: `/Users/alex/dev/pcms`.

### Build

- `make build` - builds binary to `bin/pcms`.
- `go build ./...` - verifies all packages compile.
- `make build-doc` - builds embedded doc site using doc config.
- `make build-release` - cross-platform release artifacts (destructive to `./releases` and `site-template/build`).

### Run

- `./bin/pcms -h` - show CLI usage and subcommands.
- `./bin/pcms -c pcms-config.yaml build` - build site from config.
- `./bin/pcms -c pcms-config.yaml serve` - build + serve site.
- `./bin/pcms serve-doc` - serve embedded documentation.

### Test

- `go test ./...` - run all tests.
- `go test ./processor` - run processor package tests.
- `go test -v ./...` - verbose output.

### Run a single test (important)

- By exact test name:
  - `go test ./processor -run '^TestMdProcessorTemplate$'`
  - `go test ./processor -run '^TestHtmlProcessorPrepareFilePathsOnWebroot$'`
- By name pattern:
  - `go test ./processor -run 'PrepareFilePaths'`
- Disable test cache while iterating:
  - `go test ./processor -run '^TestMdProcessorTemplate$' -count=1`

### Lint/format checks

There is no dedicated linter task in `Makefile`. Use this baseline:

- `gofmt -w .` - apply canonical formatting.
- `gofmt -l .` - list unformatted files (CI-friendly gate).
- `go vet ./...` - static checks from Go toolchain.

Recommended local quality gate before committing:

1. `gofmt -w .`
2. `go vet ./...`
3. `go test ./...`
4. `go build ./...`

## Architecture map (high-level)

- `main.go`
  - Parses CLI flags/subcommands.
  - Loads config from YAML (`model.NewConfig`).
  - Dispatches to `commands` package.
- `commands/`
  - `build.go`: recursive source traversal and processing.
  - `serve.go`: optional build, file watcher, HTTP server startup.
  - `init.go`: scaffold project from embedded `site-template`.
- `processor/`
  - `processor.go`: processor interface, exclusion logic, template context helpers.
  - `html/md/scss/raw` processors transform or copy files.
- `webserver/handler.go`
  - Static file serving with index fallback and access logging middleware.
- `model/config.go`
  - YAML config model, defaults, path normalization.

## Code style conventions

Follow existing code patterns + idiomatic Go.

### Formatting

- Always run `gofmt` on changed Go files.
- Let `gofmt` decide tabs/spaces and import ordering.
- Keep lines readable; prefer early returns over deep nesting.

### Imports

- Keep imports grouped by `gofmt` conventions.
- Typical grouping in this repo:
  1) standard library
  2) internal module imports (`alexi.ch/pcms/...`)
  3) third-party dependencies
- Remove unused imports immediately.

### Types and interfaces

- Prefer concrete types unless an interface is required by behavior boundaries.
- Reuse the existing `Processor` interface pattern for new file processors.
- Keep struct fields exported only when cross-package access is needed.
- Preserve current config modeling patterns (`Config`, nested structs, yaml tags).

### Naming

- Exported identifiers: `PascalCase`.
- Unexported identifiers: `camelCase`.
- Constants use existing style in file context:
  - enum-like constants often `UPPER_SNAKE_CASE` (e.g. log levels, serve modes).
- Test functions: `TestXxx...` and table-driven style when practical.
- File names in this repo commonly use `snake_case.go`; follow surrounding files.

### Paths, URLs, and filesystem

- Use `filepath` for local filesystem paths.
- Use `path` for URL/web paths.
- Preserve current relative-path handling behavior in processors.
- For FS abstractions, align with existing use of `fs.FS`/`os.DirFS`.

### Error handling

- Return errors with context from lower layers; avoid swallowing errors silently.
- Prefer `fmt.Errorf("...: %w", err)` when adding context.
- Command-level functions should return errors to caller (`main`) when feasible.
- Avoid introducing new `panic` paths in normal flow.
  - Existing panics are legacy/bootstrap behavior; do not expand this pattern unless required.

### Logging and output

- Use `logging.Logger` for runtime/server logs where applicable.
- Use `fmt.Fprintf(os.Stderr, ...)` for CLI-facing error output when outside logger-managed paths.
- Keep log messages concise and actionable (what failed, where, why).

### Testing guidance

- Prefer deterministic tests using fixtures under `processor/testdata`.
- Use temporary directories for generated files (`os.TempDir()` patterns already used).
- When comparing large output, normalize dynamic values (as current tests do with placeholders).
- Add package-level tests close to modified functionality.

### Change safety

- Do not change public behavior (CLI flags, config keys, template context keys) unless requested.
- Preserve compatibility for:
  - YAML config keys (`server`, `source`, `dest`, `template_dir`, etc.)
  - template context keys (`variables`, `paths`, `webroot`, helper funcs)
- For refactors, keep side effects equivalent and update tests accordingly.

## Agent workflow recommendations

- Before editing: inspect nearby files for local conventions.
- After editing: run at least targeted tests for touched package.
- Before finalizing substantial changes: run full `go test ./...` and `go build ./...`.
- If adding new commands/tooling, document them here.

## Known gaps worth noting

- No formal CI lint profile is present in-repo.
- No repo-local Cursor/Copilot instruction files currently exist.
- Some legacy comments/typos exist; prefer minimal, behavior-safe cleanups unless asked.
