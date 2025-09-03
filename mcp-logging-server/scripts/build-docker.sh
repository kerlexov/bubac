#!/bin/bash
# Smart Docker Build Script for MCP Logging Server
# Automatically detects architecture and builds with correct parameters

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
            log_warning "Unknown architecture: $arch, defaulting to amd64"
            echo "amd64"
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
            log_warning "Unknown OS: $os, defaulting to linux"
            echo "linux"
            ;;
    esac
}

# Check if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed or not in PATH"
        log_info "Please install Docker from: https://docs.docker.com/get-docker/"
        exit 1
    fi

    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        log_info "Please start Docker service"
        exit 1
    fi
}

# Check if we're in the right directory
check_directory() {
    if [[ ! -f "Dockerfile" ]]; then
        log_error "Dockerfile not found in current directory"
        log_info "Please run this script from the mcp-logging-server directory"
        exit 1
    fi

    if [[ ! -f "go.mod" ]]; then
        log_error "go.mod not found - this doesn't look like the server directory"
        exit 1
    fi
}

# Parse command line arguments
parse_args() {
    FORCE_ARCH=""
    FORCE_OS=""
    NO_CACHE=false
    TAG="mcp-logging-server:latest"

    while [[ $# -gt 0 ]]; do
        case $1 in
            --arch=*)
                FORCE_ARCH="${1#*=}"
                shift
                ;;
            --os=*)
                FORCE_OS="${1#*=}"
                shift
                ;;
            --no-cache)
                NO_CACHE=true
                shift
                ;;
            --tag=*)
                TAG="${1#*=}"
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --arch=ARCH      Force specific architecture (amd64, arm64, arm)"
                echo "  --os=OS          Force specific OS (linux, darwin, windows)"
                echo "  --no-cache       Build without using Docker cache"
                echo "  --tag=TAG        Docker image tag (default: mcp-logging-server:latest)"
                echo "  --help           Show this help message"
                echo ""
                echo "Examples:"
                echo "  $0                                    # Auto-detect architecture"
                echo "  $0 --arch=amd64                      # Force x86_64 build"
                echo "  $0 --arch=arm64 --no-cache           # Force ARM64 build without cache"
                echo "  $0 --tag=my-registry.com/mcp-server  # Custom image tag"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                log_info "Use --help for usage information"
                exit 1
                ;;
        esac
    done
}

# Main build function
main() {
    log_info "MCP Logging Server Docker Build Script"
    log_info "====================================="

    # Parse command line arguments
    parse_args "$@"

    # Pre-flight checks
    check_directory
    check_docker

    # Detect or use forced architecture
    local target_arch
    local target_os

    if [[ -n "$FORCE_ARCH" ]]; then
        target_arch="$FORCE_ARCH"
        log_info "Using forced architecture: $target_arch"
    else
        target_arch=$(detect_architecture)
        log_info "Detected architecture: $target_arch"
    fi

    if [[ -n "$FORCE_OS" ]]; then
        target_os="$FORCE_OS"
        log_info "Using forced OS: $target_os"
    else
        target_os=$(detect_os)
        log_info "Detected OS: $target_os"
    fi

    # Validate architecture/OS combination
    case "$target_arch" in
        amd64|arm64|arm)
            ;;
        *)
            log_error "Unsupported architecture: $target_arch"
            log_info "Supported architectures: amd64, arm64, arm"
            exit 1
            ;;
    esac

    case "$target_os" in
        linux|darwin|windows)
            ;;
        *)
            log_error "Unsupported OS: $target_os"
            log_info "Supported OS: linux, darwin, windows"
            exit 1
            ;;
    esac

    # Build Docker command
    local docker_cmd="docker build"

    if [[ "$NO_CACHE" == "true" ]]; then
        docker_cmd="$docker_cmd --no-cache"
    fi

    docker_cmd="$docker_cmd --build-arg TARGETARCH=$target_arch"
    docker_cmd="$docker_cmd --build-arg TARGETOS=$target_os"
    docker_cmd="$docker_cmd -t $TAG"
    docker_cmd="$docker_cmd ."

    log_info "Build Configuration:"
    echo "  Target Architecture: $target_arch"
    echo "  Target OS: $target_os"
    echo "  Image Tag: $TAG"
    echo "  No Cache: $NO_CACHE"
    echo ""

    log_info "Starting Docker build..."
    log_info "Command: $docker_cmd"

    # Execute build
    if eval "$docker_cmd"; then
        log_success "✅ Docker build completed successfully!"
        log_info "Image tagged as: $TAG"
        echo ""
        log_info "Next steps:"
        echo "  • Test the image: docker run -p 8080:8080 $TAG"
        echo "  • Deploy to Coolify: Use docker-compose.coolify-monitoring.yml"
        echo "  • Check logs: docker logs <container-name>"
    else
        log_error "❌ Docker build failed!"
        log_info "Troubleshooting tips:"
        echo "  • Check Docker daemon: docker info"
        echo "  • Clear Docker cache: docker system prune -a"
        echo "  • Check architecture: ./scripts/determine-architecture.sh"
        echo "  • Force architecture: $0 --arch=amd64"
        exit 1
    fi
}

# Run main function
main "$@"