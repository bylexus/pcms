PROJECTNAME=pcms
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)
VERSION=$(shell git describe --tags)
RELEASE_DIR=./releases

.PHONY: build exec serve-doc build-docker-image build-release

build:
	@GOBIN=$(GOBIN) go build -o $(GOBIN)/$(PROJECTNAME) $(GOFILES)

exec:
	@GOBIN=$(GOBIN) $(run)

serve-doc: build
	$(shell cd $(GOBASE)/doc && $(GOBIN)/$(PROJECTNAME) serve)

build-docker-image:
	docker build --pull -t $(PROJECTNAME) $(GOBASE)

build-release:
	# cleanup:
	rm -rf ./$(RELEASE_DIR)/
	mkdir ./$(RELEASE_DIR)
	mkdir -p $(RELEASE_DIR)/linux-{amd64,arm64}-$(VERSION) $(RELEASE_DIR)/windows-amd64-$(VERSION) $(RELEASE_DIR)/darwin-{amd64,arm64}-$(VERSION)

	# building MacOS releases:
	env GOOS=darwin GOARCH=amd64 go build -o $(RELEASE_DIR)/darwin-amd64-$(VERSION)/$(PROJECTNAME) $(GOFILES)
	env GOOS=darwin GOARCH=arm64 go build -o $(RELEASE_DIR)/darwin-arm64-$(VERSION)/$(PROJECTNAME) $(GOFILES)

	# building Linux releases:
	env GOOS=linux GOARCH=amd64 go build -o $(RELEASE_DIR)/linux-amd64-$(VERSION)/$(PROJECTNAME) $(GOFILES)
	env GOOS=linux GOARCH=arm64 go build -o $(RELEASE_DIR)/linux-arm64-$(VERSION)/$(PROJECTNAME) $(GOFILES)

	# building Windows releases:
	env GOOS=windows GOARCH=amd64 go build -o $(RELEASE_DIR)/windows-amd64-$(VERSION)/$(PROJECTNAME).exe $(GOFILES)

	# zipping:
	cd $(RELEASE_DIR); \
	for d in *; do \
		if [ -d "$$d" ]; then \
			zip -r $(PROJECTNAME)-$$d.zip $$d/; \
		fi; \
	done
