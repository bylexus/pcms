package processor

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"alexi.ch/pcms/lib"
	"alexi.ch/pcms/model"
	"github.com/flosch/pongo2/v4"
)

// Processor is the interface for processors that can render a page's index file.
type Processor interface {
	RenderFileForServe(siteFS fs.FS, sourceFSPath string, sourceFile string, config model.Config, filePaths PageInfo) ([]byte, error)
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


func BuildPageTemplateVariables(route string, indexFile string, config model.Config, page model.IndexedPage) (PageInfo, error) {
	result := PageInfo{}
	result.ActPage = page
	result.RootSourceDir = config.SourcePath
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

	if routeDir == "" {
		result.RelWebPath = "index.html"
		result.RelWebDir = "."
		result.RelWebPathToRoot = "."
	} else {
		result.RelWebPath = routeDir + "/index.html"
		result.RelWebDir = routeDir
		result.RelWebPathToRoot, err = filepath.Rel(filepath.FromSlash(routeDir), ".")
		if err != nil {
			return result, err
		}
		result.RelWebPathToRoot = filepath.ToSlash(result.RelWebPathToRoot)
	}
	result.AbsWebPath = path.Clean(path.Join("/", result.Webroot, result.RelWebPath))
	result.AbsWebDir = path.Clean(path.Join("/", result.Webroot, result.RelWebDir))

	return result, nil
}

// GetProcessor returns the appropriate PageRenderer for the given index file,
// based on its file extension.
func GetProcessor(indexFile string) (Processor, error) {
	switch strings.ToLower(path.Ext(indexFile)) {
	case ".html":
		return HtmlProcessor{}, nil
	case ".md":
		return MdProcessor{}, nil
	default:
		return nil, fmt.Errorf("unsupported page index file type: %s", indexFile)
	}
}

func AbsUrl(relPath string, Webroot string) string {
	return path.Clean(path.Join("/", Webroot, relPath))
}

func prepareTemplateContext(config model.Config, fileInfo PageInfo) (pongo2.Context, error) {
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
		"Page": fileInfo.ActPage,

		"ChildPages": childPages,
		"ChildFiles": childFiles,

		"Config": config,

		// several file path variants for the actual file:
		"Paths": fileInfo,
		// creates an absolute, webroot-based url from a relative url
		"Webroot": func(relPath string) string {
			return AbsUrl(relPath, fileInfo.Webroot)
		},
		// helper function to check a string for a prefix
		"StartsWith": strings.HasPrefix,
		// helper function to check a string for a postfix
		"EndsWith": strings.HasSuffix,
	}
	return context, nil
}
