#!/bin/bash
# Comprehensive Health Check Script for MCP Logging Server
# This script performs detailed health checks and can be used by Coolify

set -e

# Configuration
INGESTION_PORT="${INGESTION_PORT:-8080}"
MCP_PORT="${MCP_PORT:-8081}"
TIMEOUT="${HEALTH_CHECK_TIMEOUT:-10}"
VERBOSE="${VERBOSE:-false}"

# Exit codes
EXIT_OK=0
EXIT_WARNING=1
EXIT_CRITICAL=2
EXIT_UNKNOWN=3

# Colors for verbose output
if [[ "$VERBOSE" == "true" ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

# Logging functions
log_info() {
    [[ "$VERBOSE" == "true" ]] && echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    [[ "$VERBOSE" == "true" ]] && echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    [[ "$VERBOSE" == "true" ]] && echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    [[ "$VERBOSE" == "true" ]] && echo -e "${RED}[ERROR]${NC} $1"
}

# Check HTTP endpoint
check_http_endpoint() {
    local url="$1"
    local expected_status="${2:-200}"
    local name="$3"

    log_info "Checking $name endpoint: $url"

    local response
    local http_code

    # Use curl with timeout and capture both response and status
    if response=$(curl -s -w "HTTPSTATUS:%{http_code}" --max-time "$TIMEOUT" "$url" 2>/dev/null); then
        http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
        response_body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')

        if [[ "$http_code" == "$expected_status" ]]; then
            log_success "$name endpoint is healthy (HTTP $http_code)"

            # Additional checks for specific endpoints
            case "$name" in
                "Application Health")
                    if echo "$response_body" | grep -q "healthy\|ok\|up"; then
                        log_success "Health response indicates healthy status"
                        return 0
                    else
                        log_warning "Health response doesn't indicate healthy status: $response_body"
                        return $EXIT_WARNING
                    fi
                    ;;
                "Metrics")
                    if echo "$response_body" | grep -q "go_gc_duration_seconds\|prometheus"; then
                        log_success "Metrics endpoint is returning data"
                        return 0
                    else
                        log_warning "Metrics endpoint not returning expected data"
                        return $EXIT_WARNING
                    fi
                    ;;
                *)
                    return 0
                    ;;
            esac
        else
            log_error "$name endpoint returned HTTP $http_code (expected $expected_status)"
            return $EXIT_CRITICAL
        fi
    else
        log_error "$name endpoint is unreachable"
        return $EXIT_CRITICAL
    fi
}

# Check database connectivity
check_database() {
    log_info "Checking database connectivity..."

    # Try to access database file
    if [[ -f "/app/data/logs.db" ]]; then
        # Check if database is accessible
        if sqlite3 /app/data/logs.db "SELECT 1;" >/dev/null 2>&1; then
            log_success "Database is accessible"

            # Check database size
            local db_size
            db_size=$(stat -f%z "/app/data/logs.db" 2>/dev/null || stat -c%s "/app/data/logs.db" 2>/dev/null || echo "0")
            if [[ "$db_size" -gt 0 ]]; then
                log_info "Database size: $db_size bytes"
            fi

            return 0
        else
            log_error "Database is not accessible"
            return $EXIT_CRITICAL
        fi
    else
        log_warning "Database file not found (may be first startup)"
        return $EXIT_WARNING
    fi
}

