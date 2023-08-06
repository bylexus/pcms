package processor

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

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
	# yaml frontmatter: can contain arbitary data, available in the `variables` pongo template variable:
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

func (p MdProcessor) Name() string {
	return "markdown"
}

func (p MdProcessor) ProcessFile(sourceFile string, config model.Config) (destFile string, err error) {
	filePaths, err := p.prepareFilePaths(sourceFile, config)
	if err != nil {
		return "", err
	}

	// read input file
	sourceBytes, err := os.ReadFile(sourceFile)
	if err != nil {
		return "", err
	}
	sourceString := string(sourceBytes[:])

	// Extract yaml frontmatter:
	yamlFrontMatter, sourceString, err := stdlib.ExtractYamlFrontMatter(sourceString)
	if err != nil {
		return "", err
	}

	context, err := prepareTemplateContext(sourceFile, config, filePaths, yamlFrontMatter)
	if err != nil {
		return "", err
	}

	// now, we need to process the Markdown source as template:
	mdTemplate, err := pongo2.FromString(sourceString)
	if err != nil {
		return "", err
	}

	sourceString, err = mdTemplate.Execute(context)
	if err != nil {
		return "", err
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
		// templatePath = join(config.TemplateDir, template.(string))
		tpl, err = pongo2.FromFile(template.(string))
		if err != nil {
			return "", err
		}
	} else {
		tpl, err = pongo2.FromString("{{ content | safe }}")
		if err != nil {
			return "", err
		}
	}

	outWriter, err := os.Create(filePaths.AbsDestPath)
	if err != nil {
		return "", err
	}
	defer outWriter.Close()

	err = tpl.ExecuteWriter(context, outWriter)
	if err != nil {
		return "", err
	}

	return filePaths.AbsDestPath, nil
}

func (p MdProcessor) prepareFilePaths(sourceFile string, config model.Config) (ProcessingFileInfo, error) {
	var err error = nil
	result := ProcessingFileInfo{}
	result.RootSourceDir = config.SourcePath
	result.RootDestDir = config.DestPath
	result.Webroot = config.Server.Prefix
	result.AbsSourcePath = sourceFile
	result.AbsSourceDir = filepath.Dir(result.AbsSourcePath)

	// path to the input source file, relative to the base source dir:
	result.RelSourcePath, err = filepath.Rel(config.SourcePath, sourceFile)
	if err != nil {
		return result, err
	}

	// path to the input source file's directory, relative to the base source dir:
	result.RelSourceDir, err = filepath.Rel(config.SourcePath, filepath.Dir(sourceFile))
	if err != nil {
		return result, err
	}
	result.RelSourceRoot, err = filepath.Rel(result.AbsSourceDir, config.SourcePath)
	if err != nil {
		return result, err
	}

	// calc outfile path and create dest directory
	outFile := filepath.Join(config.DestPath, result.RelSourcePath)
	outBase := strings.Replace(filepath.Base(outFile), ".md", ".html", 1)
	// full output file path's dir:
	result.AbsDestDir = filepath.Dir(outFile)
	// full output file path:
	result.AbsDestPath = filepath.Join(result.AbsDestDir, outBase)
	err = os.MkdirAll(result.AbsDestDir, fs.ModeDir|0777)
	if err != nil {
		return result, err
	}

	// path to the output destionation file, relative to the base destination dir:
	result.RelDestPath, err = filepath.Rel(config.DestPath, result.AbsDestPath)
	if err != nil {
		return result, err
	}

	// path to the output destionation file's directory, relative to the base destination dir:
	result.RelDestDir, err = filepath.Rel(config.DestPath, result.AbsDestDir)
	if err != nil {
		return result, err
	}
	if result.RelDestDir == "" {
		result.RelDestDir = "."
	}
	result.RelWebPath = filepath.ToSlash(result.RelDestPath)
	result.AbsWebPath = path.Join("/", config.Server.Prefix, result.RelDestPath)

	// relative path to the destination root folder:
	result.RelDestRoot, err = filepath.Rel(result.AbsDestDir, config.DestPath)
	if err != nil {
		return result, err
	}
	if result.RelDestRoot == "" {
		result.RelDestRoot = "."
	}

	result.RelWebDir = filepath.ToSlash(result.RelDestDir)
	result.AbsWebDir = path.Join("/", config.Server.Prefix, result.RelWebDir)
	result.RelWebPathToRoot = filepath.ToSlash(result.RelDestRoot)
	return result, nil
}
