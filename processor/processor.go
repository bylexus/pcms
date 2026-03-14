package processor

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"alexi.ch/pcms/lib"
	"alexi.ch/pcms/model"
	"alexi.ch/pcms/stdlib"
	"github.com/flosch/pongo2/v4"
	"gopkg.in/yaml.v3"
)

type Processor interface {
	Name() string
	ProcessFile(sourceFile string, config model.Config) (destFile string, err error)
}

type PageInfo struct {
	// the actual page record from the index
	ActPage model.IndexedPage `yaml:"-"`

	// file paths:
	// start / top path of the source folder
	RootSourceDir string `yaml:"rootSourceDir"`
	// absolute path of the actual source file
	AbsSourcePath string `yaml:"absSourcePath"`
	// absolute path of the actual source file
	AbsSourceDir string `yaml:"absSourceDir"`
	// file path of the actual source file relative to the RootSourceDir
	RelSourcePath string `yaml:"relSourcePath"`
	// dir path of the actual source file relative to the RootSourceDir
	RelSourceDir string `yaml:"relSourceDir"`
	// relative path from the actual source file back to the RootSourceDir
	RelSourceRoot string `yaml:"relSourceRoot"`

	// start / top path of the destination folder
	RootDestDir string `yaml:"rootDestDir"`
	// absolute path of the actual destination file
	AbsDestPath string `yaml:"absDestPath"`
	// absolute path of the actual destination file
	AbsDestDir string `yaml:"absDestDir"`
	// file path of the actual destination file relative to the RootDestDir
	RelDestPath string `yaml:"relDestPath"`
	// dir path of the actual destination file relative to the RootSourceDir
	RelDestDir string `yaml:"relDestDir"`
	// relative path from the actual dest file back to the RootSourceDir
	RelDestRoot string `yaml:"relDestRoot"`

	// web paths:
	// the Webroot prefix, "/" by default
	Webroot string `yaml:"webroot"`
	// relative (to Webroot) web path to the actual output file
	RelWebPath string `yaml:"relWebPath"`
	// relative (to Webroot) web path to the actual output file's folder
	RelWebDir string `yaml:"relWebDir"`
	// relative path from the actual file back to the Webroot
	RelWebPathToRoot string `yaml:"relWebPathToRoot"`
	// absolute web path of the actual file, including the Webroot, starting always with "/"
	AbsWebPath string `yaml:"absWebPath"`
	// absolute web path of the actual file's dir, including the Webroot, starting always with "/"
	AbsWebDir string `yaml:"absWebDir"`
}

func (p PageInfo) GetStdObject() (map[string]interface{}, error) {
	yamlStr, err := yaml.Marshal(p)
	if err != nil {
		return nil, err
	}
	obj := make(map[string]interface{})
	err = yaml.Unmarshal(yamlStr, &obj)
	if err != nil {
		return nil, err
	}
	return obj, err
}

func GetProcessor(sourceFile string, config model.Config) Processor {
	fileExt := filepath.Ext(sourceFile)
	switch strings.ToLower(fileExt) {
	case ".html":
		return HtmlProcessor{}
	case ".md":
		return MdProcessor{}
	default:
		return RawProcessor{}
	}
}

func BuildPageTemplateVariables(route string, indexFile string, config model.Config, page model.IndexedPage) (PageInfo, error) {
	result := PageInfo{}
	result.ActPage = page
	result.RootSourceDir = config.SourcePath
	result.RootDestDir = config.Server.CacheDir
	result.Webroot = config.Server.Prefix

	routeDir := strings.TrimPrefix(path.Clean(route), "/")
	if routeDir == "." {
		routeDir = ""
	}

	sourceRelPath := indexFile
	if routeDir != "" {
		sourceRelPath = filepath.Join(filepath.FromSlash(routeDir), indexFile)
	}

	result.AbsSourcePath = filepath.Join(result.RootSourceDir, sourceRelPath)
	result.AbsSourceDir = filepath.Dir(result.AbsSourcePath)

	var err error
	result.RelSourcePath, err = filepath.Rel(config.SourcePath, result.AbsSourcePath)
	if err != nil {
		return result, err
	}
	result.RelSourceDir, err = filepath.Rel(config.SourcePath, result.AbsSourceDir)
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

	destRelPath := filepath.Join(filepath.FromSlash(routeDir), "index.html")
	if routeDir == "" {
		destRelPath = "index.html"
	}
	result.AbsDestPath = filepath.Join(result.RootDestDir, destRelPath)
	result.AbsDestDir = filepath.Dir(result.AbsDestPath)
	result.RelDestPath, err = filepath.Rel(result.RootDestDir, result.AbsDestPath)
	if err != nil {
		return result, err
	}
	result.RelDestDir, err = filepath.Rel(result.RootDestDir, result.AbsDestDir)
	if err != nil {
		return result, err
	}
	if result.RelDestDir == "" {
		result.RelDestDir = "."
	}
	result.RelDestRoot, err = filepath.Rel(result.AbsDestDir, result.RootDestDir)
	if err != nil {
		return result, err
	}
	if result.RelDestRoot == "" {
		result.RelDestRoot = "."
	}

	result.RelWebPath = filepath.ToSlash(result.RelDestPath)
	result.RelWebDir = filepath.ToSlash(result.RelDestDir)
	result.RelWebPathToRoot = filepath.ToSlash(result.RelDestRoot)
	result.AbsWebPath = path.Clean(path.Join("/", result.Webroot, result.RelWebPath))
	result.AbsWebDir = path.Clean(path.Join("/", result.Webroot, result.RelWebDir))

	return result, nil
}

