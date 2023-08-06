---
title: "Variables sample page"
shortTitle: "Variables"
template: "markdown.html"
iconCls: "fab fa-markdown"
mainClass: "markdown"
metaTags: 
  - name: "keywords"
    content": "pcms,cms"
  - name: "description"
    content": "Variables Sample Page"
---
# Variables demo page

Variables can be defined in the following locations:

- in `pcms-config.yaml`, in the `variables` object:

```yaml
variables:
  var1: value1
  var2: value2
```

- in `variables.yaml` files in the folder hierarchy (deeper file content override higher ones)
- in a yaml frontmatter in the file directly (as shown in this file):

```md
---
title: "Variables sample page"
template: "markdown.html"
mainClass: "markdown"
var1: value1
---
# Hello, Markdown

Access var: {{variables.var1}}
```

Defined variables in the actual context:

<pre>
{{variables | stringformat:"%#v"}}
</pre>