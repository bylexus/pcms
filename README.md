# pcms - The Programmer's Content Management System

I don't need a fancy, UI-driven CMS. I don't WANT a  UI, and I don't want a CMS that is in my way of doing things.
A CMS is too restrictive. I don't fear writing HTML and program code. I am a developer, at last, so I feel more
comfortable writing code in an editor than clicking in a UI.

This is the idea behind **pcms**, the Programmer's CMS: A clutter-free, code-centric, simple CMS to deliver web sites. For people that
love to code, also while producing content.

## Features

* simple, fast, node-based page delivery system. No database needed, file-based only.
* serves HTML and markdown (which is rendered to HTML, too :-))
* Uses [pongo template engine, a django-like template engine](https://github.com/flosch/pongo2) for html/markdown files to create pages based on templates
* separates theme / layout from content
* Support for restricted pages using basic authentication

## Project Status

This is a rewrite of the existing pcms source, which is in JavaScript: This project is now re-written in GO,
and is meant as a learning project for myself.

### TODOs, Requirements to the Go App

* + Write my own Webserver in GO
* Logging: both request and application logging to files
* + read config from .env file or similar
* + supports html and markdown as templates
* + prepares the templates by using a template engine to apply the final output
* + supports themes - aka different base layouts 
* + supports the existing pcms structure: 
  * + a folder with page.json config and content represents a page
  * + sub-folders / files are served statically, or as sub-page
* + supports route/folder authentication (basic auth)
  * add rate limiting for auth requests, to prevent brute-force attacks
* + builds page structure in memory
* watches for changes, rebuilds the page structure on the fly without restart
* configurable 404 page
* support some real webserver features:
  * Range header to seek / stream files
* support all today's server config and page.json config, see https://pcms.alexi.ch/reference


### Migrating nunjucks templates from pcms V1

pcms-go uses [pongo](https://pkg.go.dev/github.com/flosch/pongo2/v4@v4.0.2) instead of nunjucks as a template engine.
Therefore, and for the fact that I changed from nodejs to golang, there are several changes for the templates.

In general, the template engine is more or less compatible with the Django Template engine, see here: https://django.readthedocs.io/en/1.7.x/topics/templates.html

Because pcms-go uses pongo (django template language) instead of nunjucks, the templates need to be converted. The following
needs to be changed:

* Change typed comparison operators (`===`, `!==`) to standard ones (`==`, `!=`)
* Changes on the `page` object:
  * in general: All properties are `Uppercased`:
  * `page.title` ==> `page.Title`
  * `page.pageConfig` ==> `page.Metadata`
    * No default metadata anymore: `page.Metadata.enabled` is NOT true by default anymore, only if explicitely set
  * `page.childPages` ==> `page.Children`
  * `page.childPagesByRoute[routestr]` ==> `page.ChildPageByRoute(routeStr)`
* `route` ==> `page.Route`
* `rootPage`: see `page` (same object)
* Changes on the `site` object:
  * in general: All properties are `Uppercased`:
  * `site.title` ==> `site.Title`
* changes to special functions / filters:
  * `{% now %}` is now a Django Tag, but with a twist: its date format is that from the golang time format specification.
    So instead of `{% now "Y" %}` it is now `{% now "2006" %}`.
  * `{% for in in data | reverse %}` now becomes ´{% for in in data reversed %}`