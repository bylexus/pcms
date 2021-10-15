package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"alexi.ch/pcms/src/model"
	"alexi.ch/pcms/src/webserver"
)

func createPageTree(config *model.Config) webserver.PageBuilder {
	pageBuilder := webserver.NewPageBuilder()

	err := pageBuilder.ExaminePageDir(config.Site.Path, config)
	if err != nil {
		log.Fatal(err)
	}

	// add theme route manually:
	pageBuilder.AddPage(config.Site.Webroot+"/theme", &model.Page{
		Type:  model.PAGE_TYPE_THEME,
		Route: config.Site.Webroot + "/theme",
		Dir:   config.Site.ThemePath,
	})

	pageBuilder.BuildPageTree()
	return pageBuilder
}

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
	config := model.NewConfig(conffilePath)

	// create page tree:
	pageBuilder := createPageTree(&config)

	// initialize web server:
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
