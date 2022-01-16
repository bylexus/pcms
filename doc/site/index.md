# Welcome to pcms <small>the Programmer's Content management System</small>

CMS systems are great. They allow the user to produce content without caring about the underlying technology or frameworks.
But as for all frameworks, they have their limits.

I was at some point always limited by CMS systems, and I don't want a CMS that is in my way of doing things.

For me, a CMS is too restrictive. I don't fear writing HTML and program code. I am a developer, at last, so I feel more
comfortable writing code in an editor than clicking in a UI.

This is the idea behind **pcms**, the Programmer's CMS: A clutter-free, code-centric, simple CMS to deliver (simple) web sites. For people who 
love to code, and also producing content, not fiddling with the CMS.

This is the documentation of **pcms**. Be aware that you need to know the following architectures / concepts in order to use
both pcms and this documentation:

* You love (reading and writing) code.
* You have a natural aversion of UIs.
* You know **HTML**, **CSS** and know how to use them
* You have heard of **Markdown**
* If you want to dive deeper: You know [The GO Programming Language](https://go.dev/): pcms is
  written in GO, and it *may* be that you need to tweak a few things.

If you can answer "Yes, piece of cake!" to those question, then **pcms** is for you.

## Who's using it

* First of all: me, myself and I.
* This site itself is driven by pcms
* My personal site: https://alexi.ch/

## Why?

I am a programmer and I love fiddling with languages, tools, the web and stuff.
My own goal is to create as much as possible by myself. So this is first of all
an approach to build my own web engine. Because I aam currently learning GO,
pcms is also written in GO (while it was written in NodeJS/JavaScript before ALREADY).

## Table of contents

* <a href="{{base}}/">[home]</a>
{% for child in rootPage.Children %}
{% if child.Metadata.shortTitle %}
* <a href="{{child.Route}}">{{child.Metadata.shortTitle}}</a>
{% endif %}
{% endfor %}

