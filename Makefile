.PHONY: build install test lint release clean fmt vet check all

# Build configuration
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X github.com/spechtlabs/golint-sl/internal/version.Version=$(VERSION) \
	-X github.com/spechtlabs/golint-sl/internal/version.Commit=$(COMMIT) \
	-X github.com/spechtlabs/golint-sl/internal/version.Date=$(DATE)

# Default target
all: check build

# Build binary
build:
	@echo "Building golint-sl..."
	go build -ldflags "$(LDFLAGS)" -o bin/golint-sl ./cmd/golint-sl

# Install to GOPATH/bin
install:
	@echo "Installing golint-sl..."
	go install -ldflags "$(LDFLAGS)" ./cmd/golint-sl

# Run tests
test:
	@echo "Running tests..."
	go test -race -cover -covermode=atomic -coverprofile=coverage.txt ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -race -cover -v ./...

# Run golint-sl on itself
lint: build
	@echo "Running golint-sl on itself..."
	./bin/golint-sl ./...

# Run golangci-lint
golangci-lint:
	@echo "Running golangci-lint..."
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	gofmt -s -w .
	goimports -w -local github.com/spechtlabs/golint-sl .

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Run all checks (format, vet, lint, test)
check: fmt vet golangci-lint test

# Release with goreleaser (dry run)
release-dry:
	@echo "Running goreleaser (dry run)..."
	goreleaser release --snapshot --clean

# Release with goreleaser
release:
	@echo "Running goreleaser..."
	goreleaser release --clean

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/ dist/ coverage.txt

# Update dependencies
deps:
	@echo "Updating dependencies..."
	go mod tidy
	go mod verify

# Download dependencies
download:
	@echo "Downloading dependencies..."
	go mod download

# Generate (if any code generation is needed)
generate:
	@echo "Running go generate..."
	go generate ./...

# Show version info
version:
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(COMMIT)"
	@echo "Date:    $(DATE)"

# Show help
help:
	@echo "golint-sl - GoLint SpechtLabs"
	@echo ""
	@echo "Available targets:"
	@echo "  all           - Run checks and build (default)"
	@echo "  build         - Build golint-sl binary"
	@echo "  install       - Install to GOPATH/bin"
	@echo "  test          - Run tests with coverage"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  lint          - Run golint-sl on itself"
	@echo "  golangci-lint - Run golangci-lint"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  check         - Run all checks (fmt, vet, lint, test)"
	@echo "  release-dry   - Test release with goreleaser"
	@echo "  release       - Release with goreleaser"
	@echo "  clean         - Clean build artifacts"
	@echo "  deps          - Update dependencies"
	@echo "  download      - Download dependencies"
	@echo "  generate      - Run go generate"
	@echo "  version       - Show version info"
	@echo "  help          - Show this help"
