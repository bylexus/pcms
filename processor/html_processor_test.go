package processor

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"alexi.ch/pcms/model"
)

/*
Tests all the file paths returned by html_processor.prepareFilePaths
*/
func TestHtmlProcessorPrepareFilePathsOnWebroot(t *testing.T) {
	var processor = HtmlProcessor{}
	var sourceRoot = filepath.Join(os.TempDir(), filepath.FromSlash("/path/to/source"))
	var sourceFile = filepath.Join(sourceRoot, "sub", "folder", "file.html")
	var RelSourcePath = filepath.Join("sub", "folder", "file.html")
	var RelSourceRoot = filepath.Join("..", "..")

	var destRoot = filepath.Join(os.TempDir(), filepath.FromSlash("/path/to/dest"))
	var destFile = filepath.Join(destRoot, "sub", "folder", "file.html")
	var RelDestPath = filepath.Join("sub", "folder", "file.html")
	var RelDestRoot = filepath.Join("..", "..")

	var Webroot = "/web/root"

	var config = model.Config{
		SourcePath: sourceRoot,
		DestPath:   destRoot,
		Server: struct {
			Listen  string              "yaml:\"listen\""
			Watch   bool                "yaml:\"watch\""
			Prefix  string              "yaml:\"prefix\""
			Logging model.LoggingConfig "yaml:\"logging\""
		}{
			Prefix: Webroot,
		},
	}
	res, err := processor.prepareFilePaths(sourceFile, config)
	if err != nil {
		t.Fatal(err)
	}
	// start / top path of the source folder
	// RootSourceDir
	if res.RootSourceDir != sourceRoot {
		t.Errorf("RootSourceDir = %s; want %s", res.RootSourceDir, sourceRoot)
	}

	// absolute path of the actual source file
	// AbsSourcePath
	if res.AbsSourcePath != sourceFile {
		t.Errorf("AbsSourcePath = %s; want %s", res.AbsSourcePath, sourceFile)
	}

	// absolute path of the actual source file
	// AbsSourceDir string
	if res.AbsSourceDir != filepath.Dir(sourceFile) {
		t.Errorf("AbsSourceDir = %s; want %s", res.AbsSourceDir, filepath.Dir(sourceFile))
	}

	// file path of the actual source file relative to the RootSourceDir
	// RelSourcePath
	if res.RelSourcePath != RelSourcePath {
		t.Errorf("RelSourcePath = %s; want %s", res.RelSourcePath, RelSourcePath)
	}

	// dir path of the actual source file relative to the RootSourceDir
	// RelSourceDir
	if res.RelSourceDir != filepath.Dir(RelSourcePath) {
		t.Errorf("RelSourceDir = %s; want %s", res.RelSourceDir, filepath.Dir(RelSourcePath))
	}

	// // relative path from the actual source file back to the RootSourceDir
	// RelSourceRoot string
	if res.RelSourceRoot != RelSourceRoot {
		t.Errorf("RelSourceRoot = %s; want %s", res.RelSourceRoot, RelSourceRoot)
	}

	// start / top path of the destination folder
	// RootDestDir string
	if res.RootDestDir != destRoot {
		t.Errorf("RootDestDir = %s; want %s", res.RootDestDir, RelSourceRoot)
	}

	// absolute path of the actual destination file
	// AbsDestPath
	if res.AbsDestPath != destFile {
		t.Errorf("AbsDestPath = %s; want %s", res.AbsDestPath, destFile)
	}

	// absolute path of the actual destination file
	// AbsDestDir
	if res.AbsDestDir != filepath.Dir(destFile) {
		t.Errorf("AbsDestDir = %s; want %s", res.AbsDestDir, filepath.Dir(destFile))
	}

	// file path of the actual destination file relative to the RootDestDir
	// RelDestPath
	if res.RelDestPath != RelDestPath {
		t.Errorf("RelDestPath = %s; want %s", res.RelDestPath, RelDestPath)
	}

	// dir path of the actual destination file relative to the RootSourceDir
	// RelDestDir string
	if res.RelDestDir != filepath.Dir(RelDestPath) {
		t.Errorf("RelDestDir = %s; want %s", res.RelDestDir, filepath.Dir(RelDestPath))
	}

	// relative path from the actual dest file back to the RootSourceDir
	// RelDestRoot string
	if res.RelDestRoot != RelDestRoot {
		t.Errorf("RelDestRoot = %s; want %s", res.RelDestRoot, RelDestRoot)
	}

	// web paths:
	// the Webroot prefix, "/" by default
	// Webroot string
	if res.Webroot != Webroot {
		t.Errorf("Webroot = %s; want %s", res.Webroot, Webroot)
	}

	// relative (to Webroot) web path to the actual output file
	// relWebPath string
	if res.RelWebPath != filepath.ToSlash(RelDestPath) {
		t.Errorf("relWebPath = %s; want %s", res.RelWebPath, filepath.ToSlash(RelDestPath))
	}

	// relative (to Webroot) web path to the actual output file's folder
	// RelWebDir string
	if res.RelWebDir != filepath.ToSlash(filepath.Dir(RelDestPath)) {
		t.Errorf("RelWebDir = %s; want %s", res.RelWebDir, filepath.ToSlash(filepath.Dir(RelDestPath)))
	}

	// relative path from the actual file back to the Webroot
	// RelWebPathToRoot string
	if res.RelWebPathToRoot != filepath.ToSlash(RelDestRoot) {
		t.Errorf("RelWebPathToRoot = %s; want %s", res.RelWebPathToRoot, filepath.ToSlash(RelDestRoot))
	}

	// // absolute web path of the actual file, including the Webroot, starting always with "/"
	// AbsWebPath string
	if res.AbsWebPath != path.Clean(path.Join("/", Webroot, filepath.ToSlash(RelDestPath))) {
		t.Errorf("AbsWebPath = %s; want %s", res.AbsWebPath, path.Join(Webroot, filepath.ToSlash(RelDestPath)))
	}

	// absolute web path of the actual file's dir, including the Webroot, starting always with "/"
	// AbsWebDir string
	if res.AbsWebDir != path.Clean(path.Join("/", Webroot, filepath.ToSlash(filepath.Dir(RelDestPath)))) {
		t.Errorf("AbsWebDir = %s; want %s", res.AbsWebDir, path.Join(Webroot, filepath.ToSlash(filepath.Dir(RelDestPath))))
	}
}

