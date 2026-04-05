---
title: "Image Resizer"
shortTitle: "Image Resizer"
template: "page-template.html"
metaTags:
  - name: "keywords"
    content: "pcms,image,resize,backend"
  - name: "description"
    content: "pcms image resizer backend service"
---
# Image Resizer

The image resizer is a built-in backend service that resizes and converts images on demand. It is served under the `/_imageResizer/` path prefix and caches results on disk so each unique parameter+image combination is only processed once.

Only images that are indexed in the pcms database and enabled are served. Un-indexed paths and disabled pages return 404. The maximum accepted source file size is controlled by `server.max_body_size` in `pcms-config.yaml` (default: 32 MB).

## Usage

```
/_imageResizer/<params>/<image-path>
```

- `<params>` — a comma-separated list of `key:value` pairs (see [Parameter Reference](#parameter-reference) below). Use an empty string or omit parameters to get the image as-is with format conversion only.
- `<image-path>` — the site-relative path to the source image (same path used in page routes).

**Examples:**

```html
<!-- Resize to 400 px wide, keep aspect ratio: -->
<img src="/_imageResizer/width:400/images/photo.jpg">

<!-- Crop to exact 400×300, cover mode: -->
<img src="/_imageResizer/width:400,height:300,fit:cover/images/photo.jpg">

<!-- Fit inside 800×600, white letterbox, output as WebP: -->
<img src="/_imageResizer/width:800,height:600,fit:contain,format:webp/images/photo.jpg">

<!-- Downscale only if wider than 1200 px: -->
<img src="/_imageResizer/maxWidth:1200/images/hero.png">

<!-- Convert to WebP without resizing: -->
<img src="/_imageResizer/format:webp/images/photo.jpg">
```

## Parameter Reference

Parameters are passed as a comma-separated string in the URL path segment before the image path. Each parameter has the form `key:value`.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `width` | integer > 0 | — | Target width in pixels. If only `width` is set, height is scaled proportionally. Cannot be combined with `maxWidth`. |
| `height` | integer > 0 | — | Target height in pixels. If only `height` is set, width is scaled proportionally. Cannot be combined with `maxHeight`. |
| `maxWidth` | integer > 0 | — | Downscale the image so its width does not exceed this value. No-op if the source is already narrower. Cannot be combined with `width`. |
| `maxHeight` | integer > 0 | — | Downscale the image so its height does not exceed this value. No-op if the source is already shorter. Cannot be combined with `height`. |
| `fit` | string | `distort` | How to fit the image when both `width` and `height` are given. See [Fit Modes](#fit-modes) below. |
| `fillColor` | `rrggbb` hex | `ffffff` | Background fill color used by `fit:contain` to pad the image. Six hex digits, no `#` prefix. |
| `format` | string | source format | Output image format. One of `png`, `jpg`, `webp`. If omitted, the source format is preserved (JPEG stays JPEG, etc.). |
| `jpgQuality` | integer 0–100 | `80` | JPEG encoding quality. Only relevant when the output format is `jpg`. |
| `webpQuality` | integer 0–100 | `80` | WebP encoding quality (lossy). Only relevant when the output format is `webp`. |

All dimension values must be between 1 and 1920 (inclusive).

### Fit Modes

The `fit` parameter only applies when both `width` and `height` are provided.

| Value | Description |
|-------|-------------|
| `distort` | (default) Scale to exactly `width` × `height`, ignoring aspect ratio. |
| `cover` | Scale and crop to fill `width` × `height` exactly, preserving aspect ratio. The image is cropped around the center. |
| `contain` | Scale to fit within `width` × `height`, preserving aspect ratio. The remaining area is filled with `fillColor`. |

### Mixed dimension combinations

| Parameters set | Behaviour |
|---------------|-----------|
| `width` only | Scale to target width, height proportional. |
| `height` only | Scale to target height, width proportional. |
| `maxWidth` only | Downscale proportionally if wider than `maxWidth`; no-op otherwise. |
| `maxHeight` only | Downscale proportionally if taller than `maxHeight`; no-op otherwise. |
| `maxWidth` + `maxHeight` | Fit within the bounding box (`imaging.Fit`), preserving aspect ratio. |
| `width` + `maxHeight` | Scale to `width`; if the resulting height would exceed `maxHeight`, distort to `width` × `maxHeight` instead. |
| `width` + `height` | Apply `fit` mode (default: `distort`). |

### Caching

Processed images are stored in `<cache_dir>/_imageResizer/` using a SHA-256-derived filename. The cache entry is invalidated when the source file's modification time changes. The cache directory is shared with the page cache and is configured via `server.cache_dir` in `pcms-config.yaml`.
