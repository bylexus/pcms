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

pcms is a mini-framework written in [GO](https://go.dev/) to deliver web pages from HTML / Markdown from templates.
The main goal is to deliver page content and metadata from templates and / or configuration/data files, all file-system-based, no UI configuration.

In essence pcms is built upon the following infrastructure:

* a page builder that examines your `site/` dir and builds a page tree
* a Web server that handles your requests and delivers the pages
* a Template engine based on [pongo2](https://github.com/flosch/pongo2), a Django-like template engine
* all delivered in the single `pcms` binary!

## Site structure and routes

A typical `pcms` site may look as follows:

```sh
.
├── pcms-config.yaml          # The main config file for the site
├── site/                     # The site dir contains the page content, and listens to the "/" route
│   ├── page.json             # each folder with a page.json is considered a "page"
│   ├── index.html            # additional page content / templates
│   ├── favicon.png
│   ├── html-page             # another page, here route /html-page
│   │   ├── page.json
│   │   ├── index.html
│   │   └── sunset.webp
│   └── markdown-page         # another page, here a Markdown page for the route /markdown-page
│       ├── page.json
│       ├── index.md
│       └── sunset.webp
└── themes                    # the themes folder keeps all site themes
    └── default               # this is the theme named "default"
        ├── static            # static theme content, available under the /theme/static route
        │   ├── css
        │   │   └── main.css
        │   └── sunset.webp
        └── templates         # templates are available in the page.json's template config for single pages
            ├── base.html
            ├── error.html
            └── markdown.html
```

* The **site** contents are stored as directory tree in the `site/` folder.
* Every directory within `site/` (including `site/` itself) that contains a `page.json` file is considered a **page**, and answers to the corresponding URL route:
  for example, the directory `site/article/sunset/`, which contains a `page.json` file, matches the URL route `/article/sunset`.
* Sub-folders / files in non-pages folders are considered as route of a top page with URL tail relative to the parent page:
  So all content belong to a parent page. This can also be the root page. The parent page
  defines the metadata as well as access restrictions to all sub-content.
* Every URL route matches either a page directory, or a real file (e.g. static content like images).
* Pages can be HTML, HTML (pongo2) templates, Markdown with an HTML template, or JSON with an HTML template.

The `page.json` file contains the metadata information for the specific page. It may look as follows:

```js
{
    "index": "index.html",                       // required. This defines the page type as well as the included template.
    "enabled": true,                             // disabled pages are not delivered by a route, but still added to the page tree.
    "template": "template.html",                 // Only used by Markdown or JSON pages, this is the template used to embed and process the original data / markdown.
    "order": 1,                                  // child order within the parent page. This can be everything, it will be ordered naturally.
    "title": "Home",                             // additional metadata, like page title, keywords etc. This info is available in the page templates.
    "keywords": ["php", "nodejs","programming"],
    // ..... more data you want to have available in your templates
}
```

## Page delivery

The page delivery works as follows:

1. The requested route is matched against the page tree: The longest-matching page in the tree is figured out.
2. The page is checked for its availability (enabled flag is true)
3. If the route and route tail matches static content, deliver it, end here.
4. If the route matches a page, examine its metadata (`page.json`), and
5. render the page according to its type.

## Page types

The framework supports the following page types, determined **only** by the file ending of the `index` file entry in `page.json`:

* `html`: The framework looks for an HTML template named by the `index` key (e.g.: `{"index": "index.html"}`). The HTML file
    is read, processed by [pongo2](https://github.com/flosch/pongo2), and rendered back to the browser.
* `markdown`: The framework looks for a `.md`-Markdown file named by the `index` key (e.g.: `{"index": "index.md"}`). The Markdown file
    is read, processed **first** by the template engine [pongo2](https://github.com/flosch/pongo2), **then** the markdown is processed to HTML, embedded in an HTML template (if given in `page.json`, `template` property) and rendered back to the browser.
* `json`: The framework looks for a `.json`-index named by the `index` key (e.g.: `{"index": "data.json"}`).
  It then reads the (HTML) template defined in the `template` key, and processes it by [pongo2](https://github.com/flosch/pongo2), and rendered back to the browser.
  This is meant for e.g. simple index / menu files, where all the data are structured json data rendered by a template.

## Page index / page cache

The whole page tree is built upon application start: The `site` folder is walked recursively, and each page (a folder with a `page.json` file) is
examined, a page node info is created.

This simple in-memory cache should be enough for smaller sites. If things get bigger, I will search for another solution. At the moment, [Yagni](https://www.martinfowler.com/bliki/Yagni.html).

I plan to scan for changes in the file system and automatically re-build the page tree, but
for the moment, Yagni, see above. It's on my to-do-list, though. For now, just restart pcms if you do
changes at the page structure.

The page tree is also available in the page templates, to build menus, for example. See the [Reference section]({{base}}/reference/) for more information.
