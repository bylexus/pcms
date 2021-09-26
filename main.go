package main

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"alexi.ch/pcms/src/model"
	"alexi.ch/pcms/src/webserver"
	"github.com/flosch/pongo2/v4"
	"gopkg.in/yaml.v3"
)

var pages model.PageMap

func createPage(route string, dir string) (*model.Page, error) {
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

func examinePageDir(rootDir string, pageMap *model.PageMap, config *model.Config) error {
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
			route := createRouteFromRelPath(filepath.Dir(path), config)
			dir := filepath.Join(rootDir, filepath.Dir(path))
			log.Printf("Page Route: %s", route)
			page, err2 := createPage(route, dir)
			if page != nil && err2 == nil {
				pages[route] = page
			}
		}

		return nil
	})
	return err
}

/**
 * Creates a valid page route from a given site directory / file:
 */
func createRouteFromRelPath(relPath string, config *model.Config) string {
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

func configPongoTemplatePathLoader(conf *model.Config) {
	loader, err := pongo2.NewLocalFileSystemLoader(conf.Site.GetTemplatePath())
	if err != nil {
		log.Panic(err)
	}
	pongo2.DefaultSet.AddLoader(loader)
}

/**
 * Step 1: Startup:
 * - Read config from yaml file in same dir as bin
 */
func main() {
	// Read config from actual dir:
	confFile, err := ioutil.ReadFile("pcms-config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	config := model.Config{
		Server: model.ServerConfig{
			Listen: ":8080",
		},
		Site: model.SiteConfig{
			Theme: "default",
		},
	}

	basePath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	config.BasePath = basePath
	config.Site.Path, err = filepath.Abs(filepath.Join(basePath, "site"))
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(confFile, &config)
	if err != nil {
		log.Fatal(err)
	}

	if config.Site.ThemePath == "" {
		absThemePath, err3 := filepath.Abs(filepath.Join(basePath, "themes", config.Site.Theme))
		if err3 != nil {
			log.Fatal(err3)
		}
		config.Site.ThemePath = absThemePath
	}
	configPongoTemplatePathLoader(&config)

	pages = make(model.PageMap)

	err = examinePageDir(config.Site.Path, &pages, &config)

	// add theme route manually:
	pages[config.Site.Webroot+"/theme"] = &model.Page{
		Type:  model.PAGE_TYPE_THEME,
		Route: config.Site.Webroot + "/theme",
		Dir:   config.Site.ThemePath,
	}
	pages.BuildPageTree()

	h := &webserver.RequestHandler{
		ServerConfig: &config,
		Pages:        &pages,
	}
	server := &http.Server{
		Addr:    ":3000",
		Handler: h,
	}

	log.Printf("Server starting, listening to %s\n", config.Server.Listen)
	log.Printf("Serving site from %s\n", config.Site.Path)
	log.Fatal(server.ListenAndServe())
}
