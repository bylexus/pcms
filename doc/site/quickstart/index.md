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

## Step 2 - init your project

`pcms` comes with a built-in starter template, which makes it easy to get started.

Open a terminal, and create a new pcms skeleton:

```sh
$ pcms init [project-path]
$ cd [project-path]
$ pcms serve
```

And done! Your site is serving on http://localhost:3000/!


## Step 3 - Generate content!

The site skeleton is already up and running - now it's time to look at the folder structure
and implement some content.

After initializing a skeleton app with `pcms init`, you get the following file structure
(content a bit redacted for brevity):

```sh
.
├── pcms-config.yaml
├── site
│   ├── page.json
│   ├── index.html
│   ├── favicon.png
│   ├── html-page
│   │   ├── index.html
│   │   ├── page.json
│   │   └── sunset.webp
│   ├── markdown-page
│   │   ├── index.md
│   │   ├── page.json
│   │   └── sunset.webp
│   └── restricted
│       ├── index.html
│       └── page.json
└── themes
    └── default
        ├── static
        │   ├── css
        │   │   └── main.css
        │   └── sunset.webp
        └── templates
            ├── base.html
            ├── error.html
            └── markdown.html
```

Your page __content__ goes to the `site` folder, while your site __design__ (theme) goes to the `themes/[theme-name]` folder (here: `default`). The used theme is configured in `site-config.json`, and can be adapted as needed:
The theme name is just the folder name in `themes`.

Inspect the folder structure to see how they work. It is really simple:

* the main config file is `pcms-config.yaml` in the base directory. It stores all the important information to run your site.
* each page in `site/` has a `page.json` config file: This defines a "page", and is therefore available as URL route directly.
* The page folder also contains additional content / template files
* the `index` property defines the page index file as well as the page type:
  `"index": "index.html"` means you create an HTML file in `index.html`,
  while `"index": "index.md"` means you create an Markdown file in `index.md`.

## Step 4: Edit, stop, restart

Until today, pcms does not notice if you make changes in config files or at the page structure.
So for now, if you insert new pages or change config entries (even in `page.json`),
you have to restart pcms: Just `CTRL-c` it and start it again.

## Step 5: ... and beyond!

Please have a look at the other chapters for a more detailed description on how to configure
and use `pcms`.



