VERSION := 2.0.0
BINARY := efx-face
BUILD_DIR := bin
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build install clean build-all build-mac build-linux build-windows release run lint test dev help

# Default build (current platform)
build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) ./cmd/efx-face

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/efx-face

# Run the TUI
run: build
	./$(BUILD_DIR)/$(BINARY)

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)/

# ==========================================
# Cross-compilation for all platforms
# ==========================================

# Build all platforms
build-all: clean build-mac build-linux build-windows
	@echo "\n✅ All builds completed in $(BUILD_DIR)/"
	@ls -la $(BUILD_DIR)/

# macOS builds (ARM64 for M1/M2/M3/M4, AMD64 for Intel)
build-mac:
	@mkdir -p $(BUILD_DIR)
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/efx-face
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/efx-face
	@echo "✓ macOS builds done"

# Linux builds (ARM64 and AMD64)
build-linux:
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/efx-face
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/efx-face
	@echo "✓ Linux builds done"

# Windows builds (ARM64 and AMD64)
build-windows:
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/efx-face
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY)-windows-arm64.exe ./cmd/efx-face
	@echo "✓ Windows builds done"

# ==========================================
# Release packaging
# ==========================================

# Create release archives for GitHub
release: build-all
	@echo "Creating release archives..."
	@mkdir -p $(BUILD_DIR)/release
	@cp templates.example.yaml $(BUILD_DIR)/release/
	@cp README.md $(BUILD_DIR)/release/
	@cd $(BUILD_DIR) && tar -czf $(BINARY)-$(VERSION)-darwin-arm64.tar.gz $(BINARY)-darwin-arm64 release/*
	@cd $(BUILD_DIR) && tar -czf $(BINARY)-$(VERSION)-darwin-amd64.tar.gz $(BINARY)-darwin-amd64 release/*
	@cd $(BUILD_DIR) && tar -czf $(BINARY)-$(VERSION)-linux-amd64.tar.gz $(BINARY)-linux-amd64 release/*
	@cd $(BUILD_DIR) && tar -czf $(BINARY)-$(VERSION)-linux-arm64.tar.gz $(BINARY)-linux-arm64 release/*
	@cd $(BUILD_DIR) && zip -q $(BINARY)-$(VERSION)-windows-amd64.zip $(BINARY)-windows-amd64.exe release/*
	@cd $(BUILD_DIR) && zip -q $(BINARY)-$(VERSION)-windows-arm64.zip $(BINARY)-windows-arm64.exe release/*
	@echo "✅ Release archives created:"
	@ls -la $(BUILD_DIR)/*.tar.gz $(BUILD_DIR)/*.zip

# ==========================================
# Development tools
# ==========================================

# Lint
lint:
	go vet ./...
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	fi

# Test
test:
	go test -v ./...

# Development - run with live reload (requires air)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "air not installed. Run: go install github.com/air-verse/air@latest"; \
		$(MAKE) run; \
	fi

# Show help
help:
	@echo "efx-face v$(VERSION) - Makefile"
	@echo ""
	@echo "Build Commands:"
	@echo "  make build         - Build for current platform"
	@echo "  make build-all     - Build for all platforms (Mac/Linux/Windows × ARM/AMD)"
	@echo "  make build-mac     - Build for macOS (ARM64 + AMD64)"
	@echo "  make build-linux   - Build for Linux (ARM64 + AMD64)"
	@echo "  make build-windows - Build for Windows (ARM64 + AMD64)"
	@echo "  make release       - Build all + create release archives (.tar.gz/.zip)"
	@echo ""
	@echo "Development Commands:"
	@echo "  make run           - Build and run"
	@echo "  make install       - Install to GOPATH/bin"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make lint          - Run linter"
	@echo "  make test          - Run tests"
	@echo "  make dev           - Run with live reload (requires air)"
	@echo ""
	@echo "Output binaries:"
	@echo "  $(BUILD_DIR)/$(BINARY)-darwin-arm64      (macOS Apple Silicon)"
	@echo "  $(BUILD_DIR)/$(BINARY)-darwin-amd64      (macOS Intel)"
	@echo "  $(BUILD_DIR)/$(BINARY)-linux-amd64       (Linux x86_64)"
	@echo "  $(BUILD_DIR)/$(BINARY)-linux-arm64       (Linux ARM64)"
	@echo "  $(BUILD_DIR)/$(BINARY)-windows-amd64.exe (Windows x86_64)"
	@echo "  $(BUILD_DIR)/$(BINARY)-windows-arm64.exe (Windows ARM64)"
