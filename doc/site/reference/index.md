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

- [Generating a site](#generating-a-site)
- [Directory Structure of a pcms project](#directory-structure-of-a-pcms-project)
- [pcms-config.yaml](#pcms-configyaml)
- [The `site` folder](#the-site-folder)
  - [using HTML files with templates](#using-html-files-with-templates)
  - [using Markdown files with templates](#using-markdown-files-with-templates)
  - [available template variables](#available-template-variables)
  - [YAML front matter variables](#yaml-front-matter-variables)
  - [hierarchical template variables with variables.yaml files](#hierarchical-template-variables-with-variablesyaml-files)
- [pcms cli reference](#pcms-cli-reference)


## Generating a site

pcms can generate a starter skeleton for you. It has a built-in skeleton which you can generate using the following commands:

```bash
$ pcms init /path/to/pcms-project
```

You have now a working site which you can start immediately:

```bash
$ cd /path/to/pcms-project
$ pcms serve
```

## Directory Structure of a pcms project

```sh
root folder
├── pcms-config.yaml          # The config file for the site
├── site/                     # The site dir contains the page content, and listens to the "/" route
│   ├── index.html            # additional page content / templates
│   ├── variables.yaml        # A YAML file containing variables. Inherited to all (sub-)pages
│   └── ... more files/folder # add your source files/folders as needed
└── templates                 # pongo2 templates for your html / markdown content
    ├── base.html             # a pongo2 template, e.g. a base html template
    └── ... more files/folder # add as many templates as you need, and reference them from your site files
```

- `pcms-config.yaml` is the configuration file for your site. It contains all the settings and global variables.
- `site/` is the folder where all your page content goes. If you reference pongo2 templates within your files, they are searched from the `templates/` folder.
- `templates/` contains your pongo2 templates (if you need any).

## pcms-config.yaml

This is the reference of the `pcms-config.yaml` config file.

```yaml
# pcms-config.yaml: This is the pcms configuration file. It configures the whole system.
server:
  # listen address. This is an ip-address:port number pair, or a partial address: "localhost:3000", ":3000", "127.0.0.1", "0.0.0.0:3000"
  listen: ":3000"
  # watch: if true, the source folder is watched for file changes, and a rebuild
  # of changed / new files is triggered on the fly.
  watch: true
  # webroot prefix: the content is served under this webroot prefix (e.g. "/site"). Defaults to "". The webroot can be accessed by the `variables.webroot` variable in templates. 
  prefix: ""
  # Logging configuration: there are 2 diffenrent logs written:
  logging:
    # The access log: Logs all web access, like a webserver would.
    # Define the file (or STDOUT/STDERR), and the format (TBD).
    access:
      file: STDOUT
      # not yet implemented:
      format: ""
    # The error, or system log. Define the file (or STDOUT/STDERR), and the max log level:
    error:
      file: STDERR
      level: DEBUG
# The source folder of the site, containing the unbuilt site templates. Relative to
# the config file dir:
source: "site"
# The destination folder of the site, containing the built static site. 
# Note that this dir will be COMPLETELY EMPTIED on build.
# Relative to the config file dir:
dest: "build"
# Global variables for the pongo2 templates, available as "variables" pongo2 template variable as a map:
variables:
  siteTitle: My Site Title
  siteMetaTags:
    - name: foo
      content: bar
    - name: moo
      content: baz
# Where to look for pongo2 templates when inheriting / defining a template file:
# relative to the config file dir:
template_dir: templates
# Regular expression to exclude files/folders from the build process completely:
exclude_patterns:
  # Ignore .* files:
  - "/\\..*"
  # Ignore all files in the /restricted folder:
  - "^/restricted/?.*"
processors:
  scss:
    # path to the dart-scss binary, if you want to convert scss files to css:
    sass_bin: "/usr/bin/sass"
```

## The `site` folder

All your content resides under the `site` folder. Each folder (including the main folder `site`) is recognized as a `page`
as soon as it contains a `page.json` file. The folder structure corresponds directly to the page's web route:
`site/` is the webr root route `/` (or whatever your webroot is set to), `site/about/me` corresponds to the `/about/me` route.

### using HTML files with templates

`.html` files in the `site/` folder are processed as [pongo2](https://github.com/flosch/pongo2) templates. You can use the full power of this
django-like template engine. For example, you can access the pre-defined and own variables in your HTML:

```html
{% verbatim %}<div>Path of this file: {{ paths.relWebPath }}</div>{% endverbatim %}
```

You can also inherit from a base template: Templates are searched within the `templates/` folder. As an example, define a base template,
then inherit from this base template in your site folder:

```html
{% verbatim %}<!-- Base template in templates/base.html: -->
<!doctype html>
<html lang="en">
    <head>
        <title>{%if variables.title %}{{variables.title}}{% endif%}</title>
        {% for meta in variables.metaTags %}
        <meta name="{{meta.name}}" content="{{meta.content}}" />
        {% endfor %}
    </head>
    <body>
        <main id="content">
            {% block content %}{% endblock %}
        </main>
    </body>
</html> {% endverbatim %}
```

```html
{% verbatim %}<!-- partial site in /site/index.html: -->
{% extends "base.html" %}
{% block content %}
<h1>Welcome!</h1>

<p>This is your site content.</p>
{% endblock %}{% endverbatim %}
```

### using Markdown files with templates

`*.md` files are processed as a pongo2 template, rendered to HTML and optionally embedded in a HTML template. The processed Markdown content is available in the `content` variable in your templates.
The template to be used can be defined in a YAML front matter.

An example Markdown file which is rendered to a HTML template:

```text
---
# site/index.md, with a YAML front matter, defining the template:
template: base-markdown.html
title: "Welcome"
---
# Welcome!

This **Markdown** partial file has the relative path: {% verbatim %}{{paths.relWebPath}}{% endverbatim %}.
```

```html
{% verbatim %}<!-- Base template in templates/base-markdown.html: -->
<!doctype html>
<html lang="en">
    <head>
        <title>{%if variables.title %}{{variables.title}}{% endif%}</title>
    </head>
    <body>
        <main id="content">
          {{ content | safe }}
        </main>
    </body>
</html> {% endverbatim %}
```

### available template variables

pcms defines the following variables which you can use in your templates:

* `variables`: This is a map of the combined variables from `pcms-config.yaml::variables`, `variables.yaml` files and the actual file's YAML Front matter.<br>
  Example usage in a template:<br>
  {% verbatim %}`Title: {{ variables.title|default:"My Site" }}`{% endverbatim %}
* `paths`: a map of several path strings for the actual file:
  * `paths.rootSourceDir`: The full file path to the used `site` folder
  * `paths.absSourcePath`: The full file path to the actual source file
  * `paths.absSourceDir`: The full file path to the actual source file's directory
  * `paths.relSourcePath`: The relative file path of the actual file to the root source dir.
  * `paths.relSourceDir`: The relative file path of the actual file's directory to the root source dir.
  * `paths.relSourceRoot`: The relative file path of the actual file back to the `rootSourceDir` (e.g. `../../..`)
  * `paths.rootDestDir`: The full file path to the used `build` (output) folder
  * `paths.absDestPath`: The full file path to the actual dest file
  * `paths.absDestDir`: The full file path to the actual dest file's directory
  * `paths.relDestPath`: The relative file path of the actual file to the root sest dir.
  * `paths.relDestDir`: The relative file path of the actual file's directory to the root dest dir.
  * `paths.relDestRoot`: The relative file path of the actual file back to the `rootSourceDir` (e.g. `../../..`)
  * `paths.webroot`: 	The Webroot prefix, "/" by default
  * `paths.relWebPath`: relative (to Webroot) web path to the actual output file
  * `paths.relWebDir`: relative (to Webroot) web path to the actual output file's folder
  * `paths.relWebPathToRoot`: relative path from the actual output file back to the Webroot
  * `paths.absWebPath`: absolute web path of the actual output file, including the Webroot, starting always with "/"
  * `paths.absWebDir`: absolute web path of the actual output file's dir, including the Webroot, starting always with "/"
* `webroot(string)`: Function to generate an absolute web path of a given relative path.<br>
  Example usage, assuming the web prefix is set to `/mysite`:<br>
  `webroot('rel/path/to/file')` => `/mysite/rel/path/to/file`
* `startsWith(str: string, prefix: string)`: Checks if the given string `str` starts with `prefix`. Same as `strings.HasPrefix`. Useful if you want to highlight navigation markers.<br>
  Example:<br>
  {% verbatim %}`<a href="/foo" class="{% if startsWith(paths.absWebDir, webroot('/foo')) %}active{% endif%}">Nav to foo</a>`{% endverbatim %}
* `endsWith(str: string, suffix: string)`: Same as `startsWith()`, but checks if the given string `str` ends with `suffix`. Same as `strings.HasSuffix`. Useful if you want to highlight navigation markers.

### YAML front matter variables

You can define a YAML Frontmatter variable map in your `.html` and `.md` templates. This is useful to define variables which can be used in base templates.

The front matter YAML must be at the beginning of the file, encapsulated in `---` separators.

Example: You want to output a page-specific title tag, which is defined in the base template:

```html
{% verbatim %}<!-- Base template in templates/base.html: -->
<!doctype html>
<html lang="en">
    <head>
      <!-- Output page-specific title: -->
        <title>{{ variables.title|default:"My Page" }}</title>
    </head>
    <body>
        <main id="content">
            {% block content %}{% endblock %}
        </main>
    </body>
</html>{% endverbatim %}
```

```html
{% verbatim %}---
# YAML front matter for partial site in site/index.html:
title: "My page-specific title"
---
{% extends "base.html" %}
{% block content %}
<h1>Welcome!</h1>
...
{% endblock %}{% endverbatim %}
```

This generates the following output page:

```html
{% verbatim %}<!-- build/index.html: -->
<!doctype html>
<html lang="en">
    <head>
      <!-- Output page-specific title: -->
      <title>My page-specific title</title>
    </head>
    <body>
        <main id="content">
<h1>Welcome!</h1>
...
        </main>
    </body>
</html>{% endverbatim %}
```

### hierarchical template variables with variables.yaml files 

You can also define template variables outside the YAML front matter, in `variables.yaml` files. `variables.yaml` files must contain a variables map,
and will be processed in the hierarchy of the actual file up to the site root folder.

An example:

Your site folder contains the following files:

```text
└── site/
    ├── index.html
    ├── variables.yaml
    └── subpage/
        ├── index.md
        └── variables.yaml
```

Say we process the file `site/subpage/index.md`, which has the following content:

```text
---
var3: "Ice man"
var4: "Bruce Wayne"
---
Some page content
```

The file `site/variables.yaml` contains site-wide variables:

```yaml
title: "Site title"
var1: "Top"
var2: "Batman"
```

The file `site/subpage/variables.yaml` contains the following variables:


```yaml
title: "Subpage"
var1: "subfolder"
var3: "Robin"
```

If the processor processes the file `site/subpage/index.md`, the `variables` template variable contain the merged variables from all files AND the front matter (deeper variables override top ones):

```text
variables:
  title: Subpage
  var1: subfolder
  var2: Batman
  var3: Ice man
  var4: Bruce Wayne
```

## pcms cli reference

