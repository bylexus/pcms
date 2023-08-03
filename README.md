# pcms - The Programmer's Content Management System

This project, historically called `pcms`, is a static site builder and web server written in [GO](https://go.dev/).

I don't need a fancy, UI-driven CMS. I don't WANT a  UI, and I don't want a CMS that is in my way of doing things.
A CMS is too restrictive. I don't fear writing HTML and program code. I am a developer, at last, so I feel more
comfortable writing code in an editor than clicking in a UI.

This is the idea behind **pcms**, the Programmer's CMS: A clutter-free, code-centric, simple static site builder and server to deliver web sites. For people that love to code, and just want things done.

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
