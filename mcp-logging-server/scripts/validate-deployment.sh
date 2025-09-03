#!/bin/bash
# Deployment Validation Script for Coolify
# This script validates that the MCP Logging Server deployment is working correctly

set -e

# Configuration
INGESTION_PORT="${INGESTION_PORT:-8080}"
MCP_PORT="${MCP_PORT:-8081}"
API_KEY="${API_KEY:-test-api-key}"
TIMEOUT=30

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
}

# Check if service is running
check_service_running() {
    local service_name="$1"
    local container_name="$2"

    log_info "Checking if $service_name is running..."

    if ! docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
        log_error "Service $service_name ($container_name) is not running"
        return 1
    fi

    log_success "$service_name is running"
    return 0
}

# Check health endpoint
check_health_endpoint() {
    local url="$1"
    local service_name="$2"

    log_info "Checking health endpoint: $url"

    local response
    if ! response=$(curl -s --max-time $TIMEOUT "$url" 2>/dev/null); then
        log_error "Health check failed for $service_name"
        return 1
    fi

    # Check if response contains expected health indicators
    if echo "$response" | grep -q "healthy\|ok\|up"; then
        log_success "$service_name health check passed"
        return 0
    else
        log_warning "$service_name health check response: $response"
        return 1
    fi
}

