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

* Provide a simple static site generator based on Markdown or raw HTML. No magic.
* Generate final content using HTML templates
* No database needed, no setup, no strings attached. The filesystem is also the Site structure.
* Programmer friendly, or, code-first: There is no UI! Ever!
* No magic. Magic fuzzies your clear view. Magic is for wizards, not for normal human beings.

## Implemented features

* Static site generator: generates a final, statically rendered site using:
  * HTML, optionally using templates
  * Markdown within an HTML template
  * scss transpiler transforms your scss into css
  * all other files are just copied 1:1 to the destination folder
* Serves the built site with its own web server
* Watches files for changes and re-builds the output on-the-fly
* Folder structure defines URL routes
* Content is rendered using a Template engine (see pongo2 below)
* No database needed, file-based only.
* Uses [pongo2](https://github.com/flosch/pongo2), a [Django-like](https://docs.djangoproject.com/en/4.0/topics/templates/) template engine written in GO for html/markdown files to create pages based on templates
* generate starter skeleton
* self-contained binary: you just need the one single binary to run a pcms site, AND to read the docs


## Once-to-be-implemented features

* API for querying and maintaining the running app
* document indexing for implementing (server/client-side?) searching
