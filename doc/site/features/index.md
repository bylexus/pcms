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

* Provide a simple, non-intrusive Page delivery system, based on HTML templates
* No database needed, no setup, no strings. The filesystem is also the Site structure.
* Programmer friendly, or, code-first: There is no UI! Ever!
* Can render markdown as well as plain HTML, but with some kind of template engine

## Implemented features

* Web server written in GO
* serves HTML and Markdown files from a root folder
* Folder structure defines URL routes
* Content is rendered using a Template engine (see pongo2 below)
* No database needed, file-based only.
* Uses [pongo2](https://github.com/flosch/pongo2), a [Django-like](https://docs.djangoproject.com/en/4.0/topics/templates/) template engine written in GO for html/markdown files to create pages based on templates
* separates theme / layout from content
* restricted pages with HTTP Basic Auth credentials supported
* self-contained binary: you just need the one single binary to run a pcms site

## Once-to-be-implemented features

* on-the-fly sass-to-css transformation
* static output generation of whole site
* Template / HTML caching
* API for querying and maintaining the running app
* ... and a lot more: Ideas are pending...
