package webserver

import (
	"image"
	"image/color"
	"testing"
)

func TestParseHexColor(t *testing.T) {
	tests := []struct {
		input   string
		want    color.RGBA
		wantErr bool
	}{
		{"ff0000", color.RGBA{R: 255, G: 0, B: 0, A: 255}, false},
		{"00ff00", color.RGBA{R: 0, G: 255, B: 0, A: 255}, false},
		{"0000ff", color.RGBA{R: 0, G: 0, B: 255, A: 255}, false},
		{"ffffff", color.RGBA{R: 255, G: 255, B: 255, A: 255}, false},
		{"000000", color.RGBA{R: 0, G: 0, B: 0, A: 255}, false},
		{"1a2b3c", color.RGBA{R: 0x1a, G: 0x2b, B: 0x3c, A: 255}, false},
		{"fff", color.RGBA{}, true},     // too short
		{"fffffff", color.RGBA{}, true}, // too long
		{"gggggg", color.RGBA{}, true},  // invalid hex
	}

	for _, tc := range tests {
		got, err := parseHexColor(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseHexColor(%q): expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseHexColor(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseHexColor(%q): got %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestParseResizeParams(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ResizeParams
		wantErr bool
	}{
		{
			name:  "empty",
			input: "",
			want:  defaultResizeParams(),
		},
		{
			name:  "width only",
			input: "width:200",
			want: ResizeParams{Width: 200, FillColor: color.RGBA{255, 255, 255, 255}, JpgQuality: 80, WebpQuality: 80},
		},
		{
			name:  "height only",
			input: "height:300",
			want: ResizeParams{Height: 300, FillColor: color.RGBA{255, 255, 255, 255}, JpgQuality: 80, WebpQuality: 80},
		},
		{
			name:  "maxWidth and maxHeight",
			input: "maxWidth:800,maxHeight:600",
			want: ResizeParams{MaxWidth: 800, MaxHeight: 600, FillColor: color.RGBA{255, 255, 255, 255}, JpgQuality: 80, WebpQuality: 80},
		},
		{
			name:  "width height fit cover",
			input: "width:400,height:300,fit:cover",
			want: ResizeParams{Width: 400, Height: 300, Fit: fitCover, FillColor: color.RGBA{255, 255, 255, 255}, JpgQuality: 80, WebpQuality: 80},
		},
		{
			name:  "contain with fillColor",
			input: "width:400,height:300,fit:contain,fillColor:ff0000",
			want: ResizeParams{Width: 400, Height: 300, Fit: fitContain, FillColor: color.RGBA{255, 0, 0, 255}, JpgQuality: 80, WebpQuality: 80},
		},
		{
			name:  "format webp with quality",
			input: "format:webp,webpQuality:90",
			want: ResizeParams{Format: fmtWebP, WebpQuality: 90, FillColor: color.RGBA{255, 255, 255, 255}, JpgQuality: 80},
		},
		{
			name:  "format jpg with quality",
			input: "format:jpg,jpgQuality:70",
			want: ResizeParams{Format: fmtJPG, JpgQuality: 70, FillColor: color.RGBA{255, 255, 255, 255}, WebpQuality: 80},
		},
		{
			name:    "unknown parameter",
			input:   "unknown:value",
			wantErr: true,
		},
		{
			name:    "invalid fit value",
			input:   "fit:stretch",
			wantErr: true,
		},
		{
			name:    "invalid format value",
			input:   "format:bmp",
			wantErr: true,
		},
		{
			name:    "width and maxWidth both set",
			input:   "width:100,maxWidth:200",
			wantErr: true,
		},
		{
			name:    "height and maxHeight both set",
			input:   "height:100,maxHeight:200",
			wantErr: true,
		},
		{
			name:    "width zero",
			input:   "width:0",
			wantErr: true,
		},
		{
			name:    "quality out of range",
			input:   "jpgQuality:101",
			wantErr: true,
		},
		{
			name:    "malformed token no colon",
			input:   "width200",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseResizeParams(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil (result: %+v)", got)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestCacheKey(t *testing.T) {
	p := defaultResizeParams()
	p.Width = 200

	k1 := cacheKey(p, "images/photo.jpg")
	k2 := cacheKey(p, "images/photo.jpg")
	if k1 != k2 {
		t.Errorf("same params should produce same key: %s != %s", k1, k2)
	}

	p2 := defaultResizeParams()
	p2.Width = 400
	k3 := cacheKey(p2, "images/photo.jpg")
	if k1 == k3 {
		t.Errorf("different params should produce different keys")
	}

	k4 := cacheKey(p, "images/other.jpg")
	if k1 == k4 {
		t.Errorf("different paths should produce different keys")
	}

	if len(k1) != 32 {
		t.Errorf("expected 32 hex chars, got %d: %s", len(k1), k1)
	}
}

func makeTestImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	return img
}

func TestComputeTargetDimensions(t *testing.T) {
	white := color.RGBA{255, 255, 255, 255}

	tests := []struct {
		name    string
		srcW    int
		srcH    int
		params  ResizeParams
		wantOp  string
		wantW   int
		wantH   int
	}{
		{
			name:   "width only",
			srcW:   800, srcH: 600,
			params: ResizeParams{Width: 200, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "resize-ar", wantW: 200, wantH: 0,
		},
		{
			name:   "height only",
			srcW:   800, srcH: 600,
			params: ResizeParams{Height: 300, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "resize-ar", wantW: 0, wantH: 300,
		},
		{
			name:   "maxWidth and maxHeight",
			srcW:   800, srcH: 600,
			params: ResizeParams{MaxWidth: 400, MaxHeight: 300, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "fit", wantW: 400, wantH: 300,
		},
		{
			name:   "maxWidth only image larger",
			srcW:   800, srcH: 600,
			params: ResizeParams{MaxWidth: 400, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "resize-ar", wantW: 400, wantH: 0,
		},
		{
			name:   "maxWidth only image smaller",
			srcW:   300, srcH: 200,
			params: ResizeParams{MaxWidth: 400, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "noop", wantW: 0, wantH: 0,
		},
		{
			name:   "maxHeight only image larger",
			srcW:   800, srcH: 600,
			params: ResizeParams{MaxHeight: 300, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "resize-ar", wantW: 0, wantH: 300,
		},
		{
			name:   "maxHeight only image smaller",
			srcW:   800, srcH: 200,
			params: ResizeParams{MaxHeight: 300, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "noop", wantW: 0, wantH: 0,
		},
		{
			name:   "width + maxHeight height fits",
			srcW:   1000, srcH: 500,
			params: ResizeParams{Width: 1000, MaxHeight: 1000, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			// scale=1.0, projH=500 <= 1000 => resize-ar
			wantOp: "resize-ar", wantW: 1000, wantH: 0,
		},
		{
			name:   "width + maxHeight height exceeds",
			srcW:   200, srcH: 1000,
			params: ResizeParams{Width: 1000, MaxHeight: 1000, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			// scale=5.0, projH=5000 > 1000 => resize-distort
			wantOp: "resize-distort", wantW: 1000, wantH: 1000,
		},
		{
			name:   "width + height fit cover",
			srcW:   800, srcH: 600,
			params: ResizeParams{Width: 400, Height: 300, Fit: fitCover, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "fill", wantW: 400, wantH: 300,
		},
		{
			name:   "width + height fit contain",
			srcW:   800, srcH: 600,
			params: ResizeParams{Width: 400, Height: 300, Fit: fitContain, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "contain", wantW: 400, wantH: 300,
		},
		{
			name:   "width + height fit distort",
			srcW:   800, srcH: 600,
			params: ResizeParams{Width: 400, Height: 300, Fit: fitDistort, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "resize-distort", wantW: 400, wantH: 300,
		},
		{
			name:   "width + height no fit (distort)",
			srcW:   800, srcH: 600,
			params: ResizeParams{Width: 400, Height: 300, FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "resize-distort", wantW: 400, wantH: 300,
		},
		{
			name:   "nothing set",
			srcW:   800, srcH: 600,
			params: ResizeParams{FillColor: white, JpgQuality: 80, WebpQuality: 80},
			wantOp: "noop", wantW: 0, wantH: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			src := makeTestImage(tc.srcW, tc.srcH)
			gotOp, gotW, gotH := computeTargetDimensions(src, tc.params)
			if gotOp != tc.wantOp || gotW != tc.wantW || gotH != tc.wantH {
				t.Errorf("got op=%q w=%d h=%d, want op=%q w=%d h=%d",
					gotOp, gotW, gotH, tc.wantOp, tc.wantW, tc.wantH)
			}
		})
	}
}
