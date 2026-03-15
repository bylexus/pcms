package processor

import (
	"fmt"
	"io/fs"

	"alexi.ch/pcms/model"
	"alexi.ch/pcms/stdlib"
	"github.com/flosch/pongo2/v4"
	"github.com/russross/blackfriday/v2"
)

/*
The MdProcessor processes Markdown (.md) files to HTML.

 1. The input md is processed as pongo2 template,
    including yaml frontmatter support (see example below)
 2. the resulting processed Markdown is converted to HTML
 3. then it is injected to a template as the `content` variable. The template
    needs to be defined in the YAML frontmatter, as `template` key.

Example:

	---
	# yaml frontmatter: can contain arbitary data, available via the page.Metadata template object:
	title: My Site
	mainClass: my-site
	# define template to inject markdown as `content` variable:
	template: base-markdown.html
	---
	# some Markdown

	hello, Markdown!
*/
type MdProcessor struct {
}

func (p MdProcessor) RenderFileForServe(siteFS fs.FS, sourceFSPath string, sourceFile string, config model.Config, filePaths PageInfo) ([]byte, error) {
	sourceBytes, err := fs.ReadFile(siteFS, sourceFSPath)
	if err != nil {
		return nil, fmt.Errorf("read markdown source %s: %w", sourceFSPath, err)
	}

	return p.render(sourceFile, string(sourceBytes), config, filePaths)
}

func (p MdProcessor) render(sourceFile string, sourceString string, config model.Config, filePaths PageInfo) ([]byte, error) {
	// Extract yaml frontmatter:
	yamlFrontMatter, sourceString, err := stdlib.ExtractYamlFrontMatter(sourceString)
	if err != nil {
		return nil, err
	}

	context, err := prepareTemplateContext(config, filePaths)
	if err != nil {
		return nil, err
	}

	// now, we need to process the Markdown source as template:
	mdTemplate, err := pongo2.FromString(sourceString)
	if err != nil {
		return nil, err
	}

	sourceString, err = mdTemplate.Execute(context)
	if err != nil {
		return nil, err
	}
	// now, convert filled markdown to html:
	result := blackfriday.Run([]byte(sourceString), blackfriday.WithExtensions(
		blackfriday.AutoHeadingIDs|blackfriday.Autolink|blackfriday.CommonExtensions|blackfriday.Footnotes,
	))
	htmlString := string(result[:])
	context.Update(pongo2.Context{"content": htmlString})

	// Wrap processed markdown in an HTML template:
	// For markdown files, we need a 'template' file to embed the md content.
	// The template file must be defined as 'template' front matter variable.
	// If not set, we just use a very simple content.
	// The markdown content is injected as 'content' variable.
	var tpl *pongo2.Template
	template, ok := yamlFrontMatter["template"]
	if ok {
		tpl, err = pongo2.FromFile(template.(string))
		if err != nil {
			return nil, err
		}
	} else {
		tpl, err = pongo2.FromString("{{ content | safe }}")
		if err != nil {
			return nil, err
		}
	}

	out, err := tpl.Execute(context)
	if err != nil {
		return nil, err
	}

	return []byte(out), nil
}

