package webserver

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"alexi.ch/pcms/logging"
)

// cacheCleanupResult holds the outcome of one cleanup pass.
type cacheCleanupResult struct {
	FilesRemoved int
	DirsRemoved  int
	Errors       []error
}

// cleanupCacheDir does one cleanup pass over cacheDir:
//   - removes every regular file whose mtime is older than now - maxAge
//     (page cache, image cache, leftover temp files, orphaned old-format files),
//   - then removes empty sub-directories, deepest first.
//
// cacheDir itself is never removed. A non-existing cacheDir is not an error.
// Symlinks are not followed and not removed. Errors on single entries are
// collected and do not stop the pass.
func cleanupCacheDir(cacheDir string, maxAge time.Duration, now time.Time) cacheCleanupResult {
	var res cacheCleanupResult
	if cacheDir == "" {
		return res
	}
	// cacheDir itself may be a symlink (e.g. to a separate volume): resolve it,
	// because filepath.WalkDir does not descend into a symlinked root.
	root, err := filepath.EvalSymlinks(cacheDir)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			res.Errors = append(res.Errors, err)
		}
		return res
	}
	rootInfo, err := os.Stat(root)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			res.Errors = append(res.Errors, err)
		}
		return res
	}
	if !rootInfo.IsDir() {
		return res
	}
	cacheDir = root

	cutoff := now.Add(-maxAge)
	var dirs []string

	walkErr := filepath.WalkDir(cacheDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			res.Errors = append(res.Errors, err)
			// skip unreadable dirs, but continue with the rest:
			if d != nil && d.IsDir() && path != cacheDir {
				return fs.SkipDir
			}
			return nil
		}
		if path == cacheDir {
			return nil
		}
		if d.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		if !d.Type().IsRegular() {
			// symlinks, devices, sockets etc.: leave untouched
			return nil
		}
		info, err := d.Info()
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				res.Errors = append(res.Errors, err)
			}
			return nil
		}
		if !info.ModTime().Before(cutoff) {
			return nil
		}
		if err := os.Remove(path); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				res.Errors = append(res.Errors, err)
			}
			return nil
		}
		res.FilesRemoved++
		return nil
	})
	if walkErr != nil {
		res.Errors = append(res.Errors, walkErr)
	}

	// remove empty sub-dirs, deepest first (longest path first):
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				res.Errors = append(res.Errors, err)
			}
			continue
		}
		if len(entries) > 0 {
			continue
		}
		// os.Remove fails on a non-empty dir, so a file written in between is safe:
		if err := os.Remove(dir); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				res.Errors = append(res.Errors, err)
			}
			continue
		}
		res.DirsRemoved++
	}
	return res
}

// StartCacheCleanup starts the background cache cleanup goroutine.
// It does nothing when maxAge is 0 (cleanup disabled). Otherwise it runs one
// pass immediately, then one pass every interval, until ctx is done.
func StartCacheCleanup(ctx context.Context, cacheDir string, maxAge, interval time.Duration, logger *logging.Logger) {
	if maxAge <= 0 {
		if logger != nil {
			logger.Info("Cache cleanup disabled (cache_max_age is 0)")
		}
		return
	}
	if cacheDir == "" {
		if logger != nil {
			logger.Info("Cache cleanup disabled (no cache_dir)")
		}
		return
	}
	if interval <= 0 {
		// defensive: the config always sets a positive interval
		interval = time.Hour
	}
	if logger != nil {
		logger.Info("Cache cleanup enabled for %s: max age %s, interval %s", cacheDir, maxAge, interval)
	}

	go func() {
		runCacheCleanupPass(cacheDir, maxAge, logger)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runCacheCleanupPass(cacheDir, maxAge, logger)
			}
		}
	}()
}

// runCacheCleanupPass runs one cleanup pass and logs the result.
func runCacheCleanupPass(cacheDir string, maxAge time.Duration, logger *logging.Logger) {
	res := cleanupCacheDir(cacheDir, maxAge, time.Now())
	if logger == nil {
		return
	}
	for _, err := range res.Errors {
		logger.Warning("cache cleanup: %s", err.Error())
	}
	if res.FilesRemoved > 0 || res.DirsRemoved > 0 {
		logger.Info("cache cleanup: removed %d files and %d empty directories from %s", res.FilesRemoved, res.DirsRemoved, cacheDir)
	} else {
		logger.Debug("cache cleanup: nothing to remove in %s", cacheDir)
	}
}
