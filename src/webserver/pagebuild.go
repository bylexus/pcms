package webserver

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"path/filepath"

	"alexi.ch/pcms/src/logging"
	"alexi.ch/pcms/src/model"
)

// The PageBuilder's responsibility is to
// create a Page Map (a map of routes to page objects)
// from the filesystem structure.
// The PageMap represents a map between an URL route (e.g. '/projects') and a (parent) page.
type PageBuilder struct {
	pageMap *model.PageMap
	logger  *logging.Logger
}

// Creates a new PageBuilder struct, initializing the Page Map memory
func NewPageBuilder(logger *logging.Logger) PageBuilder {
	pm := model.NewPageMap()
	return PageBuilder{
		pageMap: &pm,
		logger:  logger,
	}
}

// Creates a page tree by walking the file system and creating a
// page map struct.
// The page map struct is a map of route => Page, while each Page object
// contains its children, forming a page tree.
func CreatePageTree(config *model.Config, logger *logging.Logger) *model.PageMap {
	pageBuilder := NewPageBuilder(logger)

	err := pageBuilder.ExaminePageDir(config.Site.Path, config)
	if err != nil {
		logger.Fatal(err.Error())
	}

	// add theme route manually:
	pageBuilder.AddPage(config.Site.Webroot+"/theme", &model.Page{
		Type:  model.PAGE_TYPE_THEME,
		Route: config.Site.Webroot + "/theme",
		Dir:   config.Site.ThemePath,
	})

	pageBuilder.BuildPageTree()
	return pageBuilder.GetPageMap()
}

func (pb *PageBuilder) BuildPageTree() {
	pb.pageMap.BuildPageTree()
}

func (pb *PageBuilder) AddPage(route string, page *model.Page) {
	pb.pageMap.PagesByRoute[route] = page
}

func (pb *PageBuilder) GetPageMap() *model.PageMap {
	return pb.pageMap
}

func (pb *PageBuilder) createPage(route string, dir string) (*model.Page, error) {
	page := model.Page{
		Route: route,
		Dir:   dir,
	}
	pageJson, err := ioutil.ReadFile(page.PageJsonPath())
	if err != nil {
		pb.logger.Error(err.Error())
		return nil, err
	}
	pageMeta := make(map[string]interface{})
	err = json.Unmarshal(pageJson, &pageMeta)
	if err != nil {
		pb.logger.Error(err.Error())
		return nil, err
	}
	page.Metadata = pageMeta
	page.ExtractKnownMetadata(pageMeta)

	return &page, nil
}

func (pb *PageBuilder) ExaminePageDir(rootDir string, config *model.Config) error {
	var err error
	rootDir, err = filepath.Abs(rootDir)

	if err != nil {
		return err
	}

	err = filepath.Walk(rootDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		path, err = filepath.Rel(rootDir, path)
		if err != nil {
			return filepath.SkipDir
		}
		if filepath.Base(path) == "page.json" {
			route := pb.createRouteFromRelPath(filepath.Dir(path), config)
			dir := filepath.Join(rootDir, filepath.Dir(path))
			pb.logger.Debug("Creating Page Route: %s", route)
			page, err2 := pb.createPage(route, dir)
			if page != nil && err2 == nil {
				pb.AddPage(route, page)
			}
		}

		return nil
	})
	return err
}

/**
 * Creates a valid page route from a given site directory / file:
 */
func (pb *PageBuilder) createRouteFromRelPath(relPath string, config *model.Config) string {
	if relPath == "." {
		relPath = ""
	}
	route := "/" + relPath
	if route == "/" {
		route = ""
	}
	// we prepend the webroot from the config:
	route = config.Site.Webroot + route
	if route == "" {
		route = "/"
	}
	return route
}
