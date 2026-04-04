package commands

import (
	"fmt"
	"os"

	"alexi.ch/pcms/model"
)

// RunCacheClearCmd removes all files in the configured cache directory.
func RunCacheClearCmd(config model.Config) error {
	cacheDir := config.Server.CacheDir
	if cacheDir == "" {
		return fmt.Errorf("cache_dir is not configured")
	}

	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		fmt.Printf("Cache directory does not exist, nothing to clear: %s\n", cacheDir)
		return nil
	}

	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	fmt.Printf("Cache cleared: %s\n", cacheDir)
	return nil
}