func TestHtmlProcessorPrepareFilePathsWithoutWebroot(t *testing.T) {
	var processor = HtmlProcessor{}
	var sourceRoot = filepath.Join(os.TempDir(), filepath.FromSlash("/path/to/source"))
	var sourceFile = filepath.Join(sourceRoot, "sub", "folder", "file.html")
	var RelSourcePath = filepath.Join("sub", "folder", "file.html")
	var RelSourceRoot = filepath.Join("..", "..")

	var destRoot = filepath.Join(os.TempDir(), filepath.FromSlash("/path/to/dest"))
	var destFile = filepath.Join(destRoot, "sub", "folder", "file.html")
	var RelDestPath = filepath.Join("sub", "folder", "file.html")
	var RelDestRoot = filepath.Join("..", "..")

	var Webroot = ""

	var config = model.Config{
		SourcePath: sourceRoot,
		DestPath:   destRoot,
		Server: struct {
			Listen  string              "yaml:\"listen\""
			Watch   bool                "yaml:\"watch\""
			Prefix  string              "yaml:\"prefix\""
			Logging model.LoggingConfig "yaml:\"logging\""
		}{
			Prefix: Webroot,
		},
	}
	res, err := processor.prepareFilePaths(sourceFile, config)
	if err != nil {
		t.Fatal(err)
	}
	// start / top path of the source folder
	// RootSourceDir
	if res.RootSourceDir != sourceRoot {
		t.Errorf("RootSourceDir = %s; want %s", res.RootSourceDir, sourceRoot)
	}

	// absolute path of the actual source file
	// AbsSourcePath
	if res.AbsSourcePath != sourceFile {
		t.Errorf("AbsSourcePath = %s; want %s", res.AbsSourcePath, sourceFile)
	}

	// absolute path of the actual source file
	// AbsSourceDir string
	if res.AbsSourceDir != filepath.Dir(sourceFile) {
		t.Errorf("AbsSourceDir = %s; want %s", res.AbsSourceDir, filepath.Dir(sourceFile))
	}

	// file path of the actual source file relative to the RootSourceDir
	// RelSourcePath
	if res.RelSourcePath != RelSourcePath {
		t.Errorf("RelSourcePath = %s; want %s", res.RelSourcePath, RelSourcePath)
	}

	// dir path of the actual source file relative to the RootSourceDir
	// RelSourceDir
	if res.RelSourceDir != filepath.Dir(RelSourcePath) {
		t.Errorf("RelSourceDir = %s; want %s", res.RelSourceDir, filepath.Dir(RelSourcePath))
	}

	// // relative path from the actual source file back to the RootSourceDir
	// RelSourceRoot string
	if res.RelSourceRoot != RelSourceRoot {
		t.Errorf("RelSourceRoot = %s; want %s", res.RelSourceRoot, RelSourceRoot)
	}

	// start / top path of the destination folder
	// RootDestDir string
	if res.RootDestDir != destRoot {
		t.Errorf("RootDestDir = %s; want %s", res.RootDestDir, RelSourceRoot)
	}

	// absolute path of the actual destination file
	// AbsDestPath
	if res.AbsDestPath != destFile {
		t.Errorf("AbsDestPath = %s; want %s", res.AbsDestPath, destFile)
	}

	// absolute path of the actual destination file
	// AbsDestDir
	if res.AbsDestDir != filepath.Dir(destFile) {
		t.Errorf("AbsDestDir = %s; want %s", res.AbsDestDir, filepath.Dir(destFile))
	}

	// file path of the actual destination file relative to the RootDestDir
	// RelDestPath
	if res.RelDestPath != RelDestPath {
		t.Errorf("RelDestPath = %s; want %s", res.RelDestPath, RelDestPath)
	}

	// dir path of the actual destination file relative to the RootSourceDir
	// RelDestDir string
	if res.RelDestDir != filepath.Dir(RelDestPath) {
		t.Errorf("RelDestDir = %s; want %s", res.RelDestDir, filepath.Dir(RelDestPath))
	}

	// relative path from the actual dest file back to the RootSourceDir
	// RelDestRoot string
	if res.RelDestRoot != RelDestRoot {
		t.Errorf("RelDestRoot = %s; want %s", res.RelDestRoot, RelDestRoot)
	}

	// web paths:
	// the Webroot prefix, "/" by default
	// Webroot string
	if res.Webroot != Webroot {
		t.Errorf("Webroot = %s; want %s", res.Webroot, Webroot)
	}

	// relative (to Webroot) web path to the actual output file
	// relWebPath string
	if res.RelWebPath != filepath.ToSlash(RelDestPath) {
		t.Errorf("relWebPath = %s; want %s", res.RelWebPath, filepath.ToSlash(RelDestPath))
	}

	// relative (to Webroot) web path to the actual output file's folder
	// RelWebDir string
	if res.RelWebDir != filepath.ToSlash(filepath.Dir(RelDestPath)) {
		t.Errorf("RelWebDir = %s; want %s", res.RelWebDir, filepath.ToSlash(filepath.Dir(RelDestPath)))
	}

	// relative path from the actual file back to the Webroot
	// RelWebPathToRoot string
	if res.RelWebPathToRoot != filepath.ToSlash(RelDestRoot) {
		t.Errorf("RelWebPathToRoot = %s; want %s", res.RelWebPathToRoot, filepath.ToSlash(RelDestRoot))
	}

	// // absolute web path of the actual file, including the Webroot, starting always with "/"
	// AbsWebPath string
	if res.AbsWebPath != path.Clean(path.Join("/", Webroot, filepath.ToSlash(RelDestPath))) {
		t.Errorf("AbsWebPath = %s; want %s", res.AbsWebPath, path.Join(Webroot, filepath.ToSlash(RelDestPath)))
	}

	// absolute web path of the actual file's dir, including the Webroot, starting always with "/"
	// AbsWebDir string
	if res.AbsWebDir != path.Clean(path.Join("/", Webroot, filepath.ToSlash(filepath.Dir(RelDestPath)))) {
		t.Errorf("AbsWebDir = %s; want %s", res.AbsWebDir, path.Join(Webroot, filepath.ToSlash(filepath.Dir(RelDestPath))))
	}
}
