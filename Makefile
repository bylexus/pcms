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

.PHONY: build-doc
build-doc: build
	$(GOBIN)/$(PROJECTNAME) -c $(GOBASE)/doc/pcms-config.yaml build

.PHONY: serve-doc
serve-doc: build
	$(GOBIN)/$(PROJECTNAME) serve-doc

.PHONY: build-docker-image-amd64
build-docker-image-amd64:
	docker build --pull --no-cache --platform=linux/amd64 -t $(PROJECTNAME):$(VERSION) $(GOBASE)
	docker image tag $(PROJECTNAME):$(VERSION) $(PROJECTNAME):$(VERSION)-amd64

.PHONY: build-docker-image-arm64
build-docker-image-arm64:
	docker build --pull --no-cache --platform=linux/arm64 -t $(PROJECTNAME):$(VERSION) $(GOBASE)
	docker image tag $(PROJECTNAME):$(VERSION) $(PROJECTNAME):$(VERSION)-arm64

# Note for building multi-platform docker images:
# you need to use a multiplatform builder with docker buildx:
# Create a parallel multi-platform builder
#     docker buildx create --name multiplatform-builder --use
# Make "buildx" the default
#     docker buildx install
# or used an existing multiplatform builder:
#     docker buildx use multiplatform-builder
.PHONY: docker-multibuild-to-registry
docker-multibuild-to-registry:
	docker buildx build --pull --no-cache --push --platform=linux/arm64,linux/amd64 -t $(PROJECTNAME):$(VERSION) $(GOBASE)
# 	docker image tag $(PROJECTNAME):$(VERSION) registry.alexi.ch/pcms:$(VERSION)
# 	docker image tag $(PROJECTNAME):$(VERSION) registry.alexi.ch/pcms:latest
# 	docker push registry.alexi.ch/pcms:$(VERSION)
# 	docker push registry.alexi.ch/pcms:latest


.PHONY: build-release
build-release: build-doc
	# cleanup:
	rm -rf ./$(RELEASE_DIR)/
	rm -rf ./$(GOBASE)/site-template/build/
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
