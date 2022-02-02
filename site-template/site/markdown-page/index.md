# Markdown Demo Page

[<i class="fas fa-home"></i> Back to home]({{base}}/)

Your actual Route: {{base}}{{page.Route}}

Access to pongo2 syntax, e.g. actual time: {% now "02.01.2006 15:04:05" %}

<pre>
Page config (page.json):
{{page.Metadata | stringformat:"%#v"}}
</pre>

Some static content:

* Relative addressed image: <img src="sunset.webp" class="img-fluid" />
* Absolute addressed image: <img src="{{page.Route}}/sunset2.webp" class="img-fluid" />
* Image from theme: <img src="/theme/static/sunset.webp" />
