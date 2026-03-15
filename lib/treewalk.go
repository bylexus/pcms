package lib

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"alexi.ch/pcms/model"
	"alexi.ch/pcms/stdlib"
	"github.com/gabriel-vasile/mimetype"
)

func BuildIndexSnapshot(srcFS fs.FS, excludePatterns []string) (*model.IndexSnapshot, error) {
	snapshot := &model.IndexSnapshot{
		Pages: make([]model.IndexedPage, 0),
		Files: make([]model.IndexedFile, 0),
	}

	if err := walkIndexTree(srcFS, ".", "/", nil, excludePatterns, snapshot); err != nil {
		return nil, err
	}

	return snapshot, nil
}

func walkIndexTree(srcFS fs.FS, relDir string, route string, inheritedParentPageRoute *string, excludePatterns []string, snapshot *model.IndexSnapshot) error {
	entries, err := fs.ReadDir(srcFS, relDir)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", relDir, err)
	}

	indexFileName := ""
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		entryRoute := joinRoute(route, entry.Name())
		isExcluded, _ := isFileExcluded(entryRoute, excludePatterns)
		if isExcluded {
			continue
		}
		if isSupportedIndexFile(entry.Name()) {
			if indexFileName == "" || entry.Name() < indexFileName {
				indexFileName = entry.Name()
			}
		}
	}

	currentPageRoute := (*string)(nil)
	if indexFileName != "" {
		pageTitle := defaultTitleForRoute(route)
		pageMetadata := make(stdlib.YamlFrontMatter)
		indexPath := indexFileName
		if relDir != "." {
			indexPath = path.Join(relDir, indexFileName)
		}
		metadata, title, err := parsePageIndexFrontmatter(srcFS, indexPath, pageTitle)
		if err != nil {
			return err
		}
		pageMetadata = metadata
		pageTitle = title

		snapshot.Pages = append(snapshot.Pages, model.IndexedPage{
			Route:           route,
			ParentPageRoute: inheritedParentPageRoute,
			Title:           pageTitle,
			IndexFile:       indexFileName,
			Metadata:        pageMetadata,
		})

		pageSource := indexFileName
		if relDir != "." {
			pageSource = path.Join(relDir, indexFileName)
		}
		fmt.Printf("type=page file=%s route=%s\n", pageSource, route)

		currentRoute := route
		currentPageRoute = &currentRoute
	}

	activeParentPageRoute := inheritedParentPageRoute
	if currentPageRoute != nil {
		activeParentPageRoute = currentPageRoute
	}

	for _, entry := range entries {
		entryRoute := joinRoute(route, entry.Name())
		if entry.IsDir() {
			isExcluded, _ := isFileExcluded(entryRoute, excludePatterns)
			if isExcluded {
				continue
			}

			nextRelDir := entry.Name()
			if relDir != "." {
				nextRelDir = path.Join(relDir, entry.Name())
			}
			if err := walkIndexTree(srcFS, nextRelDir, entryRoute, activeParentPageRoute, excludePatterns, snapshot); err != nil {
				return err
			}
			continue
		}

		isExcluded, _ := isFileExcluded(entryRoute, excludePatterns)
		if isExcluded {
			continue
		}
		if entry.Name() == indexFileName {
			continue
		}
		if activeParentPageRoute == nil {
			continue
		}

		entryInfo, err := entry.Info()
		if err != nil {
			return fmt.Errorf("read file info %s: %w", entryRoute, err)
		}

		filePath := entry.Name()
		if relDir != "." {
			filePath = path.Join(relDir, entry.Name())
		}
		mimeType, err := detectMimeTypeFromFS(srcFS, filePath)
		if err != nil {
			return err
		}

		snapshot.Files = append(snapshot.Files, model.IndexedFile{
			Route:           entryRoute,
			ParentPageRoute: *activeParentPageRoute,
			FileName:        entry.Name(),
			MimeType:        mimeType,
			FileSize:        entryInfo.Size(),
		})
		fmt.Printf("type=file file=%s route=%s mime=%s\n", filePath, entryRoute, mimeType)
	}

	return nil
}

func parsePageIndexFrontmatter(srcFS fs.FS, indexPath string, fallbackTitle string) (stdlib.YamlFrontMatter, string, error) {
	content, err := fs.ReadFile(srcFS, indexPath)
	if err != nil {
		return nil, "", fmt.Errorf("read index file %s: %w", indexPath, err)
	}

	metadata, _, err := stdlib.ExtractYamlFrontMatter(string(content))
	if err != nil {
		return nil, "", fmt.Errorf("parse frontmatter in %s: %w", indexPath, err)
	}

	title := fallbackTitle
	if rawTitle, hasTitle := metadata["title"]; hasTitle {
		title = fmt.Sprintf("%v", rawTitle)
	}

	return metadata, title, nil
}

func joinRoute(parentRoute string, name string) string {
	if parentRoute == "/" {
		return path.Clean("/" + name)
	}
	return path.Clean(path.Join(parentRoute, name))
}

func defaultTitleForRoute(route string) string {
	if route == "/" {
		return "/"
	}
	base := path.Base(route)
	if base == "." || base == "/" {
		return "/"
	}
	return base
}


func detectMimeTypeFromFS(srcFS fs.FS, filePath string) (string, error) {
	ext := filepath.Ext(filePath)
	switch ext {
	case ".css":
		return "text/css", nil
	case ".js":
		return "application/javascript", nil
	}

	file, err := srcFS.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file for mime type %s: %w", filePath, err)
	}
	defer file.Close()

	mtype, err := mimetype.DetectReader(file)
	if err != nil {
		return "", fmt.Errorf("detect file mime type %s: %w", filePath, err)
	}

	return mtype.String(), nil
}

func isSupportedIndexFile(fileName string) bool {
	base := strings.ToLower(filepath.Base(fileName))
	return base == "index.html" || base == "index.md"
}

// Checks if the given file matches a set of exclude regex patterns.
// The relative path within the source dir is used as input.
//
// Returns true and the matching pattern if the file name matches a exclude pattern.
func isFileExcluded(filePath string, excludePatterns []string) (bool, string) {
	skipfiles := []string{"variables.yaml"}
	if stdlib.InSlice(&skipfiles, filepath.Base(filePath)) {
		return true, filepath.Base(filePath)
	}
	for _, pattern := range excludePatterns {
		r := regexp.MustCompile(pattern)
		if r.MatchString(filePath) {
			return true, pattern
		}
	}

	return false, ""
}
