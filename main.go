package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"alexi.ch/pcms/src/logging"
	"alexi.ch/pcms/src/model"
	"alexi.ch/pcms/src/webserver"
)

type CmdArgs struct {
	confFilePath string
}

func printUsage(mainFlags *flag.FlagSet) {
	fmt.Fprint(os.Stderr, "Usage:\n\npcms [options] <sub-command>\n\n")
	fmt.Fprint(os.Stderr, "options:\n\n")
	mainFlags.PrintDefaults()
	fmt.Fprint(os.Stderr, "\nA sub-command is expected. Supported sub-commands:\n\n")
	fmt.Fprint(os.Stderr, "serve:      Starts the web server and serves the page\n")
	fmt.Fprintln(os.Stderr)
}

func parseCmdArgs() CmdArgs {
	args := CmdArgs{
		confFilePath: "",
	}

	confFilePathPtr := flag.String("c", "", "path to the pcms-config.yaml file, also defines the site dir")
	helpFlag := flag.Bool("h", false, "Prints this help")
	// serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)
	flag.Parse()

	// read conf file path:
	if len(*confFilePathPtr) > 0 {
		conffilePath, err := filepath.Abs(*confFilePathPtr)
		if err != nil {
			log.Fatal(err)
		}
		args.confFilePath = conffilePath
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
		args.confFilePath = conffilePath
	}

	if *helpFlag || flag.NArg() < 1 {
		printUsage(flag.CommandLine)
		os.Exit(1)
	}

	switch flag.Args()[0] {
	case "serve":
	default:
		printUsage(flag.CommandLine)
		os.Exit(1)
	}

	return args
}

func main() {
	args := parseCmdArgs()

	config := model.NewConfig(args.confFilePath)

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
