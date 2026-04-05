# TODO

* Sync Algorithm: on serve start, a background job should update page indexes in a separate thread.
  see REWRITE.md.
* image resize service: create a route to generate variants of an image on-the-fly

## Image resize service

pcms should offer a on-the-fly image resize service that can be used as a drop-in replacement for an image url to get a resized version of it:

* route: `/_imageResizer/[parameters]/[urlTail]`: The URL tail consists of two parts: the first tail value is the comma-separated list of parameters, while the rest of the tail is the relative URL (images on the same site), or a url-encoded absolute url.
* parameters are given as a comma-separated list in the first URL tail part. Example: `/_imageResizer/width:200,height:400/path/to/image.jpg`
* The following parameters are supported:
  * `width:n`: resizes the image to the fixed width n pixels. If no height is given, the image keeps the aspect ratio, else it will be distorted.
  * `height:n`: resizes the image to the fixed height n pixels. If no width is given, the image keeps the aspect ratio, else it will be distorted.
  * `maxWidth:n`: resizes the image to the given max width, keep it if it is already smaller. in combination with `maxHeight`, this limits an image to a certain rectangle. When possible, the aspect ratio is kept.
  * `maxHeight:n`: resizes the image to the given max height, keep it if it is already smaller. in combination with `maxWidth`, this limits an image to a certain rectangle. When possible, the aspect ratio is kept.
  * `fit:[contain|cover|distort]`: Defines how the image should be fit into a given width/height:
    * `cover`: The image will be shrinked by keeping the aspect ratio, but only one side needs to fit in: the other side is cut (centering the image).
    * `contain`: The image will be shrinked by keeping the aspect ratio, and both sides need to fit in the given sizes, creating a solid-filled border on two sides (center the image)
    * `distort`: The image will be shrinked to fit into the given size, not keeping the aspect ratio (distorted)
  * `fillColor:rrggbb`: Color (3 hex values red (rr), green(gg), blue (bb)) to use for a solid filled border (see `fit`)
  * `format:[png|jpg|webp]`: changes the output image to the apropriate format
  * `jpgQuality:n`: If format is `jpg`, this defines the jpeg quality (0-100), Defaults to 80
  * `webpQuality:n`: If format is `webp`, this defines the webp quality (0-100), Defaults to 80
* If possible, the aspect ratio should be kept, depending on the combination of the parameters, if possible. Examples:
  * `/width:1000`: The width is fixed, and the height will be changed according to the image's original aspect ratio.
  * `/width:1000,maxHeight:1000`: The width is fixed, and the aspect ratio is kept as long as the height does not go over 1000. If the height will be > 1000, the image will be distorted, and the aspect ratio cannot be kept.
  * `/maxWidth:1000,maxHeight:1000`: The image will be shrinked so that both width and height are < 1000, keeping the aspect ratio. This effectively defines a max width/height rectangle.
* The service outputs the image as binary data, including the correct mime type header
* generated images are cached (also in the page's cache dir, see `cache_dir` parameter in pcms-config.yaml): the parameters and the urls are used as a key (hash based), and delivered from cache if available.