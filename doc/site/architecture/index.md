---
title: "architecture"
shortTitle: "Architecture"
template: "page-template.html"
metaTags:
  - name: "keywords"
    content: "architecture,pcms,cms"
  - name: "description"
    content: "System and Software architecture of pcms"
---
# Architecture

pcms is a mini-webserver written in [GO](https://go.dev/) that delivers web pages from HTML / Markdown templates or static files.
Content is indexed into a **SQLite database** and served on request — no static build step required.

In essence pcms is built upon the following infrastructure:

* a **SQLite index** that tracks every page and file in your `site/` directory
* a **web server** that resolves requests against the index, renders pages on demand, and caches the output
* a **template engine** based on [pongo2](https://github.com/flosch/pongo2), a Django-like template engine,
  with support for YAML frontmatter
* all delivered in the single `pcms` binary!

## Commands

| Command | Description |
|---|---|
| `pcms init` | Initialise a new site skeleton in the current directory |
| `pcms index` | Walk the `site/` directory and build the SQLite route index |
| `pcms serve` | Start the web server and serve pages from the index |
| `pcms serve-doc` | Serve the built-in pcms documentation (embedded) |

On first `pcms serve`, if the index is empty, indexing runs automatically.

## Site structure and routes

A typical `pcms` site may look as follows:

```sh
.
├── pcms-config.yaml          # The main config file for the site
├── pcms.db                   # SQLite route index (auto-created)
├── site/                     # The site dir contains the page content, root route is "/"
│   ├── index.html            # Root page content
│   ├── favicon.png           # Static file, served at /favicon.png
│   ├── html-page/            # Route /html-page
│   │   ├── index.html        # HTML content file
│   │   └── sunset.webp       # Static file, served at /html-page/sunset.webp
│   └── markdown-page/        # Route /markdown-page
│       ├── index.md          # Markdown content file
│       └── sunset.webp
└── templates/                # pongo2 templates
    ├── base.html
    ├── error.html
    └── markdown.html
```

* Each **subdirectory** inside `site/` that contains an `index.html` or `index.md` file is a **page** in the index, addressed by its directory path as the URL route.
* Every other file inside `site/` is a **static file** in the index, addressed by its full path as the URL route.
* Routes are keyed by path only — there is no separate `build/` output folder.

## Indexing

The `pcms index` command (and the automatic initial index on `serve` start) walks the `site/` directory and creates entries in the SQLite database:

* Each directory containing an `index.html` or `index.md` file becomes a **page** entry in the `pages` table.
* YAML frontmatter in the index file is parsed: `title`, `enabled`, and all other keys are stored as `metadata_json`.
* All other files become **file** entries in the `files` table, including auto-detected MIME type.
* Excluded paths (configurable via `exclude_patterns` in `pcms-config.yaml`) are skipped.

The database is a plain SQLite file (`pcms.db` by default, configurable via `database_path` in `pcms-config.yaml`).

## Request handling

The web server resolves every incoming request **against the SQLite index**, not by probing the filesystem directly. Only indexed content is served — unknown routes get a 404.

### Page request pipeline

1. Normalize the URL path to a canonical route.
2. Look up the route in the `pages` table.
3. Check the page's effective `enabled` state (resolved recursively through parent pages).
   If disabled, return 404.
4. Check if the source index file is newer than the DB record — if so, re-index the single page on the fly.
5. Check the page cache (`server.cache_dir`):
   - If a valid cached file exists (cache mtime >= source file mtime), serve it directly.
   - Otherwise, render the page via the appropriate processor and write the result to the cache.
6. Serve the cached HTML file to the client.

### File request pipeline

1. Look up the route in the `files` table.
2. Open the file from the source `fs.FS`.
3. Set `Content-Type` from the DB `mime_type` field.
4. Stream the file to the client via `http.ServeContent`.

## Processors

Processors render a page's index file to HTML. The processor is chosen by the index file's extension.

### HTML processor

Takes `index.html` files as input:

1. YAML frontmatter is extracted. Metadata is available via the `Page.Metadata` template variable.
2. The HTML file is processed as a pongo2 template.
3. The rendered HTML is written to the page cache.

Example:

```html{% verbatim %}
---
title: 'Hello'
template: 'base.html'
---
{% extends "base.html" %}
<h1>{{ Page.Title }}</h1>{% endverbatim %}
```

### Markdown processor

Takes `index.md` files as input:

1. YAML frontmatter is extracted. Metadata is available via the `Page.Metadata` template variable.
2. The Markdown file is processed as a pongo2 template.
3. The Markdown is converted to HTML.
4. If a `template` frontmatter key is set, the converted HTML is embedded into that pongo2 template as `{{ content }}`.
5. The rendered HTML is written to the page cache.

Example:

```markdown
---
template: 'base.html'
title: 'Hello'
---
# {% verbatim %}{{ Page.Title }}{% endverbatim %}

This **Markdown** file uses the 'base.html' template.
```

The `base.html` template receives the converted markdown via the `content` variable:

```html
{% verbatim %}<!-- templates/base.html -->
<!doctype html>
<html lang="en">
    <head>
        <title>{{ Page.Title }}</title>
    </head>
    <body>
      <div id="page_content">
        {{ content | safe }}
      </div>
    </body>
</html>{% endverbatim %}
```

### Raw file handling

Files that are not page index files are served directly from the source `site/` directory without processing.
The stored MIME type from the index is used for the `Content-Type` header.

## Template context variables

The following variables are available in all page templates (HTML and Markdown):

| Variable | Type | Description |
|---|---|---|
| `Page` | `IndexedPage` | The current page record from the index |
| `Page.Route` | `string` | Canonical URL route of the page, e.g. `/blog/hello` |
| `Page.Title` | `string` | Page title (from frontmatter or filename fallback) |
| `Page.IndexFile` | `string` | Index file name, e.g. `index.md` |
| `Page.Enabled` | `bool` | Whether this page is enabled |
| `Page.Metadata` | `map[string]any` | All frontmatter keys not explicitly mapped above |
| `Page.ParentPageRoute` | `*string` | Route of the parent page, or `nil` for root |
| `ChildPages` | `[]IndexedPage` | Direct child pages of the current page |
| `ChildFiles` | `[]IndexedFile` | Files belonging to the current page |
| `Config` | `Config` | The full pcms configuration |
| `Paths` | `PageInfo` | File and web path variants for the current page |
| `Webroot(relPath)` | function | Converts a relative path to an absolute, webroot-based URL |
| `StartsWith(s, prefix)` | function | Returns true if `s` starts with `prefix` |
| `EndsWith(s, suffix)` | function | Returns true if `s` ends with `suffix` |

## Page `enabled` flag

Each page can define an `enabled` property in its YAML frontmatter:

```yaml
---
enabled: false
---
```

* Defaults to `true` if not set.
* If a parent page is disabled, all descendant pages are also treated as disabled, regardless of their own `enabled` value.
* Disabled pages return a 404 response.

## Configuration

Key `pcms-config.yaml` settings relevant to this architecture:

```yaml
source: "site"            # Source directory (relative to config file)
database_path: "pcms.db"  # SQLite index file path (relative to config file)
template_dir: "templates" # pongo2 template directory

server:
  listen: ":8080"
  prefix: ""              # URL prefix (webroot), e.g. "/app"
  cache_dir: ".pcms-cache" # Rendered page cache directory
  watch: true

exclude_patterns:
  - "^\\..*"              # Exclude hidden files/directories
```
