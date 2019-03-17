# Quick start - for the impatient

This chapter guides you very quickly from installation to a fully-running site. For more information, please refer to the rest of the documentation.

## Step 1 - init your project

Open a terminal, create a folder and init npm:

```sh
$ mkdir my-site
$ cd my-site
$ npm init
```

**Note:** You can add additional npm packages as you want, pcms does not care about other modules / tools.

## Step 2 - Install pcms

```sh
$ npm install --save pcms
```

## Step 3 - Generate a site skeleton

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

Your page __content__ goes to the `site` folder, while your site design (theme) goes to the `themes/default` folder. The used theme is configured in `site-config.json`, and can be adapted as needed:
The theme name is just the folder name in `themes`.

Now you're already set up.

## Step 4: Start up your site!

The simplest way is to just fire `node` to start your server. The `DEBUG=` env variable defines which logs you see: `server` and `pcms` are the ones that you might be interested.

```sh
$ DEBUG=server,pcms node server.js
```

That's it! If everything goes well, you can now connect to http://localhost:3000/ to see a skeleton site running.
