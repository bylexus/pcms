package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
)

const siteTemplateDir string = "site-template"

// embed the site-template/ dir into the binary:
// go:embed site-template
var templateContent embed.FS

// init creates a skeleton application in a specified dir.
func runInitCmd(args CmdArgs) {
	if len(args.flagSet.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "Error: no path given.")
		args.flagSet.Usage()
		os.Exit(1)
	}
	path := getDestPath(args.flagSet.Arg(0))

	fmt.Printf("Creating skeleton pcms site in %s...\n", path)

	files, err := templateContent.ReadDir(siteTemplateDir)
	if err != nil {
		panic(err)
	}

	for _, dEntry := range files {
		copyContentToDest("", dEntry, path)
	}
}

func copyContentToDest(baseDir string, dirEntry fs.DirEntry, destRoot string) {
	if dirEntry.Type().IsRegular() {
		embedPath := path.Join(siteTemplateDir, baseDir, dirEntry.Name())
		destPath := path.Join(destRoot, baseDir, dirEntry.Name())
		fmt.Printf("Copy %s to %s\n", embedPath, destPath)
		f, err := templateContent.Open(embedPath)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		copyFileToDest(f, destPath)
	} else if dirEntry.IsDir() {
		files, err := templateContent.ReadDir(path.Join(siteTemplateDir, baseDir, dirEntry.Name()))
		if err != nil {
			panic(err)
		}

		for _, dEntry := range files {
			copyContentToDest(path.Join(baseDir, dirEntry.Name()), dEntry, destRoot)
		}
	}
}

func copyFileToDest(file fs.File, dest string) {
	err := os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		panic(err)
	}
	destFile, err := os.Create(dest)
	if err != nil {
		panic(err)
	}
	defer destFile.Close()
	if _, err := io.Copy(destFile, file); err != nil {
		panic(err)
	}
}

func getDestPath(relPath string) string {
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		panic(err)
	}
	info, err := os.Stat(absPath)

	if err == nil && info.IsDir() != true {
		panic("File already exists and is not a dir.")
	}
	if err != nil && os.IsNotExist(err) {
		err := os.MkdirAll(absPath, 0755)
		if err != nil {
			panic(err)
		}
	} else if err != nil {
		panic(err)
	}

	return absPath
}
