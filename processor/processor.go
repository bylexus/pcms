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

type ProcessingFileInfo struct {
	// file paths:
	// start / top path of the source folder
	rootSourceDir string
	// absolute path of the actual source file
	absSourcePath string
	// absolute path of the actual source file
	absSourceDir string
	// file path of the actual source file relative to the rootSourceDir
	relSourcePath string
	// dir path of the actual source file relative to the rootSourceDir
	relSourceDir string
	// relative path from the actual source file back to the rootSourceDir
	relSourceRoot string

	// start / top path of the destination folder
	rootDestDir string
	// absolute path of the actual destination file
	absDestPath string
	// absolute path of the actual destination file
	absDestDir string
	// file path of the actual destination file relative to the rootDestDir
	relDestPath string
	// dir path of the actual destination file relative to the rootSourceDir
	relDestDir string
	// relative path from the actual dest file back to the rootSourceDir
	relDestRoot string

	// web paths:
	// the webroot prefix, "/" by default
	webroot string
	// relative (to webroot) web path to the actual output file
	relWebPath string
	// relative (to webroot) web path to the actual output file's folder
	relWebDir string
	// relative path from the actual file back to the webroot
	relWebPathToRoot string
	// absolute web path of the actual file, including the webroot, starting always with "/"
	absWebPath string
	// absolute web path of the actual file's dir, including the webroot, starting always with "/"
	absWebDir string
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
