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

	// Merge frontmatter variables with global config vars:
	variables := mergeStringMaps(config.Variables, yamlFrontMatter)

	// process template to output file
	context := pongo2.Context{
		// combined config + yaml preamble variables:
		"variables": variables,
		// several file path variants for the actual file:
		"paths": filePaths,
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

	outWriter, err := os.Create(filePaths.absDestDir)
	if err != nil {
		return "", err
	}
	defer outWriter.Close()

	err = tpl.ExecuteWriter(context, outWriter)
	if err != nil {
		return "", err
	}

	return filePaths.absDestPath, nil
}

func (p MdProcessor) prepareFilePaths(sourceFile string, config model.Config) (ProcessingFileInfo, error) {
	var err error = nil
	result := ProcessingFileInfo{}
	result.rootSourceDir = config.SourcePath
	result.rootDestDir = config.DestPath
	result.webroot = path.Join("/", config.Server.Prefix)
	result.absSourcePath = sourceFile
	result.absSourceDir = filepath.Dir(result.absSourcePath)

	// path to the input source file, relative to the base source dir:
	result.relSourcePath, err = filepath.Rel(config.SourcePath, sourceFile)
	if err != nil {
		return result, err
	}

	// path to the input source file's directory, relative to the base source dir:
	result.relSourceDir, err = filepath.Rel(config.SourcePath, filepath.Dir(sourceFile))
	if err != nil {
		return result, err
	}
	result.relSourceRoot, err = filepath.Rel(result.absSourceDir, config.SourcePath)

	// calc outfile path and create dest directory
	outFile := filepath.Join(config.DestPath, result.relSourcePath)
	outBase := strings.Replace(filepath.Base(outFile), ".md", ".html", 1)
	// full output file path's dir:
	result.absDestDir = filepath.Dir(outFile)
	// full output file path:
	result.absDestPath = filepath.Join(result.absDestDir, outBase)
	err = os.MkdirAll(result.absDestDir, fs.ModeDir|0777)
	if err != nil {
		return result, err
	}

	// path to the output destionation file, relative to the base destination dir:
	result.relDestPath, err = filepath.Rel(config.DestPath, result.absDestPath)
	if err != nil {
		return result, err
	}

	// path to the output destionation file's directory, relative to the base destination dir:
	result.relDestDir, err = filepath.Rel(config.DestPath, result.absDestDir)
	if err != nil {
		return result, err
	}
	if result.relDestDir == "" {
		result.relDestDir = "."
	}
	result.relWebPath = filepath.ToSlash(result.relDestPath)
	result.absWebPath = path.Join("/", config.Server.Prefix, result.relDestPath)

	// relative path to the destination root folder:
	result.relDestRoot, err = filepath.Rel(result.absDestDir, config.DestPath)
	if err != nil {
		return result, err
	}
	if result.relDestRoot == "" {
		result.relDestRoot = "."
	}

	result.relWebDir = filepath.ToSlash(result.relDestDir)
	result.absWebDir = path.Join("/", config.Server.Prefix, result.relWebDir)
	result.relWebPathToRoot = filepath.ToSlash(result.relDestRoot)
	return result, nil
}
