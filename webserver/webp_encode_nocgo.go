//go:build !cgo

package webserver

import (
	"fmt"
	"image"
	"io"
)

// encodeWebP has two build-tagged variants:
// - cgo build: uses github.com/chai2010/webp for real WebP encoding.
// - !cgo build (this file): keeps cross-compilation working without C toolchains.
// We return a clear runtime error when WebP output is requested in !cgo builds.
func encodeWebP(w io.Writer, img image.Image, quality int) error {
	_ = w
	_ = img
	_ = quality
	return fmt.Errorf("webp encoding is unavailable in this build (compiled without cgo)")
}
