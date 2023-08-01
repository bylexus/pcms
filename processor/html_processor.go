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
		// file path to the root source dir:
		"sourceRootDir": config.SourcePath,
		// relative dir of the actual source file to the sourceRootDir
		"sourceRelDir": filePaths.relSourceDir,
		// relative file path of the actual file to the sourceRootDir
		"sourceRelPath": filePaths.relSourcePath,
		// full path to the source file
		"sourceFullPath": sourceFile,

		// file path to the root destination dir:
		"destRootDir": config.DestPath,
		// relative dir of the actual destination file to the destRootDir
		"destRelDir": filePaths.relDestDir,
		// relative file path of the actual file to the destRootDir
		"destRelPath": filePaths.relDestPath,
		// full path to the destination file
		"destFullPath": filePaths.outFile,
		// full web path to the destination file:
		"destAbsPath": filePaths.absDestPath,
		// full web path to the destination file's dir:
		"destAbsDir": filePaths.absDestDir,

		// relative path to the web base dir, from the actual processed file:
		"base": filePaths.relDestRoot,
	}

	outWriter, err := os.Create(filePaths.outFile)
	if err != nil {
		return "", err
	}
	defer outWriter.Close()

	err = tpl.ExecuteWriter(context, outWriter)
	if err != nil {
		return "", err
	}

	return filePaths.outFile, nil
}

func (p HtmlProcessor) prepareFilePaths(sourceFile string, config model.Config) (processingFileInfo, error) {
	var err error = nil
	result := processingFileInfo{
		relSourcePath: "",
		relSourceDir:  "",
		outFile:       "",
		outDir:        "",
		relDestPath:   "",
		relDestDir:    "",
		relDestRoot:   "",
		absDestPath:   "",
		absDestDir:    "",
	}
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

	// calc outfile path and create dest directory
	result.outFile = path.Join(config.DestPath, result.relSourcePath)
	result.outDir = filepath.Dir(result.outFile)
	err = os.MkdirAll(result.outDir, fs.ModeDir|0777)
	if err != nil {
		return result, err
	}

	result.relDestPath, err = filepath.Rel(config.DestPath, result.outFile)
	if err != nil {
		return result, err
	}
	result.absDestPath = path.Join("/", config.Server.Prefix, result.relDestPath)

	result.relDestDir, err = filepath.Rel(config.DestPath, result.outDir)
	if err != nil {
		return result, err
	}
	if result.relDestDir == "" {
		result.relDestDir = "."
	}
	result.absDestDir = path.Join("/", config.Server.Prefix, result.relDestDir)

	result.relDestRoot, err = filepath.Rel(result.outDir, config.DestPath)
	if err != nil {
		return result, err
	}
	if result.relDestRoot == "" {
		result.relDestRoot = "."
	}
	return result, nil
}
