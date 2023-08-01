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

	// Merge frontmatter variables with global config vars:
	variables := mergeStringMaps(config.Variables, yamlFrontMatter)

	// create template from input file
	tpl, err := pongo2.FromString(sourceString)
	if err != nil {
		return "", err
	}

	// process template to output file
	context := pongo2.Context{
		// combined config + yaml preamble variables:
		"variables": variables,
		// several file path variants for the actual file:
		"paths": filePaths,
		"webroot": func(relPath string) string {
			return AbsUrl(relPath, filePaths.webroot)
		},
	}

	outWriter, err := os.Create(filePaths.absDestPath)
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

func (p HtmlProcessor) prepareFilePaths(sourceFile string, config model.Config) (ProcessingFileInfo, error) {
	var err error = nil
	result := ProcessingFileInfo{}
	result.rootSourceDir = config.SourcePath
	result.rootDestDir = config.DestPath
	result.webroot = config.Server.Prefix
	result.absSourcePath = sourceFile
	result.absSourceDir = filepath.Dir(result.absSourcePath)

	result.relSourcePath, err = filepath.Rel(config.SourcePath, sourceFile)
	if err != nil {
		return result, err
	}

	result.relSourceDir, err = filepath.Rel(config.SourcePath, filepath.Dir(sourceFile))
	if err != nil {
		return result, err
	}
	if result.relSourceDir == "" {
		result.relSourceDir = "."
	}
	result.relSourceRoot, err = filepath.Rel(result.absSourceDir, config.SourcePath)

	// calc destination paths and create dest directory
	result.absDestPath = filepath.Join(config.DestPath, result.relSourcePath)
	result.absDestDir = filepath.Dir(result.absDestPath)
	err = os.MkdirAll(result.absDestDir, fs.ModeDir|0777)
	if err != nil {
		return result, err
	}

	result.relDestPath, err = filepath.Rel(config.DestPath, result.absDestPath)
	if err != nil {
		return result, err
	}
	result.relWebPath = filepath.ToSlash(result.relDestPath)
	result.absWebPath = path.Join("/", config.Server.Prefix, result.relDestPath)

	result.relDestDir, err = filepath.Rel(config.DestPath, result.absDestDir)
	if err != nil {
		return result, err
	}
	if result.relDestDir == "" {
		result.relDestDir = "."
	}
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
