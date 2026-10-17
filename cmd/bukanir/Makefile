MODULE := github.com/gen2brain/bukanir
GO     ?= go
LIB    := ..

-include secrets.env

obf = $(shell printf '%s' '$(1)' | base64 | tr -d '\n' | rev)

LDFLAGS := -X '$(MODULE).builtinTMDBKey=$(call obf,$(TMDB_KEY))'

.PHONY: all gui build test test-short test-race vet fmt tidy clean help

all: gui

## gui: build the desktop GUI with injected credentials
gui:
	CGO_ENABLED=1 $(GO) build -ldflags "$(LDFLAGS)" -o bukanir .

## build: compile the library
build:
	cd $(LIB) && CGO_ENABLED=1 $(GO) build -ldflags "$(LDFLAGS)" ./...

## test: run all library tests, including network tests (CGo-free)
test:
	cd $(LIB) && CGO_ENABLED=0 $(GO) test -ldflags "$(LDFLAGS)" ./...

## test-short: run fast library tests only, no network (CGo-free)
test-short:
	cd $(LIB) && CGO_ENABLED=0 $(GO) test -short ./...

## test-race: run fast library tests with the race detector (requires CGo)
test-race:
	cd $(LIB) && CGO_ENABLED=1 $(GO) test -race -short ./...

## vet: run go vet on the library and the GUI
vet:
	cd $(LIB) && CGO_ENABLED=0 $(GO) vet ./...
	CGO_ENABLED=0 $(GO) vet ./...

## fmt: format sources
fmt:
	gofmt -w $(LIB)

## tidy: tidy module dependencies
tidy:
	cd $(LIB) && $(GO) mod tidy
	$(GO) mod tidy

## clean: clear the test cache
clean:
	$(GO) clean -testcache

## help: list targets
help:
	@grep -hE '^## ' $(MAKEFILE_LIST) | sed 's/## //'
