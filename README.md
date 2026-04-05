# pcms - The Programmer's Content Management System

This project, historically called `pcms`, is a smalle web server written in [GO](https://go.dev/), serving Markdown and HTML pages from templates, indexed via in-process sqlite db.

I don't need a fancy, UI-driven CMS. I don't WANT a  UI, and I don't want a CMS that is in my way of doing things.
A CMS is too restrictive. I don't fear writing HTML and program code. I am a developer, at last, so I feel more
comfortable writing code in an editor than clicking in a UI.

This is the idea behind **pcms**, the Programmer's CMS: A clutter-free, code-centric, simple server to deliver web sites with a template system. For people that love to code, and just want things done.

---
## ⚠⚠⚠ V.0.9: Refactor / Rewrite! ⚠⚠⚠

_"Wait, what? Another restart?"_

Yes. I wanted to give pcms another direction - instead of a static site builder, which in reality I didn't need, I wanted to implement it as an application server that indexes its 
page tree, not in memory, as today, but in a small in-process DB (sqlite).

This enables some features I wanted to see in pcms:

* The system should know the whole page document tree. Templates should be able to access its childs / anchestors / other pages / content.
* The template system can support helper functions to query/filter/search the document tree
* The db approach allows for searching / indexing.

Have a look at the [Vision](#for-reference-refactor-vision) section below for more information.

Alex, 28.03.2026

---

## Why "CMS"?

The name `pcms`, a "Programmer's Content Management System" is rooted in the early days of the project: At the beginning, the idea was to build a content management system to deliver
content from a DB / dynamically instead of building static websites. Thus the name.

The project was re-written multiple times in multiple languages, and I only realized over time that I ALSO don't need a CMS at all, but really just a web server to serve my file content.

But after I started the project, it was too late, the name has already burned in :-)

## Features

* **Dynamic web server** — serves pages on request directly from source; no static build step needed.
* **SQLite page index** — the full page and file tree is indexed in a local SQLite database (`pcms.db`).
* **Folder-based routing** — the site's file structure is also the URL structure.
* **HTML and Markdown pages** — `.html` files are processed as [pongo2](https://github.com/flosch/pongo2) (django-like) templates. `.md` files are rendered to HTML via a pongo2 base template defined in the YAML front matter.
* **YAML front matter** — page metadata (`title`, `template`, `enabled`, and arbitrary fields) is read from the front matter of `.html` and `.md` files and stored in the index.
* **Page cache** — rendered pages are stored in a file cache once the template is rendered to HTML.
* **Page query builder** — the `PageQuery()` function available in templates lets you query, filter, sort, and paginate indexed pages directly from within a template (e.g. list child pages, filter by metadata tag, order by date).
* **Enabled/disabled pages** — pages can be hidden via `enabled: false` in front matter. The flag is resolved recursively: disabling a parent page also hides all its children.
* **Configurable exclusion patterns** — regex patterns in `pcms-config.yaml` exclude files and folders from indexing and serving.
* **Access and error logging** — separate, configurable access log and system log, each with a configurable output target and level.
* **Single binary** — the complete tool, built-in documentation, and a starter project skeleton are all embedded in the single `pcms` binary. No external dependencies required.

## Project Status

A first viable product is already available - All features to drive a full, real website are implemented. The first production site is already using
pcms: <https://alexi.ch/> is driven by the actual pcms version.

Still, this is a work in progress, I change the tool as I need it, and changes have to be expected.

## Quick Start

A getting started guide can be found in the documentation - see `doc/site/quickstart/index.md`, or check it out online: <https://pcms.alexi.ch/quickstart>

For the even more impatient:

```sh
# build pcms (golang / go tools needed)
$ make build

# create a new site:
$ bin/pcms init path/to/site/

# serve it at localhost:3000
$ cd path/to/site/
$ bin/pcms serve

# read the doc at localhost:8888
$ bin/pcms serve-doc -listen :8888
```

## For reference: Refactor Vision

### Overview / Ideas


- The whole site / document tree is indexed in an in-memory-DB (sqlite, duckdb) once, and/or before the web server starts
- All "objects" in the site folder is indexed and created as an entry in the DB - pages (see below) as well as additional / arbitary files
- Each folder is a route, so the system uses a folder-based routing system
- Each folder is a "Page". A page is a website direcly addressable via URL. A Page consists of:
  - its Page struct, knowing:
  - its child pages
  - arbitary files contained in the page's folder
  - an index file, e.g. index.html, index.md, index.you-name-it. The index file contains the content and meta info to display the page
  - the index file contains arbitary meta data that can be used by the page templates
  - The index file itself is not part of the page object / db entry - it defines a page object
- Each other file in a folder is also indexed in the db/tree (as Page.Files), and are served 1:1 when requested by URL
- index files are still processed by the/a template engine. It now gets more possibilities:
  - it has access to the actual Page struct and the whole page structure ("Root page").
  - each page has query/filter functions to search for sub-content (e.g. 'give me all children, ordered by the metadata.date field')

### Architecture

#### The Content Tree

The whole site folder tree gets indexed, and creates entries in the index DB. It "emits" different type of Content objects:

**The Page object:**

The main object we're talking about is the `Page` struct: Each folder, starting with the root of the site, is converted to a `Page` struct that resembles a PCMS-managed page.

A `Page` object looks like:

```golang
type Page struct {
	ParentPage *Page
	Title string
	IndexFile string
	Route string
	Metadata any
	ChildPages []*Page
	Files []File
}
```

**The File object:**

Everything not considered as part of a "Page" object is considered as a raw `File` object, and is also entered in the index DB, e.g. like this:

```golang
type File struct {
	ParentPage *Page
	FileName string
	MimeType string
	FileSize int
	Route string
	Metadata any
}
```


#### Delivering the site

Delivering the page is done by the built-in web server. I don't have a use case for a static build anymore, so I skip that part (for now).
In fact, as I create a page cache of already rendered pages, it would be easy to just allow static builds again in the future.

The steps to deliver the page conists of:

1. Building the Page Tree by walking the project's file tree and creating the index DB - IF necessary (e.g. not yet done)
2. Routing the request to an object in the Page Tree by querying the index db
3. generate the final HTML from the template, and cache it, or delivering it directly from the cache

#### Generating content

Each page will generate HTML content based on its type (the index file define its type)

- index.html (HTML template): the index.html is processed by the template engine and output as final HTML stream
- index.md (Markdown template): The Yaml Frontmatter defines an HTML template (`template` parameter) used to render the html page as a stream.
- empty/no index file: Just a sub-folder: it is not rendered, e.g. it will output a 404 if it is directly addressed.

The output stream can now be picked up by the web server to directly deliver it to the client, and/or the caching mechanism to store it in its file cache.

All other files are not processed, but just streamed as-is (raw content).

If the content is already present in the cache, it checks if the un-cached version is newer. If not, it uses the cached version.

#### Web Server: Routing requests

Based on the requested URL route, the apropriate `Page` object is searched in the pre-generated content tree. If found, the page is being processed (see [Generating content](#generating-content)) and output. If not, the system checks if it is an (allowed) file from within a page. 

So an URL route either matches with a `Page` object, a raw file, or it ends in a 404.

---

## Changelog

### v0.11.1

- [bugfix] Fixed `make build-release` cross-platform builds failing on `github.com/chai2010/webp` when compiled without `cgo`
- [change] WebP encoding now uses build-tagged implementations: `cgo` builds keep full WebP encoding, `!cgo` builds compile cleanly and return a clear runtime error if WebP output is requested

### v0.11.0

- [feature] Image Resizer backend service for on-the-fly image resizing
- [bugfix] Fixed error template rendering broken since v0.9.0

### v0.10.0

- `PageQuery()` builder available in templates to query, filter, sort, and paginate indexed pages
- `enabled`/`disabled` page flag resolved recursively at index time (disabling a parent hides all children)
- `disable-page` and `enable-page` CLI commands to toggle pages without editing front matter
- `cache-clear` CLI command to wipe the rendered page cache
- Page routes redirect to trailing slash (`/page/`) for consistent URL handling
- Docker image build cleaned up with multistage builder

### v0.9.0

- Full rewrite: page tree indexed in an in-process SQLite database
- Folder-based routing — the file structure is the URL structure
- HTML pages processed as pongo2 (Django-like) templates
- Markdown pages rendered via a pongo2 base template defined in YAML front matter
- YAML front matter for page metadata (`title`, `template`, `enabled`, arbitrary fields)
- File cache for rendered pages, invalidated when source is newer
- Automatic re-indexing when files change on disk
- Configurable exclusion patterns (regex) in `pcms-config.yaml`
- Separate, configurable access log and system log
- `init` command to scaffold a starter project from the embedded skeleton
- `serve` and `serve-doc` commands
- Embedded documentation served from the single binary
- Docker support
