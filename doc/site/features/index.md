---
title: "features"
shortTitle: "Features"
template: page-template.html
metaTags:
  - name: "keywords"
    content: "features,feature list,pcms,cms"
  - name: "description"
    content: "Feature list of pcms"
---
# Features

**Note**: pcms serves my own needs (only?), so I just implement what is necessary for my projects. If something is missing, that means I just had no use for that feature.

## Design Goals

* Provide a simple site generator based on Markdown or raw HTML and a template engine. No magic.
* Generate final content using HTML templates
* Single binary, in-process sqlite db for indexing. No setup needed.
* The filesystem is the URL route structure.
* Programmer friendly, or, code-first: There is no UI! Ever!
* No magic. Magic is opaque, it does things you don't see directly. Magic is for wizards, not for programmers.

## Implemented features

* Web server as site generator: serves dynamically rendered content using:
  * HTML, optionally using templates
  * Markdown within an HTML template
  * all other files are just served 1:1 as content
* Folder structure defines URL routes
* Content is rendered using a Template engine (see pongo2 below)
* Uses [pongo2](https://github.com/flosch/pongo2), a [Django-like](https://docs.djangoproject.com/en/4.0/topics/templates/) template engine written in GO for html/markdown files to create pages based on templates
* SQLite-backed route index: pages and files are indexed into an in-process SQLite DB (no external dependencies, pure Go driver)
* `pcms index` command: walks the source file tree and populates the index DB with page and file entries, including front matter metadata
* DB-first request routing: all requests are resolved against the index — only indexed content is served, everything else returns 404
* Page render cache: rendered pages are cached on disk; cache is invalidated automatically when the source file changes
* Automatic re-indexing: individual pages are re-indexed on serve start if their source file is newer than the index entry
* Per-page `enabled` flag: pages can be disabled via front matter; disabled pages (and their children) return 404
* generates starter skeleton
* self-contained binary: you just need the one single binary to run a pcms site, AND to read the docs

## Once-to-be-implemented features

* API for querying and maintaining the running app
* Async DB/filesystem sync on serve start (background re-index of new, stale, and deleted entries)
* Page content indexing and search mechanism, based on the db-indexed content
