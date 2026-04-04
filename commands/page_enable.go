package commands

import (
	"fmt"

	"alexi.ch/pcms/lib"
	"alexi.ch/pcms/model"
)

// RunEnablePageCmd enables a page (and optionally its descendants) in the index DB.
func RunEnablePageCmd(config model.Config, route string, recursive bool) error {
	dbh, err := lib.OpenDBH(config.DatabasePath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer dbh.Close()

	if err := dbh.SetPageEnabled(route, true, recursive); err != nil {
		return err
	}

	if recursive {
		fmt.Printf("enabled page and all descendants: %s\n", route)
	} else {
		fmt.Printf("enabled page: %s\n", route)
	}
	return nil
}

// RunDisablePageCmd disables a page and all its descendants in the index DB.
func RunDisablePageCmd(config model.Config, route string) error {
	dbh, err := lib.OpenDBH(config.DatabasePath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer dbh.Close()

	if err := dbh.SetPageEnabled(route, false, false); err != nil {
		return err
	}

	fmt.Printf("disabled page and all descendants: %s\n", route)
	return nil
}
