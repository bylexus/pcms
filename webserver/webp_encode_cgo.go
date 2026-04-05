//go:build cgo

package webserver

import (
	"image"
	"io"

	"github.com/chai2010/webp"
)

// encodeWebP has two build-tagged variants:
// - this cgo variant keeps WebP encoding functional in normal/native builds.
// - the !cgo variant lets cross-platform release builds compile without a C toolchain.
// The split exists because github.com/chai2010/webp encoder requires cgo.
func encodeWebP(w io.Writer, img image.Image, quality int) error {
	return webp.Encode(w, img, &webp.Options{Lossless: false, Quality: float32(quality)})
}
