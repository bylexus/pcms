pcms - The Programmer's Content Management System
================================================

I don't need a fancy, UI-driven CMS. I don't WANT a  UI, and I don't want a CMS that is in my way of doing things.
A CMS is too restrictive. I don't fear writing HTML and program code. I am a developer, at last, so I feel more
comfortable writing code in an editor than clicking in a UI.

This is the idea behind **pcms**, the Programmer's CMS: A clutter-free, code-centric, simple CMS to deliver web sites. For people that
love to code, also while producing content.

Features
----------

* simple, fast, node-based page delivery system. No database needed, file-based only.
* serves HTML and markdown (which is rendered to HTML, too :-))
* Uses [Nunjucks](https://mozilla.github.io/nunjucks) for html/markdown files to create pages based on templates
* separates theme / layout from content

Getting Started
-----------------

### Create a new site

To start from scratch, you can generate a skeleton site from a template:

```
$ mkdir my-new-site
$ cd my-new-site
$ npm init
$ npm install --save pcms
$ node_modules/.bin/pcms-generate
# or, of you have npx:
$ npx pcms-generate
```
This will generate a fully-working demo site. Your content live in the `site/` folder.

### Start your server

You can start the site immediately:

```
$ DEBUG=server,pcms node server.js
```

Open a browser and browse to http://localhost:3000/

For more information, see below or read the official docs.

The Architecture
-----------------

pcms is a NodeJS / expressjs mini-framework to deliver web pages. The main goal is to deliver page content and metadata from templates
and / or configuration/data files, all file-system-based.

### URL and site structure

* The site contents are stored as directory tree in the `site/` folder.
* Every directory within `site/` that contains a `page.json` file is considered a "page", and answers to the corresponding URL route:
  for example, the directory `site/article/sunset/`, which contains a `page.json` file, matches the URL route `/article/sunset`.
* Every URL route matches either a page directory, or a real file (e.g. static content like images).
* Pages can be HTML, HTML templates, Markdown with an HTML template, JSON with an HTML template, or you can do whatever you want
  in server-side JS.

The `page.json` file is the metadata information for the specific page. It may look as follows:

```javascript
{
    "index": "index.html",                       // required. This defines the page type as well as the included template.
    "enabled": true,                             // can be used to disable a page, e.g. to stop it from being shown.
    "template": "template.html",                 // Only used by JSON data, this is the template used to process the JSON index data.
    "preprocessor": "prep.js",                   // A script that is executed before the page gets rendered, to deliver additional functionality.
    "title": "Home",                             // additional metadata, like page title, keywords etc. This info is available in the page templates.
    "keywords": ["php", "nodejs","programming"],
    "order": 1,
    // ..... more data you have available in your templates
}
```

### Page delivery

The page delivery works as follows:

1. The requested route is matched against the page tree: The longest-matching page in the tree is figured out.
2. The page is checked for its availability (enabled flag is true), and its legimity (not in the `preventDelivery` match array)
3. If the route and route tail matches static content, deliver it, end here.
4. If given, the preprocessor function is executed. This allows the site dev to process further data for the page
5. Finally, load the template and render it to the browser.

### Page types

The framework supports the following page types, determined **only** by the file ending of the `index` file entry in `page.json`:

* `html`: The framework looks for an HTML template named by the `index` key (e.g.: `{"index": "index.html"}`). The HTML file
    is read, processed by [Nunjucks](https://mozilla.github.io/nunjucks), and rendered back to the browser.
* `markdown`: The framework looks for a `.md`-Markdown template named by the `index` key (e.g.: `{"index": "index.md"}`). The Markdown file
    is read, processed **first** by [marked](https://marked.js.org/), then processed by [Nunjucks](https://mozilla.github.io/nunjucks), and rendered back to the browser.
* `json`: The framework looks for a `.json`-index named by the `index` key (e.g.: `{"index": "data.json"}`).
      It then reads the (HTML) template defined in the `template` key, and processes it by [Nunjucks](https://mozilla.github.io/nunjucks), and rendered back to the browser.
      This is meant for e.g. simple index / menu files, where all the data are structured json data.
* `js: This is the most complex but also most flexible page type: The `index` property can point to a .js-File, which then is included by the framework and
   taken as an Express Router middleware: It is then in the responsibility of the js developer to process some kind of output.

### Page index / page cache

The whole page tree is built upon application start: The `site` folder is walked recursively, and each page (a folder with a `page.json` file) is
examined, a page node info is created.

This simple in-memory cache should be enough for smaller sites. If things get bigger, I will search for another solution. At the moment, [Yagni](https://www.martinfowler.com/bliki/Yagni.html).

It should be possible to re-scanning the page tree by sending a unix signal to the server process. How this is achieved has to be evaluated, still.

The page tree is also available in the page templates, to build menus, for example.

A future version of the framework will also include some kind of watch mechanism, for re-building the page tree at runtime. At the moment, this is not possible, so if your
page tree structure changes, you have to restart the server.

Setup a new project
--------------------

Running the site
-------------------------

### Configuration

//TODO: @see site-config.json

### Run the server

//TODO: document. basically: node server.js
- .env file
- docker compose
- pm2:
  - prod: NODE_ENV=production npx pm2 start ecosystem.config.js --env production



TODO: Framework documentation
------------------------

### site-config.json

### Anatomy of a page

#### page.json

- preprocessing
- preventDelivery
- enabled

#### Page type: html

#### Page type: markdown

#### Page type: json

#### Page type: js

### Page templating, themes


(c) 2019 Alexander Schenkel
