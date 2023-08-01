package processor

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"alexi.ch/pcms/model"
)

/*
Tests all the file paths returned by md_processor.prepareFilePaths
*/
func TestMdProcessorPrepareFilePaths(t *testing.T) {
	var processor = MdProcessor{}
	var sourceRoot = filepath.Join(os.TempDir(), filepath.FromSlash("/path/to/source"))
	var sourceFile = filepath.Join(sourceRoot, "sub", "folder", "file.md")
	var relSourcePath = filepath.Join("sub", "folder", "file.md")
	var relSourceRoot = filepath.Join("..", "..")

	var destRoot = filepath.Join(os.TempDir(), filepath.FromSlash("/path/to/dest"))
	var destFile = filepath.Join(destRoot, "sub", "folder", "file.html")
	var relDestPath = filepath.Join("sub", "folder", "file.html")
	var relDestRoot = filepath.Join("..", "..")

	var webroot = "/web/root"

	var config = model.Config{
		SourcePath: sourceRoot,
		DestPath:   destRoot,
		Server: struct {
			Listen  string              "yaml:\"listen\""
			Prefix  string              "yaml:\"prefix\""
			Logging model.LoggingConfig "yaml:\"logging\""
		}{
			Prefix: webroot,
		},
	}
	res, err := processor.prepareFilePaths(sourceFile, config)
	if err != nil {
		t.Fatal(err)
	}
	// start / top path of the source folder
	// rootSourceDir
	if res.rootSourceDir != sourceRoot {
		t.Errorf("rootSourceDir = %s; want %s", res.rootSourceDir, sourceRoot)
	}

	// absolute path of the actual source file
	// absSourcePath
	if res.absSourcePath != sourceFile {
		t.Errorf("absSourcePath = %s; want %s", res.absSourcePath, sourceFile)
	}

	// absolute path of the actual source file
	// absSourceDir string
	if res.absSourceDir != filepath.Dir(sourceFile) {
		t.Errorf("absSourceDir = %s; want %s", res.absSourceDir, filepath.Dir(sourceFile))
	}

	// file path of the actual source file relative to the rootSourceDir
	// relSourcePath
	if res.relSourcePath != relSourcePath {
		t.Errorf("relSourcePath = %s; want %s", res.relSourcePath, relSourcePath)
	}

	// dir path of the actual source file relative to the rootSourceDir
	// relSourceDir
	if res.relSourceDir != filepath.Dir(relSourcePath) {
		t.Errorf("relSourceDir = %s; want %s", res.relSourceDir, filepath.Dir(relSourcePath))
	}

	// // relative path from the actual source file back to the rootSourceDir
	// relSourceRoot string
	if res.relSourceRoot != relSourceRoot {
		t.Errorf("relSourceRoot = %s; want %s", res.relSourceRoot, relSourceRoot)
	}

	// start / top path of the destination folder
	// rootDestDir string
	if res.rootDestDir != destRoot {
		t.Errorf("rootDestDir = %s; want %s", res.rootDestDir, relSourceRoot)
	}

	// absolute path of the actual destination file
	// absDestPath
	if res.absDestPath != destFile {
		t.Errorf("absDestPath = %s; want %s", res.absDestPath, destFile)
	}

	// absolute path of the actual destination file
	// absDestDir
	if res.absDestDir != filepath.Dir(destFile) {
		t.Errorf("absDestDir = %s; want %s", res.absDestDir, filepath.Dir(destFile))
	}

	// file path of the actual destination file relative to the rootDestDir
	// relDestPath
	if res.relDestPath != relDestPath {
		t.Errorf("relDestPath = %s; want %s", res.relDestPath, relDestPath)
	}

	// dir path of the actual destination file relative to the rootSourceDir
	// relDestDir string
	if res.relDestDir != filepath.Dir(relDestPath) {
		t.Errorf("relDestDir = %s; want %s", res.relDestDir, filepath.Dir(relDestPath))
	}

	// relative path from the actual dest file back to the rootSourceDir
	// relDestRoot string
	if res.relDestRoot != relDestRoot {
		t.Errorf("relDestRoot = %s; want %s", res.relDestRoot, relDestRoot)
	}

	// web paths:
	// the webroot prefix, "/" by default
	// webroot string
	if res.webroot != webroot {
		t.Errorf("webroot = %s; want %s", res.webroot, webroot)
	}

	// relative (to webroot) web path to the actual output file
	// relWebPath string
	if res.relWebPath != filepath.ToSlash(relDestPath) {
		t.Errorf("relWebPath = %s; want %s", res.relWebPath, filepath.ToSlash(relDestPath))
	}

	// relative (to webroot) web path to the actual output file's folder
	// relWebDir string
	if res.relWebDir != filepath.ToSlash(filepath.Dir(relDestPath)) {
		t.Errorf("relWebDir = %s; want %s", res.relWebDir, filepath.ToSlash(filepath.Dir(relDestPath)))
	}

	// relative path from the actual file back to the webroot
	// relWebPathToRoot string
	if res.relWebPathToRoot != filepath.ToSlash(relDestRoot) {
		t.Errorf("relWebPathToRoot = %s; want %s", res.relWebPathToRoot, filepath.ToSlash(relDestRoot))
	}

	// // absolute web path of the actual file, including the webroot, starting always with "/"
	// absWebPath string
	if res.absWebPath != path.Join(webroot, filepath.ToSlash(relDestPath)) {
		t.Errorf("absWebPath = %s; want %s", res.absWebPath, path.Join(webroot, filepath.ToSlash(relDestPath)))
	}

	// absolute web path of the actual file's dir, including the webroot, starting always with "/"
	// absWebDir string
	if res.absWebDir != path.Join(webroot, filepath.ToSlash(filepath.Dir(relDestPath))) {
		t.Errorf("absWebDir = %s; want %s", res.absWebDir, path.Join(webroot, filepath.ToSlash(filepath.Dir(relDestPath))))
	}
}
