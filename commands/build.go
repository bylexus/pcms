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
func RunBuildCmd(config model.Config) error {
	if config.ServeMode != model.SERVE_MODE_EMBEDDED_DOC {
		cleanDir(config.DestPath)
	}
	srcFS := os.DirFS(config.SourcePath)
	return processInputFS(srcFS, config.SourcePath, config)
}

func cleanDir(dir string) error {
	if len(dir) == 0 {
		return fmt.Errorf("path empty")
	}
	return os.RemoveAll(dir)
}

func processInputFS(srcFS fs.FS, basePath string, config model.Config) error {
	entries, err := fs.ReadDir(srcFS, ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return err
	}
	for _, entry := range entries {
		sourcePath := path.Join(basePath, entry.Name())
		if entry.IsDir() {
			childPath := path.Join(basePath, entry.Name())
			childFS := os.DirFS(childPath)
			err := processInputFS(childFS, childPath, config)
			if err != nil {
				return err
			}
		} else {
			fmt.Printf("Working on: %s\n", sourcePath)
			processSourceFile(sourcePath, config)
		}
	}
	return nil
}

func processSourceFile(sourcePath string, config model.Config) error {
	isExcluded, pattern := processor.IsFileExcluded(sourcePath, config.ExcludePatterns)
	if isExcluded {
		fmt.Printf("  Skip file due to exclude pattern match: %s\n", pattern)
		return nil
	}

	processor := processor.GetProcessor(sourcePath, config)
	outFile, err := processor.ProcessFile(sourcePath, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s: %s\n", sourcePath, err)
		return err
	}
	fmt.Printf("  %s: %s\n", processor.Name(), outFile)
	return nil
}
