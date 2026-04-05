package webserver

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp" // register WebP decoder
)

const imageResizerPrefix = "/_imageResizer/"
const maxResizeDimension = 1920

type fitMode string
type outFormat string

const (
	fitContain fitMode = "contain"
	fitCover   fitMode = "cover"
	fitDistort fitMode = "distort"

	fmtPNG  outFormat = "png"
	fmtJPG  outFormat = "jpg"
	fmtWebP outFormat = "webp"
)

type ResizeParams struct {
	Width, Height       int
	MaxWidth, MaxHeight int
	Fit                 fitMode
	FillColor           color.RGBA
	Format              outFormat
	JpgQuality          int
	WebpQuality         int
}

func defaultResizeParams() ResizeParams {
	return ResizeParams{
		FillColor:   color.RGBA{R: 255, G: 255, B: 255, A: 255},
		JpgQuality:  80,
		WebpQuality: 80,
	}
}

func parseResizeParams(paramStr string) (ResizeParams, error) {
	p := defaultResizeParams()
	if paramStr == "" {
		return p, nil
	}

	for _, token := range strings.Split(paramStr, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		parts := strings.SplitN(token, ":", 2)
		if len(parts) != 2 {
			return p, fmt.Errorf("invalid parameter token %q: expected key:value", token)
		}
		key, val := parts[0], parts[1]

		switch key {
		case "width":
			n, err := parsePositiveInt(val)
			if err != nil {
				return p, fmt.Errorf("width: %w", err)
			}
			p.Width = n
		case "height":
			n, err := parsePositiveInt(val)
			if err != nil {
				return p, fmt.Errorf("height: %w", err)
			}
			p.Height = n
		case "maxWidth":
			n, err := parsePositiveInt(val)
			if err != nil {
				return p, fmt.Errorf("maxWidth: %w", err)
			}
			p.MaxWidth = n
		case "maxHeight":
			n, err := parsePositiveInt(val)
			if err != nil {
				return p, fmt.Errorf("maxHeight: %w", err)
			}
			p.MaxHeight = n
		case "fit":
			switch fitMode(val) {
			case fitContain, fitCover, fitDistort:
				p.Fit = fitMode(val)
			default:
				return p, fmt.Errorf("fit: unknown value %q (must be contain, cover, or distort)", val)
			}
		case "fillColor":
			c, err := parseHexColor(val)
			if err != nil {
				return p, fmt.Errorf("fillColor: %w", err)
			}
			p.FillColor = c
		case "format":
			switch outFormat(val) {
			case fmtPNG, fmtJPG, fmtWebP:
				p.Format = outFormat(val)
			default:
				return p, fmt.Errorf("format: unknown value %q (must be png, jpg, or webp)", val)
			}
		case "jpgQuality":
			n, err := parseQuality(val)
			if err != nil {
				return p, fmt.Errorf("jpgQuality: %w", err)
			}
			p.JpgQuality = n
		case "webpQuality":
			n, err := parseQuality(val)
			if err != nil {
				return p, fmt.Errorf("webpQuality: %w", err)
			}
			p.WebpQuality = n
		default:
			return p, fmt.Errorf("unknown parameter %q", key)
		}
	}

	if p.Width > 0 && p.MaxWidth > 0 {
		return p, fmt.Errorf("width and maxWidth cannot both be set")
	}
	if p.Height > 0 && p.MaxHeight > 0 {
		return p, fmt.Errorf("height and maxHeight cannot both be set")
	}

	for _, dim := range []struct {
		name string
		val  int
	}{
		{"width", p.Width}, {"height", p.Height},
		{"maxWidth", p.MaxWidth}, {"maxHeight", p.MaxHeight},
	} {
		if dim.val > maxResizeDimension {
			return p, fmt.Errorf("%s: must be <= %d, got %d", dim.name, maxResizeDimension, dim.val)
		}
	}

	return p, nil
}

func parsePositiveInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("expected integer, got %q", s)
	}
	if n <= 0 {
		return 0, fmt.Errorf("must be > 0, got %d", n)
	}
	return n, nil
}

