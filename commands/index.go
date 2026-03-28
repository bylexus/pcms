package commands

import (
	"fmt"
	"io/fs"
	"os"
	"time"

	"alexi.ch/pcms/lib"
	"alexi.ch/pcms/model"
)

// RunIndexCmd initializes or updates the local db schema.
func RunIndexCmd(config model.Config) error {
	start := time.Now()

	sourceFS, sourceLabel, err := getIndexSourceFS(config)
	if err != nil {
		return err
	}

	snapshot, err := lib.BuildIndexSnapshot(sourceFS, config.ExcludePatterns)
	if err != nil {
		return err
	}

	dbh, shouldClose, err := lib.GetDBHForConfig(config)
	if err != nil {
		return err
	}
	if shouldClose {
		defer dbh.Close()
	}

	if err := dbh.BeginIndexRun(); err != nil {
		return err
	}
	defer dbh.RollbackIndexRun()

	if err := dbh.CleanIndex(); err != nil {
		return err
	}

	for _, page := range snapshot.Pages {
		if err := dbh.ReplacePage(page); err != nil {
			return err
		}
	}

	for _, file := range snapshot.Files {
		if err := dbh.ReplaceFile(file); err != nil {
			return err
		}
	}

	if err := dbh.SetLastIndexInfo(sourceLabel, len(snapshot.Pages), len(snapshot.Files)); err != nil {
		return err
	}

	if err := dbh.CommitIndexRun(); err != nil {
		return err
	}

	pagesCount, err := dbh.CountPages()
	if err != nil {
		return err
	}
	filesCount, err := dbh.CountFiles()
	if err != nil {
		return err
	}

	duration := time.Since(start).Round(time.Millisecond)
	fmt.Printf("Index done: %d pages, %d files (%s)\n", pagesCount, filesCount, duration)
	fmt.Printf("DB: %s (schema version %d)\n", dbh.Path(), dbh.SchemaVersion())
	return nil
}

func getIndexSourceFS(config model.Config) (fs.FS, string, error) {
	if config.ServeMode == model.SERVE_MODE_EMBEDDED_DOC {
		return model.GetEmbeddedSourceFS(config)
	}

	if config.SourcePath == "" {
		return nil, "", fmt.Errorf("source path empty")
	}

	return os.DirFS(config.SourcePath), config.SourcePath, nil
}
