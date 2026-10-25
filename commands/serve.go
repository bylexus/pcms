package commands

import (
	"context"
	"fmt"
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
	var dbh *lib.DBH
	var shouldCloseDBH bool
	// re-index in the background after start (files mode with an existing index only):
	var backgroundIndex bool
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
	switch config.ServeMode {
	case model.SERVE_MODE_EMBEDDED_DOC:
		// Clear stale cache from previous runs — the in-memory DB is fresh each
		// time, so cached HTML must be rebuilt.
		if config.Server.CacheDir != "" {
			os.RemoveAll(config.Server.CacheDir)
		}

		// index the doc on every start:
		if err := RunIndexCmd(config); err != nil {
			return err
		}

		// determine the embedded doc FS:
		errorLogger.Info("Serving embedded doc site")
		siteFS, _, err = model.GetEmbeddedSourceFS(config)
		if err != nil {
			return err
		}

		dbh, shouldCloseDBH, err = lib.GetDBHForConfig(config)
		if err != nil {
			return err
		}
	case model.SERVE_MODE_FILES:
		errorLogger.Info("Serving content from %s", config.SourcePath)
		siteFS = os.DirFS(config.SourcePath)
		dbh, shouldCloseDBH, err = lib.GetDBHForConfig(config)
		if err != nil {
			return err
		}

		// Auto-index on first startup: if the pages table is empty, build the index.
		pageCount, err := dbh.CountPages()
		if err != nil {
			return err
		}
		if pageCount == 0 {
			errorLogger.Info("No pages indexed yet — running initial index automatically")
			log.Println("No pages indexed yet — running initial index automatically")
			if err := RunIndexCmd(config); err != nil {
				return err
			}
		} else {
			backgroundIndex = true
		}
	default:
		return fmt.Errorf("unknown serve mode: %s", config.ServeMode)
	}
	if shouldCloseDBH {
		defer dbh.Close()
	}

	defer accessLogger.Close()
	defer errorLogger.Close()

	// background jobs (run until the process exits):
	bgCtx, cancelBg := context.WithCancel(context.Background())
	defer cancelBg()

	// background re-index (runs once):
	if backgroundIndex {
		lib.StartBackgroundIndex(bgCtx, config, dbh, errorLogger)
	}

	// background cache cleanup:
	webserver.StartCacheCleanup(
		bgCtx,
		config.Server.CacheDir,
		*config.Server.CacheMaxAge,
		config.Server.CacheCleanupInterval,
		errorLogger,
	)

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
