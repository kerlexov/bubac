#!/bin/bash
# Security Configuration Validation Script for Coolify Deployment
# This script validates all security configurations before starting the application

set -e

echo "🔒 Validating MCP Logging Server Security Configuration..."

# Function to check if variable is set and not empty
check_var() {
    local var_name="$1"
    local var_value="${!var_name}"

    if [[ -z "$var_value" ]]; then
        echo "❌ Required variable $var_name is not set"
        return 1
    fi

    echo "✅ $var_name = $var_value"
    return 0
}

# Function to check boolean values
check_boolean() {
    local var_name="$1"
    local var_value="${!var_name}"

    case "$var_value" in
        "true"|"false"|1|0)
            echo "✅ $var_name = $var_value"
            return 0
            ;;
        *)
            echo "❌ $var_name must be true/false or 1/0, got: $var_value"
            return 1
            ;;
    esac
}

# Check required environment variables
echo "📋 Checking required environment variables..."

# Authentication settings
check_boolean "API_KEY_REQUIRED" || exit 1

# Rate limiting
check_boolean "RATE_LIMIT_ENABLED" || exit 1
if [[ "$RATE_LIMIT_ENABLED" == "true" ]]; then
    check_var "RATE_LIMIT_REQUESTS_PER_MINUTE" || exit 1
    check_var "RATE_LIMIT_BURST" || exit 1
fi

# Data protection
check_boolean "MASK_SENSITIVE_FIELDS" || exit 1
if [[ "$MASK_SENSITIVE_FIELDS" == "true" ]]; then
    check_var "SENSITIVE_FIELDS" || exit 1
fi

# TLS settings
check_boolean "TLS_ENABLED" || exit 1

# Audit settings
check_boolean "AUDIT_ENABLED" || exit 1

# Health check settings
check_boolean "HEALTH_CHECK_ENABLED" || exit 1

echo "📁 Checking configuration files..."

# Check API key configuration if required
if [[ "$API_KEY_REQUIRED" == "true" ]]; then
    if [[ ! -f "/app/config/api-keys.yaml" ]]; then
        echo "❌ API keys required but configuration file not found at /app/config/api-keys.yaml"
        exit 1
    fi

    if [[ ! -s "/app/config/api-keys.yaml" ]]; then
        echo "❌ API keys file exists but is empty"
        exit 1
    fi

    echo "✅ API keys configuration file exists and has content"
fi

# Check main configuration file
if [[ ! -f "/app/config/config.yaml" ]]; then
    echo "❌ Main configuration file not found at /app/config/config.yaml"
    exit 1
fi

echo "✅ Main configuration file exists"

# Check volume mounts
echo "💾 Checking volume mounts..."

volumes=(
    "/app/data"
    "/app/config"
    "/app/recovery"
    "/app/audit"
)

for volume in "${volumes[@]}"; do
    if [[ ! -d "$volume" ]]; then
        echo "❌ Volume mount $volume does not exist"
        exit 1
    fi

    # Check if volume is writable
    if ! touch "$volume/.test" 2>/dev/null; then
        echo "❌ Volume mount $volume is not writable"
        rm -f "$volume/.test" 2>/dev/null || true
        exit 1
    fi

    rm -f "$volume/.test"
    echo "✅ Volume mount $volume exists and is writable"
done

# Validate security headers configuration
echo "🔒 Checking security headers configuration..."

if [[ "${SECURITY_HEADERS_ENABLED:-true}" == "true" ]]; then
    echo "✅ Security headers are enabled"
else
    echo "⚠️  Security headers are disabled - ensure reverse proxy provides security headers"
fi

# Validate CORS configuration
if [[ "${CORS_ENABLED:-false}" == "true" ]]; then
    echo "⚠️  CORS is enabled - ensure CORS_ALLOWED_ORIGINS is properly configured"
    check_var "CORS_ALLOWED_ORIGINS" || exit 1
else
    echo "✅ CORS is disabled (recommended for security)"
fi

# Check for insecure configurations
echo "🚨 Checking for insecure configurations..."

if [[ "$TLS_ENABLED" == "true" ]] && [[ "$TLS_CERT_FILE" == "" ]]; then
    echo "❌ TLS enabled but no certificate file specified"
    exit 1
fi

if [[ "$RATE_LIMIT_ENABLED" == "false" ]]; then
    echo "⚠️  Rate limiting is disabled - this reduces security"
fi

if [[ "$AUDIT_ENABLED" == "false" ]]; then
    echo "⚠️  Audit logging is disabled - this reduces security monitoring"
fi

# Check resource limits
echo "⚡ Checking resource configuration..."

if [[ -n "${MEMORY_LIMIT:-}" ]]; then
    echo "✅ Memory limit configured: $MEMORY_LIMIT"
fi

if [[ -n "${CPU_LIMIT:-}" ]]; then
    echo "✅ CPU limit configured: $CPU_LIMIT"
fi

# Final validation
echo ""
echo "🎉 Security configuration validation completed successfully!"
echo ""
echo "📊 Summary:"
echo "   • Authentication: $(if [[ "$API_KEY_REQUIRED" == "true" ]]; then echo "Enabled"; else echo "Disabled"; fi)"
echo "   • Rate Limiting: $(if [[ "$RATE_LIMIT_ENABLED" == "true" ]]; then echo "Enabled"; else echo "Disabled"; fi)"
echo "   • Data Protection: $(if [[ "$MASK_SENSITIVE_FIELDS" == "true" ]]; then echo "Enabled"; else echo "Disabled"; fi)"
echo "   • TLS: $(if [[ "$TLS_ENABLED" == "true" ]]; then echo "Enabled"; else echo "Disabled"; fi)"
echo "   • Audit Logging: $(if [[ "$AUDIT_ENABLED" == "true" ]]; then echo "Enabled"; else echo "Disabled"; fi)"
echo ""
echo "🚀 Starting MCP Logging Server..."