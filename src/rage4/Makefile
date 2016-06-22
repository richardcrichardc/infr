## simple makefile to log workflow
# Targets:
#   all: builds the code
#   build: builds the code
#   fmt: formats the source files
#   clean: cleans the code
#   install: installs the code on the GOPATH
#   def:  installs dependencies
#   test:  run the tests
#

GOFLAGS ?= $(GOFLAGS:)

all: install test

build:
	@go build $(GOFLAGS) ./...

install:
	@go get $(GOFLAGS) ./...

test: install
	@go test $(GOFLAGS) ./...

clean:
	@go clean $(GOFLAGS) -i ./...

## EOF