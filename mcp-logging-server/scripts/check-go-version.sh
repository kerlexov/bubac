#!/bin/bash
# Go Version Validation Script
# Ensures the Go version meets the minimum requirements

set -e

REQUIRED_GO_VERSION="1.23.0"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

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

# Check if Go is installed
check_go_installed() {
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        log_info "Please install Go 1.23 or higher from https://golang.org/dl/"
        exit 1
    fi
}

# Get current Go version
get_go_version() {
    local version
    version=$(go version | awk '{print $3}' | sed 's/go//')
    echo "$version"
}

# Compare versions
version_compare() {
    local version1="$1"
    local version2="$2"

    if [[ "$version1" == "$version2" ]]; then
        return 0
    fi

    local IFS=.
    local v1=($version1)
    local v2=($version2)

    for ((i=0; i<${#v1[@]} || i<${#v2[@]}; i++)); do
        local num1=${v1[i]:-0}
        local num2=${v2[i]:-0}

        if (( num1 > num2 )); then
            return 1
        elif (( num1 < num2 )); then
            return 2
        fi
    done

    return 0
}

# Main validation
main() {
    log_info "Checking Go version requirements..."
    log_info "Required Go version: $REQUIRED_GO_VERSION"

    check_go_installed

    local current_version
    current_version=$(get_go_version)
    log_info "Current Go version: $current_version"

    version_compare "$current_version" "$REQUIRED_GO_VERSION"
    local comparison_result=$?

    case $comparison_result in
        0)
            log_success "Go version $current_version matches exactly"
            ;;
        1)
            log_success "Go version $current_version is newer than required"
            ;;
        2)
            log_error "Go version $current_version is older than required $REQUIRED_GO_VERSION"
            log_info "Please upgrade Go to version $REQUIRED_GO_VERSION or higher"
            log_info "Download from: https://golang.org/dl/"
            exit 1
            ;;
    esac

    # Check go.mod files
    log_info "Checking go.mod files..."

    local go_mod_files=(
        "$PROJECT_ROOT/mcp-logging-server/go.mod"
        "$PROJECT_ROOT/mcp-logging-go-sdk/go.mod"
    )

    for go_mod_file in "${go_mod_files[@]}"; do
        if [[ -f "$go_mod_file" ]]; then
            local mod_version
            mod_version=$(grep '^go ' "$go_mod_file" | awk '{print $2}')
            log_info "Found go.mod: $(basename "$(dirname "$go_mod_file")") requires Go $mod_version"

            if [[ "$mod_version" != "$REQUIRED_GO_VERSION" ]]; then
                log_warning "go.mod version ($mod_version) differs from required version ($REQUIRED_GO_VERSION)"
            fi
        fi
    done

    log_success "Go version validation completed successfully!"
    log_info "Environment is ready for MCP Logging System development"
}

# Run main function
main "$@"