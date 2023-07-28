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

type HtmlProcessor struct {
}

func (p HtmlProcessor) Name() string {
	return "html"
}

func (p HtmlProcessor) ProcessFile(sourceFile string, config model.Config) (destFile string, err error) {
	// processorConfig := config.Processors.Html

	relSourcePath, err := filepath.Rel(config.SourcePath, sourceFile)
	if err != nil {
		return "", err
	}

	relSourceDir, err := filepath.Rel(config.SourcePath, filepath.Dir(sourceFile))
	if err != nil {
		return "", err
	}

	// calc outfile path and create dest directory
	outFile := path.Join(config.DestPath, relSourcePath)
	outDir := filepath.Dir(outFile)
	err = os.MkdirAll(outDir, fs.ModeDir|0777)
	if err != nil {
		return "", err
	}

	relDestPath, err := filepath.Rel(config.DestPath, outFile)
	if err != nil {
		return "", err
	}

	relDestDir, err := filepath.Rel(config.DestPath, outDir)
	if err != nil {
		return "", err
	}

	relDestRoot, err := filepath.Rel(outDir, config.DestPath)
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
	yamlFrontMatter, sourceString := stdlib.ExtractYamlFrontMatter(sourceString)

	// Merge frontmatter variables with global config vars:
	variables := make(map[string]interface{})
	for k, v := range config.Variables {
		variables[k] = v
	}
	for k, v := range yamlFrontMatter {
		variables[k] = v
	}

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
		"sourceRelDir": relSourceDir,
		// relative file path of the actual file to the sourceRootDir
		"sourceRelPath": relSourcePath,
		// full path to the source file
		"sourceFullPath": sourceFile,

		// file path to the root destination dir:
		"destRootDir": config.DestPath,
		// relative dir of the actual destination file to the destRootDir
		"destRelDir": relDestDir,
		// relative file path of the actual file to the destRootDir
		"destRelPath": relDestPath,
		// full path to the destination file
		"destFullPath": outFile,

		"base": relDestRoot,
	}

	outWriter, err := os.Create(outFile)
	if err != nil {
		return "", err
	}
	defer outWriter.Close()

	err = tpl.ExecuteWriter(context, outWriter)
	if err != nil {
		return "", err
	}

	return outFile, nil
}