func parseQuality(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("expected integer, got %q", s)
	}
	if n < 0 || n > 100 {
		return 0, fmt.Errorf("must be 0-100, got %d", n)
	}
	return n, nil
}

func parseHexColor(s string) (color.RGBA, error) {
	if len(s) != 6 {
		return color.RGBA{}, fmt.Errorf("expected 6 hex characters (rrggbb), got %q", s)
	}
	r, err := strconv.ParseUint(s[0:2], 16, 8)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid hex color %q: %w", s, err)
	}
	g, err := strconv.ParseUint(s[2:4], 16, 8)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid hex color %q: %w", s, err)
	}
	b, err := strconv.ParseUint(s[4:6], 16, 8)
	if err != nil {
		return color.RGBA{}, fmt.Errorf("invalid hex color %q: %w", s, err)
	}
	return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}, nil
}

func cacheKey(params ResizeParams, urlTail string) string {
	canonical := fmt.Sprintf("w%d:h%d:mw%d:mh%d:fit%s:fill%02x%02x%02x:fmt%s:jq%d:wq%d:%s",
		params.Width, params.Height, params.MaxWidth, params.MaxHeight,
		params.Fit,
		params.FillColor.R, params.FillColor.G, params.FillColor.B,
		params.Format,
		params.JpgQuality, params.WebpQuality,
		urlTail,
	)
	h := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", h[:16])
}

func computeTargetDimensions(src image.Image, p ResizeParams) (op string, w, h int) {
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()

	switch {
	case p.Width > 0 && p.Height == 0 && p.MaxWidth == 0 && p.MaxHeight == 0:
		return "resize-ar", p.Width, 0

	case p.Height > 0 && p.Width == 0 && p.MaxWidth == 0 && p.MaxHeight == 0:
		return "resize-ar", 0, p.Height

	case p.MaxWidth > 0 && p.MaxHeight > 0 && p.Width == 0 && p.Height == 0:
		return "fit", p.MaxWidth, p.MaxHeight

	case p.MaxWidth > 0 && p.MaxHeight == 0 && p.Width == 0 && p.Height == 0:
		if srcW <= p.MaxWidth {
			return "noop", 0, 0
		}
		return "resize-ar", p.MaxWidth, 0

	case p.MaxHeight > 0 && p.MaxWidth == 0 && p.Width == 0 && p.Height == 0:
		if srcH <= p.MaxHeight {
			return "noop", 0, 0
		}
		return "resize-ar", 0, p.MaxHeight

	case p.Width > 0 && p.MaxHeight > 0 && p.Height == 0 && p.MaxWidth == 0:
		scale := float64(p.Width) / float64(srcW)
		projH := int(math.Round(float64(srcH) * scale))
		if projH <= p.MaxHeight {
			return "resize-ar", p.Width, 0
		}
		return "resize-distort", p.Width, p.MaxHeight

	case p.Width > 0 && p.Height > 0:
		switch p.Fit {
		case fitCover:
			return "fill", p.Width, p.Height
		case fitContain:
			return "contain", p.Width, p.Height
		default:
			return "resize-distort", p.Width, p.Height
		}

	default:
		return "noop", 0, 0
	}
}

func applyResize(src image.Image, op string, w, h int, p ResizeParams) image.Image {
	filter := imaging.Lanczos
	switch op {
	case "resize-ar", "resize-distort":
		return imaging.Resize(src, w, h, filter)
	case "fit":
		return imaging.Fit(src, w, h, filter)
	case "fill":
		return imaging.Fill(src, w, h, imaging.Center, filter)
	case "contain":
		fitted := imaging.Fit(src, w, h, filter)
		bg := imaging.New(w, h, p.FillColor)
		return imaging.PasteCenter(bg, fitted)
	default:
		return src
	}
}

