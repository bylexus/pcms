package webserver

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

func TestResolveOutputFormat(t *testing.T) {
	tests := []struct {
		name      string
		format    outFormat
		srcFormat string
		want      outFormat
	}{
		{"jpeg source", "", "jpeg", fmtJPG},
		{"webp source", "", "webp", fmtWebP},
		{"png source", "", "png", fmtPNG},
		{"gif source", "", "gif", fmtPNG},
		{"tiff source", "", "tiff", fmtPNG},
		{"bmp source", "", "bmp", fmtPNG},
		{"unknown source", "", "", fmtPNG},
		{"param png wins over jpeg source", fmtPNG, "jpeg", fmtPNG},
		{"param jpg wins over png source", fmtJPG, "png", fmtJPG},
		{"param webp wins over jpeg source", fmtWebP, "jpeg", fmtWebP},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := defaultResizeParams()
			p.Format = tc.format
			if got := resolveOutputFormat(p, tc.srcFormat); got != tc.want {
				t.Errorf("resolveOutputFormat(%q, %q) = %q, want %q", tc.format, tc.srcFormat, got, tc.want)
			}
		})
	}
}

func TestFormatContentTypeAndExtension(t *testing.T) {
	tests := []struct {
		format      outFormat
		wantCT      string
		wantExt     string
		wantPathEnd string
	}{
		{fmtJPG, "image/jpeg", "jpg", "abc.jpg"},
		{fmtPNG, "image/png", "png", "abc.png"},
		{fmtWebP, "image/webp", "webp", "abc.webp"},
		{"", "image/png", "png", "abc.png"},
	}

	for _, tc := range tests {
		t.Run(string(tc.format), func(t *testing.T) {
			if got := formatContentType(tc.format); got != tc.wantCT {
				t.Errorf("formatContentType(%q) = %q, want %q", tc.format, got, tc.wantCT)
			}
			if got := formatExtension(tc.format); got != tc.wantExt {
				t.Errorf("formatExtension(%q) = %q, want %q", tc.format, got, tc.wantExt)
			}
			wantPath := filepath.Join("cache", "_imageResizer", tc.wantPathEnd)
			if got := imageCachePath("cache", "abc", tc.format); got != wantPath {
				t.Errorf("imageCachePath(%q) = %q, want %q", tc.format, got, wantPath)
			}
		})
	}
}

func encodeTestImage(t *testing.T, format string) []byte {
	t.Helper()
	img := makeTestImage(4, 3)
	var buf bytes.Buffer
	var err error
	switch format {
	case "jpeg":
		err = jpeg.Encode(&buf, img, nil)
	case "png":
		err = png.Encode(&buf, img)
	case "gif":
		err = gif.Encode(&buf, img, nil)
	default:
		t.Fatalf("unsupported test format %q", format)
	}
	if err != nil {
		t.Fatalf("encode %s: %v", format, err)
	}
	return buf.Bytes()
}

func TestDetectOutputFormat(t *testing.T) {
	fsys := fstest.MapFS{
		"img.jpg":     {Data: encodeTestImage(t, "jpeg")},
		"img.png":     {Data: encodeTestImage(t, "png")},
		"img.gif":     {Data: encodeTestImage(t, "gif")},
		"jpeg-as.png": {Data: encodeTestImage(t, "jpeg")},
		"broken.jpg":  {Data: []byte("not an image")},
	}

	tests := []struct {
		name    string
		path    string
		format  outFormat
		want    outFormat
		wantErr bool
	}{
		{"jpeg source", "img.jpg", "", fmtJPG, false},
		{"png source", "img.png", "", fmtPNG, false},
		{"gif source", "img.gif", "", fmtPNG, false},
		{"content wins over file extension", "jpeg-as.png", "", fmtJPG, false},
		{"param format wins", "img.jpg", fmtWebP, fmtWebP, false},
		{"param format does not read source", "missing.jpg", fmtPNG, fmtPNG, false},
		{"broken source", "broken.jpg", "", "", true},
		{"missing source", "missing.jpg", "", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := defaultResizeParams()
			p.Format = tc.format
			got, err := detectOutputFormat(fsys, tc.path, p)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got format %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDetectOutputFormatMatchesDecode(t *testing.T) {
	// The format from the header (cache lookup) must be the same as the format from
	// the full decode, so the cache file extension matches the encoded content.
	for _, srcFormat := range []string{"jpeg", "png", "gif"} {
		t.Run(srcFormat, func(t *testing.T) {
			data := encodeTestImage(t, srcFormat)
			fsys := fstest.MapFS{"img": {Data: data}}
			p := defaultResizeParams()

			got, err := detectOutputFormat(fsys, "img", p)
			if err != nil {
				t.Fatalf("detectOutputFormat: %v", err)
			}
			_, decodedFormat, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("image.Decode: %v", err)
			}
			if want := resolveOutputFormat(p, decodedFormat); got != want {
				t.Errorf("detectOutputFormat = %q, from decode = %q", got, want)
			}
		})
	}
}

func TestTouchCacheFile(t *testing.T) {
	tests := []struct {
		name    string
		create  bool
		wantErr bool
	}{
		{"existing file gets new mtime", true, false},
		{"missing file returns error", false, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "_imageResizer", "abc.png")
			old := time.Now().Add(-48 * time.Hour).Truncate(time.Second)
			if tc.create {
				if err := writeCacheFile(path, []byte("data")); err != nil {
					t.Fatalf("writeCacheFile: %v", err)
				}
				if err := os.Chtimes(path, old, old); err != nil {
					t.Fatalf("Chtimes: %v", err)
				}
			}

			now := time.Now().Truncate(time.Second)
			err := touchCacheFile(path, now)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("touchCacheFile: %v", err)
			}

			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("Stat: %v", err)
			}
			if !info.ModTime().Equal(now) {
				t.Errorf("mtime = %v, want %v", info.ModTime(), now)
			}
			data, err := os.ReadFile(path)
			if err != nil || string(data) != "data" {
				t.Errorf("content changed: %q, err %v", data, err)
			}
		})
	}
}
