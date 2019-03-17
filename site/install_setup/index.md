# Installation / Setup of pcms-driven sites

pcms has two parts:

* a site generator that helps you starting a new site. It will create a skeleton site / structure for you so you don't have to start from scratch.
  You don't need to generate a skeleton site if you want to do it manually.
* The server part that serves a pcms-compatible site structure. This is yet another npm package which allows you to start / deliver the site.

This documentation shows you both parts: Setup a skeleton site and run the server.

## Install pcms

In your project directory, require `pcms` from npm. If you don't have a project directory yet, create one:

```sh
# Optional steps: setup a new site directory:
$ mkdir my-site
$ cd my-site
$ npm init
# install pcms:
$ npm install --save pcms
```

## Setup a skeleton site

If you don't want to start from scratch, generate a skeleton site:

```sh
# if you have npx (https://www.npmjs.com/package/npx) installed:
$ npx pcms-generate
# if you don't have npx:
$ node_modules/.bin/pcms-generate
```

This will generate a fully-working demo site into the local folder. Note that no existing files will be overwritten. You end with a folder structure like this:

```sh
.
├── package.json
├── server.js
├── site
│   ├── index.html
│   └── page.json
├── site-config.json
└── themes
    └── default
        ├── static
        │   └── css
        │       ├── main.css
        │       └── main.css.map
        └── templates
            ├── base.html
            ├── error.html
            └── markdown-template.md
```

* `server.js` is the entrypoint of your application. This is the script you can start with node right away. It sets up and starts the pcms-driven server.
* `site-config.json` is the site-wide configuration. It contains pcms-specific settings like server port as well as user-defined content which can be used in your site templates.
* `site/` is the folder where all your page content goes. Every pcms-visible page is a folder within `site` with a `page.json` file.
* `themes/` contains (the / several) layouts and styles for your site: It is the folder where your site-wide HTML and CSS layouts / files are stored and fetched. Each theme corresponds to
  the folder name within `themes/`. The actual theme can be set up in `site-config.json`.


## Site-wide configuration: `site-config.json`

The main config file is `site-config.json`. It servers the following purposes:

* It defines the folder where pcms looks for content: The folder where `site-config.json` is placed must contain the `themes` and `site` folder.
* It defines pcms configuration like server settings (port).
* It offers you a global configuration for additional data / configs that is available in all your templates.

An example `site-config.json`:

```json
{
    "title": "pcms documentation",
    "port": 3000,
    "webroot": "",
    "theme": "pcms-doc",
    "metaTags": [
        {"name": "author", "content": "Alexander Schenkel"}
    ]
}
```

There are only two properties you must define in order to run pcms:

* `port`: This is the TCP port the webserver listens to
* `theme`: The theme to use (a folder with the same name in `themes/`)

All the other config options can be chosen freely by the theme / site creator.

## Page config with`page.json`

Each (sub-)folder within `site/` that contains a `page.json` is considered a "page" by pcms, and is indexed in the site tree. So each page has its own `page.json`.
Other folders or sub-folders that exists are not indexed in the site tree, but can server as static content containers, belonging to an upper page.

An example `page.json` file:

```json
{
    "enabled": true,
    "index": "index.md",
    "template": "page-template.html",
    "preprocessor": "script.js",
    "shortTitle": "pcms doc",
    "iconCls": "fas fa-home",
    "mainClass": "home",
    "metaTags": [
        {"name": "keywords", "content": "pcms,cms"},
        {"name": "description", "content": "Documentation for the pcms system, the programmer's cms"}
    ]
}
```

Also here, only the pcms relevant configs must be set, all other configs are available in that page's templates / context.

* `enabled`: If false, the page including its sub-pages and static content are not delivered. You can use this flag to disable a page temporarily.
* `index`: The file that is read as content file. This can be html, Markdown, json or js, depending of the file ending pcms will act accordingly.
* `template`: for non-HTML index files, this html template file is read and processed by the template engine. The content from `index` is available as `content` template variable.
* `preprocessor`: A JS script that is executed before the page rendering happens. It has access to the actual page's context information and can enrich it with additional
  data. It must return either an object with additional data that is merged with the Page's context data, or a `Promise` that resolves that Object later.

## Page content

The `index` property of the `page.json` file defines the page content file. In the simplest form, this is a plain HTML file that is rendered via Nunjucks template engine.
The template engine injects a page context: This is an object with available template variables that can be used within the template.

The page context contains the following variables:

* site: This object contains the `site-config.json` entries.
* base: the configured webroot in `site-config.json`
* route: the actual page's route, without the index part (so e.g. /folder/to/page/)
* fullRoute: the actual's page full route, inluding the index part (e.g. /folder/to/page/index.html)
* page: The Page Node object, containing several information about the page, including the `page.json` entries (`page.pageConfig`)
* rootPage: The site's root node object. Same entries as in `page`, but for the site's root page.
* now: The actual date (JavaScript `Date` object)

All these variables can be used within the template. An example:

```html
{% raw %}<!-- index.html -->
{% extends "base.html" %}

<h1>{{page.pageConfig.title}}</h1>
<p>Hello. It is now {{now}}.</p>
<a href="{{rootPage.route}}">Back to home</a>

A relative link: <a href="{{route}}/sub/folder/picture.png">Image</a>
{% endraw %}
```

## Starting the site

The pre-generated server.js file is the main file you can run using nodejs:

```sh
$ DEBUG=server,pcms node server.js
```

The pre-configured `server.js` file can be modified to satisfy your needs. The initial script works as follows:


```javascript
const server = require('pcms');
const path = require('path');
const debug = require('debug')('server');

/**
 * Start the server, which is an expressjs app. You can get the express app instance if you
 * need to configure things in advance:
 *
 * const app = server.expressApp;
 */
server
    // give the full path to your site-config.json: The server needs it to gather information
    // about your directory origin:
    .start(path.join(__dirname, 'site-config.json'))
    .then(app => {
        debug(`Site is now serving at port: ${app.serverConfig.siteConfig.port}`);
    })
    .catch(err => {
        debug('ERROR: ', err);
    });
```

This is it. You can modify the server.js file as needed, as long as you start the server somewhen using
`server.start(path-to-site-config.json)`.

For more options on how to start the site see the <a href="{{base}}/running">Running</a> section.
