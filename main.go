package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"alexi.ch/pcms/src/logging"
	"alexi.ch/pcms/src/model"
	"alexi.ch/pcms/src/webserver"
)

func createPageTree(config *model.Config, logger *logging.Logger) webserver.PageBuilder {
	pageBuilder := webserver.NewPageBuilder(logger)

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

	// setup logging:
	accessLogger := logging.NewLogger(
		config.Server.Logging.Access.File,
		logging.DEBUG,
		"{{message}}",
	)
	errorLogger := logging.NewLogger(
		config.Server.Logging.Error.File,
		logging.StrToLevel(config.Server.Logging.Error.Level),
		"",
	)
	log.Printf("Server is starting. System log goes to %s\n", errorLogger.Filepath)
	defer accessLogger.Close()
	defer errorLogger.Close()

	// create page tree:
	pageBuilder := createPageTree(&config, errorLogger)

	// initialize web server:
	h := &webserver.RequestHandler{
		ServerConfig: &config,
		PageMap:      pageBuilder.GetPageMap(),
		ErrorLogger:  errorLogger,
	}
	server := &http.Server{
		Addr:    ":3000",
		Handler: webserver.CreateAccessLoggerMiddleware(accessLogger, h),
	}

	errorLogger.Info("Server starting, listening to %s", config.Server.Listen)
	errorLogger.Info("Serving site from %s", config.Site.Path)
	log.Printf("Server starting, listening to %s\n", config.Server.Listen)
	log.Printf("Serving site from %s\n", config.Site.Path)
	errorLogger.Fatal(server.ListenAndServe().Error())
}
