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

[<i class="fas fa-home"></i> Back to home]({{webroot("/")}})

Your actual Route: {{paths.absWebPath}}

Access to pongo2 syntax, e.g. actual time: {% now "02.01.2006 15:04:05" %}

<pre>
Page variables:
{{variables | stringformat:"%#v"}}
Page paths:
{{paths | stringformat:"%#v"}}
</pre>

Some static content:

* Relative addressed image: ./sunset.webp<br>
  ![relative addressd image](./sunset.webp)
* Relative addressed from webroot: {{paths.relWebPathToRoot}}/{{paths.relWebDir}}/sunset.webp<br>
  ![relative addressed from webroot]({{paths.relWebPathToRoot}}/{{paths.relWebDir}}/sunset.webp)
* Absolute addressed image: {{paths.absWebDir}}/sunset2.webp<br>
  ![absolute addressed image]({{paths.absWebDir}}/sunset2.webp)
