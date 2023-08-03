package processor

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"alexi.ch/pcms/model"
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
