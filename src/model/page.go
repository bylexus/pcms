package model

import (
	"path"
	"path/filepath"
)

const (
	PAGE_TYPE_UNKNOWN  = iota
	PAGE_TYPE_HTML     = iota
	PAGE_TYPE_MARKDOWN = iota
	PAGE_TYPE_JSON     = iota
	PAGE_TYPE_THEME    = iota
)

type PageType int

type Page struct {
	Type       PageType
	Route      string
	Title      string
	ShortTitle string
	IndexFile  string
	Dir        string
	Metadata   Metadata
	Parent     *Page
	Template   string
	Children   []*Page
}

func (p *Page) ExtractKnownMetadata(meta Metadata) {
	var present bool

	p.Title, present = meta["title"].(string)
	if present != true {
		p.Title = ""
	}

	p.ShortTitle, present = meta["shortTitle"].(string)
	if present != true {
		p.ShortTitle = ""
	}

	p.IndexFile, present = meta["index"].(string)
	if present != true {
		p.IndexFile = "index.html"
	}

	p.Template, present = meta["template"].(string)
	if present != true {
		p.Template = "index.html"
	}

	ext := filepath.Ext(p.IndexFile)
	switch ext {
	case ".html":
		p.Type = PAGE_TYPE_HTML
	case ".md":
		p.Type = PAGE_TYPE_MARKDOWN
	case ".json":
		p.Type = PAGE_TYPE_JSON
	default:
		p.Type = PAGE_TYPE_UNKNOWN
	}
}

func (p *Page) PageJsonPath() string {
	return path.Join(p.Dir, "page.json")
}

func (p *Page) PageIndexPath() string {
	return path.Join(p.Dir, p.IndexFile)
}

func (p *Page) IsPageRoute(route string) bool {
	if p.Route+"/page.json" == route {
		return true
	}

	if p.Route+"/"+p.IndexFile == route {
		return true
	}
	if p.Route == route {
		return true
	}
	return false
}
