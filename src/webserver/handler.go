package webserver

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/flosch/pongo2/v4"
	"github.com/gabriel-vasile/mimetype"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/crypto/bcrypt"

	"alexi.ch/pcms/src/logging"
	"alexi.ch/pcms/src/model"
)

type RequestHandler struct {
	ServerConfig *model.Config
	PageMap      *model.PageMap
	ErrorLogger  *logging.Logger
}

func NewRequestHandler(
	config *model.Config,
	pageMap *model.PageMap,
	accessLogger *logging.Logger,
	errorLogger *logging.Logger,
) *RequestHandler {
	r := RequestHandler{
		ServerConfig: config,
		PageMap:      pageMap,
		ErrorLogger:  errorLogger,
	}
	return &r
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	route := getRoute(req, h.ServerConfig)
	page, err := h.findMatchingRoute(route)
	if page != nil {
		// do not deliver content from disabled pages:
		if enabled, present := page.Metadata["enabled"]; present && enabled.(bool) == false {
			h.errorHandler(w, errors.New("Page not found"), 404)
			return
		}

		if page.IsProtected() {
			ok := h.handleBasicAuth(page, w, req)
			if !ok {
				return
			}
		}

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
	return h.PageMap.FindMatchingRoute(routePath)
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
	w.Header().Add("Content-Type", "text/html")

	content, err := ioutil.ReadFile(page.PageIndexPath())
	if err != nil {
		h.errorHandler(w, err, 500)
		return
	}
	switch page.Type {
	case model.PAGE_TYPE_HTML:
		h.renderHtmlPageContent(page, w, req, &content)
	case model.PAGE_TYPE_MARKDOWN:
		h.renderMarkdownPageContent(page, w, req, &content)

	default:
		w.Write([]byte(fmt.Sprintf("Not yet implemented - page type %v", page.Type)))
	}
}

/**
 * Does the rendering for a html page content. 'content' is the raw content read by the
 * page's index file. It is rendered using the pongo template engine, to
 * fill in the placeholder.
 *
 * Outputs the result using the given response writer.
 */
func (h *RequestHandler) renderHtmlPageContent(page *model.Page, w http.ResponseWriter, req *http.Request, content *[]byte) {
	result, err := h.renderTemplateFromBytes(page, content, pongo2.Context{})
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
	}
	w.Write(result)
}

/**
 * Does the rendering for a Markdown page content. 'content' is the raw Markdown content read by the
 * page's index file.
 *
 * Markdown is rendered in the following steps:
 * 1. the Markdown file is read from the Page's index property and processed with
 *    the pongo template engine
 * 2. Then HTML is generated from it
 * 3. The page's html template (Metadata.template property) is load and also parsed using pongo,
 *    in addition with the rendered Markdown html in the "content" context variable
 *
 * Outputs the result using the given response writer.
 */
func (h *RequestHandler) renderMarkdownPageContent(page *model.Page, w http.ResponseWriter, req *http.Request, content *[]byte) {
	// markdown uses a html template, which has to be set in the "Metadata.template" page property
	template, ok := page.Metadata["template"].(string)
	if ok {
		// content is the markdown content of the page
		result, err := h.renderTemplateFromBytes(page, content, pongo2.Context{})
		if err != nil {
			h.errorHandler(w, err, http.StatusInternalServerError)
		}
		result = blackfriday.Run(result, blackfriday.WithExtensions(
			blackfriday.AutoHeadingIDs|blackfriday.Autolink|blackfriday.CommonExtensions|blackfriday.Footnotes,
		))
		// template content is the html template, filled with the result from the content rendering above:
		templateContent, err := h.renderTemplateFromFile(page, template, pongo2.Context{"content": string(result)})
		if err != nil {
			h.errorHandler(w, err, http.StatusInternalServerError)
		}
		w.Write(templateContent)
	}
}

/**
 * Takes a template file name and a partial pongo context, then renders the
 * template. The template file is searched using its configured loaders.
 *
 * Returns the rendered content as byte array.
 */
func (h *RequestHandler) renderTemplateFromFile(page *model.Page, templateFile string, context pongo2.Context) ([]byte, error) {
	tpl, err := pongo2.FromFile(templateFile)
	if err != nil {
		return nil, err
	}
	return h.renderTemplate(page, tpl, context)
}

/**
 * Takes a byte array and a partial pongo context, then renders the
 * content using pongo.
 *
 * Returns the rendered content as byte array.
 */
func (h *RequestHandler) renderTemplateFromBytes(page *model.Page, content *[]byte, context pongo2.Context) ([]byte, error) {
	tpl, err := pongo2.FromBytes(*content)
	if err != nil {
		return nil, err
	}
	return h.renderTemplate(page, tpl, context)
}

/**
 * Renders a pongo template with the given partial pongo context. The context is
 * enriched with the data below, so the context is meant to provide additional data, if needed.

 * Returns the rendered content as byte array.
 */
