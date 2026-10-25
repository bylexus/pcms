package commands

import (
	"context"
	"fmt"

	"alexi.ch/pcms/lib"
	"alexi.ch/pcms/model"
)

// RunIndexCmd initializes or updates the local db schema.
func RunIndexCmd(config model.Config) error {
	res, err := lib.RunIndexForConfig(context.Background(), config)
	if err != nil {
		return err
	}

	fmt.Printf("Index done: %d pages, %d files (%s)\n", res.Pages, res.Files, res.Duration)
	fmt.Printf("DB: %s (schema version %d)\n", res.DBPath, res.SchemaVersion)
	return nil
}
