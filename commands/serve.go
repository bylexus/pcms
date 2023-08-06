package commands

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"alexi.ch/pcms/logging"
	"alexi.ch/pcms/model"
	"alexi.ch/pcms/webserver"
	"github.com/fsnotify/fsnotify"
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
		// start file watcher, if enabled in config:
		if config.Server.Watch {
			watchers, _ := startFileWatcher(config.SourcePath, config, errorLogger)
			for _, w := range watchers {
				defer w.Close()
			}
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
	log.Printf("Serving site from %s%s", config.Server.Listen, path.Join("/", config.Server.Prefix))
	err = server.ListenAndServe()
	if err != nil {
		errorLogger.Fatal(err.Error())
	}
	return err
}

func startFileWatcher(watchPath string, config model.Config, logger *logging.Logger) ([]*fsnotify.Watcher, error) {
	watchers := make([]*fsnotify.Watcher, 0)
	singleFileBuildWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("Cannot start single build file watcher: %s", err.Error())
		return nil, err
	}
	watchers = append(watchers, singleFileBuildWatcher)
	fullFileBuildWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("Cannot start full build file watcher: %s", err.Error())
		return nil, err
	}
	watchers = append(watchers, fullFileBuildWatcher)

	// starting the watcher processes in a separate thread, as this watches
	// indefinitely:
	go processSingleBuildWatcherEvents(singleFileBuildWatcher, config, logger)
	go processFullBuildWatcherEvents(fullFileBuildWatcher, config, logger)

	// Add whole source directory tree to single or full file build watcher
	err = filepath.WalkDir(watchPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Error("File Watcher Error: %s", err.Error())
			return err
		}
		if !d.IsDir() && filepath.Base(path) == "variables.yaml" {
			fullFileBuildWatcher.Add(path)
		} else if d.IsDir() {
			return singleFileBuildWatcher.Add(path)
		}

		return nil
	})
	if err != nil {
		logger.Error("File Watcher Error: %s", err.Error())
		return nil, err
	}

	// adding pcms config file to full build watcher:
	fullFileBuildWatcher.Add(config.ConfigFile)

	// adding template dir to full build watcher:
	err = filepath.WalkDir(config.TemplateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			logger.Error("File Watcher Error: %s", err.Error())
			return err
		}
		if d.IsDir() {
			return fullFileBuildWatcher.Add(path)
		}
		return nil
	})
	if err != nil {
		logger.Error("File Watcher Error: %s", err.Error())
		return nil, err
	}

	logger.Info("File Watcher started for root dir %s", watchPath)
	return watchers, nil
}

/*
This method watches for fsnotify.Watcher events indefinitely and should be started
as a goroutine in parallel to the main process.
*/
func processSingleBuildWatcherEvents(watcher *fsnotify.Watcher, config model.Config, logger *logging.Logger) {
	var err error = nil
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				logger.Info("File Watcher: Initiating single rebuild: File %s: %s", event.Op, event.Name)
				err = triggerSingleRebuild(event.Name, config)
				if err != nil {
					logger.Error("File Rebuild Error: %s", err.Error())
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error("File Watcher Error: %s", err.Error())
		}
	}
}

/*
This method watches for fsnotify.Watcher events indefinitely and should be started
as a goroutine in parallel to the main process.
*/
func processFullBuildWatcherEvents(watcher *fsnotify.Watcher, config model.Config, logger *logging.Logger) {
	var err error = nil
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				logger.Info("File Watcher: Initiating Full rebuild due to file change: %s", event.Name)
				err = triggerFullRebuild(config)
				if err != nil {
					logger.Error("File Rebuild Error: %s", err.Error())
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error("File Watcher Error: %s", err.Error())
		}
	}
}

func triggerSingleRebuild(file string, config model.Config) error {
	return ProcessSourceFile(file, config)
}

func triggerFullRebuild(config model.Config) error {
	return RunBuildCmd(config)
}
