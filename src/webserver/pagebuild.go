package webserver

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"log"
	"path/filepath"

	"alexi.ch/pcms/src/model"
)

type PageBuilder struct {
	pages *model.PageMap
}

func NewPageBuilder() PageBuilder {
	return PageBuilder{
		pages: &model.PageMap{},
	}
}

func (pb *PageBuilder) BuildPageTree() {
	pb.pages.BuildPageTree()
}

func (pb *PageBuilder) AddPage(route string, page *model.Page) {
	(*pb.pages)[route] = page
}

func (pb *PageBuilder) GetPages() *model.PageMap {
	return pb.pages
}

func (pb *PageBuilder) createPage(route string, dir string) (*model.Page, error) {
	page := model.Page{
		Route: route,
		Dir:   dir,
	}
	pageJson, err := ioutil.ReadFile(page.PageJsonPath())
	if err != nil {
		log.Println(err)
		return nil, err
	}
	pageMeta := make(map[string]interface{})
	err = json.Unmarshal(pageJson, &pageMeta)
	if err != nil {
		log.Println(err)
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
			log.Printf("Page Route: %s", route)
			page, err2 := pb.createPage(route, dir)
			if page != nil && err2 == nil {
				(*pb.pages)[route] = page
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
