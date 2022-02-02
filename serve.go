package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"alexi.ch/pcms/src/logging"
	"alexi.ch/pcms/src/model"
	"alexi.ch/pcms/src/webserver"
)

func getConfFilePath(flagSet *flag.FlagSet) string {
	confFlag := flagSet.Lookup("c")
	// read conf file path:
	if confFlag != nil && len((*confFlag).Value.String()) > 0 {
		conffilePath, err := filepath.Abs((*confFlag).Value.String())
		if err != nil {
			log.Fatal(err)
		}
		return conffilePath
	} else {
		// Read config from actual dir:
		basePath, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		conffilePath, err := filepath.Abs(filepath.Join(basePath, "pcms-config.yaml"))
		if err != nil {
			log.Fatal(err)
		}
		return conffilePath
	}
}

// Run the 'serve' sub-command:
// build the page tree and start the web engine.
func runServeCmd(args CmdArgs) {
	confFilePath := getConfFilePath(args.flagSet)

	// change the app's CWD to the conf file location's dir:
	cwd := path.Dir(confFilePath)
	err := os.Chdir(cwd)
	if err != nil {
		panic(err)
	}
	config := model.NewConfig(confFilePath)

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
	log.Printf("App's Current Working Dir: %s\n", cwd)
	errorLogger.Info("App's Current Working Dir: %s\n", cwd)
	defer accessLogger.Close()
	defer errorLogger.Close()

	// create page tree:
	pageMap := webserver.CreatePageTree(&config, errorLogger)

	// initialize web server:
	h := &webserver.RequestHandler{
		ServerConfig: &config,
		PageMap:      pageMap,
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
