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
* Support for restricted pages using basic authentication

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

For more information, [check the official documentation](https://pcms.alexi.ch).

