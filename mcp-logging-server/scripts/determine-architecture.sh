#!/bin/bash
# Architecture Detection Script for MCP Logging Server
# Helps determine the correct build parameters for different server architectures

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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
}

# Detect system architecture
detect_architecture() {
    local arch
    arch=$(uname -m)

    case "$arch" in
        x86_64|amd64)
            echo "amd64"
            ;;
        aarch64|arm64)
            echo "arm64"
            ;;
        armv7l|armv7)
            echo "arm"
            ;;
        *)
            log_warning "Unknown architecture: $arch"
            echo "amd64"  # Default fallback
            ;;
    esac
}

# Detect operating system
detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')

    case "$os" in
        linux)
            echo "linux"
            ;;
        darwin)
            echo "darwin"
            ;;
        windows)
            echo "windows"
            ;;
        *)
            log_warning "Unknown OS: $os"
            echo "linux"  # Default fallback
            ;;
    esac
}

# Get Docker platform string
get_docker_platform() {
    local arch="$1"
    local os="$2"

    case "$arch" in
        amd64)
            echo "linux/amd64"
            ;;
        arm64)
            echo "linux/arm64"
            ;;
        arm)
            echo "linux/arm/v7"
            ;;
        *)
            echo "linux/amd64"
            ;;
    esac
}

# Main detection
main() {
    log_info "Detecting system architecture and OS..."

    local detected_arch
    local detected_os
    local docker_platform

    detected_arch=$(detect_architecture)
    detected_os=$(detect_os)
    docker_platform=$(get_docker_platform "$detected_arch" "$detected_os")

    log_success "System Information:"
    echo "  Architecture: $detected_arch"
    echo "  Operating System: $detected_os"
    echo "  Docker Platform: $docker_platform"
    echo ""

    log_info "Recommended Docker build commands:"
    echo ""
    echo "# Build for current architecture:"
    echo "docker build --build-arg TARGETARCH=$detected_arch --build-arg TARGETOS=$detected_os -t mcp-logging-server ."
    echo ""
    echo "# Build for specific platform (e.g., Hetzner x86_64):"
    echo "docker build --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux -t mcp-logging-server ."
    echo ""
    echo "# Build for ARM64 (e.g., AWS Graviton, Apple Silicon):"
    echo "docker build --build-arg TARGETARCH=arm64 --build-arg TARGETOS=linux -t mcp-logging-server ."
    echo ""
    echo "# Build multi-platform (requires Docker Buildx):"
    echo "docker buildx build --platform linux/amd64,linux/arm64 -t mcp-logging-server ."
    echo ""

    # Check if running in Docker
    if [[ -f "/.dockerenv" ]]; then
        log_info "Running inside Docker container"
        echo "  Container Architecture: $(uname -m)"
        echo "  Go Version: $(go version 2>/dev/null || echo 'Go not found')"
        echo "  GCC Version: $(gcc --version 2>/dev/null | head -1 || echo 'GCC not found')"
    fi

    # Provide troubleshooting tips
    echo ""
    log_info "Troubleshooting Tips:"
    echo "• If you get ARM64 assembly errors on x86_64, use: --build-arg TARGETARCH=amd64"
    echo "• If you get x86_64 assembly errors on ARM64, use: --build-arg TARGETARCH=arm64"
    echo "• For Hetzner servers, use: --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux"
    echo "• For AWS Graviton/ARM64, use: --build-arg TARGETARCH=arm64 --build-arg TARGETOS=linux"
    echo ""

    # Architecture-specific warnings
    case "$detected_arch" in
        arm64)
            if [[ "$detected_os" == "linux" ]]; then
                log_warning "ARM64 Linux detected. Make sure your GCC supports ARM64 assembly."
                echo "  If build fails, try: apt-get install gcc-aarch64-linux-gnu"
            fi
            ;;
        amd64)
            log_success "x86_64 architecture detected - should work with standard GCC."
            ;;
    esac
}

# Run main function
main "$@"