---
title: "requirements"
shortTitle: "Requirements"
template: "page-template.html"
metaTags: 
  - name: "keywords"
    content: "requirements,pcms,cms"
  - name: "description"
    content: "Requirements of running pcms"
---
# Requirements

In order to use and run pcms, you need to fullfil the following requirements:

* If you want to compile for yourself: [ GO Lang >= 1.20 ]( https://go.dev/dl/ )
* The `pcms` binary. Check the [Releases Page on Github](https://github.com/bylexus/pcms/releases) for pre-built binaries.
* If you want to compile `scss` to `css`, you need the [dart sass](https://sass-lang.com/dart-sass/) compiler binary

... Really, that's it?? Well, yes! That's all you need. No database, no fancy infrastructure, no air plane. Oh, yes, and some idea for your content, but this is your problem ;-)

There is also a Dockerfile for running pcms as docker service. So if you want to run it
in a Docker container, you need a container runtime (like [docker](https://www.docker.com/), or [podman](https://podman.io/)). Check the [Installation section]({{ webroot("/install_setup/")}}) for information on how to use pcms as container service.