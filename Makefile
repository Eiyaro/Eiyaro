# Makefile for Eiyaro Full Node
# Production-grade build system

.PHONY: all build build-race test test-race clean install fmt lint vet help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOINSTALL=$(GOCMD) install
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Binary name
BINARY_NAME=eyarod

# Build flags
LDFLAGS=-ldflags="-s -w"
RACE_FLAG=-race
VET_FLAG=-vet

# Output directory
OUTPUT_DIR=bin

# Default target
all: build

# Build without race detector (faster, for development)
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) ./cmd/eyarod

# Build with race detector (slower, for testing)
build-race:
	@echo "Building $(BINARY_NAME) with race detector..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) $(RACE_FLAG) $(VET_FLAG) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-race ./cmd/eyarod

# Run tests without race detector
test:
	@echo "Running tests..."
	$(GOTEST) -timeout 20m ./...

# Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) -race -timeout 20m ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -race -timeout 20m -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format all Go files
fmt:
	@echo "Formatting code..."
	$(GOFMT) -w -s .

# Check formatting (don't modify files)
fmt-check:
	@echo "Checking formatting..."
	@UNFORMATTED=$$($(GOFMT) -l . | grep -v vendor | head -1); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "ERROR: Files are not properly formatted. Run 'make fmt' to fix."; \
		exit 1; \
	fi
	@echo "All files are properly formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Install staticcheck
install-staticcheck:
	@echo "Installing staticcheck..."
	$(GOINSTALL) honnef.co/go/tools/cmd/staticcheck@latest

# Run staticcheck
lint: install-staticcheck
	@echo "Running staticcheck..."
	staticcheck -checks SA4006,SA4008,SA4009,SA4010,SA5003,SA1004,SA1014,SA1021,SA1023,SA1024,SA1025,SA1026,SA1027,SA2000,SA2001,SA2003,SA4000,SA4001,SA4003,SA4004,SA4011,SA4012,SA4013,SA4014,SA4015,SA4016,SA4017,SA4018,SA4019,SA4020,SA4021,SA4022,SA4023,SA5000,SA5002,SA5004,SA5005,SA5007,SA5008,SA5009,SA5010,SA5011,SA5012,SA6001,SA6002,SA9001,SA9002,SA9003,SA9004,SA9005,SA9006,ST1019 ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

# Tidy go.mod
tidy:
	@echo "Tidying go.mod..."
	$(GOMOD) tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(OUTPUT_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	$(GOINSTALL) $(LDFLAGS) ./cmd/eyarod

# Production build (no race detector, stripped binary)
production: build
	@echo "Production build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"

# Full CI/CD pipeline
ci: deps fmt-check vet lint test-race build-race
	@echo "CI pipeline completed successfully"

# Help
help:
	@echo "Eiyaro Full Node Build System"
	@echo ""
	@echo "Targets:"
	@echo "  all          - Build without race detector (default)"
	@echo "  build        - Build without race detector"
	@echo "  build-race   - Build with race detector"
	@echo "  test         - Run tests without race detector"
	@echo "  test-race    - Run tests with race detector"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  fmt          - Format all Go files"
	@echo "  fmt-check    - Check formatting (CI mode)"
	@echo "  vet          - Run go vet"
	@echo "  lint         - Run staticcheck"
	@echo "  deps         - Download dependencies"
	@echo "  tidy         - Tidy go.mod"
	@echo "  clean        - Remove build artifacts"
	@echo "  install      - Install binary to GOPATH/bin"
	@echo "  production   - Production build (optimized)"
	@echo "  ci           - Full CI/CD pipeline"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build           # Build for development"
	@echo "  make test-race       # Run tests with race detection"
	@echo "  make production      # Build for production deployment"
	@echo "  make ci              # Run full CI pipeline"
