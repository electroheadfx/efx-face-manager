#!/bin/bash
#
# efx-face installer
# Downloads and installs efx-face for your platform
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/electroheadfx/efx-face-manager/main/install.sh | bash
#

set -e

VERSION="0.2.0"
REPO="electroheadfx/efx-face-manager"
BINARY_NAME="efx-face"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() { echo -e "${BLUE}→${NC} $1"; }
success() { echo -e "${GREEN}✓${NC} $1"; }
warn() { echo -e "${YELLOW}!${NC} $1"; }
error() { echo -e "${RED}✗${NC} $1"; exit 1; }

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Darwin*) echo "darwin" ;;
        Linux*)  echo "linux" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) error "Unsupported OS: $(uname -s)" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# Main installation
main() {
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║${NC}     efx-face installer v${VERSION}       ${BLUE}║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════╝${NC}"
    echo ""

    OS=$(detect_os)
    ARCH=$(detect_arch)
    
    info "Detected: ${OS}/${ARCH}"

    # Determine file extension and archive type
    if [ "$OS" = "windows" ]; then
        ARCHIVE_EXT="zip"
        BINARY_EXT=".exe"
    else
        ARCHIVE_EXT="tar.gz"
        BINARY_EXT=""
    fi

    ARCHIVE_NAME="${BINARY_NAME}-${VERSION}-${OS}-${ARCH}.${ARCHIVE_EXT}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${ARCHIVE_NAME}"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Download
    info "Downloading ${ARCHIVE_NAME}..."
    if command -v curl &> /dev/null; then
        curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/$ARCHIVE_NAME" || error "Download failed. Check if release v${VERSION} exists."
    elif command -v wget &> /dev/null; then
        wget -q "$DOWNLOAD_URL" -O "$TMP_DIR/$ARCHIVE_NAME" || error "Download failed. Check if release v${VERSION} exists."
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
    success "Downloaded"

    # Extract
    info "Extracting..."
    cd "$TMP_DIR"
    if [ "$ARCHIVE_EXT" = "zip" ]; then
        unzip -q "$ARCHIVE_NAME"
    else
        tar -xzf "$ARCHIVE_NAME"
    fi
    success "Extracted"

    # Install
    BINARY_FILE="${BINARY_NAME}-${OS}-${ARCH}${BINARY_EXT}"
    
    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        info "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
        mv "$BINARY_FILE" "${INSTALL_DIR}/${BINARY_NAME}${BINARY_EXT}"
        chmod +x "${INSTALL_DIR}/${BINARY_NAME}${BINARY_EXT}"
    else
        info "Installing to ${INSTALL_DIR}/${BINARY_NAME} (requires sudo)..."
        sudo mv "$BINARY_FILE" "${INSTALL_DIR}/${BINARY_NAME}${BINARY_EXT}"
        sudo chmod +x "${INSTALL_DIR}/${BINARY_NAME}${BINARY_EXT}"
    fi
    success "Installed to ${INSTALL_DIR}/${BINARY_NAME}"

    # Create default config directory
    CONFIG_DIR="$HOME/.config/efx-face"
    if [ ! -d "$CONFIG_DIR" ]; then
        mkdir -p "$CONFIG_DIR"
        success "Created config directory: ${CONFIG_DIR}"
    fi

    # Create default config file if it doesn't exist
    CONFIG_FILE="$HOME/.efx-face-manager.conf"
    if [ ! -f "$CONFIG_FILE" ]; then
        # Detect default model path
        if [ -d "/Volumes/T7/mlx-server" ]; then
            MODEL_DIR="/Volumes/T7/mlx-server"
        else
            MODEL_DIR="$HOME/mlx-server"
            mkdir -p "$MODEL_DIR"
        fi
        
        echo "MODEL_DIR=$MODEL_DIR" > "$CONFIG_FILE"
        success "Created config: ${CONFIG_FILE}"
        info "Model directory: ${MODEL_DIR}"
    else
        info "Config already exists: ${CONFIG_FILE}"
    fi

    # Verify installation
    echo ""
    if command -v efx-face &> /dev/null; then
        success "Installation complete!"
        echo ""
        echo -e "  Run ${GREEN}efx-face${NC} to start"
        echo ""
    else
        warn "Installation complete, but efx-face not found in PATH"
        echo ""
        echo "  Add ${INSTALL_DIR} to your PATH:"
        echo "  export PATH=\"\$PATH:${INSTALL_DIR}\""
        echo ""
    fi
}

main "$@"
