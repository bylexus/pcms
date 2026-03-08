package commands

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"

	"alexi.ch/pcms/lib"
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

	// serve mode: either by the configured file folder,
	// or serve the embedded doc:
	var siteFS fs.FS
	var dbh *lib.DBH
	switch config.ServeMode {
	case model.SERVE_MODE_EMBEDDED_DOC:
		errorLogger.Info("Serving embedded doc site")
		siteFS, err = fs.Sub(config.EmbeddedDocFS, "doc/build")
		if err != nil {
			return err
		}
	default:
		errorLogger.Info("Serving content from %s", config.SourcePath)
		siteFS = os.DirFS(config.SourcePath)
		dbh, err = lib.GetDBH()
		if err != nil {
			return err
		}
	}

	defer accessLogger.Close()
	defer errorLogger.Close()

	// initialize web server:
	// register own handler with the web prefix removed:
	h := webserver.CreateAccessLoggerMiddleware(
		accessLogger,
		http.StripPrefix(
			config.Server.Prefix,
			webserver.NewRequestHandler(config, accessLogger, errorLogger, siteFS, dbh),
		),
	)

	// Now, fire up the barbequeue:
	server := &http.Server{
		Addr:    config.Server.Listen,
		Handler: h,
	}

	errorLogger.Info("Server starting, listening to %s", config.Server.Listen)
	errorLogger.Info("Serving site from %s", config.SourcePath)
	log.Printf("Server starting, listening to %s\n", config.Server.Listen)
	log.Printf("Serving site from %s%s", config.Server.Listen, path.Join("/", config.Server.Prefix))
	err = server.ListenAndServe()
	if err != nil {
		errorLogger.Fatal("%s", err.Error())
	}
	return err
}
