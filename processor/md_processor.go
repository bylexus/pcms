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

type MdProcessor struct {
}

func (p MdProcessor) Name() string {
	return "markdown"
}

func (p MdProcessor) ProcessFile(sourceFile string, config model.Config) (destFile string, err error) {

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
	outBase := strings.Replace(filepath.Base(outFile), ".md", ".html", 1)
	outDir := filepath.Dir(outFile)
	outFile = path.Join(outDir, outBase)
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
	yamlFrontMatter, sourceString, err := stdlib.ExtractYamlFrontMatter(sourceString)
	if err != nil {
		return "", err
	}

	// Merge frontmatter variables with global config vars:
	variables := make(map[string]interface{})
	for k, v := range config.Variables {
		variables[k] = v
	}
	for k, v := range yamlFrontMatter {
		variables[k] = v
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
