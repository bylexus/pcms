---
title: "Markdown sample page"
shortTitle: "Markdown"
template: "markdown.html"
iconCls: "fab fa-markdown"
mainClass: "markdown"
metaTags: 
  - name: "keywords"
    content": "pcms,cms"
  - name: "description"
    content": "Markdown sample page"
---
# Markdown Demo Page

[<i class="fas fa-home"></i> Back to home]({{base}}/)

Your actual Route: {{base}}{{destRelPath}}

Access to pongo2 syntax, e.g. actual time: {% now "02.01.2006 15:04:05" %}

<pre>
Page variables:
{{variables | stringformat:"%#v"}}
</pre>

Some static content:

* Relative addressed image: ![relative addressd image](sunset.webp)
* Relative addressed from webroot: ![relative addressed from webroot](/{{destRelDir}}/sunset.webp)
* Absolute addressed image: ![absolute addressed image]({{destAbsDir}}/sunset2.webp)
