package webserver

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/flosch/pongo2/v4"
	"github.com/gabriel-vasile/mimetype"

	"alexi.ch/pcms/logging"
	"alexi.ch/pcms/model"
)

type RequestHandler struct {
	ServerConfig model.Config
	ErrorLogger  *logging.Logger
}

func NewRequestHandler(
	config model.Config,
	accessLogger *logging.Logger,
	errorLogger *logging.Logger,
) *RequestHandler {
	r := RequestHandler{
		ServerConfig: config,
		ErrorLogger:  errorLogger,
	}
	return &r
}

func (h *RequestHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// create a file path from the requested URL path:
	relUrl := req.URL
	filePath := filepath.Join(h.ServerConfig.DestPath, relUrl.Path)

	// check if file exists, output an error if not:
	info, err := os.Stat(filePath)
	if errors.Is(err, fs.ErrNotExist) {
		h.errorHandler(w, fmt.Errorf("not found: %s", relUrl), http.StatusNotFound)
		return
	}
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}

	// if the requested file is a dir, add index.html to the path and try again:
	if info.IsDir() {
		filePath = filepath.Join(filePath, "index.html")
		_, err = os.Stat(filePath)
		if errors.Is(err, fs.ErrNotExist) {
			h.errorHandler(w, fmt.Errorf("not found: %s", relUrl), http.StatusNotFound)
			return
		}
		if err != nil {
			h.errorHandler(w, err, http.StatusInternalServerError)
			return
		}
	}

	// all well, deliver it!
	staticHandler := http.FileServer(http.Dir(h.ServerConfig.DestPath))
	staticHandler.ServeHTTP(w, req)
}

/*
 * Generic error handler, sets an HTTP error code and outputs the specified error
 */
func (h *RequestHandler) errorHandler(w http.ResponseWriter, err error, status int) {
	w.WriteHeader(status)
	w.Write([]byte(fmt.Sprintf("error: %v\n", err)))
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
