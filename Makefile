# Makefile for xorchid Go project

VERSION := 0.1.0
BINARY := xorchid
MODULE := xorchid

# Directories
BUILD_DIR := build

# Go tools
GO := go
GOFMT := gofmt
GOLINT := golint

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

.PHONY: all build run test clean install fmt lint vet help

all: build

# Build the binary
BINARY_EXE := $(BINARY).exe
build: $(BUILD_DIR)/$(BINARY_EXE)

$(BUILD_DIR)/$(BINARY_EXE):
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	$(GO) build -o $@ $(LDFLAGS) ./main.go

# Run the program
run: build
	@echo "Running $(BINARY)..."
	cmd /c $(BUILD_DIR)\(BINARY_EXE) --help 2>&1 || true

# Run tests
test:
	$(GO) test -v ./...

# Run tests with coverage
test-coverage:
	$(GO) test -cover ./...

# Run tests with race detector
test-race:
	$(GO) test -race ./...

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Install the binary
install:
	$(GO) install ./...

# Format code
fmt:
	$(GOFMT) -w .

# Lint code
lint:
	$(GOLINT) ./...

# Run vet
vet:
	$(GO) vet ./...

# Build for all platforms
build-all: clean build-linux build-windows build-mac

build-linux:
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY)-linux $(LDFLAGS) ./main.go

build-windows:
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY).exe $(LDFLAGS) ./main.go

build-mac:
	@if not exist $(BUILD_DIR) mkdir $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BUILD_DIR)/$(BINARY)-mac $(LDFLAGS) ./main.go

# Show help
help:
	@echo "Available targets:"
	@echo "  all       - Build the project (default)"
	@echo "  build     - Build the binary to $(BUILD_DIR)/"
	@echo "  run       - Build and run the program"
	@echo "  test      - Run all tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  test-race - Run tests with race detector"
	@echo "  clean     - Remove build artifacts"
	@echo "  install   - Install the binary"
	@echo "  fmt       - Format code"
	@echo "  lint      - Lint code"
	@echo "  vet       - Run go vet"
	@echo "  build-all - Build for all platforms"
	@echo "  help      - Show this help message"
