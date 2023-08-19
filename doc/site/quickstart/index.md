---
title: "quick start"
shortTitle: "Quick Start"
template: "page-template.html"
metaTags: 
  - name: "keywords"
    content: "quick start,pcms,cms"
  - name: "description"
    content: "Quick start of running pcms"
---
# Quick start - for the impatient

This chapter guides you very quickly from installation to a fully-running site. For more information, please refer to the rest of the documentation.

## Step 1 - get pcms

You only need a single `pcms` binary to run a website. Check out the sections below for a brief description how you get one.

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

## Step 2 - init your project

`pcms` comes with a built-in starter template, which makes it easy to get started.

Open a terminal, and create a new pcms skeleton:

```sh
$ pcms init [project-path]
$ cd [project-path]
$ pcms serve
```

And done! Your site is serving on http://localhost:3000/!

Your static pages are also generated to the `build/` folder.


## Step 3 - Generate content!

The site skeleton is already up and running - now it's time to look at the folder structure
and implement some content.

After initializing a skeleton app with `pcms init`, you get the following file structure
(content a bit redacted for brevity):

```text
.
├── log
├── pcms-config.yaml        # The main configuration file. Adapt as needed
├── site                    # your site's source files
│   ├── index.html
│   ├── favicon.png
│   ├── html-page
│   │   ├── index.html
│   │   └── sunset.webp
│   ├── markdown-page
│   │   ├── index.md
│   │   └── sunset.webp
│   ├── static
│   │   └── css
│   │       └── main.css
│   └── variables.yaml
└── templates               # pongo2 templates for your html / markdown content
    ├── base.html
    ├── error.html
    └── markdown.html
```

Your page's __content__ goes to the `site` folder: This folder contains all your source files, markdown and html partials,
static content and whatnot.

Inspect the folder structure to see how they work. It is really simple:

* the main config file is `pcms-config.yaml` in the base directory. It stores all the important information to run your site.
* The `site/` folder contains your source files: html, markdown partials, static content, scss files etc. You mainly work in here
  to generate content.
  * `.html` files are processed as [`pongo2`](https://github.com/flosch/pongo2) templates, and support a YAML frontmatter (see next chapter)
  * `.md` files are processed as [`pongo2`](https://github.com/flosch/pongo2) templates, converted to `.html` and support a YAML frontmatter (see next chapter)
  * `.scss` files are processed with the [`dart-sass`](https://sass-lang.com/dart-sass/) compiler (need to be installed separately)
* The `templates/` folder contains your `pongo2` base templates for your site, if needed

### YAML front matter

`.html` and `.md` files support a YAML front matter to define template variables: See `site/markdown-page/index.md` as an example:

```text
---
template: "markdown.html"
title: "Markdown sample page"
shortTitle: "Markdown"
iconCls: "fab fa-markdown"
mainClass: "markdown"
metaTags:
  - name: "keywords"
    content": "pcms,cms"
  - name: "description"
    content": "Markdown sample page"
---
# Markdown Demo Page

[<i class="fas fa-home"></i> Back to home]({{webroot("/")}})

Your actual Route: {{paths.absWebPath}}

Access to pongo2 syntax, e.g. actual time: {% now "02.01.2006 15:04:05" %}

Use an icon: <i class="{{variales.iconCls}}"></i>
```

The YAML front matter is the start part beteween the `---` separators, and (may) contain valid YAML content.
This content then will become availabe in your `pongo2` templates in the `variables` map.
For Markdown files, the only needed entry is `template: `, which defines the HTML template to be used from the `templates/` folder
to generate the final HTML.

## Step 4: Build and serve

`pcms serve` watches your files for changes and re-builds the output when any file changes in the `site/` or `template/` folder.
If you just want to build your files without starting a server and watch, you can just issue 

```
pcms build
```

The final static site is built into the `build/` folder. You can now copy this folder to your public web server, if needed.

## Step 5: ... and beyond!

Please have a look at the other chapters for a more detailed description on how to configure
and use `pcms`.



