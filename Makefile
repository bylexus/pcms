PROJECTNAME=pcms
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

build:
	@GOBIN=$(GOBIN) go build -o $(GOBIN)/$(PROJECTNAME) $(GOFILES)

exec:
	@GOBIN=$(GOBIN) $(run)