# Check configuration files
check_configuration() {
    log_info "Checking configuration files..."

    local config_files=(
        "/app/config/config.yaml"
        "/app/config/api-keys.yaml"
    )

    local missing_configs=()

    for config_file in "${config_files[@]}"; do
        if [[ ! -f "$config_file" ]]; then
            missing_configs+=("$config_file")
        fi
    done

    if [[ ${#missing_configs[@]} -gt 0 ]]; then
        log_warning "Missing configuration files: ${missing_configs[*]}"
        return $EXIT_WARNING
    else
        log_success "All configuration files are present"
        return 0
    fi
}

# Check volumes and permissions
check_volumes() {
    log_info "Checking volume mounts and permissions..."

    local volumes=(
        "/app/data:rw"
        "/app/config:ro"
        "/app/recovery:rw"
        "/app/audit:rw"
    )

    local failed_volumes=()

    for volume_spec in "${volumes[@]}"; do
        IFS=':' read -r volume_path permissions <<< "$volume_spec"

        if [[ ! -d "$volume_path" ]]; then
            log_error "Volume mount $volume_path does not exist"
            failed_volumes+=("$volume_path")
            continue
        fi

        # Check permissions
        case "$permissions" in
            "rw")
                if ! touch "$volume_path/.test" 2>/dev/null; then
                    log_error "Volume mount $volume_path is not writable"
                    failed_volumes+=("$volume_path")
                else
                    rm -f "$volume_path/.test"
                fi
                ;;
            "ro")
                if touch "$volume_path/.test" 2>/dev/null; then
                    log_warning "Volume mount $volume_path should be read-only but is writable"
                    rm -f "$volume_path/.test"
                fi
                ;;
        esac
    done

    if [[ ${#failed_volumes[@]} -gt 0 ]]; then
        log_error "Volume checks failed for: ${failed_volumes[*]}"
        return $EXIT_CRITICAL
    else
        log_success "All volume mounts are properly configured"
        return 0
    fi
}

# Check system resources
check_resources() {
    log_info "Checking system resources..."

    # Check memory usage
    local mem_info
    mem_info=$(free -m 2>/dev/null || echo "Memory info not available")
    if [[ "$mem_info" != "Memory info not available" ]]; then
        log_info "Memory usage: $mem_info"
    fi

    # Check disk usage
    local disk_usage
    disk_usage=$(df -h /app/data 2>/dev/null | tail -1 || echo "Disk usage not available")
    if [[ "$disk_usage" != "Disk usage not available" ]]; then
        local disk_percent
        disk_percent=$(echo "$disk_usage" | awk '{print $5}' | sed 's/%//')

        if [[ "$disk_percent" -gt 90 ]]; then
            log_error "Disk usage is critically high: $disk_percent%"
            return $EXIT_CRITICAL
        elif [[ "$disk_percent" -gt 80 ]]; then
            log_warning "Disk usage is high: $disk_percent%"
            return $EXIT_WARNING
        else
            log_success "Disk usage is normal: $disk_percent%"
        fi
    fi

    return 0
}

# Check application metrics
check_metrics() {
    log_info "Checking application metrics..."

    local metrics_url="http://localhost:$INGESTION_PORT/metrics"

    if curl -s --max-time "$TIMEOUT" "$metrics_url" > /dev/null 2>&1; then
        log_success "Metrics endpoint is accessible"

        # Check for specific metrics
        local metrics_response
        metrics_response=$(curl -s --max-time "$TIMEOUT" "$metrics_url")

        # Check for Go runtime metrics
        if echo "$metrics_response" | grep -q "go_gc_duration_seconds"; then
            log_success "Go runtime metrics are available"
        else
            log_warning "Go runtime metrics not found"
        fi

        # Check for application-specific metrics
        if echo "$metrics_response" | grep -q "http_requests_total\|log_entries_total"; then
            log_success "Application metrics are available"
        else
            log_warning "Application-specific metrics not found"
        fi

        return 0
    else
        log_warning "Metrics endpoint is not accessible"
        return $EXIT_WARNING
    fi
}

# Check MCP server
check_mcp_server() {
    log_info "Checking MCP server..."

    local mcp_health_url="http://localhost:$MCP_PORT/mcp/health"

    if check_http_endpoint "$mcp_health_url" "200" "MCP Server"; then
        log_success "MCP server is healthy"
        return 0
    else
        log_error "MCP server health check failed"
        return $EXIT_CRITICAL
    fi
}

# Generate health report
generate_health_report() {
    local exit_code="$1"
    local report_file="/tmp/health-report-$(date +%Y%m%d_%H%M%S).json"

    cat > "$report_file" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "hostname": "$(hostname)",
    "status": "$([[ $exit_code -eq 0 ]] && echo "healthy" || echo "unhealthy")",
    "exit_code": $exit_code,
    "checks": {
        "application_health": $(check_http_endpoint "http://localhost:$INGESTION_PORT/health" "200" "Application Health" >/dev/null 2>&1 && echo "true" || echo "false"),
        "mcp_server": $(check_mcp_server >/dev/null 2>&1 && echo "true" || echo "false"),
        "database": $(check_database >/dev/null 2>&1 && echo "true" || echo "false"),
        "configuration": $(check_configuration >/dev/null 2>&1 && echo "true" || echo "false"),
        "volumes": $(check_volumes >/dev/null 2>&1 && echo "true" || echo "false"),
        "resources": $(check_resources >/dev/null 2>&1 && echo "true" || echo "false"),
        "metrics": $(check_metrics >/dev/null 2>&1 && echo "true" || echo "false")
    },
    "system_info": {
        "uptime": "$(uptime)",
        "load_average": "$(uptime | awk -F'load average:' '{ print $2 }' | sed 's/,//g')",
        "memory": "$(free -h 2>/dev/null | grep '^Mem:' | awk '{print $3 "/" $2}' || echo "N/A")",
        "disk": "$(df -h /app/data 2>/dev/null | tail -1 | awk '{print $3 "/" $2 " (" $5 ")"}' || echo "N/A")"
    }
}
EOF

    if [[ "$VERBOSE" == "true" ]]; then
        echo "Health report saved to: $report_file"
        cat "$report_file"
    fi
}

# Main execution
main() {
    log_info "Starting comprehensive health check for MCP Logging Server..."
    log_info "Ingestion Port: $INGESTION_PORT, MCP Port: $MCP_PORT, Timeout: $TIMEOUT"

    local overall_status=$EXIT_OK
    local check_results=()

    # Perform all checks
    log_info "Performing health checks..."

    # 1. Application health
    if ! check_http_endpoint "http://localhost:$INGESTION_PORT/health" "200" "Application Health"; then
        overall_status=$EXIT_CRITICAL
        check_results+=("application_health:FAILED")
    else
        check_results+=("application_health:PASSED")
    fi

    # 2. MCP server
    if ! check_mcp_server; then
        overall_status=$EXIT_CRITICAL
        check_results+=("mcp_server:FAILED")
    else
        check_results+=("mcp_server:PASSED")
    fi

    # 3. Database
    if ! check_database; then
        if [[ $overall_status -eq $EXIT_OK ]]; then
            overall_status=$EXIT_WARNING
        fi
        check_results+=("database:FAILED")
    else
        check_results+=("database:PASSED")
    fi

    # 4. Configuration
    if ! check_configuration; then
        if [[ $overall_status -eq $EXIT_OK ]]; then
            overall_status=$EXIT_WARNING
        fi
        check_results+=("configuration:FAILED")
    else
        check_results+=("configuration:PASSED")
    fi

    # 5. Volumes
    if ! check_volumes; then
        overall_status=$EXIT_CRITICAL
        check_results+=("volumes:FAILED")
    else
        check_results+=("volumes:PASSED")
    fi

    # 6. Resources
    if ! check_resources; then
        if [[ $overall_status -eq $EXIT_OK ]]; then
            overall_status=$EXIT_WARNING
        fi
        check_results+=("resources:FAILED")
    else
        check_results+=("resources:PASSED")
    fi

    # 7. Metrics
    if ! check_metrics; then
        if [[ $overall_status -eq $EXIT_OK ]]; then
            overall_status=$EXIT_WARNING
        fi
        check_results+=("metrics:FAILED")
    else
        check_results+=("metrics:PASSED")
    fi

    # Generate report
    generate_health_report "$overall_status"

    # Final status
    case $overall_status in
        $EXIT_OK)
            log_success "✅ All health checks passed - MCP Logging Server is healthy"
            ;;
        $EXIT_WARNING)
            log_warning "⚠️  Some health checks failed with warnings"
            ;;
        $EXIT_CRITICAL)
            log_error "❌ Critical health check failures detected"
            ;;
        *)
            log_error "❓ Unknown health check status"
            ;;
    esac

    if [[ "$VERBOSE" == "true" ]]; then
        echo ""
        echo "Check Results Summary:"
        printf '%s\n' "${check_results[@]}"
    fi

    exit $overall_status
}

# Run main function
main "$@"