VERSION := 1.0.0
BINARY := efx-face
BUILD_DIR := bin

.PHONY: build install clean build-all run lint

# Default build
build:
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY) ./cmd/efx-face

# Install to GOPATH/bin
install:
	go install -ldflags "-X main.version=$(VERSION)" ./cmd/efx-face

# Run the TUI
run: build
	./$(BUILD_DIR)/$(BINARY)

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)/

# Cross-compilation for multi-platform
build-all: clean
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/efx-face
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/efx-face
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/efx-face
	GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/efx-face

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
	@echo "efx-face-manager Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build      - Build the binary"
	@echo "  make install    - Install to GOPATH/bin"
	@echo "  make run        - Build and run"
	@echo "  make clean      - Remove build artifacts"
	@echo "  make build-all  - Build for all platforms"
	@echo "  make lint       - Run linter"
	@echo "  make test       - Run tests"
	@echo "  make dev        - Run with live reload (requires air)"
	@echo "  make help       - Show this help"
