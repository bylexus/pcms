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

	if err := walkIndexTree(srcFS, ".", "/", nil, excludePatterns, snapshot, true); err != nil {
		return nil, err
	}

	return snapshot, nil
}

// walkIndexTree recursively walks the source filesystem and builds the index snapshot.
// parentEffectivelyEnabled carries the effective enabled state of the nearest ancestor
// page so that disabled parents force all descendants to also be disabled in the index.
func walkIndexTree(srcFS fs.FS, relDir string, route string, inheritedParentPageRoute *string, excludePatterns []string, snapshot *model.IndexSnapshot, parentEffectivelyEnabled bool) error {
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
		fm, err := parsePageIndexFrontmatter(srcFS, indexPath, pageTitle)
		if err != nil {
			return err
		}
		pageMetadata = fm.Metadata
		pageTitle = fm.Title

		// Effective enabled: own flag AND all ancestor pages must be enabled.
		// parentEffectivelyEnabled already encodes the full ancestor chain, so
		// a single AND is sufficient.
		effectiveEnabled := fm.Enabled && parentEffectivelyEnabled

		snapshot.Pages = append(snapshot.Pages, model.IndexedPage{
			Route:           route,
			ParentPageRoute: inheritedParentPageRoute,
			Title:           pageTitle,
			IndexFile:       indexFileName,
			Enabled:         effectiveEnabled,
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
	// childEffectivelyEnabled tracks the effective enabled state to pass into
	// subdirectories. If this directory introduced a page, use its effective
	// enabled state; otherwise propagate the inherited one.
	childEffectivelyEnabled := parentEffectivelyEnabled
	if currentPageRoute != nil {
		activeParentPageRoute = currentPageRoute
		// Look up the effective enabled that was stored for this page.
		// Since snapshot.Pages is append-only and we just added it, it's the last element.
		childEffectivelyEnabled = snapshot.Pages[len(snapshot.Pages)-1].Enabled
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
			if err := walkIndexTree(srcFS, nextRelDir, entryRoute, activeParentPageRoute, excludePatterns, snapshot, childEffectivelyEnabled); err != nil {
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
			Enabled:         childEffectivelyEnabled,
		})
		fmt.Printf("type=file file=%s route=%s mime=%s\n", filePath, entryRoute, mimeType)
	}

	return nil
}

type parsedFrontmatter struct {
	Metadata stdlib.YamlFrontMatter
	Title    string
	Enabled  bool
}

func parsePageIndexFrontmatter(srcFS fs.FS, indexPath string, fallbackTitle string) (parsedFrontmatter, error) {
	content, err := fs.ReadFile(srcFS, indexPath)
	if err != nil {
		return parsedFrontmatter{}, fmt.Errorf("read index file %s: %w", indexPath, err)
	}

	metadata, _, err := stdlib.ExtractYamlFrontMatter(string(content))
	if err != nil {
		return parsedFrontmatter{}, fmt.Errorf("parse frontmatter in %s: %w", indexPath, err)
	}

	title := fallbackTitle
	if rawTitle, hasTitle := metadata["title"]; hasTitle {
		title = fmt.Sprintf("%v", rawTitle)
	}

	enabled := true
	if rawEnabled, hasEnabled := metadata["enabled"]; hasEnabled {
		switch v := rawEnabled.(type) {
		case bool:
			enabled = v
		}
	}

	return parsedFrontmatter{Metadata: metadata, Title: title, Enabled: enabled}, nil
}

// ReindexSinglePage re-reads the frontmatter from the source file and returns
// an updated IndexedPage. The caller is responsible for persisting it via ReplacePage.
func ReindexSinglePage(srcFS fs.FS, route string, existingPage model.IndexedPage) (model.IndexedPage, error) {
	indexPath := existingPage.IndexFile
	if route != "/" {
		indexPath = path.Join(strings.TrimPrefix(route, "/"), existingPage.IndexFile)
	}

	fm, err := parsePageIndexFrontmatter(srcFS, indexPath, defaultTitleForRoute(route))
	if err != nil {
		return model.IndexedPage{}, err
	}

	return model.IndexedPage{
		Route:           route,
		ParentPageRoute: existingPage.ParentPageRoute,
		Title:           fm.Title,
		IndexFile:       existingPage.IndexFile,
		Enabled:         fm.Enabled,
		Metadata:        fm.Metadata,
	}, nil
}

func joinRoute(parentRoute string, name string) string {
	if parentRoute == "/" {
		return path.Clean("/" + name)
	}
	return path.Clean(path.Join(parentRoute, name))
}

func defaultTitleForRoute(route string) string {
	if route == "/" {
		return ""
	}
	base := path.Base(route)
	if base == "." || base == "/" {
		return ""
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
	for _, pattern := range excludePatterns {
		r := regexp.MustCompile(pattern)
		if r.MatchString(filePath) {
			return true, pattern
		}
	}

	return false, ""
}
