package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path"

	"alexi.ch/pcms/model"
	"alexi.ch/pcms/processor"
)

// Run the 'build' sub-command:
// build the site to an output folder
func RunBuildCmd(config model.Config) {
	cleanDir(config.DestPath)
	srcFS := os.DirFS(config.SourcePath)
	processInputFS(srcFS, config.SourcePath, config)

	// setup logging:
	// accessLogger := logging.NewLogger(
	// 	config.Server.Logging.Access.File,
	// 	logging.DEBUG,
	// 	"{{message}}",
	// )
	// errorLogger := logging.NewLogger(
	// 	config.Server.Logging.Error.File,
	// 	logging.StrToLevel(config.Server.Logging.Error.Level),
	// 	"",
	// )
	// log.Printf("Server is starting. System log goes to %s\n", errorLogger.Filepath)
	// log.Printf("App's Current Working Dir: %s\n", cwd)
	// errorLogger.Info("App's Current Working Dir: %s\n", cwd)
	// defer accessLogger.Close()
	// defer errorLogger.Close()

	// // create page tree:
	// pageMap := webserver.CreatePageTree(&config, errorLogger)

	// // initialize web server:
	// h := &webserver.RequestHandler{
	// 	ServerConfig: &config,
	// 	PageMap:      pageMap,
	// 	ErrorLogger:  errorLogger,
	// }
	// server := &http.Server{
	// 	Addr:    ":3000",
	// 	Handler: webserver.CreateAccessLoggerMiddleware(accessLogger, h),
	// }

	// errorLogger.Info("Server starting, listening to %s", config.Server.Listen)
	// errorLogger.Info("Serving site from %s", config.Site.Path)
	// log.Printf("Server starting, listening to %s\n", config.Server.Listen)
	// log.Printf("Serving site from %s\n", config.Site.Path)
	// errorLogger.Fatal(server.ListenAndServe().Error())
}

func cleanDir(dir string) error {
	if len(dir) == 0 {
		return fmt.Errorf("path empty")
	}
	return os.RemoveAll(dir)
}

func processInputFS(srcFS fs.FS, basePath string, config model.Config) {
	entries, err := fs.ReadDir(srcFS, ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
	for _, entry := range entries {
		sourcePath := path.Join(basePath, entry.Name())
		if entry.IsDir() {
			childPath := path.Join(basePath, entry.Name())
			childFS := os.DirFS(childPath)
			processInputFS(childFS, childPath, config)
		} else {
			fmt.Printf("Working on: %s\n", sourcePath)
			processSourceFile(sourcePath, config)
		}
	}
}

func processSourceFile(sourcePath string, config model.Config) {
	isExcluded, pattern := processor.IsFileExcluded(sourcePath, config.ExcludePatterns)
	if isExcluded {
		fmt.Printf("  Skip file due to exclude pattern match: %s\n", pattern)
		return
	}

	processor := processor.GetProcessor(sourcePath, config)
	outFile, err := processor.ProcessFile(sourcePath, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s: %s\n", sourcePath, err)
		return
	}
	fmt.Printf("  %s: %s\n", processor.Name(), outFile)
}
