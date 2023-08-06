# TODO

- [ ] file exclude patterns on build: better globs instead of regex? more intuitive
- [ ] per-processor config
- [ ] build:
  - [ ] cmd flag for output folder (overrides config)
  - [x] MUST NOT allow empty source/dest dirs
  - [x] MUST NOT clear wrong directory (aka source folder)
  - [x] MUST NOT build when in serve-doc mode
  - [x] don't delete output folder, just empty it
  - [x] inherit variables from upper (parent) page
  - [ ] more url functions:
    - [x] beginsWith() checks if a string begins with a (shorter) one: needed for sub-page check
    - [x] endsWith() checks if a string ends with a (shorter) one: needed for super-page check
    - [ ] concat() to concat multiple strings

- [ ] serve:
  - [x] watch for file changes, and rebuild
  - [ ] watch for template file changes, and rebuild
  - [ ] watch for variables.yaml file changes, and rebuild
  - [ ] watch for pcms-config.yaml file changes, and rebuild
  - [ ] optionally build before serve (cli flag)
  - [x] embedded doc
  - [ ] config access log format
  - [ ] configurable error pages (specific templates for http errors)
  - [ ] indexing using sqlite
- [ ] Update docs
- [ ] Templates: support for reading json / yaml data files while building