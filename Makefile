GO             = go
GOFMT          = go fmt
GOLINT         = golint
GOVERALLS      = goveralls
GOVERALLS_ARGS = -service=travis-ci

PACKAGE = nagopher-checks
TARGET  = $(CURDIR)/bin
PKGS    = $(shell $(GO) list ./... | grep -v "$(PACKAGE)/shared")

.PHONY: all
all: lint test build

.PHONY: build
build: deps
	for pkg in $(PKGS); do \
		$(GO) build -o "$(TARGET)/$$(basename "$$pkg")" $$pkg; \
	done

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