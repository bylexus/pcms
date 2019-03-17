# Architecture

pcms is a [ NodeJS ](https://nodejs.org/) / [ ExpressJS ](https://expressjs.com/) mini-framework to deliver web pages.
The main goal is to deliver page content and metadata from templates and / or configuration/data files, all file-system-based.

In essence pcms is built upon the following infrastructure:

* a NodeJS / ExpressJS Server to handle all web requests and deliver dynamic and static content
* Several Express Middlewares that contains the pcms application logic to build routes and render pages
* a simple file-based site-structure, aka your content, that is examined / parsed / rendered by pcms

## Site structure and routes

* The **site** contents are stored as directory tree in the `site/` folder.
* Every directory within `site/` (including `site/ itself`) that contains a `page.json` file is considered a **page**, and answers to the corresponding URL route:
  for example, the directory `site/article/sunset/`, which contains a `page.json` file, matches the URL route `/article/sunset`.
* Sub-pages / files in non-pages folders are considered as route of a top page with URL tail realtive the top page:
  So all content belong to a top page. This can also be the root page.
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
    "order": 1,                                  // child order within the parent page. This can be everything, it will be ordered naturally.
    "title": "Home",                             // additional metadata, like page title, keywords etc. This info is available in the page templates.
    "keywords": ["php", "nodejs","programming"],
    // ..... more data you have available in your templates
}
```

## Page delivery

The page delivery works as follows:

1. The requested route is matched against the page tree: The longest-matching page in the tree is figured out.
2. The page is checked for its availability (enabled flag is true), and its legimity (not in the `preventDelivery` match array)
3. If the route and route tail matches static content, deliver it, end here.
4. If given, the preprocessor function is executed. This allows the site dev to process further data for the page
5. Finally, load the template and render it to the browser.

## Page types

The framework supports the following page types, determined **only** by the file ending of the `index` file entry in `page.json`:

* `html`: The framework looks for an HTML template named by the `index` key (e.g.: `{"index": "index.html"}`). The HTML file
    is read, processed by [Nunjucks](https://mozilla.github.io/nunjucks), and rendered back to the browser.
* `markdown`: The framework looks for a `.md`-Markdown template named by the `index` key (e.g.: `{"index": "index.md"}`). The Markdown file
    is read, processed **first** by [marked](https://marked.js.org/), then processed by [Nunjucks](https://mozilla.github.io/nunjucks), and rendered back to the browser.
* `json`: The framework looks for a `.json`-index named by the `index` key (e.g.: `{"index": "data.json"}`).
  It then reads the (HTML) template defined in the `template` key, and processes it by [Nunjucks](https://mozilla.github.io/nunjucks), and rendered back to the browser.
  This is meant for e.g. simple index / menu files, where all the data are structured json data.
* `js`: This is the most complex but also most flexible page type: The `index` property can point to a .js-File, which then is included by the framework and
   taken as an Express Router middleware: It is then in the responsibility of the js developer to process some kind of output.

## Page index / page cache

The whole page tree is built upon application start: The `site` folder is walked recursively, and each page (a folder with a `page.json` file) is
examined, a page node info is created.

This simple in-memory cache should be enough for smaller sites. If things get bigger, I will search for another solution. At the moment, [Yagni](https://www.martinfowler.com/bliki/Yagni.html).

It should be possible to re-scanning the page tree by sending a unix signal to the server process. How this is achieved has to be evaluated, still.

The page tree is also available in the page templates, to build menus, for example.

A future version of the framework will also include some kind of watch mechanism, for re-building the page tree at runtime. At the moment, this is not possible, so if your
page tree structure changes, you have to restart the server. Of course this can also be done by using something like [nodemon](https://nodemon.io/) or [pm2](https://pm2.io/runtime/).
