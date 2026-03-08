package webserver

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"alexi.ch/pcms/lib"
	"alexi.ch/pcms/logging"
	"alexi.ch/pcms/model"
	"alexi.ch/pcms/processor"
	"github.com/flosch/pongo2/v4"
)

type RequestHandler struct {
	ServerConfig model.Config
	ErrorLogger  *logging.Logger
	DBH          *lib.DBH
	// the site FS is the root of the served file system
	siteFS fs.FS
}

func NewRequestHandler(
	config model.Config,
	accessLogger *logging.Logger,
	errorLogger *logging.Logger,
	siteFS fs.FS,
	dbh *lib.DBH,
) *RequestHandler {
	r := RequestHandler{
		ServerConfig: config,
		ErrorLogger:  errorLogger,
		siteFS:       siteFS,
		DBH:          dbh,
	}
	return &r
}

/*
The RequestHandler's Serve function. This should be the inner-most
handler, so middlewares should already be applied (e.g. the StripPrefix handler).
We take the URL's path and prefix the local destination path to find a matching
local file, then deliver it.
*/
func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if h.DBH == nil {
		h.errorHandler(w, fmt.Errorf("db handler not configured"), http.StatusInternalServerError)
		return
	}

	rawRoutePath := req.URL.Path
	route := normalizeRoute(rawRoutePath)

	page, found, err := h.DBH.GetPageByRoute(route)
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}
	if found {
		h.servePage(w, req, route, page)
		return
	}

	fileRoute, allowFileLookup := normalizeFileLookupRoute(rawRoutePath)
	if !allowFileLookup {
		h.errorHandler(w, fmt.Errorf("not found: %s", route), http.StatusNotFound)
		return
	}

	file, found, err := h.DBH.GetFileByRoute(fileRoute)
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}
	if found {
		h.serveFile(w, req, file)
		return
	}

	h.errorHandler(w, fmt.Errorf("not found: %s", route), http.StatusNotFound)
}

func normalizeRoute(rawPath string) string {
	cleaned := path.Clean("/" + rawPath)
	if cleaned == "." || cleaned == "" {
		return "/"
	}
	return cleaned
}

func normalizeFileLookupRoute(rawPath string) (string, bool) {
	if rawPath != "/" && strings.HasSuffix(rawPath, "/") {
		return "", false
	}

	return normalizeRoute(rawPath), true
}

func routeToFSPath(route string) string {
	trimmed := strings.TrimPrefix(route, "/")
	if trimmed == "" {
		return "."
	}
	return trimmed
}

func (h *RequestHandler) servePage(w http.ResponseWriter, req *http.Request, route string, page lib.IndexedPageRecord) {
	fileInfo, err := processor.BuildPageProcessingFileInfo(route, page.IndexFile, h.ServerConfig)
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}

	sourceFSPath := path.Clean(path.Join(strings.TrimPrefix(route, "/"), page.IndexFile))
	if route == "/" {
		sourceFSPath = page.IndexFile
	}

	sourceStat, err := fs.Stat(h.siteFS, sourceFSPath)
	if errors.Is(err, fs.ErrNotExist) {
		h.errorHandler(w, fmt.Errorf("indexed page source missing: %s", sourceFSPath), http.StatusNotFound)
		return
	}
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}

	isValid, err := isPageCacheValid(fileInfo.AbsDestPath, sourceStat.ModTime())
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}

	if !isValid {
		rendered, err := h.renderPage(page.IndexFile, sourceFSPath, fileInfo)
		if err != nil {
			h.errorHandler(w, err, http.StatusInternalServerError)
			return
		}
		if err := writeCacheFile(fileInfo.AbsDestPath, rendered); err != nil {
			h.errorHandler(w, err, http.StatusInternalServerError)
			return
		}
	}

	http.ServeFile(w, req, fileInfo.AbsDestPath)
}

func (h *RequestHandler) renderPage(indexFile string, sourceFSPath string, fileInfo processor.ProcessingFileInfo) ([]byte, error) {
	switch strings.ToLower(path.Ext(indexFile)) {
	case ".html":
		return (processor.HtmlProcessor{}).RenderFileForServe(h.siteFS, sourceFSPath, fileInfo.AbsSourcePath, h.ServerConfig, fileInfo)
	case ".md":
		return (processor.MdProcessor{}).RenderFileForServe(h.siteFS, sourceFSPath, fileInfo.AbsSourcePath, h.ServerConfig, fileInfo)
	default:
		return nil, fmt.Errorf("unsupported page index file type: %s", indexFile)
	}
}

func (h *RequestHandler) serveFile(w http.ResponseWriter, req *http.Request, file lib.IndexedFileRecord) {
	fsPath := routeToFSPath(file.Route)
	f, err := h.siteFS.Open(fsPath)
	if errors.Is(err, fs.ErrNotExist) {
		h.errorHandler(w, fmt.Errorf("indexed file missing: %s", file.Route), http.StatusNotFound)
		return
	}
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if file.MimeType != "" {
		w.Header().Set("Content-Type", file.MimeType)
	}

	if readSeeker, ok := f.(io.ReadSeeker); ok {
		var modTime time.Time
		if info, err := fs.Stat(h.siteFS, fsPath); err == nil {
			modTime = info.ModTime()
		}
		http.ServeContent(w, req, file.FileName, modTime, readSeeker)
		return
	}

	_, err = io.Copy(w, f)
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
	}
}

func isPageCacheValid(cacheFile string, sourceModTime time.Time) (bool, error) {
	cacheInfo, err := os.Stat(cacheFile)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return !cacheInfo.ModTime().Before(sourceModTime), nil
}

func writeCacheFile(cacheFile string, content []byte) error {
	dir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return err
	}

	return os.WriteFile(cacheFile, content, 0o666)
}

/*
 * Generic error handler, sets an HTTP error code and outputs the specified error
 */
func (h *RequestHandler) errorHandler(w http.ResponseWriter, err error, status int) {
	if h.ErrorLogger != nil {
		h.ErrorLogger.Error("request failed (%d): %s", status, err.Error())
	}
	w.WriteHeader(status)
	w.Write([]byte(fmt.Sprintf("error: %v\n", err)))
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
