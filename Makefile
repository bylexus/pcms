PROJECTNAME=pcms
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)
VERSION=$(shell git describe --tags)
RELEASE_DIR=./releases


.PHONY: build
build:
	@GOBIN=$(GOBIN) go build -o $(GOBIN)/$(PROJECTNAME) $(GOFILES)

.PHONY: exec
exec:
	@GOBIN=$(GOBIN) $(run)

.PHONY: serve-doc
serve-doc: build
	$(GOBIN)/$(PROJECTNAME) -c $(GOBASE)/doc/pcms-config.yaml serve

.PHONY: build-docker-image-amd64
build-docker-image-amd64:
	docker build --pull --platform=linux/amd64 -t $(PROJECTNAME):amd64 $(GOBASE)

.PHONY: build-docker-image-arm64
build-docker-image-arm64:
	docker build --pull --platform=linux/arm64 -t $(PROJECTNAME):arm64 $(GOBASE)

.PHONY: docker-push-to-registry
docker-push-to-registry: build-docker-image-amd64 build-docker-image-arm64
	docker image tag $(PROJECTNAME):amd64 registry.alexi.ch/pcms:amd64
	docker image tag $(PROJECTNAME):arm64 registry.alexi.ch/pcms:arm64
	docker push registry.alexi.ch/pcms:amd64
	docker push registry.alexi.ch/pcms:arm64


.PHONY: build-release
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
	echo "Releases exported in $(RELEASE_DIR)."
