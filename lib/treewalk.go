package lib

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"

	"alexi.ch/pcms/processor"
	"alexi.ch/pcms/stdlib"
	"github.com/gabriel-vasile/mimetype"
)

type IndexedPageRecord struct {
	Route           string
	ParentPageRoute *string
	Title           string
	IndexFile       string
	MetadataJSON    string
}

type IndexedFileRecord struct {
	Route           string
	ParentPageRoute string
	FileName        string
	MimeType        string
	FileSize        int64
	MetadataJSON    string
}

type IndexSnapshot struct {
	Pages []IndexedPageRecord
	Files []IndexedFileRecord
}

func BuildIndexSnapshot(srcFS fs.FS, excludePatterns []string) (*IndexSnapshot, error) {
	snapshot := &IndexSnapshot{
		Pages: make([]IndexedPageRecord, 0),
		Files: make([]IndexedFileRecord, 0),
	}

	if err := walkIndexTree(srcFS, ".", "/", nil, excludePatterns, snapshot); err != nil {
		return nil, err
	}

	return snapshot, nil
}

func walkIndexTree(srcFS fs.FS, relDir string, route string, parentRoute *string, excludePatterns []string, snapshot *IndexSnapshot) error {
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
		isExcluded, _ := processor.IsFileExcluded(entryRoute, excludePatterns)
		if isExcluded {
			continue
		}
		if isSupportedIndexFile(entry.Name()) {
			if indexFileName == "" || entry.Name() < indexFileName {
				indexFileName = entry.Name()
			}
		}
	}

	pageTitle := defaultTitleForRoute(route)
	pageMetadata := make(stdlib.YamlFrontMatter)
	if indexFileName != "" {
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
	}

	pageMetadataJSON, err := toJSON(pageMetadata)
	if err != nil {
		return fmt.Errorf("marshal page metadata for %s: %w", route, err)
	}

	snapshot.Pages = append(snapshot.Pages, IndexedPageRecord{
		Route:           route,
		ParentPageRoute: parentRoute,
		Title:           pageTitle,
		IndexFile:       indexFileName,
		MetadataJSON:    pageMetadataJSON,
	})

	pageSource := relDir
	if pageSource == "." {
		pageSource = "/"
	}
	if indexFileName != "" {
		if relDir == "." {
			pageSource = indexFileName
		} else {
			pageSource = path.Join(relDir, indexFileName)
		}
	}
	fmt.Printf("type=page file=%s route=%s\n", pageSource, route)

	for _, entry := range entries {
		entryRoute := joinRoute(route, entry.Name())
		if entry.IsDir() {
			isExcluded, _ := processor.IsFileExcluded(entryRoute, excludePatterns)
			if isExcluded {
				continue
			}

			nextRelDir := entry.Name()
			if relDir != "." {
				nextRelDir = path.Join(relDir, entry.Name())
			}
			currentRoute := route
			if err := walkIndexTree(srcFS, nextRelDir, entryRoute, &currentRoute, excludePatterns, snapshot); err != nil {
				return err
			}
			continue
		}

		isExcluded, _ := processor.IsFileExcluded(entryRoute, excludePatterns)
		if isExcluded {
			continue
		}
		if entry.Name() == indexFileName {
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

		snapshot.Files = append(snapshot.Files, IndexedFileRecord{
			Route:           entryRoute,
			ParentPageRoute: route,
			FileName:        entry.Name(),
			MimeType:        mimeType,
			FileSize:        entryInfo.Size(),
			MetadataJSON:    "{}",
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

func isSupportedIndexFile(name string) bool {
	return name == "index.html" || name == "index.md"
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

func toJSON(obj interface{}) (string, error) {
	raw, err := json.Marshal(obj)
	if err != nil {
		return "", err
	}
	if len(raw) == 0 {
		return "{}", nil
	}
	return string(raw), nil
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
