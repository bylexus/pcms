package processor

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"alexi.ch/pcms/model"
	"alexi.ch/pcms/stdlib"
	"github.com/flosch/pongo2/v4"
	"gopkg.in/yaml.v3"
)

type Processor interface {
	Name() string
	ProcessFile(sourceFile string, config model.Config) (destFile string, err error)
}

type ProcessingFileInfo struct {
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

func (p ProcessingFileInfo) GetStdObject() (map[string]interface{}, error) {
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
	case ".scss":
		return ScssProcessor{}
	default:
		return RawProcessor{}
	}
}

// Checks if the given file matches a set of exclude regex patterns.
// The relative path within the source dir is used as input.
//
// Returns true and the matching pattern if the file name matches a exclude pattern.
func IsFileExcluded(filePath string, excludePatterns []string) (bool, string) {
	skipfiles := []string{"variables.yaml"}
	if stdlib.InSlice(&skipfiles, filepath.Base(filePath)) {
		return true, filepath.Base(filePath)
	}
	for _, pattern := range excludePatterns {
		r := regexp.MustCompile(pattern)
		if r.MatchString(filePath) {
			return true, pattern
		}
	}

	return false, ""
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

func prepareTemplateContext(sourceFile string, config model.Config, filePaths ProcessingFileInfo, yamlFrontMatter stdlib.YamlFrontMatter) (pongo2.Context, error) {

	// combined variables object:
	variables := collectPageVariables(sourceFile, config, yamlFrontMatter)

	// convert paths object to an anonymous, lower-cased map:
	pathObj, err := filePaths.GetStdObject()
	if err != nil {
		return nil, err
	}

	var context = pongo2.Context{
		// contains the combined variables
		"variables": variables,
		// several file path variants for the actual file:
		"paths": pathObj,
		// creates an absolute, webroot-based url from a relative url
		"webroot": func(relPath string) string {
			return AbsUrl(relPath, filePaths.Webroot)
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
