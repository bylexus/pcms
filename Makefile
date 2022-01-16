PROJECTNAME=pcms
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

.PHONY: build exec serve-doc build-docker-image

build:
	@GOBIN=$(GOBIN) go build -o $(GOBIN)/$(PROJECTNAME) $(GOFILES)

exec:
	@GOBIN=$(GOBIN) $(run)

serve-doc: build
	$(shell cd $(GOBASE)/doc && $(GOBIN)/$(PROJECTNAME))

build-docker-image:
	docker build --pull -t $(PROJECTNAME) $(GOBASE)
