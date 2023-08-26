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

or, if you checked out the source code locally:

```sh
$ cd path/to/pcms
$ docker build -t pcms ./
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
├── build/
├── log/
├── pcms-config.yaml
├── site
│   ├── favicon.png
│   ├── index.html
│   └── ... more files ...
└── templates/
    ├── base.html
    └── ... more files ...
```

* `pcms-config.yaml` is the site-wide configuration. It contains pcms-specific settings like server port as well as user-defined content which can be used in your site templates.
* `site/` is the folder where all your page content goes, and which is processed by the build process.
* `build/` is where your generated static content goes by default.
* `templates/` contains the `pongo2` templates used for your site.

## Site configuration: `pcms-config.yaml`

The main config file is `pcms-config.yaml`.

An example `pcms-config.yaml`:

```yaml
# server config:
server:
  listen: ":3000"
  prefix: ""
  watch: true
  logging:
    access:
      file: STDOUT
      format: ""
    error:
      file: STDERR
      level: DEBUG
# source filder:
source: "site"
# output / build folder:
dest: "build"
# site-specific global variables:
variables:
  siteTitle: pcms-go - Documentation
  siteMetaTags:
    - name: "keywords"
      content: "reference,pcms,cms, golang"
template_dir: templates
exclude_patterns:
  - "^\\..*"
processors:
  scss:
    sass_bin: "/usr/bin/sass"
```

Adapt the config as needed by your page.

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
$ pcms -c path-to-site/pcms-config.yaml serve
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

