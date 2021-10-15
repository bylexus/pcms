package main

import (
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
	basePath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	conffilePath, err := filepath.Abs(filepath.Join(basePath, "pcms-config.yaml"))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Reading config file: %v", conffilePath)

	confFile, err := ioutil.ReadFile(conffilePath)
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

	pageBuilder := webserver.NewPageBuilder()

	err = pageBuilder.ExaminePageDir(config.Site.Path, &config)

	// add theme route manually:
	pageBuilder.AddPage(config.Site.Webroot+"/theme", &model.Page{
		Type:  model.PAGE_TYPE_THEME,
		Route: config.Site.Webroot + "/theme",
		Dir:   config.Site.ThemePath,
	})

	pageBuilder.BuildPageTree()

	h := &webserver.RequestHandler{
		ServerConfig: &config,
		PageMap:      pageBuilder.GetPageMap(),
	}
	server := &http.Server{
		Addr:    ":3000",
		Handler: h,
	}

	log.Printf("Server starting, listening to %s\n", config.Server.Listen)
	log.Printf("Serving site from %s\n", config.Site.Path)
	log.Fatal(server.ListenAndServe())
}
