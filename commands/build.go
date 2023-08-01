package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	fmt.Printf("Cleaning Dir: %s\n", dir)
	for _, entry := range entries {
		file := filepath.Join(dir, entry.Name())
		err = os.RemoveAll(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func processInputFS(srcFS fs.FS, basePath string, config model.Config) error {
	entries, err := fs.ReadDir(srcFS, ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(basePath, entry.Name())
		if entry.IsDir() {
			childPath := filepath.Join(basePath, entry.Name())
			childFS := os.DirFS(childPath)
			err := processInputFS(childFS, childPath, config)
			if err != nil {
				return err
			}
		} else {
			processSourceFile(sourcePath, config)
		}
	}
	return nil
}

func processSourceFile(sourcePath string, config model.Config) error {
	relPath, err := filepath.Rel(config.SourcePath, sourcePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s: %s\n", sourcePath, err)
		return err
	}
	relPath = filepath.Join("/", relPath)
	isExcluded, pattern := processor.IsFileExcluded(relPath, config.ExcludePatterns)
	if isExcluded {
		fmt.Printf("Skip file due to exclude pattern match: %s\n", pattern)
		return nil
	}

	processor := processor.GetProcessor(sourcePath, config)
	outFile, err := processor.ProcessFile(sourcePath, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s: %s\n", sourcePath, err)
		return err
	}
	if len(outFile) > 0 {
		fmt.Printf("%s: %s\n", processor.Name(), outFile)
	}
	return nil
}