# Test API key authentication
test_api_authentication() {
    log_info "Testing API key authentication..."

    local test_url="http://localhost:$INGESTION_PORT/v1/logs"
    local test_payload='{"logs":[{"level":"INFO","message":"test","service_name":"test","agent_id":"test"}]}'

    # Test without API key (should fail if auth is required)
    local response_no_key
    response_no_key=$(curl -s -w "%{http_code}" --max-time $TIMEOUT \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$test_payload" \
        "$test_url" 2>/dev/null || echo "000")

    local status_code="${response_no_key: -3}"

    if [[ "$API_KEY_REQUIRED" == "true" ]]; then
        if [[ "$status_code" == "401" ]] || [[ "$status_code" == "403" ]]; then
            log_success "API key authentication is working (correctly rejected without key)"
        else
            log_warning "API key authentication may not be working (status: $status_code)"
        fi
    else
        log_info "API key authentication is disabled"
    fi

    # Test with API key
    if [[ -n "$API_KEY" ]]; then
        local response_with_key
        response_with_key=$(curl -s -w "%{http_code}" --max-time $TIMEOUT \
            -X POST \
            -H "Content-Type: application/json" \
            -H "X-API-Key: $API_KEY" \
            -d "$test_payload" \
            "$test_url" 2>/dev/null || echo "000")

        local status_code_with_key="${response_with_key: -3}"

        if [[ "$status_code_with_key" == "200" ]] || [[ "$status_code_with_key" == "201" ]]; then
            log_success "API key authentication works with valid key"
        else
            log_error "API key authentication failed with valid key (status: $status_code_with_key)"
            return 1
        fi
    fi
}

# Test rate limiting
test_rate_limiting() {
    if [[ "$RATE_LIMIT_ENABLED" != "true" ]]; then
        log_info "Rate limiting is disabled, skipping test"
        return 0
    fi

    log_info "Testing rate limiting..."

    local test_url="http://localhost:$INGESTION_PORT/health"
    local request_count=10
    local rate_limited=false

    for i in $(seq 1 $request_count); do
        local response
        response=$(curl -s -w "%{http_code}" --max-time 5 \
            -H "X-API-Key: $API_KEY" \
            "$test_url" 2>/dev/null || echo "000")

        local status_code="${response: -3}"

        if [[ "$status_code" == "429" ]]; then
            rate_limited=true
            break
        fi

        # Small delay between requests
        sleep 0.1
    done

    if [[ "$rate_limited" == "true" ]]; then
        log_success "Rate limiting is working correctly"
    else
        log_warning "Rate limiting may not be working (no 429 status received)"
    fi
}

# Test data protection
test_data_protection() {
    if [[ "$MASK_SENSITIVE_FIELDS" != "true" ]]; then
        log_info "Data protection is disabled, skipping test"
        return 0
    fi

    log_info "Testing data protection..."

    local test_url="http://localhost:$INGESTION_PORT/v1/logs"
    local test_payload='{"logs":[{"level":"INFO","message":"test","password":"secret123","token":"abc123","service_name":"test","agent_id":"test"}]}'

    local response
    response=$(curl -s --max-time $TIMEOUT \
        -X POST \
        -H "Content-Type: application/json" \
        -H "X-API-Key: $API_KEY" \
        -d "$test_payload" \
        "$test_url" 2>/dev/null || echo "")

    if echo "$response" | grep -q "***"; then
        log_success "Data protection is working (sensitive fields masked)"
    else
        log_warning "Data protection may not be working (no masking detected)"
    fi
}

# Test MCP server
test_mcp_server() {
    log_info "Testing MCP server..."

    local mcp_url="http://localhost:$MCP_PORT/mcp/health"

    if curl -s --max-time $TIMEOUT "$mcp_url" > /dev/null 2>&1; then
        log_success "MCP server is responding"
    else
        log_error "MCP server is not responding"
        return 1
    fi
}

# Test monitoring stack
test_monitoring_stack() {
    log_info "Testing monitoring stack..."

    local services=(
        "prometheus:9090"
        "grafana:3000"
        "loki:3100"
        "alertmanager:9093"
        "node-exporter:9100"
    )

    local failed_services=()

    for service in "${services[@]}"; do
        local name="${service%%:*}"
        local port="${service##*:}"
        local url="http://localhost:$port"

        if curl -s --max-time 5 "$url" > /dev/null 2>&1; then
            log_success "$name is responding on port $port"
        else
            log_warning "$name is not responding on port $port"
            failed_services+=("$name")
        fi
    done

    if [[ ${#failed_services[@]} -gt 0 ]]; then
        log_warning "Some monitoring services are not responding: ${failed_services[*]}"
    else
        log_success "All monitoring services are responding"
    fi
}

# Check volumes
check_volumes() {
    log_info "Checking Docker volumes..."

    local volumes=(
        "mcp_logging_data"
        "mcp_logging_config"
        "mcp_logging_recovery"
        "mcp_logging_audit"
    )

    for volume in "${volumes[@]}"; do
        if docker volume ls --format "{{.Name}}" | grep -q "^${volume}$"; then
            log_success "Volume $volume exists"
        else
            log_error "Volume $volume does not exist"
            return 1
        fi
    done
}

# Generate report
generate_report() {
    log_info "Generating deployment validation report..."

    local report_file="/tmp/deployment-validation-$(date +%Y%m%d_%H%M%S).txt"

    cat > "$report_file" << EOF
MCP Logging Server Deployment Validation Report
==============================================

Generated: $(date)
Environment: $(hostname)

SERVICES STATUS:
$(docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}")

HEALTH CHECKS:
- MCP Server: $(curl -s http://localhost:$INGESTION_PORT/health 2>/dev/null || echo "FAILED")
- MCP API: $(curl -s http://localhost:$MCP_PORT/mcp/health 2>/dev/null || echo "FAILED")

CONFIGURATION:
- API Key Required: ${API_KEY_REQUIRED:-false}
- Rate Limiting: ${RATE_LIMIT_ENABLED:-false}
- Data Protection: ${MASK_SENSITIVE_FIELDS:-false}
- Audit Logging: ${AUDIT_ENABLED:-false}

VOLUMES:
$(docker volume ls)

SYSTEM RESOURCES:
$(df -h /app/data 2>/dev/null || echo "Volume mount info not available")

EOF

    log_success "Report generated: $report_file"
    echo "Report location: $report_file"
}

# Main execution
main() {
    log_info "Starting MCP Logging Server deployment validation..."
    log_info "Ingestion Port: $INGESTION_PORT"
    log_info "MCP Port: $MCP_PORT"

    local failed_checks=0

    # Check main service
    if ! check_service_running "MCP Logging Server" "mcp-logging-server"; then
        ((failed_checks++))
    fi

    # Check health endpoints
    if ! check_health_endpoint "http://localhost:$INGESTION_PORT/health" "MCP Server"; then
        ((failed_checks++))
    fi

    # Test API authentication
    if ! test_api_authentication; then
        ((failed_checks++))
    fi

    # Test rate limiting
    if ! test_rate_limiting; then
        ((failed_checks++))
    fi

    # Test data protection
    if ! test_data_protection; then
        ((failed_checks++))
    fi

    # Test MCP server
    if ! test_mcp_server; then
        ((failed_checks++))
    fi

    # Test monitoring stack
    if ! test_monitoring_stack; then
        ((failed_checks++))
    fi

    # Check volumes
    if ! check_volumes; then
        ((failed_checks++))
    fi

    # Generate report
    generate_report

    # Final result
    echo ""
    if [[ $failed_checks -eq 0 ]]; then
        log_success "✅ Deployment validation completed successfully!"
        log_success "All checks passed - MCP Logging Server is ready for production use."
        exit 0
    else
        log_error "❌ Deployment validation failed with $failed_checks check(s) failing"
        log_error "Please review the issues above and fix them before using in production."
        exit 1
    fi
}

# Run main function
main "$@"