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

Only images that are indexed in the pcms database and enabled are served. Un-indexed paths and disabled pages return 404. The maximum accepted source file size is controlled by `server.image_resizer.max_source_size` in `pcms-config.yaml` (default: 32 MB).

## Usage

```
/_imageResizer/<params>/<image-path>
```

- `<params>` — a comma-separated list of `key:value` pairs (see [Parameter Reference](#parameter-reference) below). Use an empty string or omit parameters to get the image as-is with format conversion only (the image is still downscaled if it is larger than the maximum image size, see [Size limits](#size-limits)).
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

### Size limits

The maximum image size is configured in `server.image_resizer.max_width` and `server.image_resizer.max_height` (default: 1920 × 1920, see the [Reference]({{ Webroot("/reference/") }}#pcms-configyaml)):

- `width` and `maxWidth` must be between 1 and `max_width` (inclusive), `height` and `maxHeight` between 1 and `max_height` (inclusive). Larger values return HTTP 400.
- The output image always fits into `max_width` × `max_height`, also when no resize is requested (e.g. format conversion only, or `maxWidth` larger than the source). Larger images are downscaled, preserving aspect ratio.

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
| no size parameter | Keep the source size; downscale proportionally only if larger than `max_width` × `max_height`. |
| `width` only | Scale to target width, height proportional. If the resulting height would exceed `max_height`, scale to `max_height` instead (the width is then smaller than `width`). |
| `height` only | Scale to target height, width proportional. If the resulting width would exceed `max_width`, scale to `max_width` instead (the height is then smaller than `height`). |
| `maxWidth` only | Downscale proportionally if wider than `maxWidth` (or taller than `max_height`); no-op otherwise. |
| `maxHeight` only | Downscale proportionally if taller than `maxHeight` (or wider than `max_width`); no-op otherwise. |
| `maxWidth` + `maxHeight` | Fit within the bounding box (`imaging.Fit`), preserving aspect ratio. |
| `width` + `maxHeight` | Scale to `width`; if the resulting height would exceed `maxHeight`, distort to `width` × `maxHeight` instead. |
| `height` + `maxWidth` | Scale to `height`; if the resulting width would exceed `maxWidth`, distort to `maxWidth` × `height` instead. |
| `width` + `height` | Apply `fit` mode (default: `distort`). |

`max_width` and `max_height` are the configured [size limits](#size-limits).

### Concurrency limit

Resizing an image that is not cached yet uses a lot of CPU. The number of such resize operations running at the same time is limited by `server.image_resizer.max_concurrent` (default: number of CPUs). Other requests wait for a free slot, up to `server.image_resizer.wait_timeout` (default: 10 seconds). After that, the request returns HTTP 503 with a `Retry-After: 5` header. Cached images are not limited.

### Caching

Processed images are stored in `<cache_dir>/_imageResizer/<key>.<jpg|png|webp>`, where `<key>` is derived (SHA-256) from the parameters and the image path. The cache directory is shared with the page cache and is configured via `server.cache_dir` in `pcms-config.yaml`.

- A cache entry is valid as long as it is not older than the source image. When the source file changes, the image is processed again.
- Cached images are sent with `Last-Modified` set to the modification time of the source image, so browsers can revalidate them (HTTP 304).
- Cache files that were not used for `server.cache_max_age` (default: 24 hours) are removed automatically. The cleanup runs at server startup and then every `server.cache_cleanup_interval` (default: 1 hour).
- If the cache file cannot be written (e.g. the disk is full), the image is processed and sent without caching, and a warning is logged.
