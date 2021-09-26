package webserver

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/flosch/pongo2/v4"
	"github.com/gabriel-vasile/mimetype"

	"alexi.ch/pcms/src/model"
)

type RequestHandler struct {
	ServerConfig *model.Config
	Pages        *model.PageMap
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route := getRoute(req, h.ServerConfig)
	page, err := h.findMatchingRoute(route)
	if page != nil {

		if page.IsPageRoute(route) {
			// deliver page itself
			h.renderPage(page, w, req)
		} else {
			// deliver a child file of the route:
			h.deliverStaticFile(route, page, w, req)
		}

	} else {
		h.errorHandler(w, err, 404)
		return
	}
}

func getRoute(req *http.Request, serverConf *model.Config) string {
	// remove trailing '/'
	route := strings.TrimRight(req.URL.Path, "/")
	if route == "" {
		return "/"
	} else {
		return route
	}
}

/**
 * Generic error handler, sets an HTTP error code and outputs the specified erro
 */
func (h *RequestHandler) errorHandler(w http.ResponseWriter, err error, status int) {
	w.WriteHeader(status)
	w.Write([]byte(fmt.Sprintf("error: %v\n", err)))
}

func (h *RequestHandler) findMatchingRoute(routePath string) (*model.Page, error) {
	return h.Pages.FindMatchingRoute(routePath)
}

func (h *RequestHandler) findMimeType(file string) (string, error) {
	ext := filepath.Ext(file)
	switch ext {
	case ".css":
		return "text/css", nil
	case ".js":
		return "application/javascript", nil
	}
	mtype, err := mimetype.DetectFile(file)
	if err != nil {
		return "", err
	}
	return mtype.String(), nil
}

/**
 * renders a page and outputs it (html, markdown, ...).
 */
func (h *RequestHandler) renderPage(page *model.Page, w http.ResponseWriter, req *http.Request) {
	// w.Write([]byte(fmt.Sprintf("URL Path: %v\n", req.URL.Path)))
	// w.Write([]byte(fmt.Sprint("Delivering page ", page.Title)))
	w.Header().Add("Content-Type", "text/html")

	content, err := ioutil.ReadFile(page.PageIndexPath())
	if err != nil {
		h.errorHandler(w, err, 500)
		return
	}
	switch page.Type {
	case model.PAGE_TYPE_HTML:
		content, err = h.renderTemplate(page, &content)
		if err != nil {
			h.errorHandler(w, err, http.StatusInternalServerError)
		}
		w.Write(content)
	default:
		w.Write([]byte(fmt.Sprintf("Not yet implemented - page type %v", page.Type)))
	}
}

var templateLoaderAdded bool = false

func (h *RequestHandler) renderTemplate(page *model.Page, content *[]byte) ([]byte, error) {
	if templateLoaderAdded != true {
		loader, err3 := pongo2.NewLocalFileSystemLoader(filepath.Join(h.ServerConfig.Site.ThemePath, "templates"))
		if err3 != nil {
			return nil, err3
		}
		pongo2.DefaultSet.AddLoader(loader)
		templateLoaderAdded = true
	}

	tpl, err := pongo2.FromBytes(*content)
	if err != nil {
		return nil, err
	}
	out, err2 := tpl.ExecuteBytes(pongo2.Context{
		"site":     h.ServerConfig.Site,
		"page":     page,
		"rootPage": h.Pages.GetRootPage(),
		"base":     h.ServerConfig.Site.Webroot,
		"meta":     page.Metadata["metaTags"],
	})

	rp := h.Pages.GetRootPage()
	fmt.Print(rp.Title)

	if err2 != nil {
		return nil, err2
	}
	return out, nil
}

/**
 * Delivers a static file, which must be part of the site root
 */
func (h *RequestHandler) deliverStaticFile(route string, page *model.Page, w http.ResponseWriter, req *http.Request) {
	// create an absolute path, to strip out path traversal sequences like ".."
	// create a relative route: the route given must/should be a sub-route of the page's route, so strip
	// away the page part (e.g. route = /foo/bar/image.jpg, page.Route = /foo/bar --> relativeRoute = /image.jpg)
	relativeRoute := strings.TrimPrefix(route, page.Route)

	// create a file path for the route:
	filePath, err := filepath.Abs(filepath.Join(page.Dir, relativeRoute))

	if err != nil {
		h.errorHandler(w, err, 500)
		return
	}

	// security check: is the requested file in the route dir?
	if strings.HasPrefix(filePath, page.Dir) != true {
		h.errorHandler(w, errors.New(fmt.Sprintf("error: %v\n", "not in route dir")), 404)
		return
	}

	// If the route points to a regular file, deliver ite:
	finfo, err2 := os.Stat(filePath)
	if err2 != nil {
		h.errorHandler(w, err2, 404)
		return
	}
	if finfo.Mode().IsRegular() {
		mtype, err4 := h.findMimeType(filePath)
		if err4 == nil {
			w.Header().Add("Content-Type", mtype)
		}
		// TODO: stream file content instead of read it all
		content, err4 := ioutil.ReadFile(filePath)
		if err4 != nil {
			h.errorHandler(w, err4, 500)
			return
		}
		// w.Write([]byte(fmt.Sprintf("URL Path: %v\n", req.URL.Path)))
		// w.Write([]byte(fmt.Sprintf("URL Query: %v\n", req.URL.Query())))
		// w.Write([]byte(fmt.Sprintf("RequestURI: %v\n", req.RequestURI)))
		// w.Write([]byte(fmt.Sprintf("Request URL: %v\n", req.URL)))
		// w.Write([]byte(fmt.Sprintf("Found page: %v\n", page.Route)))
		// w.Write([]byte(fmt.Sprintf("   Title: %s\n", page.Title)))

		w.Write(content)
	} else {
		h.errorHandler(w, errors.New("Not found"), 404)
		return
	}
}
