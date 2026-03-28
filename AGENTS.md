# AGENTS.md

Guidance for autonomous coding agents working in `pcms`.

## Scope and intent

- This repository is a Go project (`module alexi.ch/pcms`).
- Main binary entrypoint: `main.go`.
- Core behavior: web server that serves markdown and html files from templates, using an sqlite db for indexing the pages / routes

## Quick command reference

Run commands from repo root: `/Users/alex/dev/pcms`.

### Build

- `make build` - builds binary to `bin/pcms`.
- `go build ./...` - verifies all packages compile.
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
- `webserver/*.go`
  - Web server and routing logic
- `model/*.go`
  - used model structs for config and page / file objects
