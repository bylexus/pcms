---
shortTitle: "pcms doc"
template: page-template.html
mainClass: home
iconCls: "fas fa-home"
metaTags:
  - name: "keywords"
    content: "pcms,cms"
  - name: "description"
    content": "Documentation for the pcms system, the programmer's cms"
---
# Welcome to pcms <small>the Programmer's Content management System</small>

CMS systems are great. They allow the user to produce content without caring about the underlying technology or frameworks.
But as for all frameworks, they have their limits.

I was at some point always limited by CMS systems, and I don't want a CMS that is in my way of doing things.

For me, a CMS is too restrictive. I don't fear writing HTML and program code. I am a developer, at last, so I feel more
comfortable writing code in an editor than clicking in a UI.

This is the idea behind **pcms**, the Programmer's CMS: A clutter-free, code-centric, simple CMS to deliver (simple) web sites. For people who 
love to code, and just want to produce content as they want, not fiddling with or around the CMS.

This is the documentation of **pcms**. Be aware that you need to know the following architectures / concepts in order to use
both pcms and this documentation:

* You love (reading and writing) code.
* You have a natural aversion of UIs.
* You know **HTML**, **CSS** and know how to use them
* You have heard of **Markdown**
* You don't say "Bless you!" if you hear someone saying "JSON" or "YAML"!
* If you want to dive deeper: You know [The GO Programming Language](https://go.dev/): pcms is
  written in GO, and it *may* be that you need to tweak a few things. At least you may want to
  compile it for yourself.

If you can answer "Yes, piece of cake!" to those question, then **pcms** is for you.

## Who's using it

* First of all: me, myself and I.
* This site itself is driven by pcms
* My personal site: https://alexi.ch/

## Why?

I am a programmer and I love fiddling with languages, tools, the web and stuff.
My own goal is to create as much as possible by myself. So this is first of all
an approach to build my own web engine. Because I am currently learning GO,
pcms is also written in GO (while it was written in NodeJS/JavaScript before ALREADY).

## Table of contents

* <a href="{{webroot("/")}}">[home]</a>
{% for child in variables.toc %}* <a href="{{webroot(child.destRelDir)}}/">{{child.title}}
{% endfor %}

