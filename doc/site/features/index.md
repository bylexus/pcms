# Features

**Note**: pcms serves my own needs (only?), so I just implement what is necessary for my projects. If something is missing, that means I just had no use for that feature.

## Design Goals

* Provide a simple, non-intrusive Page delivery system, based on HTML templates
* No database needed, no setup, no strings. The filesystem is also the Site structure.
* Programmer friendly, or, coding should not fear you.
* Understand markdown as well as plain HTML, but with some kind of template engine

## Implemented features

* simple, fast, node-based page delivery system.
* No database needed, file-based only.
* serves HTML and markdown (which is rendered to HTML, too :-))
* Uses [pongo2](https://github.com/flosch/pongo2), a [Django-like](https://docs.djangoproject.com/en/4.0/topics/templates/) template engine written in GO for html/markdown files to create pages based on templates
* separates theme / layout from content
* restricted pages with HTTP Basic Auth credentials

## Once-to-be-implemented features

* on-the-fly sass-to-css transformation
* static output generation of whole site
* Template / HTML caching
* API for querying and maintaining the running app
