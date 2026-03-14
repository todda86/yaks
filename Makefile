# yaks - Kubernetes Context Switcher
# Cross-platform build support

BINARY_NAME := yaks
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-s -w -X github.com/todda86/yaks/cmd.version=$(VERSION) -X github.com/todda86/yaks/cmd.commit=$(COMMIT)"

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOVET := $(GOCMD) vet
GOMOD := $(GOCMD) mod
GOFMT := gofmt

# Output directory
DIST_DIR := dist

.PHONY: all build clean test vet fmt lint install uninstall cross-compile help

## help: Show this help message
help:
	@echo "yaks - Kubernetes Context Switcher"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed 's/^/  /'

## build: Build for the current platform
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

## install: Install to GOPATH/bin
install:
	$(GOCMD) install $(LDFLAGS) .

## uninstall: Remove from GOPATH/bin
uninstall:
	rm -f $(shell go env GOPATH)/bin/$(BINARY_NAME)

## test: Run tests
test:
	$(GOTEST) -v -race ./...

## vet: Run go vet
vet:
	$(GOVET) ./...

## fmt: Format code
fmt:
	$(GOFMT) -s -w .

## lint: Run go vet and check formatting
lint: vet
	@test -z "$$($(GOFMT) -l .)" || (echo "Files need formatting:" && $(GOFMT) -l . && exit 1)

## clean: Remove build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(DIST_DIR)

## tidy: Tidy go modules
tidy:
	$(GOMOD) tidy

## cross-compile: Build for all platforms
cross-compile: clean
	@mkdir -p $(DIST_DIR)
	# Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .
	GOOS=linux GOARCH=arm GOARM=7 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-armv7 .
	# macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .
	# Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-arm64.exe .

## checksums: Generate checksums for dist binaries
checksums: cross-compile
	@cd $(DIST_DIR) && shasum -a 256 * > checksums.txt
	@cat $(DIST_DIR)/checksums.txt

all: lint test build
