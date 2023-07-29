---
title: "reference"
shortTitle: "Reference"
template: "page-template.html"
metaTags: 
  - name: "keywords"
    content: "reference,pcms,cms"
  - name: "description"
    content: "pcms reference"
---
# Reference documentation

## Generating a site

pcms comes with a simple command line tool, `pcms-generate` to create an initial site skeleton. Creating an empty site skeleton is very simple:

```bash
$ mkdir my-project
$ cd my-project
$ npm init
$ npm install --save pcms
$ npx pcms-generate
```

You have now a working site which you can start immediately:

```bash
$ DEBUG=server,pcms node server.js
```

## Directory Structure

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

- `server.js` is the entrypoint of your application. This is the script you can start with node right away. It sets up and starts the pcms-driven server.
- `site-config.json` is the site-wide configuration. It contains pcms-specific settings like server port as well as user-defined content which can be used in your site templates.
- `site/` is the folder where all your page content goes. Every pcms-visible page is a folder within `site` with a `page.json` file.
- `themes/` contains (the / several) layouts and styles for your site: It is the folder where your site-wide HTML and CSS layouts / files are stored and fetched. Each theme corresponds to
  the folder name within `themes/`. The actual theme can be set up in `site-config.json`.

## site-config.json

This is the server-wide configuration file. A full and annotated example is shown below (note that JSON does not support comments, so this file is not working as-is).

```js
{
    // The TCP port the servier is listening for web requests
    "port": 3000,

    // The theme to use. This corresponds to a folder in the `themes/` dir.
    "theme": "default",

    // Define users for restricted page access. Each entry is a username / bcrypt password pair.
    "users": {
        "alex": "$2b$10$ORq5bwxW1oxQxra5s0V16uEzPTXlW9VF2/2jLxzsm8sOeaWnbTQTO"
    },

    // Define the starting web path if your site does not respond to the root folder:
    "webroot": "",

    // Additional data that can be used in your templates. Some examples below:
    "title": "My Site",
    "metaTags": [
        {"name": "author", "content": "Here comes your name"}
    ]
}
```

## The `site` folder

All your content resides under the `site` folder. Each folder (including the main folder `site`) is recognized as a `page`
as soon as it contains a `page.json` file. The folder structure corresponds directly to the page's web route:
`site/` is the webr root route `/` (or whatever your webroot is set to), `site/about/me` corresponds to the `/about/me` route.

### Anatomy of a page

A page consists of:

- a `page.json` configuration file
- a template file in html or Markdown
- optional JSON data for json-driven pages
- optional JavaScripts for JS-driven pages
- optional static content like images, css etc.

A Page can contain child pages with the same structure: As soon as it contains a `page.json`, it ist threatened as its own page,
otherwise it is just static sub-content.

### page.json

The `page.json` file is the page-wide configuration file for a single page. A full example is shown below:

```js
{
// Set to false if you want to disable a page. If true, the page including sub-resources / pages will
// return a 404 not found error. Useful for e.g. temporarily disable a page. Defaults to true.
"enabled": true,

// For HTML pages, this is the HTML template that is rendered. It can / should inherit from
// the theme's Nunjucks base page (see Themes below)
"index": "index.html",

// You can also write your pages in Markdown:
"index": "index.md",

// ... or you deliver some data as JSON:
"index": "data.json",

// If you are adventurous, you can do all by yourselve by defining a JS script:
"index": "index.js",

// If you have a Markdown or JSON page, you need to define a render-template which renders the Markdown / JSON:
"template": "page-template.html",

// The preprocessor script is called before the page rendering takes place, and allows you to
// execute arbitary JS code to e.g. fetch data from a database or the like:
"preprocessor": "preprocessor.js",

// Prevent sub-resources from delivery: Relative paths to static files that must not be delivered directly:
"preventDelivery": [
    "passwords.txt",
    "folder2/secrets.txt"
],

// Restrict access to a page using basic auth:
// select users from `site-config.json`, or just require a valid user:
"requiredUsers": [
    // allowed users from site-config.json
    "user1","user2"
    // or any valid user if successfully authenticated:
    "valid-user"
],

// Additional data that are available in the template: This can be anything, from page titles, meta data
// menu ordering etc. and is available as `page.pageConfig` in the page template:
"order": 6,
"title": "reference",
"shortTitle": "Reference",
"metaTags": [
    {"name": "keywords", "content": "reference,pcms,cms"},
    {"name": "description", "content": "pcms reference"}
],
// ......
}
```

The most important property is the `index` property: It defines 1) the type of the page (by looking at the ending) as well as the main entrypoint / template / data
file for the page.

Valid page types are:

- `.html`: A HTML template that should inherit from your theme's base page Nunjucks template
- `.md`: A Markdown page that can contain Page data Nunjucks variables. Needs the `template` property to render the content.
- `.json`: A JSON datafile. Needs the `template` property to render the content. Useful if you have e.g. tabular data to render.
- `.js`: A nodjs script if you want to have the full freedom: It must export a expressjs middleware function that is called from the framework.

