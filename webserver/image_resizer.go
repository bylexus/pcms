package webserver

import (
	"bytes"
	"crypto/sha256"
	"errors"
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
	"time"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp" // register WebP decoder
)

const imageResizerPrefix = "/_imageResizer/"

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

// parseResizeParams parses the resizer parameter string. width and maxWidth must be
// <= maxWidthLimit, height and maxHeight must be <= maxHeightLimit.
func parseResizeParams(paramStr string, maxWidthLimit, maxHeightLimit int) (ResizeParams, error) {
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
		name  string
		val   int
		limit int
	}{
		{"width", p.Width, maxWidthLimit}, {"height", p.Height, maxHeightLimit},
		{"maxWidth", p.MaxWidth, maxWidthLimit}, {"maxHeight", p.MaxHeight, maxHeightLimit},
	} {
		if dim.val > dim.limit {
			return p, fmt.Errorf("%s: must be <= %d, got %d", dim.name, dim.limit, dim.val)
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

// computeTargetDimensions returns the resize operation and target dimensions for
// applyResize. The result always fits into the max box maxWidthLimit x maxHeightLimit
// (with a single resample). The params must be validated by parseResizeParams with
// the same limits.
func computeTargetDimensions(src image.Image, p ResizeParams, maxWidthLimit, maxHeightLimit int) (op string, w, h int) {
	srcW := src.Bounds().Dx()
	srcH := src.Bounds().Dy()

	switch {
	case p.Width > 0 && p.Height == 0 && p.MaxWidth == 0 && p.MaxHeight == 0:
		// derived height = srcH * Width / srcW; cap it at maxHeightLimit
		if int64(srcH)*int64(p.Width) > int64(maxHeightLimit)*int64(srcW) {
			return "resize-ar", 0, maxHeightLimit
		}
		return "resize-ar", p.Width, 0

	case p.Height > 0 && p.Width == 0 && p.MaxWidth == 0 && p.MaxHeight == 0:
		// derived width = srcW * Height / srcH; cap it at maxWidthLimit
		if int64(srcW)*int64(p.Height) > int64(maxWidthLimit)*int64(srcH) {
			return "resize-ar", maxWidthLimit, 0
		}
		return "resize-ar", 0, p.Height

	case p.MaxWidth > 0 && p.MaxHeight > 0 && p.Width == 0 && p.Height == 0:
		return "fit", p.MaxWidth, p.MaxHeight

	case p.MaxWidth > 0 && p.MaxHeight == 0 && p.Width == 0 && p.Height == 0:
		return fitIntoBox(srcW, srcH, p.MaxWidth, maxHeightLimit)

	case p.MaxHeight > 0 && p.MaxWidth == 0 && p.Width == 0 && p.Height == 0:
		return fitIntoBox(srcW, srcH, maxWidthLimit, p.MaxHeight)

	case p.Width > 0 && p.MaxHeight > 0 && p.Height == 0 && p.MaxWidth == 0:
		scale := float64(p.Width) / float64(srcW)
		projH := int(math.Round(float64(srcH) * scale))
		if projH <= p.MaxHeight {
			return "resize-ar", p.Width, 0
		}
		return "resize-distort", p.Width, p.MaxHeight

	case p.Height > 0 && p.MaxWidth > 0 && p.Width == 0 && p.MaxHeight == 0:
		scale := float64(p.Height) / float64(srcH)
		projW := int(math.Round(float64(srcW) * scale))
		if projW <= p.MaxWidth {
			return "resize-ar", 0, p.Height
		}
		return "resize-distort", p.MaxWidth, p.Height

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
		return fitIntoBox(srcW, srcH, maxWidthLimit, maxHeightLimit)
	}
}

// fitIntoBox returns "noop" if the source fits into boxW x boxH, else a
// downscale ("resize-ar") that keeps the aspect ratio and fits into the box.
func fitIntoBox(srcW, srcH, boxW, boxH int) (op string, w, h int) {
	if srcW <= boxW && srcH <= boxH {
		return "noop", 0, 0
	}
	// width is the limiting side if srcW/boxW >= srcH/boxH
	if int64(srcW)*int64(boxH) >= int64(srcH)*int64(boxW) {
		return "resize-ar", boxW, 0
	}
	return "resize-ar", 0, boxH
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

// resolveOutputFormat returns the output format of a resize request: params.Format if
// set, else derived from the source format name (as returned by image.Decode /
// image.DecodeConfig): "jpeg" -> jpg, "webp" -> webp, everything else -> png.
func resolveOutputFormat(params ResizeParams, srcFormat string) outFormat {
	if params.Format != "" {
		return params.Format
	}
	switch srcFormat {
	case "jpeg":
		return fmtJPG
	case "webp":
		return fmtWebP
	default:
		return fmtPNG
	}
}

// detectOutputFormat returns the output format of a resize request before the source
// image is decoded. If params.Format is not set, only the image header of the source
// file is read (image.DecodeConfig).
func detectOutputFormat(fsys fs.FS, fsPath string, params ResizeParams) (outFormat, error) {
	if params.Format != "" {
		return params.Format, nil
	}
	f, err := fsys.Open(fsPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, srcFormat, err := image.DecodeConfig(f)
	if err != nil {
		return "", err
	}
	return resolveOutputFormat(params, srcFormat), nil
}

// formatContentType returns the Content-Type of an output format.
func formatContentType(format outFormat) string {
	switch format {
	case fmtJPG:
		return "image/jpeg"
	case fmtWebP:
		return "image/webp"
	default:
		return "image/png"
	}
}

// formatExtension returns the cache file extension (without dot) of an output format.
func formatExtension(format outFormat) string {
	switch format {
	case fmtJPG, fmtWebP:
		return string(format)
	default:
		return string(fmtPNG)
	}
}

// imageCachePath returns the image cache file path: <cacheDir>/_imageResizer/<key>.<ext>.
func imageCachePath(cacheDir, key string, format outFormat) string {
	return filepath.Join(cacheDir, "_imageResizer", key+"."+formatExtension(format))
}

// touchCacheFile sets the access and modification time of a cache file to now, so the
// file mtime is the time of the last use.
func touchCacheFile(cacheFile string, now time.Time) error {
	return os.Chtimes(cacheFile, now, now)
}

// encodeImage encodes img in the given output format.
func encodeImage(img image.Image, params ResizeParams, format outFormat) ([]byte, error) {
	var buf bytes.Buffer
	var err error

	switch format {
	case fmtJPG:
		err = imaging.Encode(&buf, img, imaging.JPEG, imaging.JPEGQuality(params.JpgQuality))
	case fmtWebP:
		err = encodeWebP(&buf, img, params.WebpQuality)
	default:
		err = png.Encode(&buf, img)
	}

	return buf.Bytes(), err
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

	maxW := h.ServerConfig.Server.ImageResizer.MaxWidth
	maxH := h.ServerConfig.Server.ImageResizer.MaxHeight

	params, err := parseResizeParams(paramStr, maxW, maxH)
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

	info, statErr := fs.Stat(h.siteFS, fsPath)
	if statErr != nil {
		h.errorHandler(w, statErr, http.StatusInternalServerError)
		return
	}
	if info.Size() > h.ServerConfig.Server.ImageResizer.MaxSourceSize {
		h.errorHandler(w, fmt.Errorf("image too large"), http.StatusRequestEntityTooLarge)
		return
	}
	sourceModTime := info.ModTime()

	format, err := detectOutputFormat(h.siteFS, fsPath, params)
	if err != nil {
		h.errorHandler(w, fmt.Errorf("detect image format: %w", err), http.StatusInternalServerError)
		return
	}
	contentType := formatContentType(format)

	key := cacheKey(params, fileRoute)
	cachePath := imageCachePath(h.ServerConfig.Server.CacheDir, key, format)

	cacheValid, err := isPageCacheValid(cachePath, sourceModTime)
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}

	if cacheValid {
		if err := touchCacheFile(cachePath, time.Now()); err != nil && h.ErrorLogger != nil {
			h.ErrorLogger.Warning("touch image cache file %s: %s", cachePath, err.Error())
		}
		cacheFile, err := os.Open(cachePath)
		if err != nil {
			h.errorHandler(w, fmt.Errorf("open image cache: %w", err), http.StatusInternalServerError)
			return
		}
		defer cacheFile.Close()
		w.Header().Set("Content-Type", contentType)
		http.ServeContent(w, req, filepath.Base(cachePath), sourceModTime, cacheFile)
		return
	}

	// Uncached render: take a slot of the concurrency limit for decode + resize + encode.
	// The slot is released right after encoding; cache write and response writing do not
	// hold it (a slow client must not block a render slot).
	if err := h.resizeSem.acquire(req.Context(), h.ServerConfig.Server.ImageResizer.WaitTimeout); err != nil {
		if errors.Is(err, errResizeSlotTimeout) {
			w.Header().Set("Retry-After", "5")
			h.errorHandler(w, err, http.StatusServiceUnavailable)
			return
		}
		// Client disconnected while waiting: no response possible, nothing to render.
		if h.ErrorLogger != nil {
			h.ErrorLogger.Debug("image resize cancelled while waiting for a slot: %s: %s", fileRoute, err.Error())
		}
		return
	}
	slotHeld := true
	releaseSlot := func() {
		if slotHeld {
			slotHeld = false
			h.resizeSem.release()
		}
	}
	defer releaseSlot()

	f, err := h.siteFS.Open(fsPath)
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		h.errorHandler(w, err, http.StatusInternalServerError)
		return
	}

	op, tw, th := computeTargetDimensions(img, params, maxW, maxH)
	resized := applyResize(img, op, tw, th, params)

	data, err := encodeImage(resized, params, format)
	if err != nil {
		h.errorHandler(w, fmt.Errorf("encode image: %w", err), http.StatusInternalServerError)
		return
	}
	releaseSlot()

	if err := writeCacheFile(cachePath, data); err != nil && h.ErrorLogger != nil {
		// Fallback: the cache cannot be written (e.g. disk full, read-only),
		// the rendered image is served from memory below.
		h.ErrorLogger.Warning("write image cache %s: %s (serving from memory)", cachePath, err.Error())
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
