package model

import (
	"errors"
	"log"
	"path/filepath"
	"sort"
	"strings"
)

type PageMap map[string]*Page

var rootPage *Page = nil

func (pm *PageMap) BuildPageTree() {
	rootPage = nil
	// build tree:
	for route, page := range *pm {
		log.Printf("Finding parent for %s\n", route)
		parentPage, _ := pm.FindMatchingRoute(filepath.Dir(page.Route))
		if parentPage != nil && parentPage != page {
			page.Parent = parentPage
			parentPage.Children = append(parentPage.Children, page)
			log.Printf("    found parent: %s\n", parentPage.Route)
		} else {
			log.Print("    found no parent.\n")
		}
	}

	// sort children by their Metadata.order property:
	for _, page := range *pm {
		sortChildsByOrderMeta(page)
	}
}

func sortChildsByOrderMeta(page *Page) {
	sort.Slice(page.Children, func(i, j int) bool {
		c1 := page.Children[i]
		c2 := page.Children[j]
		switch c1.Metadata["order"].(type) {
		case int:
			v1 := c1.Metadata["order"].(int)
			v2, ok := c2.Metadata["order"].(int)
			if ok {
				return v1 < v2
			} else {
				return false
			}
		case float64:
			v1 := c1.Metadata["order"].(float64)
			v2, ok := c2.Metadata["order"].(float64)
			if ok {
				return v1 < v2
			} else {
				return false
			}
		case string:
			v1 := c1.Metadata["order"].(string)
			v2, ok := c2.Metadata["order"].(string)
			if ok {
				return v1 < v2
			} else {
				return false
			}
		default:
			return true
		}
	})
}

func (pm *PageMap) GetRootPage() *Page {
	if rootPage == nil {
		for _, page := range *pm {
			if page.Parent == nil {
				rootPage = page
				break
			}
		}
	}
	return rootPage
}

func (pm *PageMap) FindMatchingRoute(routePath string) (*Page, error) {
	for parts := strings.Split(routePath, "/"); len(parts) > 0; parts = parts[:len(parts)-1] {
		partialRoute := strings.Join(parts, "/")
		if partialRoute == "" {
			partialRoute = "/"
		}
		page, _ := (*pm)[partialRoute]
		if page != nil {
			return page, nil
		}
	}
	return nil, errors.New("Page not found for route " + routePath)
}
