---
title: "install / setup"
shortTitle: "Install / Setup"
template: "page-template.html"
metaTags: 
  - name: "keywords"
    content: "install, setup,pcms,cms"
  - name: "description"
    content: "Installation and setup of sites using pcms"
---
# Installation / Setup of pcms-driven sites

`pcms` comes with a site template which helps you getting startet. It will create a skeleton site / structure for you so you don't have to start from scratch.
  You don't need to generate a skeleton site if you want to do it manually.

This documentation shows you how to install `pcms` and generate a new site from scratch, as well as running it / an existing site.

- [Install pcms](#install-pcms)
  - [As pre-built binary](#as-pre-built-binary)
  - [As go source code](#as-go-source-code)
  - [Build a Docker image](#build-a-docker-image)
- [Generate a skeleton site](#generate-a-skeleton-site)
- [Site configuration: `pcms-config.yaml`](#site-configuration-pcms-configyaml)
- [Page config with `page.json`](#page-config-with-pagejson)
- [Page content](#page-content)
- [Starting the site](#starting-the-site)
  - [Start `pcms` with the local binary](#start-pcms-with-the-local-binary)
  - [Start `pcms` in a docker container](#start-pcms-in-a-docker-container)

## Install pcms

This section was already covered in the [Quick Start]({{webroot("quickstart/")}}) section. If you already followed there, you don't have to repeat this steps.

You only need a single `pcms` binary to run a website. Check out the next chapters for a brief description how you get one.

### As pre-built binary

You can download a `pcms` binary from a pre-built release from github:

https://github.com/bylexus/pcms/releases

Download and extract the `pcms` binary that fits your OS and architecture.

### As go source code

You can build `pcms` from the go source code:

* install the [go build toolchain](https://go.dev/dl/)
* clone the `pcms` source code from git:

```sh
$ git clone https://github.com/bylexus/pcms.git
```
* build it:

```sh
$ cd pcms
$ make build # ==> build goes to bin/pcms
```

### Build a Docker image

You can build and run `pcms` as a Linux container, using [Docker](https://www.docker.com/) or [Podman](https://podman.io/).

**Build the Docker image:**

```sh
$ docker build -t pcms https://github.com/bylexus/pcms.git
```


## Generate a skeleton site

If you don't want to start from scratch, generate a skeleton site:

```sh
# local binary:
$ pcms init path-to-site

# with the docker image:
$ mkdir path-to-site
$ docker run --rm -v $(pwd)/path-to-site:/site pcms pcms init /site
```

This will generate a fully-working demo site into the given folder (here: `path-to-site`). Note that existing files **will be overwritten**. You end up with a folder structure like this:

```sh
path-to-site/
├── pcms-config.yaml
├── site
│   ├── favicon.png
│   ├── page.json
│   └── ... more files ...
└── themes
    └── default
        ├── static
        └── templates
```

* `pcms-config.yaml` is the site-wide configuration. It contains pcms-specific settings like server port as well as user-defined content which can be used in your site templates.
* `site/` is the folder where all your page content goes. Every pcms-visible page is a folder within `site` with a `page.json` file.
* `themes/` contains (the / several) layouts and styles for your site: It is the folder where your site-wide HTML and CSS layouts / files are stored and fetched. Each theme corresponds to
  the folder name within `themes/`. The actual theme can be set up in `pcms-config.yaml`.


## Site configuration: `pcms-config.yaml`

The main config file is `pcms-config.yaml`. It servers the following purposes:

* It defines the folder where pcms looks for content: The folder where `pcms-config.yaml` is placed must contain the `themes` and `site` folder.
* It defines pcms configuration like server settings (port).
* It offers you a global configuration for additional data / configs that is available in all your templates.

A minimal example `pcms-config.yaml`:

```yaml
# Server config:
server:
  listen: ":3000"
  logging:
    access:
      file: STDOUT
    error:
      file: STDERR
      level: DEBUG

# The site config defines all things about your site.
site:
  theme: default
  title: My Site Title
```

There are only two properties you must define in order to run pcms:

* `port`: This is the TCP port the webserver listens to
* `site.theme`: The theme to use (a folder with the same name in `themes/`)

All the other config options can be chosen freely by the theme / site creator.

## Page config with `page.json`

Each (sub-)folder within `site/` that contains a `page.json` is considered a "page" by pcms, and is indexed in the site tree. So each page has its own `page.json`.
Other folders or sub-folders that exists are not indexed in the site tree, but can serve as static content containers, belonging to an upper page.

An example `page.json` file:

```json
{
    "enabled": true,
    "index": "index.md",
    "template": "page-template.html",
    "shortTitle": "pcms doc",
    "iconCls": "fas fa-home",
    "mainClass": "home",
    "metaTags": [
        {"name": "keywords", "content": "pcms,cms"},
        {"name": "description", "content": "Documentation for the pcms system, the programmer's cms"}
    ]
}
```

The page config consists of pcms-needed configs, but can contain any other value you may need to
generate your page (e.g. here, 'shortTitle' or 'iconCls' are used by the template, not by pcms). All  config is available in that page's template / context.

Configs needed by pcms are:

* `enabled`: If false, the page including its sub-pages and static content are not delivered. You can use this flag to disable a page temporarily.
* `index`: The file that is read as content file. This can be html, Markdown, json or js, depending of the file ending pcms will act accordingly.
* `template`: for non-HTML index files, this html template file is read and processed by the template engine. The content from `index` is available as `content` template variable.

## Page content

The `index` property of the `page.json` file defines the page content file. In the simplest form, this is a plain HTML file that is rendered via pongo2 (django-like) template engine.
The template engine injects a page context: This is an object with available template variables that can be used within the template.

The page context contains the following variables.

* `site`: This object contains the `pcms-config.yaml` entries.
* `base`: the configured webroot in `pcms-config.yaml`
* `page`: The Page Node object, containing several information about the page, including the `page.json` entries (`page.Metadata`), actual route (`page.Route`), child pages (`page.Children`).
* `rootPage`: The site's root node object. Same type as `page`, but for the site's root page.
* `now`: The actual date, to be used as in django, but with the Go time format strings

All these variables can be used within the template. An example:

```html
{% verbatim %}<!-- index.html -->
{% extends "base.html" %}

<h1>{{variables.title}}</h1>
<p>Hello. It is now {% now "03:04 on 02.01.2006" %}.</p>
<a href="{{webroot("/")}}">Back to home</a>

A relative link: <a href="{{path.absWebDir}}/sub/folder/picture.png">Image</a>
{% endverbatim %}
```

## Starting the site

Now your site is generated / ready, you can start the server. The site's root folder is the
directory where `pcms-config.yaml` is located. The easiest way to start `pcms` is from within
this directory.

### Start `pcms` with the local binary

Execute `pcms serve` from within your site's root folder:

```sh
$ cd path-to-site/
$ pcms serve
```

You can also give the location of the config file, so you don't have to change to the dir:

```sh
$ pcms serve -c path-to-site/pcms-config.yaml
```

### Start `pcms` in a docker container

If you are using the `pcms` docker image, you can create and run a container as follows:

```sh
$ docker run -d \
    --name mysite \
    -v path/to/site:/site \
    -p 3000:3000 \
    -w /site
    pcms
```

This will mount your local site directory to the docker's `/site` directory, and also
uses this dir as working dir. Export the port you configured, and you're good to go!

