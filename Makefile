.PHONY: build install clean dev test deps

VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
BINARY := zplay
BUILD_DIR := dist
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -s -w"

# Detect OS and ARCH
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

build:
	@echo "Building $(BINARY) $(VERSION) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/zplay
	@echo "Built: $(BUILD_DIR)/$(BINARY)"

build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/zplay
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/zplay
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/zplay
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/zplay
	@echo "Built all platforms in $(BUILD_DIR)/"

install: build
	@echo "Installing to /usr/local/bin/$(BINARY)..."
	sudo cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installed!"

uninstall:
	@echo "Removing /usr/local/bin/$(BINARY)..."
	sudo rm -f /usr/local/bin/$(BINARY)
	@echo "Uninstalled!"

clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	@echo "Clean!"

deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies installed!"

dev:
	@echo "Running in development mode..."
	go run ./cmd/zplay

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Running linter..."
	golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...

# Release helpers
version:
	@echo $(VERSION)

bump-patch:
	@echo "Bumping patch version..."
	@./scripts/bump-version.sh patch

bump-minor:
	@echo "Bumping minor version..."
	@./scripts/bump-version.sh minor

bump-major:
	@echo "Bumping major version..."
	@./scripts/bump-version.sh major
