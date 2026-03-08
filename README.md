# pcms - The Programmer's Content Management System

This project, historically called `pcms`, is a static site builder and web server written in [GO](https://go.dev/).

I don't need a fancy, UI-driven CMS. I don't WANT a  UI, and I don't want a CMS that is in my way of doing things.
A CMS is too restrictive. I don't fear writing HTML and program code. I am a developer, at last, so I feel more
comfortable writing code in an editor than clicking in a UI.

This is the idea behind **pcms**, the Programmer's CMS: A clutter-free, code-centric, simple static site builder and server to deliver web sites. For people that love to code, and just want things done.

---
## ⚠⚠⚠ Refactor / Rewrite in progress! ⚠⚠⚠

_"Wait, what? Another restart?"_

Yes. I want to give pcms another direction - instead of a static site builder, which in fact I don't need, I want to implement it as an application server that indexes its 
page tree, not in memory, as today, but in a small in-process DB, like sqlite or duckdb.

This enables some features I'd like to see in pcms:

* The system should know the whole page document tree. Templates should be able to access its childs / anchestors / other pages / content.
* The template system can support helper functions to query/filter/search the document tree
* The db approach allows for searching / indexing.

So, please give the project some time to reinvent itself, again :-)

Have a look at the [Vision](#refactor-vision) section below for more information.

Alex, 08.03.2026

---

## Refactor Vision

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

**existing (old) doc, needs updating:**

## Why "CMS"?

The name `pcms`, a "Programmer's Content Management System" is rooted in the early days of the project: At the beginning, the idea was to build a content management system to deliver
content from a DB / dynamically instead of building static websites. Thus the name.

The project was re-written multiple times in multiple langauges, and I only realized over time that I ALSO don't need a CMS at all, but really just a static site builder.
But then it was too late, the name has already burned in :-)

## Features

* Static site builder: Builds static HTML from html templates, Markdown templates, scss/sass.
* Uses [pongo2 template engine, a django-like template engine](https://github.com/flosch/pongo2) for html/markdown files to create pages based on templates
* Uses [Dart Sass](https://github.com/sass/dart-sass) to create CSS from scss files
* Built-in Web-Server for deliver the page as web site
* Single-binary: The whole tool including documentation and a skeleton project is included in the single `pcms` binary. No other requirements.

## Project Status

A first viable product is already available - All features to drive a full, real website are implemented. The first production site is already using
pcms: <https://alexi.ch/> is driven by the actual pcms version.

Still, this is an early stage, and changes have to be expected.

## Quick Start

A getting started guide can be found in the documentation - see `doc/site/quickstart/index.md`, or check it out online: <https://pcms.alexi.ch/quickstart>

For the even more impatient:

```sh
# build PMCS (golang / go tools needed)
$ make build

# create a new site:
$ bin/pcms init path/to/site/

# serve it at localhost:3000
$ cd path/to/site/
$ bin/pcms serve

# read the doc at localhost:8888
$ bin/pcms serve-doc -listen :8888
```