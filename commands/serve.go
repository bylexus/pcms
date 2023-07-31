package commands

import (
	"io/fs"
	"log"
	"net/http"
	"os"

	"alexi.ch/pcms/logging"
	"alexi.ch/pcms/model"
	"alexi.ch/pcms/webserver"
)

// Run the 'serve' sub-command:
// build the page tree and start the web engine.
func RunServeCmd(config model.Config) error {
	var err error = nil
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

	// first, we run the Build command on start:
	if config.ServeMode == model.SERVE_MODE_FILES {
		errorLogger.Info("Building static site ...")
		err := RunBuildCmd(config)
		if err != nil {
			return err
		}
	}

	// serve mode: either by the configured file folder,
	// or serve the embedded doc:
	var siteFS fs.FS
	switch config.ServeMode {
	case model.SERVE_MODE_EMBEDDED_DOC:
		errorLogger.Info("Serving embedded doc site")
		siteFS, err = fs.Sub(config.EmbeddedDocFS, "doc/build")
		if err != nil {
			return err
		}
	default:
		// static site folder as FS:
		errorLogger.Info("Serving content from %s", config.DestPath)
		siteFS = os.DirFS(config.DestPath)

	}

	defer accessLogger.Close()
	defer errorLogger.Close()

	// initialize web server:
	// register own handler with the web prefix removed:
	h := webserver.CreateAccessLoggerMiddleware(
		accessLogger,
		http.StripPrefix(
			config.Server.Prefix,
			webserver.NewRequestHandler(config, accessLogger, errorLogger, siteFS),
		),
	)

	// Now, fire up the barbequeue:
	server := &http.Server{
		Addr:    config.Server.Listen,
		Handler: h,
	}

	errorLogger.Info("Server starting, listening to %s", config.Server.Listen)
	errorLogger.Info("Serving site from %s", config.DestPath)
	log.Printf("Server starting, listening to %s\n", config.Server.Listen)
	log.Println("Serving site")
	err = server.ListenAndServe()
	if err != nil {
		errorLogger.Fatal(err.Error())
	}
	return err
}
