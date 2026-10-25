package lib

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"time"

	"alexi.ch/pcms/logging"
	"alexi.ch/pcms/model"
)

// IndexResult holds the outcome of one index run.
type IndexResult struct {
	Source        string
	Pages         int
	Files         int
	Duration      time.Duration
	DBPath        string
	SchemaVersion int
}

// IndexSourceFS returns the source FS to index and a label for it (the source path).
func IndexSourceFS(config model.Config) (fs.FS, string, error) {
	if config.ServeMode == model.SERVE_MODE_EMBEDDED_DOC {
		return model.GetEmbeddedSourceFS(config)
	}

	if config.SourcePath == "" {
		return nil, "", fmt.Errorf("source path empty")
	}

	return os.DirFS(config.SourcePath), config.SourcePath, nil
}

// RunIndex rebuilds the page and file index in dbh from sourceFS.
// All changes are done in one transaction: other users of dbh see the old
// index until the commit. When ctx is done before the commit, the run is
// rolled back and ctx.Err() is returned.
func RunIndex(ctx context.Context, dbh *DBH, sourceFS fs.FS, sourceLabel string, excludePatterns []string) (IndexResult, error) {
	start := time.Now()

	snapshot, err := BuildIndexSnapshot(sourceFS, excludePatterns)
	if err != nil {
		return IndexResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return IndexResult{}, err
	}

	run, err := dbh.BeginIndexRun()
	if err != nil {
		return IndexResult{}, err
	}
	defer run.Rollback()

	if err := run.CleanIndex(); err != nil {
		return IndexResult{}, err
	}
	for _, page := range snapshot.Pages {
		if err := run.ReplacePage(page); err != nil {
			return IndexResult{}, err
		}
	}
	for _, file := range snapshot.Files {
		if err := run.ReplaceFile(file); err != nil {
			return IndexResult{}, err
		}
	}
	if err := run.SetLastIndexInfo(sourceLabel, len(snapshot.Pages), len(snapshot.Files)); err != nil {
		return IndexResult{}, err
	}

	if err := ctx.Err(); err != nil {
		return IndexResult{}, err
	}
	if err := run.Commit(); err != nil {
		return IndexResult{}, err
	}

	pagesCount, err := dbh.CountPages()
	if err != nil {
		return IndexResult{}, err
	}
	filesCount, err := dbh.CountFiles()
	if err != nil {
		return IndexResult{}, err
	}

	return IndexResult{
		Source:        sourceLabel,
		Pages:         pagesCount,
		Files:         filesCount,
		Duration:      time.Since(start).Round(time.Millisecond),
		DBPath:        dbh.Path(),
		SchemaVersion: dbh.SchemaVersion(),
	}, nil
}

// RunIndexForConfig rebuilds the index of the configured source in the configured DB.
func RunIndexForConfig(ctx context.Context, config model.Config) (IndexResult, error) {
	sourceFS, sourceLabel, err := IndexSourceFS(config)
	if err != nil {
		return IndexResult{}, err
	}

	dbh, shouldClose, err := GetDBHForConfig(config)
	if err != nil {
		return IndexResult{}, err
	}
	if shouldClose {
		defer dbh.Close()
	}

	return RunIndex(ctx, dbh, sourceFS, sourceLabel, config.ExcludePatterns)
}

// StartBackgroundIndex rebuilds the index once in a background goroutine.
// It does nothing when server.index_on_start is false. The old index stays
// visible until the run commits. Errors are logged; the run is not repeated.
// The returned channel is closed when the run is done (or immediately when disabled).
func StartBackgroundIndex(ctx context.Context, config model.Config, dbh *DBH, logger *logging.Logger) <-chan struct{} {
	done := make(chan struct{})

	if config.Server.IndexOnStart != nil && !*config.Server.IndexOnStart {
		logInfo(logger, "Background index on start disabled (index_on_start is false)")
		close(done)
		return done
	}

	sourceFS, sourceLabel, err := IndexSourceFS(config)
	if err != nil {
		logError(logger, "Background index: %s", err.Error())
		close(done)
		return done
	}

	go func() {
		defer close(done)
		logInfo(logger, "Background index started for %s", sourceLabel)

		res, err := RunIndex(ctx, dbh, sourceFS, sourceLabel, config.ExcludePatterns)
		if err != nil {
			if ctx.Err() != nil {
				logInfo(logger, "Background index canceled: %s", err.Error())
				return
			}
			logError(logger, "Background index failed: %s", err.Error())
			return
		}
		logInfo(logger, "Background index done: %d pages, %d files (%s)", res.Pages, res.Files, res.Duration)
	}()

	return done
}

// logInfo writes msg to the logger (if any) and to stdout.
func logInfo(logger *logging.Logger, msg string, params ...any) {
	if logger != nil {
		logger.Info(msg, params...)
	}
	log.Printf(msg+"\n", params...)
}

// logError writes msg to the logger (if any) and to stdout.
func logError(logger *logging.Logger, msg string, params ...any) {
	if logger != nil {
		logger.Error(msg, params...)
	}
	log.Printf(msg+"\n", params...)
}