func (h *RequestHandler) renderTemplate(page *model.Page, template *pongo2.Template, context pongo2.Context) ([]byte, error) {
	finalContext := pongo2.Context{
		"site":     h.ServerConfig.Site,
		"page":     page,
		"rootPage": h.PageMap.RootPage,
		"base":     h.ServerConfig.Site.Webroot,
		"meta":     page.Metadata["metaTags"],
	}
	finalContext.Update(context)
	out, err2 := template.ExecuteBytes(finalContext)

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
	finfo, err := os.Stat(filePath)
	if err != nil {
		h.errorHandler(w, err, 404)
		return
	}
	if finfo.Mode().IsRegular() {
		mtype, err := h.findMimeType(filePath)
		if err == nil {
			w.Header().Add("Content-Type", mtype)
		}
		// stream file content instead of read it all
		fh, err := os.Open(filePath)
		if err != nil {
			h.errorHandler(w, err, 500)
			return
		}
		defer fh.Close()
		buffer := make([]byte, 8192)
		for {
			count, err := fh.Read(buffer)
			if err == io.EOF {
				break
			}
			w.Write(buffer[:count])
		}
	} else {
		h.errorHandler(w, errors.New("Not found"), 404)
		return
	}
}

// Checks and handles the basic auth mechanism for a given page.
// The golang mechanism is taken from the following blog:
// https://www.alexedwards.net/blog/basic-authentication-in-go
//
// If the authentication was successful, then true is returned.
// If not, then a 401 is returned and the WWW-Authenticate header is set,
// while returning false to indicate a failure.
func (r *RequestHandler) handleBasicAuth(page *model.Page, w http.ResponseWriter, req *http.Request) bool {
	username, password, ok := req.BasicAuth()
	if ok {
		pageUsers := getPageUsers(page)
		ok := r.validateLogin(username, password, pageUsers)
		if ok {
			return true
		}
	}

	w.Header().Add("WWW-Authenticate", `Basic realm="Restricted Area", charset="UTF-8"`)
	r.errorHandler(w, errors.New("Unauthorized"), http.StatusUnauthorized)
	return false
}

// Validates a username / password pair:
// - username must be present in pageUsers (or be 'valid-user').
// - then the username/password pair must match an entry in the server's site config (requiredUsers)
// returns true if ok, false if not.
func (r *RequestHandler) validateLogin(username string, password string, pageUsers []string) bool {
	configUsers := r.ServerConfig.Site.Users
	expectedPassword := "noop"
	present := false

	// check if the given username is a valid user for the page:
	for _, pageUser := range pageUsers {
		if pageUser == "valid-user" || pageUser == username {
			present = true
			break
		}
	}
	if !present {
		return false
	}

	// OK so far, so now validate the username / password
	configPassword, present := configUsers[username]
	if present {
		expectedPassword = configPassword
	}

	// compare the password in any case, even if not present in usertable,
	// to generate the same timing for all tries (to prevent timing attacks)
	err := bcrypt.CompareHashAndPassword([]byte(expectedPassword), []byte(password))
	if err == nil {
		return true
	}

	return false
}

func getPageUsers(page *model.Page) []string {
	extractedUsers := make([]string, 0)
	users, ok := page.Metadata["requiredUsers"]

	if ok {
		users, ok := users.([]interface{})
		if ok {
			for _, user := range users {
				extractedUsers = append(extractedUsers, user.(string))
			}
		}
	}
	return extractedUsers
}

// Factory function to create a http.Handler middleware that logs all access.
// Should be used when constructing the http server to act as a middleware between the
// real request handler.
func CreateAccessLoggerMiddleware(logger *logging.Logger, next http.Handler) http.Handler {
	// TODO: missing:
	// - last entry (-) should be response bytes
	// - make it configurable in config
	t, err := pongo2.FromString("{{clientIp}} - {{userId}} [{{time}}] {{httpMethod}} {{url}} {{protocol}} {{statusCode}} -")
	if err != nil {
		panic(err)
	}

	middleware := &AccessLoggerMiddleware{
		AccessLogger:      logger,
		AccessLogTemplate: t,
	}

	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// replacing the original writer to use our own, to capture the HTTP status code:
		loggingWriter := loggingResponseWriter{
			rw, http.StatusOK,
		}

		next.ServeHTTP(&loggingWriter, r)
		middleware.LogAccess(&loggingWriter, r)
	})
}

type AccessLoggerMiddleware struct {
	AccessLogTemplate *pongo2.Template
	AccessLogger      *logging.Logger
}

func (l *AccessLoggerMiddleware) LogAccess(rw *loggingResponseWriter, req *http.Request) {
	username, _, _ := req.BasicAuth()
	if len(username) == 0 {
		username = "-"
	}
	url := req.URL.String()
	clientIp := req.RemoteAddr
	now := time.Now().Format(time.RFC3339)
	ctx := pongo2.Context{
		"clientIp":   clientIp,
		"userId":     username,
		"time":       now,
		"httpMethod": req.Method,
		"url":        url,
		"protocol":   req.Proto,
		"statusCode": rw.StatusCode,
	}
	msg, err := l.AccessLogTemplate.Execute(ctx)
	if err == nil {
		l.AccessLogger.Info(msg)
	}
}

// Used to capture status code when writing
type loggingResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.StatusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
