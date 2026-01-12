.PHONY: build build-all clean test lint install help

# Build variables
BINARY_NAME=s3finder
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Directories
BUILD_DIR=build
CMD_DIR=./cmd/s3finder

# Default target
all: test build

## help: Show this help message
help:
	@echo "s3finder - AI-Powered S3 Bucket Enumeration Tool"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build        Build for current OS/arch"
	@echo "  build-all    Build for all supported platforms"
	@echo "  build-linux  Build for Linux (amd64, arm64)"
	@echo "  build-darwin Build for macOS (amd64, arm64)"
	@echo "  build-windows Build for Windows (amd64, arm64)"
	@echo "  install      Install to GOPATH/bin"
	@echo "  test         Run tests"
	@echo "  test-cover   Run tests with coverage"
	@echo "  lint         Run linter"
	@echo "  clean        Clean build artifacts"
	@echo "  deps         Download dependencies"
	@echo "  help         Show this help"

## build: Build for current OS/arch
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(CMD_DIR)

## install: Install to GOPATH/bin
install:
	$(GOCMD) install $(LDFLAGS) $(CMD_DIR)

## build-all: Build for all supported platforms
build-all: clean build-linux build-darwin build-windows
	@echo "All builds complete. Binaries in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

## build-linux: Build for Linux
build-linux:
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Linux amd64..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	@echo "Building for Linux arm64..."
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

## build-darwin: Build for macOS
build-darwin:
	@mkdir -p $(BUILD_DIR)
	@echo "Building for macOS amd64..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	@echo "Building for macOS arm64 (Apple Silicon)..."
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

## build-windows: Build for Windows
build-windows:
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Windows amd64..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "Building for Windows arm64..."
	GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe $(CMD_DIR)

## test: Run tests
test:
	$(GOTEST) -v ./...

## test-cover: Run tests with coverage
test-cover:
	$(GOTEST) -v -cover -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Run linter (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

## clean: Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

## deps: Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## release: Create release archives (requires build-all first)
release: build-all
	@mkdir -p $(BUILD_DIR)/release
	@echo "Creating release archives..."
	@cd $(BUILD_DIR) && tar -czvf release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czvf release/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	@cd $(BUILD_DIR) && tar -czvf release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(BUILD_DIR) && tar -czvf release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@cd $(BUILD_DIR) && zip release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@cd $(BUILD_DIR) && zip release/$(BINARY_NAME)-$(VERSION)-windows-arm64.zip $(BINARY_NAME)-windows-arm64.exe
	@echo "Release archives in $(BUILD_DIR)/release/"
	@ls -la $(BUILD_DIR)/release/
