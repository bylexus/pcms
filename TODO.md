# TODO

* Sync Algorithm: on serve start, a background job should update page indexes in a separate thread.
  see REWRITE.md.
* enabled flag: instead of checking the enabled state upwards on request, the enabled flag should
  already be calculated during index time, and "forced" down the sub-pages in the index. This makes
  the index run more complex, but page serving is simpler. Must also be applied to file routes.
* implement clear cache dir command
