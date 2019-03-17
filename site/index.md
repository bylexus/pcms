# Welcome to pcms <small>the Programmer's Content management System</small>

CMS systems are great. They allow the user to produce content without caring about the underlying technology or frameworkds.
But as for all frameworks, they have their limits.

I was at some point always limited by CMS systems, and I don't want a CMS that is in my way of doing things.

For me, a CMS is too restrictive. I don't fear writing HTML and program code. I am a developer, at last, so I feel more
comfortable writing code in an editor than clicking in a UI.

This is the idea behind **pcms**, the Programmer's CMS: A clutter-free, code-centric, simple CMS to deliver web sites. For people that
love to code, also while producing content.

This is the documentation of **pcms**. Be aware that you need to know the following architectures / concepts in order to use
both pcms and this documentation:

* You love (reading and writing) code.
* You have a natural aversion of UIs.
* You know **NodeJS** and **JavaScript**
* You know the **npm ecosystem**
* You know **HTML**, **CSS** and know how to use them
* You have heard of **Markdown**

If you can answer "Yes, piece of cake!" to those question, then **pcms** is for you.

## Who's using it

* This site itself is driven by pcms
* My personal site: https://alexi.ch/

## Table of contents

* <a href="{{base}}/">[home]</a>
{% for child in rootPage.childPages %}
* <a href="{{child.route}}">{{child.pageConfig.shortTitle}}</a>
{% endfor %}

