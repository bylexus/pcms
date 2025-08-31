package commands

import (
	"fmt"

	"alexi.ch/pcms/model"
)

// Run the 'dev' sub-command:
// playground during development
func RunDevCmd(config model.Config) error {
	fmt.Println("dev command")
	fmt.Printf("Source dir: %s\n", config.SourcePath)

	// config.ServeMode = model.SERVE_MODE_EMBEDDED_DOC
	page, err := model.BuildPageTree(config)

	fmt.Printf("%#v\n", page)
	fmt.Printf("%v\n", page.Routes())

	// srcFS := os.DirFS(config.SourcePath)
	// return processInputFS(srcFS, config.SourcePath, config)
	return err
}
