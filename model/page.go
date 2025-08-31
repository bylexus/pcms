package model

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"strings"

	"alexi.ch/pcms/stdlib"
)

type PageType int

const (
	PAGE_TYPE_HTML = iota
	PAGE_TYPE_MARKDOWN
)

type Page struct {
	ParentPage *Page
	PageFS     fs.FS
	Type       PageType
	IndexFile  string
	Route      string
	Metadata   stdlib.YamlFrontMatter
	ChildPages []*Page
	Files      []string
}

func BuildPageTree(config Config) (*Page, error) {

	var siteFS fs.FS
	var err error

	switch config.ServeMode {
	case SERVE_MODE_EMBEDDED_DOC:
		fmt.Println("Using embedded doc site to build page tree")
		siteFS, err = fs.Sub(config.EmbeddedDocFS, "doc/build")
		if err != nil {
			return nil, err
		}
	default:
		// static site folder as FS:
		fmt.Printf("Using content from %s to build page tree\n", config.SourcePath)
		siteFS = os.DirFS(config.SourcePath)
	}

	page, err := BuildPageFromFS(siteFS, nil)
	return page, err

}

func BuildPageFromFS(dirFS fs.FS, parentPage *Page) (*Page, error) {
	// Read the given FS's info, to gather the name and meta info:
	dirInfo, err := fs.Stat(dirFS, ".")
	if err != nil {
		return nil, err
	}
	if !dirInfo.IsDir() {
		return nil, errors.New("not a dir")
	}

	var page = Page{
		PageFS:     dirFS,
		ParentPage: parentPage,
		ChildPages: make([]*Page, 0),
		Files:      make([]string, 0),
	}
	if parentPage == nil {
		page.Route = "/"
	} else {
		page.Route, err = url.JoinPath(parentPage.Route, dirInfo.Name())
		if err != nil {
			return nil, err
		}
	}

	// Process the dir's entries:
	entries, err := fs.ReadDir(dirFS, ".")
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() {
			// fmt.Printf("Dir: %s/%s\n", page.Route, e.Name())
			subFS, err := fs.Sub(dirFS, e.Name())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while processing dir %s/%s: %s\n", page.Route, e.Name(), err)
			}
			subPage, err := BuildPageFromFS(subFS, &page)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error while constructing Page from dir %s/%s: %s\n", page.Route, e.Name(), err)
			}
			page.ChildPages = append(page.ChildPages, subPage)
		} else {
			// fmt.Printf("File: %s/%s\n", page.Route, e.Name())

			// Index pages define the type of the page. All other files to go the raw Files array:
			switch strings.ToLower(e.Name()) {
			case "index.html":
				page.Type = PAGE_TYPE_HTML
				page.IndexFile = e.Name()
			case "index.md":
				page.Type = PAGE_TYPE_MARKDOWN
				page.IndexFile = e.Name()
			default:
				page.Files = append(page.Files, e.Name())
			}
		}
	}

	// read all page's metadata from the YAML front matter:
	page.extractMetadata()

	return &page, nil
}

func (p *Page) Routes() []string {
	routes := make([]string, 0)
	actRoute := p.Route
	routes = append(routes, actRoute)
	for _, child := range p.ChildPages {
		childRoutes := child.Routes()
		routes = append(routes, childRoutes...)
	}

	for _, child := range p.Files {
		childRoute, err := url.JoinPath(actRoute, child)
		if err == nil {
			routes = append(routes, childRoute)
		}
	}
	return routes
}

func (p *Page) extractMetadata() {
	if len(p.IndexFile) > 0 {
		meta, _, err := stdlib.ExtractYamlFrontMatterFromFS(p.PageFS, p.IndexFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error extracting metadata: %s\n", err)
		}
		p.Metadata = meta
	}
}
