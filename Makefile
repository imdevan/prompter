# Makefile for prompter CLI

# Variables
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Build directory
BUILD_DIR := dist

.PHONY: all build clean test install dev-build cross-platform help

# Default target
all: build

# Build for current platform
build:
	go build -o ./prompter ./cmd/prompter

# build:
# 	@echo "Building prompter $(VERSION) for current platform..."
# 	@mkdir -p $(BUILD_DIR)
# 	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/prompter ./cmd/prompter
# 	@echo "Build complete: $(BUILD_DIR)/prompter"


# Development build (no optimization)
dev-build:
	@echo "Building development version..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/prompter ./cmd/prompter
	@echo "Development build complete: $(BUILD_DIR)/prompter"

# Cross-platform builds
cross-platform:
	@echo "Building prompter $(VERSION) for all platforms..."
	@./scripts/build.sh $(VERSION)

# Install to local system
install: build
	@echo "Installing prompter to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/prompter /usr/local/bin/prompter
	@echo "Installation complete!"

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete!"

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build for current platform"
	@echo "  dev-build      - Build development version (no optimization)"
	@echo "  cross-platform - Build for all supported platforms"
	@echo "  install        - Install to /usr/local/bin"
	@echo "  test           - Run tests"
	@echo "  clean          - Clean build artifacts"
	@echo "  help           - Show this help"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION        - Version to build (default: dev)"
	@echo ""
	@echo "Examples:"
	@echo "  make build VERSION=1.0.0"
	@echo "  make cross-platform VERSION=1.0.0"