// Takes multiple string maps, and merges them.
// later map entries override previous ones.
func mergeStringMaps(maps ...map[string]interface{}) map[string]interface{} {
	resultMap := make(map[string]interface{})
	for _, m := range maps {
		for k, v := range m {
			resultMap[k] = v
		}
	}
	return resultMap
}

func AbsUrl(relPath string, Webroot string) string {
	return path.Clean(path.Join("/", Webroot, relPath))
}

func prepareTemplateContext(sourceFile string, config model.Config, fileInfo PageInfo, yamlFrontMatter stdlib.YamlFrontMatter) (pongo2.Context, error) {

	// combined variables object:
	variables := collectPageVariables(sourceFile, config, yamlFrontMatter)

	// convert paths object to an anonymous, lower-cased map:
	pathObj, err := fileInfo.GetStdObject()
	if err != nil {
		return nil, err
	}

	dbh, err := lib.GetDBH()
	if err != nil {
		return nil, err
	}
	childPages, err := dbh.GetChildPages(fileInfo.ActPage.Route)
	if err != nil {
		return nil, err
	}
	childFiles, err := dbh.GetChildFiles(fileInfo.ActPage.Route)
	if err != nil {
		return nil, err
	}

	var context = pongo2.Context{
		"page": fileInfo.ActPage,

		"childPages": childPages,
		"childFiles": childFiles,

		// contains the combined variables
		"variables": variables,
		// several file path variants for the actual file:
		"paths": pathObj,
		// creates an absolute, webroot-based url from a relative url
		"webroot": func(relPath string) string {
			return AbsUrl(relPath, fileInfo.Webroot)
		},
		// helper function to check a string for a prefix
		"startsWith": strings.HasPrefix,
		// helper function to check a string for a postfix
		"endsWith": strings.HasSuffix,
	}
	return context, nil
}

func collectPageVariables(sourceFile string, config model.Config, yamlFrontMatter stdlib.YamlFrontMatter) stdlib.YamlFrontMatter {
	// The variables content is prepared in the follwing priority (higher prio overrides lower prio).
	// 4. frontmatter content
	// 3. variables.yaml file in the actual source file's directory
	// 2. variables.yaml files further up the directory tree, until the root source dir is reached
	// 1. global variables from pcms-config.yaml
	variables := make(stdlib.YamlFrontMatter)

	// merge pcms-config variables:
	variables = mergeStringMaps(variables, config.Variables)

	// find and combine all variables.yaml files in the hierarchy:
	variableFiles := make([]string, 0)
	for actPath := filepath.Dir(sourceFile); strings.HasPrefix(actPath, config.SourcePath); actPath = filepath.Dir(actPath) {
		yamlFile := filepath.Join(actPath, "variables.yaml")
		fileInfo, err := os.Stat(yamlFile)
		if err != nil {
			continue
		}
		if !fileInfo.IsDir() {
			variableFiles = append(variableFiles, yamlFile)
		}
	}
	// process the collected variables.yaml file backward (top file first):
	for i := len(variableFiles) - 1; i >= 0; i-- {
		yamlFile := variableFiles[i]
		fileContent, err := os.ReadFile(yamlFile)
		if err != nil {
			continue
		}

		// parse yaml file
		variablesObj := make(stdlib.YamlFrontMatter)
		err = yaml.Unmarshal(fileContent, &variablesObj)
		if err != nil {
			fmt.Printf("ERROR while reading yaml from %s: %s\n", yamlFile, err)
			continue
		}
		variables = mergeStringMaps(variables, variablesObj)
	}

	// merge frontmatter variables:
	variables = mergeStringMaps(variables, yamlFrontMatter)

	return variables
}