func encodeImage(img image.Image, params ResizeParams, srcFormat string) ([]byte, string, error) {
	targetFmt := string(params.Format)
	if targetFmt == "" {
		targetFmt = srcFormat
		if targetFmt == "jpeg" {
			targetFmt = "jpg"
		}
	}

	var buf bytes.Buffer
	var contentType string
	var err error

	switch outFormat(targetFmt) {
	case fmtJPG:
		contentType = "image/jpeg"
		err = imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(params.JpgQuality))
	case fmtWebP:
		contentType = "image/webp"
		err = webp.Encode(&buf, img, &webp.Options{Lossless: false, Quality: float32(params.WebpQuality)})
	default:
		contentType = "image/png"
		err = png.Encode(&buf, img)
	}

	return buf.Bytes(), contentType, err
}

func cachedContentType(cachePath string, params ResizeParams) string {
	data, err := os.ReadFile(cachePath + ".ct")
	if err == nil && len(data) > 0 {
		return strings.TrimSpace(string(data))
	}
	switch params.Format {
	case fmtJPG:
		return "image/jpeg"
	case fmtWebP:
		return "image/webp"
	default:
		return "image/png"
	}
}

func (h *RequestHandler) serveResizedImage(w http.ResponseWriter, req *http.Request, rawPath string) {
	idx := strings.Index(rawPath, "/")
	if idx < 0 {
		h.errorHandler(w, fmt.Errorf("malformed image resizer path: %q", rawPath), http.StatusBadRequest)
		return
	}
	paramStr := rawPath[:idx]
	urlTail, err := url.PathUnescape(rawPath[idx+1:])
	if err != nil {
		h.errorHandler(w, fmt.Errorf("invalid URL encoding in image path"), http.StatusBadRequest)
		return
	}

	if urlTail == "" {
		h.errorHandler(w, fmt.Errorf("missing image path in resizer URL"), http.StatusBadRequest)
		return
	}

	params, err := parseResizeParams(paramStr)
	if err != nil {
		h.errorHandler(w, fmt.Errorf("parse resize params: %w", err), http.StatusBadRequest)
		return
	}

	// Only serve files that are indexed in the DB and enabled.
	// This enforces the existing content security model and prevents SSRF
	// by rejecting remote URLs and un-indexed paths alike.
	fileRoute := normalizeRoute(urlTail)
	file, found, err := h.DBH.GetFileByRoute(fileRoute)
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}
	if !found || !file.Enabled {
		h.errorHandler(w, fmt.Errorf("not found: %s", fileRoute), http.StatusNotFound)
		return
	}

	fsPath := routeToFSPath(fileRoute)

	key := cacheKey(params, fileRoute)
	cachePath := filepath.Join(h.ServerConfig.Server.CacheDir, "_imageResizer", key)

	info, statErr := fs.Stat(h.siteFS, fsPath)
	if statErr != nil {
		h.errorHandler(w, statErr, http.StatusInternalServerError)
		return
	}
	if info.Size() > h.ServerConfig.Server.MaxBodySize {
		h.errorHandler(w, fmt.Errorf("image too large"), http.StatusRequestEntityTooLarge)
		return
	}
	sourceModTime := info.ModTime()

	cacheValid, err := isPageCacheValid(cachePath, sourceModTime)
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}

	if cacheValid {
		contentType := cachedContentType(cachePath, params)
		w.Header().Set("Content-Type", contentType)
		http.ServeFile(w, req, cachePath)
		return
	}

	f, err := h.siteFS.Open(fsPath)
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}
	defer f.Close()

	img, srcFmt, err := image.Decode(f)
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}

	op, tw, th := computeTargetDimensions(img, params)
	resized := applyResize(img, op, tw, th, params)

	data, contentType, err := encodeImage(resized, params, srcFmt)
	if err != nil {
		h.errorHandler(w, fmt.Errorf("encode image: %w", err), http.StatusInternalServerError)
		return
	}

	if err := writeCacheFile(cachePath, data); err != nil {
		h.errorHandler(w, fmt.Errorf("write image cache: %w", err), http.StatusInternalServerError)
		return
	}
	_ = writeCacheFile(cachePath+".ct", []byte(contentType))

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
