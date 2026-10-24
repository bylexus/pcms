# TODO

* Sync Algorithm: on serve start, a background job should update page indexes in a separate thread.
  see REWRITE.md.
* image resize service: create a route to generate variants of an image on-the-fly
* cache invalidation: when a template file changes, the html cache needs to be invalidated - at the moment, the cache has to be
  cleared manually when a template changes.
