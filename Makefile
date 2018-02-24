GO             = go
GOFMT          = go fmt
GOLINT         = golint
GOVERALLS      = goveralls
GOVERALLS_ARGS = -service=travis-ci
GORELEASER     = goreleaser

BUILD_DATE := $(shell date +'%Y-%m-%d_%H:%M:%S')
BUILD_COMMIT := $(shell git show -q --format='%H' HEAD)

PACKAGE = nagopher-checks
TARGET  = $(CURDIR)/$(PACKAGE)
PKGS    = $(shell $(GO) list ./... | grep -v "$(PACKAGE)/shared")

.PHONY: all
all: lint test build

.PHONY: build
build: deps
	$(GO) build \
		-ldflags "-X main.BuildDate=$(BUILD_DATE) -X main.BuildCommit=$(BUILD_COMMIT)" \
		-o $(TARGET) .

.PHONY: deps
deps:
	$(GO) get -d -v -t ./...

.PHONY: devel-deps
devel-deps: deps
	$(GO) get github.com/golang/lint/golint
	$(GO) get github.com/mattn/goveralls

.PHONY: lint
lint: devel-deps
	$(GO) vet ./...
	$(GOLINT) -set_exit_status ./...

.PHONY: test
test: devel-deps
	$(GO) test -v ./...

.PHONY: coverage
coverage: devel-deps
	$(GOVERALLS) $(GOVERALLS_ARGS)
