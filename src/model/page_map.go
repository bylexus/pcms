package model

import (
	"errors"
	"log"
	"path/filepath"
	"strings"
)

type PageMap map[string]*Page

var rootPage *Page = nil

func (pm *PageMap) BuildPageTree() {
	rootPage = nil
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