### page context / template variables

The page templates are rendered using the [ Nunjucks template engine ](https://mozilla.github.io/nunjucks/). The template gets a bunch of variables assigned
that can be used in the template.

For example, you can access the actual Page's route and arbitary `page.json` data in your template:

```html
{% verbatim %}
<html>
  <head>
    <title>{{page.pageConfig.title}}</title>
  </head>
  <body>
    <p>The actual page's route: {{route}}</p>
  </body>
</html>
{% endverbatim %}
```

#### `site`: The site-config content

The `site` variable contains the contents of `site-config.json`.

Example:

```html
{% verbatim %}
<div>This server runs on port: {{site.port}}</div>
{% endverbatim %}
```

#### `base`: The webroot property

The `base` variable contains just the contents of the site-config's `webroot` property. This is important if you
need absolute-addressed links, for example.

Example:

```html
{% verbatim %}
<div>
  See my fancy image: <img src="{{base}}/my-image.jpg" />
</div>
{% endverbatim %}
```

#### `route` and `fullRoute`: The actual page's / resource's route

`route` contains the page's route, e.g. this page's `route` variable resolves to `{{route}}`.

The `fullRoute` variable contains the route to the requested sub-content, e.g. an image. This is only useful in e.g. preprocessor scripts
or js-based pages.

#### `page`: Information about the actual page

This variable contains all page-related information:

- `page.urlTail`: Access to the page-relative url tail, if requesting a sub-resource (Useful only in a pre-processor or verbatim JS script)
- `page.depth`: Distance to the root page (root page distance = 0, `/page` = 1, `/page/subpage` = 2)
- `page.fullPath`: Full file path to this page's folder
- `page.routePart`: The page's last part of the route: e.g. for the route `/page/subpage`, `page.routePart` contains `subpage`
- `page.route`: The page's route
- `page.pageConfig`: The content of the page's `page.json` file
- `page.pageIndex`: The index entry of the page (e.g. `index.html`)
- `page.childPages`: An array with child page objects, in the same structure as the actual `page` property
- `page.childPagesByRoute`: page objects by route, the key representing the route. The values are also `page` objects.
- `page.template`: The template file used
- `page.type`: The page type: `html`, `markdown`, `json`, `js`

### Page type: html

HTML pages contain a `page.json` and a Nunjucks HTML template file. Most probably the HTML template will inherit from the theme's base page template.

An example:

```json
// page.json:
{
  "title": "HTML Demo",
  "shortTitle": "HTML",
  "index": "index.html"
}
```

```html
{% verbatim %}
<!-- index.html -->
{% extends "base.html" %}

{% block content %}
<h1>{{page.pageConfig.title}}</h1>
<p><a href="{{base}}/">Back to home</a></p>
<p>Your actual Route: {{base}}{{route}}</page>
{% endblock %}
{% endverbatim %}
```

### Page type: markdown

A markdown page consists of the page content in a Markdown file, and an HTML template where the content will be rendered in.
The most common scenario is that you have a common HTML template for all your Markdown files (or at least for all topic-related files).

The Markdown file will also be prepared by Nunjucks, so you also have all page variables at hand.

In the HTML template, the variable `content` contains the rendered HTML from the Markdown file.

An example:

```json
// page.json:
{
  "title": "My Markdown Page",
  "index": "index.md",
  "template": "page-template.html"
}
```

```md
{% verbatim %}
Hello, {{page.pageConfig.title}}!

---

This is a **Markdown** page. Your route is: {{route}}
{% endverbatim %}
```

```html
{% verbatim %}
<!-- page-template.html -->
{% extends "base.html" %} 
{% block content %}
<div id="page_content">
  <!-- here, the markdown content will be rendered, `safe` for not encoding HTML tags: -->
  {{ content | safe }}
</div>

{% endblock %}
{% endverbatim %}
```

### Page type: json

The `json` type is meant for displaying list or tabular data, like table of contens, structured information and the like. A `json` page consists
of a `page.json`, a json data file and a HTML template where the data can be rendered.

An example:

```json
// page.json:
{
  "title": "Enterprise Personell",
  "index": "data.json",
  "template": "page-template.html"
}
```

```json
// data.json:
{
  "records": [
    {
      "name": "Piccard",
      "surname": "Jean-Luc",
      "rank": "Captain"
    },
    {
      "name": "Riker",
      "surname": "William T.",
      "rank": "Commander"
    },
    {
      "name": "La Forge",
      "surname": "Geordi",
      "rank": "Lietenant Commander"
    }
  ],
  "totalCount": 2300
}
```

```html
{% verbatim %}
<!-- page-template.html -->
{% extends "base.html" %}
{% block content %}

<h1>{{page.pageConfig.title}}</h1>

<table>
  <thead>
    <tr>
      <th>Name</th>
      <th>Surname</th>
      <th>Rank</th>
    </tr>
  </thead>
  <tbody>
    <!-- `content` contains the JSON data: -->
    {% for character in content.records %}
    <tr>
      <td>{{character.name}}</td>
      <td>{{character.surname}}</td>
      <td>{{character.rank}}</td>
    </tr>
    {% endfor %}
  </tbody>
</table>
{% endblock %}
{% endverbatim %}
```

### Page type: js

If you need complete freedom, you can create a NodeJS script page: In essence, this is a JS function
like an expressjs middleware, which has the full control over the output.

Example:

```json
// page.json:
{
  "title": "JS Test Page",
  "shortTitle": "JTP",
  "index": "script.js"
}
```

```js
// script.js:
// export a function that takes the request, response and next() middleware helper:
module.exports = function process(req, res, next) {
  // the page context is available as req.context:
  res.send(`Hello, world! Title: ${req.context.page.pageConfig.title}`).end();
};
```

### Preprocessor script

Each page type can define a preprocessor script that is executed before any rendering occurs.
The preprocessor script must return a Promise that resolves additional data that is injected
into the page context.

This is useful if you want to fetch external data from a database or an external data source to
display on the page.

An example:

```json
// page.json
{
  "title": "Chuck Norris Joke",
  "index": "index.html",
  "preprocessor": "preprocessor.js"
}
```

```js
// preprocessor.js
// The preprocessor functon gets the page node and root node object:
module.exports = function(pageNode, rootNode) {
  // It must return a Promise that is resolved with additional data:
  return new Promise((resolve, reject) => {
    fetchChuckNorrisJoke().then(joke => {
      resolve({
        joke: joke,
        actTime: new Date()
      });
    });
  });
};
```

```html
{% verbatim %}
<!-- HTML page template -->
{% extends "base.html" %}
{% block content %}
<h1>{{page.pageConfig.title}}</h1>

<p>Actual time injected from preprocessor script: {{actTime}}</p>
<p>Your favorite joke: {{joke}}</p>
{% endblock %}
{% endverbatim %}
```

## Page templating, Themes

A Site has a site-wide theme which is the base for the page layout.
Each page in `pcms` is build from a Nunjucks HTML template that resides in the theme folder.

Most probably you want to create a theme for your site. A theme is a subfolder in the `theme` folder with
all the needed templates / resources. The folder structure of a theme must look as follows:

```
themes/
└── pcms-doc/
    ├── static/
    └── templates/
```

You need at least one `templates` folder: This is the folder where Nunjucks is looking for templates
if you extend a page template from it.

The `static` folder is where you can put theme-wide static contents like images, fonts, CSS files etc.
The static folder is available in your templates as `{% verbatim %}{{base}}/theme/static{% endverbatim %}`: This route is mapped to
the configured theme.

You can add any additional theme-wide resources in your theme folder. As an example, the theme of
this documentation looks as follows in full:

```
themes
└── pcms-doc
    ├── sass
    │   ├── _footer.scss
    │   ├── _header.scss
    │   ├── _home.scss
    │   └── main.scss
    ├── static
    │   └── css
    │       ├── main.css
    │       └── main.css.map
    └── templates
        ├── base.html
        ├── error.html
        └── markdown-template.md
```

The site theme can be configured in `site-config.json`:

```json
{
  "theme": "pcms-doc"
}
```

This is just the folder name of your theme.

Now you can define page templates as shown in the chapters above, by inheriting from a base theme:

```html
{% verbatim %}
<!-- page.html -->
<!-- inherit from a base theme in your theme: -->
{% extends "base.html" %}
{% block content %}

<!-- Route to the Theme's folder, for accessing theme resources: -->
<img src="{{base}}/theme/static/theme-image.jpg" />

{% endblock %}
{% endverbatim %}
```

## Restricted pages with HTTP Basic Auth

Access to certain pages can be restricted by standard Basic Auth mechanism. `pcms` supports
locally configured users which you can assign to pages.

To secure a page with Basic Auth, follow these steps:

### 1. Add the users to `site-config.json`:

In your `site-config.json`, define a `users` object that consists of username / bcrypt password pairs:

```js
{
    // Define users for restricted page access. Each entry is a username / bcrypt password pair.
    "users": {
        "alex": "$2b$10$ORq5bwxW1oxQxra5s0V16uEzPTXlW9VF2/2jLxzsm8sOeaWnbTQTO",
        "admin": "$2b$10$stx0pVSdyk0gkJDctKZBh.zhhXvVADM0Mvn0dFbhhJ04gnkYdoPQa"
    }
}
```

To generate a bcrypt password, use the `pcms-password` helper script:

```
$ npx pcms-generate "your secret password"
```

Enter this password to the config file.

### 2. Add a `requiredUsers` to your `page.json`

In the `page.json` file of the page you want to secure, add a `requiredUsers` array with
allowed users:

```json
{
    "requiredUsers": [
        // allowed users from site-config.json
        "user1","user2"
        // or any valid user if successfully authenticated:
        "valid-user"
    ]
}
```

The special entry `valid-users` can be used if just any authenticated user should be able to log in.
