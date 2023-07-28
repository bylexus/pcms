package model

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

type CmdArgs struct {
	ConfigFilePath string
	FlagSet        *flag.FlagSet
}

func GetConfFilePath(confPath string) string {
	// read conf file path:
	if len(confPath) > 0 {
		conffilePath, err := filepath.Abs(confPath)
		if err != nil {
			log.Fatal(err)
		}
		return conffilePath
	} else {
		// Read config from actual dir:
		basePath, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		conffilePath, err := filepath.Abs(filepath.Join(basePath, "pcms-config.yaml"))
		if err != nil {
			log.Fatal(err)
		}
		return conffilePath
	}
}
