package processor

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"alexi.ch/pcms/model"
)

type ScssProcessor struct {
}

func (p ScssProcessor) Name() string {
	return "scss"
}

// Processes scss files using the external dart sass binary,
// for a lack of a golang available processor.
// _*.scss files are skipped, as they are included in a master scss file.
func (p ScssProcessor) ProcessFile(sourceFile string, config model.Config) (destFile string, err error) {
	sassBin := config.Processors.Scss.SassBin
	if len(sassBin) == 0 {
		sassBin = "sass"
	}

	fileBase := filepath.Base(sourceFile)
	match, err := filepath.Match("_*.scss", fileBase)
	if err != nil {
		return "", err
	}
	if match {
		fmt.Fprintf(os.Stderr, "skipping _*.scss, as those are included by main sass file\n")
		return "", nil
	}

	relPath, err := filepath.Rel(config.SourcePath, sourceFile)
	if err != nil {
		return "", err
	}

	// calc outfile path and create dest directory
	outFile := path.Join(config.DestPath, relPath)
	outBase := strings.Replace(filepath.Base(outFile), ".scss", ".css", 1)
	outDir := filepath.Dir(outFile)
	outFile = path.Join(outDir, outBase)
	err = os.MkdirAll(outDir, fs.ModeDir|0777)
	if err != nil {
		return "", err
	}

	cmd := exec.Command(sassBin, sourceFile, outFile)
	err = cmd.Run()
	if err != nil {
		return "", err
	}

	return outFile, nil
}
