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

pcms is a static site builder and mini-webserver written in [GO](https://go.dev/) to build and deliver web pages from HTML / Markdown templates or static files.
The main goal is to create static page content from templates and / or configuration/data files, all file-system-based, no UI configuration.

In essence pcms is built upon the following infrastructure:

* a page builder that examines your `site/` dir and builds as static version to `build/`
* a Web server that handles your requests and delivers the static pages
* a Template engine based on [pongo2](https://github.com/flosch/pongo2), a Django-like template engine,
  with support for YAML frontmatter and configuration variables
* all delivered in the single `pcms` binary!

## Site structure and routes

A typical `pcms` site may look as follows:

```sh
.
├── pcms-config.yaml          # The main config file for the site
├── site/                     # The site dir contains the page content, and listens to the "/" route
│   ├── index.html            # additional page content / templates
│   ├── styles.scss           # an scss file which may be transpiled to css using dart-css
│   ├── variables.yaml        # A YAML file containing variables. Inherited to all (sub-)pages
│   ├── favicon.png
│   ├── html-page             # another page, here route /html-page
│   │   ├── index.html        # a html content file
│   │   └── sunset.webp       # some static content
│   └── markdown-page         # another page, here a Markdown page for the route /markdown-page
│       ├── index.md          # a Markdown content file
│       ├── variables.yaml    # A local YAML file containing variables. Overrides upper variables
│       ├── favicon.png
│       └── sunset.webp
└── templates               # pongo2 templates for your html / markdown content
    ├── base.html
    ├── error.html
    └── markdown.html
```

* The **`site`** contents are stored as directory tree in the `site/` folder.
* Every file within `site/` (including `site/` itself) is directly rendered to the same folder structure in the `build/` folder:
  The files are renamed if processed:
  * `.md` files become `.html` files
  * `.html` files stay `.html` files
  * `.scss` files become `.css` files
  * all other files are copied 1:1 to the destination folder.

The `build/` folder is the webroot folder, and the files correspond relatively to the build folder to the same URL route.

## Site building

With the `pcms build` or `pcms serve` command, the site is built from the `site/` folder. Each file
is examined, processed and written to the output (`build/`) folder.

The system supports different processors, determined by the source file's file ending:

* `*.html` files are processed by the `html_processor`:
  * A YAML Frontmatter is extracted from the file, if present. The Frontmatter is available as `variables` map in the
    `pongo2` template
  * The HTML file is processed as `pongo2` template, having access to the variables defined in the frontmatter or in `variables` files.
  * Finally, the processd file is written to the output folder to the same relative path.
* `*.md` files are processed by the `md_processor`:
  * A YAML Frontmatter is extracted from the file, if present. The Frontmatter is available as `variables` map in the
    `pongo2` template
  * The Markdown file is processed as `pongo2` template, having access to the variables defined in the frontmatter or in `variables` files.
  * The Markdown file is converted to HTML.
  * Optionally, the converted HTML can be embedded into a template, defined in the `template` YAML frontmatter variable.
  * Finally, the processd file is written to the output folder to the same relative path, but with a `.html` ending.
* `*.scss` files are processed by the `dart-sass` compiler, which must be present as an external binary, and
  written as `.css` file to the output folder.
* `variables.yaml` files are read and merged into the `variables` map, which is available in the templates.
  Deeper `variables.yaml` file entries override values in higher files, and are merged within the hierarchy.
  `variables.yaml` files are not written to the output folder. 
* All other files are treatened as raw files and copied 1:1 to the output folder.


## Processors

### HTML processor

The HTML processor takes `.html` files as input, and processes them:

An HTML file is processed as pongo2 template, and may be configured using a YAML Frontmatter.

Example:

```html{% verbatim %}
---
# YAML front matter
title: 'Hello'
---
{% extends "base-template.html" %}
<h1>{{title}}</h1>
<p>This file is processed using the 'base-template.html' template file</p>{% endverbatim %}
```

1. All variables from `variables.yaml` files in the hierarchy from this file's folder up to the root folder are
   merged into the `variables` map.
2. The YAML Frontmatter is extracted from the HTML file, and merged with the `variables` map
3. The complete HTML then is processed using the `pongo2` engine, and written as final HTML to the output folder.


### Markdown processor

The Markdown processor is working very similar to the HTML processor, but takes `.md` files as input, and processes them to HTML:

A Markdown file is processed as pongo2 template with a YAML Frontmatter, converted to HTML using a template, 
and again processed as a pongo2 HTML template.

Example:

```markdown
---
# YAML front matter
template: 'base-template.html'
title: 'Hello'
---
# {% verbatim %}{{title}}{% endverbatim %}

This **Markdown** file is processed using the 'base-template.html' template file
```

1. All variables from `variables.yaml` files in the hierarchy from this file's folder up to the root folder are
   merged into the `variables` map.
2. The YAML Frontmatter is extracted from the Markdown file, and merged with the `variables` map
3. The `template` variable defines the used pongo2 template file: The (processed) contents of this file is available as `content` variable.
   For example, the `base-template` file may look as follows:

```html
{% verbatim %}<!-- templates/base-template.html file -->
<!doctype html>
<html lang="en">
    <head>
        <title>{{variables.title}}</title>
    </head>
    <body>
      <div id="page_content">
      <!-- The processed markdown content is placed here: -->
      {{ content | safe }}
      </div>
    </body>
</html>{% endverbatim %}
```
4. The complete HTML then is processed using the `pongo2` engine, and written as final HTML to the output folder.


### SCSS Processor

### Raw File Processor

This is the simplest processor: It just copies the input file 1:1 to the output file, keeping the same relative path
with no further processing.