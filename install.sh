#!/bin/bash
set -e

# PipeGen Installation Script
# Usage: curl -sSL https://raw.githubusercontent.com/owner/pipegen/main/install.sh | bash

REPO="mcolomerc/pipegen"
BINARY_NAME="pipegen"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $OS in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        mingw*|cygwin*|msys*)
            OS="windows"
            ;;
        *)
            log_error "Unsupported operating system: $OS"
            ;;
    esac
    
    case $ARCH in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        arm64|aarch64)
            ARCH="arm64"
            ;;
        i386|i686)
            ARCH="386"
            ;;
        armv6l|armv7l)
            ARCH="arm"
            ;;
        *)
            log_error "Unsupported architecture: $ARCH"
            ;;
    esac
    
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="pipegen.exe"
        EXTENSION="zip"
    else
        EXTENSION="tar.gz"
    fi
    
    PLATFORM="${OS}-${ARCH}"
    log_info "Detected platform: $PLATFORM"
}

# Get the latest release version
get_latest_version() {
    log_info "Fetching latest release information..."
    
    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
    elif command -v wget >/dev/null 2>&1; then
        VERSION=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
    else
        log_error "curl or wget is required"
    fi
    
    if [ -z "$VERSION" ]; then
        log_error "Failed to fetch latest version"
    fi
    
    log_info "Latest version: $VERSION"
}

# Download and install binary
install_binary() {
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY_NAME%.*}-${PLATFORM}.${EXTENSION}"
    TEMP_DIR=$(mktemp -d)
    
    log_info "Downloading $BINARY_NAME $VERSION for $PLATFORM..."
    
    if command -v curl >/dev/null 2>&1; then
        curl -L "$DOWNLOAD_URL" -o "$TEMP_DIR/pipegen.${EXTENSION}"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$TEMP_DIR/pipegen.${EXTENSION}" "$DOWNLOAD_URL"
    else
        log_error "curl or wget is required"
    fi
    
    log_info "Extracting archive..."
    
    cd "$TEMP_DIR"
    if [ "$EXTENSION" = "zip" ]; then
        unzip -q "pipegen.${EXTENSION}"
    else
        tar -xzf "pipegen.${EXTENSION}"
    fi
    
    # Find the binary in the extracted files
    EXTRACTED_BINARY=$(find . -name "$BINARY_NAME" -type f | head -1)
    if [ -z "$EXTRACTED_BINARY" ]; then
        log_error "Binary not found in archive"
    fi
    
    # Make binary executable
    chmod +x "$EXTRACTED_BINARY"
    
    # Install binary
    if [ -w "$INSTALL_DIR" ]; then
        cp "$EXTRACTED_BINARY" "$INSTALL_DIR/$BINARY_NAME"
        log_success "$BINARY_NAME installed to $INSTALL_DIR"
    else
        log_warning "No write permission to $INSTALL_DIR, installing with sudo..."
        sudo cp "$EXTRACTED_BINARY" "$INSTALL_DIR/$BINARY_NAME"
        log_success "$BINARY_NAME installed to $INSTALL_DIR (with sudo)"
    fi
    
    # Cleanup
    rm -rf "$TEMP_DIR"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        VERSION_OUTPUT=$("$BINARY_NAME" version 2>/dev/null || "$BINARY_NAME" --version 2>/dev/null || echo "version command not available")
        log_success "Installation verified!"
        log_info "Version: $VERSION_OUTPUT"
        log_info "Location: $(which $BINARY_NAME)"
    else
        log_error "Installation failed - $BINARY_NAME not found in PATH"
    fi
}

# Show usage information
show_usage() {
    echo
    log_info "🚀 PipeGen is now installed!"
    echo
    echo "To get started:"
    echo "  1. Create a new pipeline project:"
    echo "     $ pipegen init my-pipeline"
    echo
    echo "  2. Generate with AI (optional):"
    echo "     $ pipegen init my-pipeline --ai --description 'e-commerce analytics'"
    echo
    echo "  3. Start the dashboard:"
    echo "     $ cd my-pipeline && pipegen dashboard"
    echo
    echo "  4. Run the pipeline:"
    echo "     $ pipegen run --dashboard"
    echo
    echo "Documentation: https://github.com/$REPO/blob/main/README.md"
    echo "Issues: https://github.com/$REPO/issues"
    echo
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --help)
            echo "PipeGen Installation Script"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --version VERSION    Install specific version (default: latest)"
            echo "  --install-dir DIR    Installation directory (default: /usr/local/bin)"
            echo "  --help              Show this help message"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            ;;
    esac
done

# Main installation process
main() {
    log_info "🔧 Starting PipeGen installation..."
    
    detect_platform
    
    if [ -z "$VERSION" ]; then
        get_latest_version
    fi
    
    install_binary
    verify_installation
    show_usage
}

# Check if running as root (not recommended)
if [ "$EUID" -eq 0 ]; then
    log_warning "Running as root is not recommended"
fi

# Run main installation
main
