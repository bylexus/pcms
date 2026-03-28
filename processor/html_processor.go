package processor

import (
	"fmt"
	"io/fs"

	"alexi.ch/pcms/model"
	"alexi.ch/pcms/stdlib"
	"github.com/flosch/pongo2/v6"
)

/*
The HtmlProcessor processes .html files

The input HTML is processed as pongo2 template.
It also supports a YAML front matter:

	---
	# yaml frontmatter: can contain arbitary data, available via the page.Metadata template object:
	title: My Site
	mainClass: my-site
	---
	{% extends base.tpl.html %}
	<p>some html</p>
*/
type HtmlProcessor struct {
}

func (p HtmlProcessor) RenderFileForServe(siteFS fs.FS, sourceFSPath string, sourceFile string, config model.Config, pageInfo PageInfo) ([]byte, error) {
	sourceBytes, err := fs.ReadFile(siteFS, sourceFSPath)
	if err != nil {
		return nil, fmt.Errorf("read html source %s: %w", sourceFSPath, err)
	}

	return p.render(sourceFile, string(sourceBytes), config, pageInfo)
}

func (p HtmlProcessor) render(sourceFile string, sourceString string, config model.Config, pageInfo PageInfo) ([]byte, error) {
	// Extract yaml frontmatter (strip it from source):
	_, sourceString, err := stdlib.ExtractYamlFrontMatter(sourceString)
	if err != nil {
		return nil, err
	}

	// create template from input file
	tpl, err := pongo2.FromString(sourceString)
	if err != nil {
		return nil, err
	}

	context, err := prepareTemplateContext(config, pageInfo)
	if err != nil {
		return nil, err
	}

	out, err := tpl.Execute(context)
	if err != nil {
		return nil, err
	}

	return []byte(out), nil
}

