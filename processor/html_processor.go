package processor

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"alexi.ch/pcms/model"
	"alexi.ch/pcms/stdlib"
	"github.com/flosch/pongo2/v4"
)

/*
The HtmlProcessor processes .html files

The input HTML is processed as pongo2 template.
It also supports a YAML front matter:

	---
	# yaml frontmatter: can contain arbitary data, available in the `variables` pongo template variable:
	title: My Site
	mainClass: my-site
	---
	{% extends base.tpl.html %}
	<p>some html</p>
*/
type HtmlProcessor struct {
}

func (p HtmlProcessor) Name() string {
	return "html"
}

func (p HtmlProcessor) ProcessFile(sourceFile string, config model.Config) (destFile string, err error) {
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

	// create template from input file
	tpl, err := pongo2.FromString(sourceString)
	if err != nil {
		return "", err
	}

	context, err := prepareTemplateContext(sourceFile, config, filePaths, yamlFrontMatter)
	if err != nil {
		return "", err
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

func (p HtmlProcessor) prepareFilePaths(sourceFile string, config model.Config) (ProcessingFileInfo, error) {
	var err error = nil
	result := ProcessingFileInfo{}
	result.RootSourceDir = config.SourcePath
	result.RootDestDir = config.DestPath
	result.Webroot = config.Server.Prefix
	result.AbsSourcePath = sourceFile
	result.AbsSourceDir = filepath.Dir(result.AbsSourcePath)

	result.RelSourcePath, err = filepath.Rel(config.SourcePath, sourceFile)
	if err != nil {
		return result, err
	}

	result.RelSourceDir, err = filepath.Rel(config.SourcePath, filepath.Dir(sourceFile))
	if err != nil {
		return result, err
	}
	if result.RelSourceDir == "" {
		result.RelSourceDir = "."
	}
	result.RelSourceRoot, err = filepath.Rel(result.AbsSourceDir, config.SourcePath)
	if err != nil {
		return result, err
	}

	// calc destination paths and create dest directory
	result.AbsDestPath = filepath.Join(config.DestPath, result.RelSourcePath)
	result.AbsDestDir = filepath.Dir(result.AbsDestPath)
	err = os.MkdirAll(result.AbsDestDir, fs.ModeDir|0777)
	if err != nil {
		return result, err
	}

	result.RelDestPath, err = filepath.Rel(config.DestPath, result.AbsDestPath)
	if err != nil {
		return result, err
	}
	result.RelWebPath = filepath.ToSlash(result.RelDestPath)
	result.AbsWebPath = path.Join("/", config.Server.Prefix, result.RelDestPath)

	result.RelDestDir, err = filepath.Rel(config.DestPath, result.AbsDestDir)
	if err != nil {
		return result, err
	}
	if result.RelDestDir == "" {
		result.RelDestDir = "."
	}
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
