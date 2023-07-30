package commands

import (
	"log"
	"net/http"
	"os"
	"path"

	"alexi.ch/pcms/logging"
	"alexi.ch/pcms/model"
	"alexi.ch/pcms/webserver"
)

// Run the 'serve' sub-command:
// build the page tree and start the web engine.
func RunServeCmd(config model.Config) error {
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
	errorLogger.Info("Building static site ...")
	err := RunBuildCmd(config)
	if err != nil {
		return err
	}

	confFilePath := config.ConfigFile

	// change the app's CWD to the conf file location's dir:
	cwd := path.Dir(confFilePath)
	err = os.Chdir(cwd)
	if err != nil {
		return err
	}

	errorLogger.Info("App's Current Working Dir: %s\n", cwd)
	defer accessLogger.Close()
	defer errorLogger.Close()

	// initialize web server:
	// register own handler with the web prefix removed:
	h := webserver.CreateAccessLoggerMiddleware(
		accessLogger,
		http.StripPrefix(
			config.Server.Prefix,
			webserver.NewRequestHandler(config, accessLogger, errorLogger),
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
	log.Printf("Serving site from %s\n", config.DestPath)
	err = server.ListenAndServe()
	if err != nil {
		errorLogger.Fatal(err.Error())
	}
	return err
}
