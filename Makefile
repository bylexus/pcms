PROJECTNAME=pcms
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

build:
	@GOBIN=$(GOBIN) go build -o $(GOBIN)/$(PROJECTNAME) $(GOFILES)

exec:
	@GOBIN=$(GOBIN) $(run)

serve-doc: build
	$(shell cd $(GOBASE)/doc && $(GOBIN)/$(PROJECTNAME))


