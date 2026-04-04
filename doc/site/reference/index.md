---
title: "reference"
shortTitle: "Reference"
template: "page-template.html"
metaTags: 
  - name: "keywords"
    content: "reference,pcms,cms"
  - name: "description"
    content: "pcms reference"
---
# Reference documentation

- [Generating a site](#generating-a-site)
- [Directory Structure of a pcms project](#directory-structure-of-a-pcms-project)
- [pcms-config.yaml](#pcms-configyaml)
- [The `site` folder](#the-site-folder)
  - [using HTML files with templates](#using-html-files-with-templates)
  - [using Markdown files with templates](#using-markdown-files-with-templates)
  - [available template variables](#available-template-variables)
  - [YAML front matter variables](#yaml-front-matter-variables)
- [PageQuery — querying pages from templates](#pagequery--querying-pages-from-templates)
- [pcms cli reference](#pcms-cli-reference)
  - [init](#init)
  - [index](#index)
  - [serve](#serve)
  - [serve-doc](#serve-doc)
  - [cache-clear](#cache-clear)
  - [enable-page](#enable-page)
  - [disable-page](#disable-page)


## Generating a site

pcms can generate a starter skeleton for you. It has a built-in skeleton which you can generate using the following commands:

```bash
$ pcms init /path/to/pcms-project
```

You have now a working site which you can start immediately:

```bash
$ cd /path/to/pcms-project
$ pcms serve
```

## Directory Structure of a pcms project

```sh
root folder
├── pcms-config.yaml          # The config file for the site
├── pcms.db                   # SQLite page index (auto-created)
├── .pcms-cache/              # Rendered page cache (auto-created)
├── site/                     # The site dir contains the page content, and listens to the "/" route
│   ├── index.html            # additional page content / templates
│   └── ... more files/folder # add your source files/folders as needed
└── templates                 # pongo2 templates for your html / markdown content
    ├── base.html             # a pongo2 template, e.g. a base html template
    └── ... more files/folder # add as many templates as you need, and reference them from your site files
```

- `pcms-config.yaml` is the configuration file for your site. It contains all the settings and global variables.
- `pcms.db` is the SQLite index database. It is created and populated by `pcms index` (or automatically on first `pcms serve`). The path is configurable via `database_path` in `pcms-config.yaml`.
- `.pcms-cache/` holds rendered HTML output cached by the serve process. The path is configurable via `server.cache_dir` in `pcms-config.yaml`.
- `site/` is the folder where all your page content goes. If you reference pongo2 templates within your files, they are searched from the `templates/` folder.
- `templates/` contains your pongo2 templates (if you need any).

## pcms-config.yaml

This is the reference of the `pcms-config.yaml` config file.

```yaml
# pcms-config.yaml: This is the pcms configuration file. It configures the whole system.
server:
  # listen address. This is an ip-address:port number pair, or a partial address: "localhost:3000", ":3000", "127.0.0.1", "0.0.0.0:3000"
  listen: ":3000"
  # webroot prefix: the content is served under this webroot prefix (e.g. "/site"). Defaults to "". The webroot can be accessed by the `Paths.Webroot` variable or the `Webroot()` function in templates.
  prefix: ""
  # cache dir for rendered pages in serve mode. Relative to the config file dir, or absolute.
  # Defaults to ".pcms-cache".
  cache_dir: ".pcms-cache"
  # Logging configuration: there are 2 different logs written:
  logging:
    # The access log: Logs all web access, like a webserver would.
    # Define the file (or STDOUT/STDERR), and the format (TBD).
    access:
      file: STDOUT
      format: ""
    # The error, or system log. Define the file (or STDOUT/STDERR), and the max log level:
    error:
      file: STDERR
      level: DEBUG
# Path to the SQLite database file. Relative to the config file dir, or absolute.
# Defaults to "pcms.db" in the project dir.
# Useful for Docker setups where the database should live on a separate volume.
# database_path: "pcms.db"
# The source folder of the site, containing the page content.
# Relative to the config file dir:
source: "site"
# Global variables available in pongo2 templates as "Config.Variables.xxx":
variables:
  siteTitle: My Site Title
  siteMetaTags:
    - name: foo
      content: bar
    - name: moo
      content: baz
# Where to look for pongo2 templates when inheriting / defining a template file.
# Relative to the config file dir:
template_dir: templates
# Regular expressions to exclude files/folders from indexing and serving:
exclude_patterns:
  # Ignore .* files:
  - "/\\..*"
  # Ignore all files in the /restricted folder:
  - "^/restricted/?.*"
```

## The `site` folder

All your content resides under the `site` folder. Each folder (including the main folder `site`) is recognized as a `page`
as soon as it contains a `page.json` file. The folder structure corresponds directly to the page's web route:
`site/` is the webr root route `/` (or whatever your webroot is set to), `site/about/me` corresponds to the `/about/me` route.

### using HTML files with templates

`.html` files in the `site/` folder are processed as [pongo2](https://github.com/flosch/pongo2) templates. You can use the full power of this
django-like template engine. For example, you can access the pre-defined and own variables in your HTML:

```html
{% verbatim %}<div>Path of this file: {{ Paths.RelWebPath }}</div>{% endverbatim %}
```

You can also inherit from a base template: Templates are searched within the `templates/` folder. As an example, define a base template,
then inherit from this base template in your site folder:

```html
{% verbatim %}<!-- Base template in templates/base.html: -->
<!doctype html>
<html lang="en">
    <head>
        <title>{%if Page.Title %}{{Page.Title}}{% endif%}</title>
        {% for meta in Page.Metadata.metaTags %}
        <meta name="{{meta.name}}" content="{{meta.content}}" />
        {% endfor %}
    </head>
    <body>
        <main id="content">
            {% block content %}{% endblock %}
        </main>
    </body>
</html> {% endverbatim %}
```

```html
{% verbatim %}<!-- partial site in /site/index.html: -->
{% extends "base.html" %}
{% block content %}
<h1>Welcome!</h1>

<p>This is your site content.</p>
{% endblock %}{% endverbatim %}
```

### using Markdown files with templates

`*.md` files are processed as a pongo2 template, rendered to HTML and optionally embedded in a HTML template. The processed Markdown content is available in the `content` variable in your templates.
The template to be used can be defined in a YAML front matter.

An example Markdown file which is rendered to a HTML template:

```text
---
# site/index.md, with a YAML front matter, defining the template:
template: base-markdown.html
title: "Welcome"
---
# Welcome!

This **Markdown** partial file has the relative path: {% verbatim %}{{Paths.RelWebPath}}{% endverbatim %}.
```

```html
{% verbatim %}<!-- Base template in templates/base-markdown.html: -->
<!doctype html>
<html lang="en">
    <head>
        <title>{%if Page.Title %}{{Page.Title}}{% endif%}</title>
    </head>
    <body>
        <main id="content">
          {{ content | safe }}
        </main>
    </body>
</html> {% endverbatim %}
```

### available template variables

pcms defines the following variables which you can use in your templates:

* `Page`: The page object from the index database. Contains the page's metadata (from YAML front matter) as `Page.Metadata`.<br>
  Example usage in a template:<br>
  {% verbatim %}`Title: {{ Page.Title|default:"My Site" }}`{% endverbatim %}
* `ChildPages`: A list of child pages of the current page.
* `ChildFiles`: A list of child files of the current page.
* `Config`: The global configuration object. Access site-wide variables via `Config.Variables`.<br>
  Example: {% verbatim %}`{{ Config.Variables.siteTitle }}`{% endverbatim %}
* `Paths`: a map of several path strings for the actual file:
  * `Paths.RootSourceDir`: The full file path to the used `site` folder
  * `Paths.AbsSourcePath`: The full file path to the actual source file
  * `Paths.AbsSourceDir`: The full file path to the actual source file's directory
  * `Paths.RelSourcePath`: The relative file path of the actual file to the root source dir.
  * `Paths.RelSourceDir`: The relative file path of the actual file's directory to the root source dir.
  * `Paths.RelSourceRoot`: The relative file path of the actual file back to the `RootSourceDir` (e.g. `../../..`)
  * `Paths.Webroot`: 	The Webroot prefix, "/" by default
  * `Paths.RelWebPath`: relative (to Webroot) web path to the actual output file
  * `Paths.RelWebDir`: relative (to Webroot) web path to the actual output file's folder
  * `Paths.RelWebPathToRoot`: relative path from the actual output file back to the Webroot
  * `Paths.AbsWebPath`: absolute web path of the actual output file, including the Webroot, starting always with "/"
  * `Paths.AbsWebDir`: absolute web path of the actual output file's dir, including the Webroot, starting always with "/"
* `Webroot(string)`: Function to generate an absolute web path of a given relative path.<br>
  Example usage, assuming the web prefix is set to `/mysite`:<br>
  `Webroot('rel/path/to/file')` => `/mysite/rel/path/to/file`
* `StartsWith(str: string, prefix: string)`: Checks if the given string `str` starts with `prefix`. Same as `strings.HasPrefix`. Useful if you want to highlight navigation markers.<br>
  Example:<br>
  {% verbatim %}`<a href="/foo" class="{% if StartsWith(Paths.AbsWebDir, Webroot('/foo')) %}active{% endif%}">Nav to foo</a>`{% endverbatim %}
* `EndsWith(str: string, suffix: string)`: Same as `StartsWith()`, but checks if the given string `str` ends with `suffix`. Same as `strings.HasSuffix`. Useful if you want to highlight navigation markers.
* `PageQuery()`: Returns a chainable query builder for searching indexed pages. See the [PageQuery](#pagequery--querying-pages-from-templates) section for full documentation.
* `List(items: ...string)`: Helper function that creates a string list from its arguments. Used with `PageQuery()` filter methods that accept multiple field paths.<br>
  Example: {% verbatim %}`List("tags", "categories")`{% endverbatim %}

### YAML front matter variables

You can define a YAML Frontmatter variable map in your `.html` and `.md` templates. This is useful to define variables which can be used in base templates.

The front matter YAML must be at the beginning of the file, encapsulated in `---` separators.

Example: You want to output a page-specific title tag, which is defined in the base template:

```html
{% verbatim %}<!-- Base template in templates/base.html: -->
<!doctype html>
<html lang="en">
    <head>
      <!-- Output page-specific title: -->
        <title>{{ Page.Title|default:"My Page" }}</title>
    </head>
    <body>
        <main id="content">
            {% block content %}{% endblock %}
        </main>
    </body>
</html>{% endverbatim %}
```

```html
{% verbatim %}---
# YAML front matter for partial site in site/index.html:
title: "My page-specific title"
---
{% extends "base.html" %}
{% block content %}
<h1>Welcome!</h1>
...
{% endblock %}{% endverbatim %}
```

This generates the following output page:

```html
{% verbatim %}<!-- build/index.html: -->
<!doctype html>
<html lang="en">
    <head>
      <!-- Output page-specific title: -->
      <title>My page-specific title</title>
    </head>
    <body>
        <main id="content">
<h1>Welcome!</h1>
...
        </main>
    </body>
</html>{% endverbatim %}
```

### Reserved front matter properties

The following front matter properties have special meaning and are handled by pcms directly
(they are not just passed through to `Page.Metadata`):

| Property  | Type    | Default | Description |
|-----------|---------|---------|-------------|
| `title`   | string  | directory name | Sets `Page.Title`. Used for page titles and navigation. |
| `enabled` | boolean | `true`  | Controls whether the page is active. A disabled page returns 404 and is hidden from `ChildPages`. |

#### The `enabled` property

Each page can define an `enabled` property in its YAML front matter to control whether the page is active:

```yaml
---
title: "My hidden page"
enabled: false
---
```

**Behavior:**

* If `enabled` is not set, the page defaults to **active** (`true`).
* A disabled page returns **404** on direct requests.
* Disabled pages are **excluded from `ChildPages`** lists and `PageQuery` results, so they do not appear in navigation or template loops.
* The flag is **propagated downward at index time**: when a parent page is disabled, all its descendant pages are stored as disabled in the index — even if the child pages themselves have `enabled: true` or no `enabled` property at all.

**Example:** Given this page tree:

```text
/           (enabled: true)
/blog       (enabled: false)
/blog/post1 (enabled: true)
```

Both `/blog` and `/blog/post1` will return 404. When the index is built, `/blog/post1` is stored as disabled because its parent is disabled.

> **Note:** After changing the `enabled` flag in a page's front matter, run `pcms index` to rebuild the index so the new state is propagated to all descendant pages.

## PageQuery — querying pages from templates

`PageQuery()` is a chainable query builder that lets you search and filter indexed pages directly from pongo2 templates. It queries the SQLite page index and returns `IndexedPage` objects.

### Creating a query

Call the `PageQuery()` function to get a new builder instance:

```text
{% verbatim %}{% with qb=PageQuery() %}...{% endwith %}{% endverbatim %}
```

### The `List()` helper

Several filter methods accept a list of JSON field paths. Since pongo2 does not support inline array literals, use the `List()` helper to create string lists:

```text
{% verbatim %}{{ List("tags", "categories") }}{% endverbatim %}
```

### Filter methods

All filter methods return a new builder copy and can be chained. Multiple filters are ANDed.

#### `WhereParentRoute(route: string)`

Filters pages by their parent page route.

```html
{% verbatim %}{% for child in PageQuery().WhereParentRoute("/blog").FetchAll() %}
    <li>{{ child.Title }}</li>
{% endfor %}{% endverbatim %}
```

#### `WhereRoute(route: string)`

Filters pages by their route. Supports exact match or prefix match with a trailing wildcard (`*`).

```html
{% verbatim %}{# exact match — find a single page by route: #}
{% with p=PageQuery().WhereRoute("/blog/post-1").First() %}
    <h2>{{ p.Title }}</h2>
{% endwith %}

{# wildcard — find all pages under /blog/ (including /blog itself): #}
{% for p in PageQuery().WhereRoute("/blog/*").OrderBy("title", "asc").FetchAll() %}
    <li>{{ p.Title }}</li>
{% endfor %}{% endverbatim %}
```

#### `WhereMetadataEquals(fields: List, value: string)`

Matches pages where at least one of the given metadata JSON paths has the exact value. Multiple fields are ORed.

```html
{% verbatim %}{# find all pages by author "alice": #}
{% for p in PageQuery().WhereMetadataEquals(List("author"), "alice").FetchAll() %}
    <li>{{ p.Title }}</li>
{% endfor %}

{# search in multiple fields (ORed): #}
{% for p in PageQuery().WhereMetadataEquals(List("author", "editor"), "alice").FetchAll() %}
    <li>{{ p.Title }}</li>
{% endfor %}{% endverbatim %}
```

#### `WhereMetadataContains(fields: List, value: string)`

Matches pages where at least one of the given metadata fields contains the value. Works for both string values (substring match) and JSON array values (element match).

```html
{% verbatim %}{# find pages where the "tags" array contains "go": #}
{% for p in PageQuery().WhereMetadataContains(List("tags"), "go").FetchAll() %}
    <li>{{ p.Title }}</li>
{% endfor %}

{# substring match on a string field: #}
{% for p in PageQuery().WhereMetadataContains(List("description"), "tutorial").FetchAll() %}
    <li>{{ p.Title }}</li>
{% endfor %}{% endverbatim %}
```

#### `WhereMetadataIsOneOf(fields: List, values: List)`

Matches pages where at least one of the given metadata fields matches at least one of the given values. Works for both string and array metadata values.

```html
{% verbatim %}{# find pages tagged with "go" OR "rust": #}
{% for p in PageQuery().WhereMetadataIsOneOf(List("tags"), List("go", "rust")).FetchAll() %}
    <li>{{ p.Title }}</li>
{% endfor %}{% endverbatim %}
```

#### `WhereMetadataLT(fields: List, value: string)`

Matches pages where at least one of the given metadata fields is less than the value (string comparison).

```html
{% verbatim %}{# pages published before 2025-06-01: #}
{% for p in PageQuery().WhereMetadataLT(List("publish_date"), "2025-06-01").FetchAll() %}
    <li>{{ p.Title }}</li>
{% endfor %}{% endverbatim %}
```

#### `WhereMetadataLTE(fields: List, value: string)`

Same as `WhereMetadataLT`, but less than or equal.

#### `WhereMetadataGT(fields: List, value: string)`

Matches pages where at least one of the given metadata fields is greater than the value.

```html
{% verbatim %}{# pages published after 2025-01-01: #}
{% for p in PageQuery().WhereMetadataGT(List("publish_date"), "2025-01-01").FetchAll() %}
    <li>{{ p.Title }}</li>
{% endfor %}{% endverbatim %}
```

#### `WhereMetadataGTE(fields: List, value: string)`

Same as `WhereMetadataGT`, but greater than or equal.

```html
{% verbatim %}{% for p in PageQuery().WhereMetadataGTE(List("publish_date"), "2025-01-01").FetchAll() %}
    <li>{{ p.Title }} ({{ p.Metadata.publish_date }})</li>
{% endfor %}{% endverbatim %}
```

### Ordering and paging methods

#### `OrderBy(field: string, direction: string)`

Adds a sort clause. The field can be a standard page column (`route`, `title`, `updated_at`, `created_at`, `enabled`) or a metadata JSON path. The direction must be `"asc"` or `"desc"`. Multiple calls are cumulative.

```html
{% verbatim %}{# order by title ascending: #}
{% for p in PageQuery().WhereParentRoute("/blog").OrderBy("title", "asc").FetchAll() %}
    <li>{{ p.Title }}</li>
{% endfor %}

{# order by a metadata field, e.g. publish_date descending: #}
{% for p in PageQuery().OrderBy("publish_date", "desc").FetchAll() %}
    <li>{{ p.Title }} — {{ p.Metadata.publish_date }}</li>
{% endfor %}{% endverbatim %}
```

#### `PageSize(size: int)`

Sets the maximum number of results per page. Non-cumulative: the last call wins.

#### `Page(page: int)`

Sets the 1-based page number for pagination. Has no effect unless `PageSize` is set.

```html
{% verbatim %}{# paginated blog listing, 5 per page, showing page 2: #}
{% for p in PageQuery().WhereParentRoute("/blog").OrderBy("publish_date", "desc").PageSize(5).Page(2).FetchAll() %}
    <li>{{ p.Title }}</li>
{% endfor %}{% endverbatim %}
```

### Terminal methods

These methods execute the query and return results. They end the builder chain.

#### `FetchAll() -> []IndexedPage`

Executes the query and returns all matching pages as a list.

```html
{% verbatim %}{% for p in PageQuery().WhereParentRoute(Page.Route).OrderBy("title", "asc").FetchAll() %}
    <a href="{{ p.Route }}">{{ p.Title }}</a>
{% endfor %}{% endverbatim %}
```

#### `First() -> IndexedPage or nil`

Returns the first matching result, or nil if no result is found.

```html
{% verbatim %}{% with featured=PageQuery().WhereMetadataEquals(List("featured"), "true").First() %}
    {% if featured %}<h2>Featured: {{ featured.Title }}</h2>{% endif %}
{% endwith %}{% endverbatim %}
```

#### `Count() -> int`

Returns the total number of matching pages (ignoring `PageSize`/`Page`).

```html
{% verbatim %}<p>Total blog posts: {{ PageQuery().WhereParentRoute("/blog").Count() }}</p>{% endverbatim %}
```

#### `NrOfPages() -> int`

Returns the number of available result pages based on `Count()` and `PageSize`. Returns 1 if `PageSize` is not set.

```html
{% verbatim %}{% with qb=PageQuery().WhereParentRoute("/blog").PageSize(10) %}
    <p>Page count: {{ qb.NrOfPages() }}</p>
{% endwith %}{% endverbatim %}
```

### Enabled page filtering

The query builder automatically filters out disabled pages. The `enabled` flag stored in the index already encodes the full ancestor chain (propagated downward at index time), so a page is excluded whenever its stored `enabled` value is `false` — no recursive parent lookup is needed at query time.

### Complete example

```html
{% verbatim %}{# Blog listing with tag filter and pagination: #}
{% with qb=PageQuery().WhereParentRoute("/blog").WhereMetadataContains(List("tags"), "go").OrderBy("publish_date", "desc").PageSize(5) %}

<h2>Blog posts tagged "go" ({{ qb.Count() }} total, {{ qb.NrOfPages() }} pages)</h2>

<ul>
{% for post in qb.Page(1).FetchAll() %}
    <li>
        <a href="{{ post.Route }}">{{ post.Title }}</a>
        <small>{{ post.Metadata.publish_date }}</small>
    </li>
{% endfor %}
</ul>

{% endwith %}{% endverbatim %}
```

## pcms cli reference

```text
pcms [options] <command> [command options]
```

**Global options** (must appear before the command):

| Option | Default | Description |
|--------|---------|-------------|
| `-c <path>` | `pcms-config.yaml` | Path to the config file. All relative paths in the config are resolved relative to the config file's directory. |
| `-h` | — | Print help and exit. |

---

### init

Creates a new pcms project skeleton in the given directory. The directory is created if it does not exist.

```bash
pcms init <path>
```

**Example:**

```bash
pcms init /path/to/my-site
cd /path/to/my-site
pcms serve
```

---

### index

Builds (or rebuilds) the SQLite page index from the source folder. Walks the `source` directory tree, extracts front matter metadata, and stores all pages and files in `pcms.db`. Run this after adding or changing content outside of `pcms serve`.

```bash
pcms index
pcms -c /path/to/pcms-config.yaml index
```

The database path defaults to `pcms.db` next to the config file and can be changed via `database_path` in `pcms-config.yaml`.

**Note:** `pcms serve` runs an initial index automatically when the database is empty, so a separate `pcms index` call is only needed when you want to pre-build the index or refresh it without starting the server.

---

### serve

Starts the web server and serves the indexed site. On the first start, if the page index is empty, it builds the index automatically.

```bash
pcms serve
pcms -c /path/to/pcms-config.yaml serve
pcms serve -listen :8080
```

**Options:**

| Option | Default | Description |
|--------|---------|-------------|
| `-listen <addr>` | `:3000` | TCP/IP listen address. Accepts `host:port`, `:port`, or a full address such as `127.0.0.1:8888`. Overrides `server.listen` from the config file. |

---

### serve-doc

Starts a web server that serves the built-in pcms documentation. No config file or project directory is required.

```bash
pcms serve-doc
pcms serve-doc -listen :9000
```

**Options:**

| Option | Default | Description |
|--------|---------|-------------|
| `-listen <addr>` | `:3000` | TCP/IP listen address, same format as in `serve`. |

---

### cache-clear

Removes all files in the page file cache directory (configured via `server.cache_dir` in `pcms-config.yaml`, defaults to `.pcms-cache`). The cache is rebuilt automatically on the next `pcms serve` request.

Use this when cached HTML is stale and you want to force a full re-render without restarting the server, or before deploying updated templates.

```bash
pcms cache-clear
pcms -c /path/to/pcms-config.yaml cache-clear
```

---

### enable-page

Enables a page in the index database. By default only the specified page and its direct files are enabled; child pages are left unchanged. Pass `-r` to also enable all descendant pages and their files recursively.

```bash
pcms enable-page <route>
pcms enable-page -r <route>
pcms -c /path/to/pcms-config.yaml enable-page /blog
```

**Options:**

| Option | Description |
|--------|-------------|
| `-r` | Recursively enable all descendant pages and files. |

**Example:**

```bash
# Enable only /blog (its child pages remain as-is):
pcms enable-page /blog

# Enable /blog and every page and file beneath it:
pcms enable-page -r /blog
```

> **Cache note:** Enabling a page updates the index database, but any previously rendered HTML in the cache may still reflect the old state. For example, a parent page whose template lists child pages will continue to show the cached version — the newly enabled child will not appear until the cache entry is invalidated. Run `pcms cache-clear` after enabling pages to ensure the next request re-renders all affected pages.

---

### disable-page

Disables a page and all its descendant pages and files in the index database. Disabled pages return 404 and are excluded from `ChildPages` and `PageQuery` results.

Disabling always cascades to all descendants — there is no non-recursive variant.

```bash
pcms disable-page <route>
pcms -c /path/to/pcms-config.yaml disable-page /blog
```

**Example:**

```bash
# Disable /blog and every page and file beneath it:
pcms disable-page /blog
```

> **Cache note:** Disabling a page updates the index database, but previously cached HTML may still be served until it is invalidated. A parent page that lists child pages in its template will continue to show the disabled child in the cache until that entry is rebuilt. Run `pcms cache-clear` after disabling pages to prevent stale content from being served.