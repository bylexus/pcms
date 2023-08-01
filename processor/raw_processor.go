package processor

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"alexi.ch/pcms/model"
	"alexi.ch/pcms/stdlib"
)

type RawProcessor struct {
}

func (p RawProcessor) Name() string {
	return "raw"
}

func (p RawProcessor) ProcessFile(sourceFile string, config model.Config) (destFile string, err error) {
	relPath, err := filepath.Rel(config.SourcePath, sourceFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s: %s\n", sourceFile, err)
	}
	outFile := filepath.Join(config.DestPath, relPath)
	outDir := filepath.Dir(outFile)
	err = os.MkdirAll(outDir, fs.ModeDir|0777)
	if err != nil {
		return "", err
	}
	_, err = stdlib.CopyFile(sourceFile, outFile)
	if err != nil {
		return "", err
	}
	return outFile, nil
}
