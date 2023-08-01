package processor

import (
	"path/filepath"
	"regexp"
	"strings"

	"alexi.ch/pcms/model"
)

type Processor interface {
	Name() string
	ProcessFile(sourceFile string, config model.Config) (destFile string, err error)
}

type processingFileInfo struct {
	relSourcePath string
	relSourceDir  string
	outFile       string
	outDir        string
	relDestPath   string
	relDestDir    string
	relDestRoot   string
	absDestPath   string
	absDestDir    string
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
