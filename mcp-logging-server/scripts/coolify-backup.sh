#!/bin/bash
# Coolify Backup Script for MCP Logging Server
# This script creates comprehensive backups of all MCP Logging Server data

set -e

# Configuration
BACKUP_ROOT="/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="${BACKUP_ROOT}/${TIMESTAMP}"
COMPRESSION_LEVEL=6
RETENTION_DAYS=30

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

# Create backup directory
create_backup_dir() {
    log_info "Creating backup directory: $BACKUP_DIR"
    mkdir -p "$BACKUP_DIR"
    if [[ ! -d "$BACKUP_DIR" ]]; then
        log_error "Failed to create backup directory"
        exit 1
    fi
}

# Backup database
backup_database() {
    log_info "Backing up database..."

    if [[ -f "/app/data/logs.db" ]]; then
        # Create database backup
        sqlite3 /app/data/logs.db ".backup '$BACKUP_DIR/database.db'" || {
            log_error "Failed to backup database"
            return 1
        }

        # Compress database backup
        gzip -${COMPRESSION_LEVEL} "$BACKUP_DIR/database.db"
        log_success "Database backup completed"
    else
        log_warning "Database file not found, skipping database backup"
    fi
}

# Backup configuration
backup_config() {
    log_info "Backing up configuration..."

    if [[ -d "/app/config" ]]; then
        tar -czf "$BACKUP_DIR/config.tar.gz" -C /app config/ || {
            log_error "Failed to backup configuration"
            return 1
        }
        log_success "Configuration backup completed"
    else
        log_warning "Configuration directory not found"
    fi
}

# Backup audit logs
backup_audit() {
    log_info "Backing up audit logs..."

    if [[ -d "/app/audit" ]]; then
        tar -czf "$BACKUP_DIR/audit.tar.gz" -C /app audit/ || {
            log_error "Failed to backup audit logs"
            return 1
        }
        log_success "Audit logs backup completed"
    else
        log_warning "Audit directory not found"
    fi
}

# Backup recovery data
backup_recovery() {
    log_info "Backing up recovery data..."

    if [[ -d "/app/recovery" ]]; then
        tar -czf "$BACKUP_DIR/recovery.tar.gz" -C /app recovery/ || {
            log_error "Failed to backup recovery data"
            return 1
        }
        log_success "Recovery data backup completed"
    else
        log_warning "Recovery directory not found"
    fi
}

# Backup monitoring data
backup_monitoring() {
    log_info "Backing up monitoring data..."

    # Backup Prometheus data
    if docker volume ls | grep -q prometheus_data; then
        docker run --rm \
            -v prometheus_data:/data \
            -v "$BACKUP_DIR:/backup" \
            alpine:latest \
            tar czf "/backup/prometheus.tar.gz" -C /data . || {
                log_warning "Failed to backup Prometheus data"
            }
    fi

    # Backup Grafana data
    if docker volume ls | grep -q grafana_data; then
        docker run --rm \
            -v grafana_data:/data \
            -v "$BACKUP_DIR:/backup" \
            alpine:latest \
            tar czf "/backup/grafana.tar.gz" -C /data . || {
                log_warning "Failed to backup Grafana data"
            }
    fi

    # Backup Loki data
    if docker volume ls | grep -q loki_data; then
        docker run --rm \
            -v loki_data:/data \
            -v "$BACKUP_DIR:/backup" \
            alpine:latest \
            tar czf "/backup/loki.tar.gz" -C /data . || {
                log_warning "Failed to backup Loki data"
            }
    fi

    log_success "Monitoring data backup completed"
}

# Create backup manifest
create_manifest() {
    log_info "Creating backup manifest..."

    local manifest_file="$BACKUP_DIR/manifest.json"

    cat > "$manifest_file" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "version": "1.0",
    "backup_type": "full",
    "compression_level": $COMPRESSION_LEVEL,
    "components": {
        "database": $([[ -f "$BACKUP_DIR/database.db.gz" ]] && echo "true" || echo "false"),
        "config": $([[ -f "$BACKUP_DIR/config.tar.gz" ]] && echo "true" || echo "false"),
        "audit": $([[ -f "$BACKUP_DIR/audit.tar.gz" ]] && echo "true" || echo "false"),
        "recovery": $([[ -f "$BACKUP_DIR/recovery.tar.gz" ]] && echo "true" || echo "false"),
        "prometheus": $([[ -f "$BACKUP_DIR/prometheus.tar.gz" ]] && echo "true" || echo "false"),
        "grafana": $([[ -f "$BACKUP_DIR/grafana.tar.gz" ]] && echo "true" || echo "false"),
        "loki": $([[ -f "$BACKUP_DIR/loki.tar.gz" ]] && echo "true" || echo "false")
    },
    "total_size_bytes": $(du -bc "$BACKUP_DIR"/* 2>/dev/null | tail -1 | cut -f1 || echo "0"),
    "created_by": "coolify-backup-script",
    "coolify_version": "4.0"
}
EOF

    log_success "Backup manifest created"
}

# Calculate backup size
calculate_size() {
    local size=$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1)
    log_info "Backup size: $size"
}

# Cleanup old backups
cleanup_old_backups() {
    log_info "Cleaning up backups older than $RETENTION_DAYS days..."

    if [[ -d "$BACKUP_ROOT" ]]; then
        find "$BACKUP_ROOT" -name "20*" -type d -mtime +$RETENTION_DAYS -exec rm -rf {} \; 2>/dev/null || true
        local cleaned=$(find "$BACKUP_ROOT" -name "20*" -type d -mtime +$RETENTION_DAYS 2>/dev/null | wc -l)
        if [[ $cleaned -gt 0 ]]; then
            log_success "Cleaned up $cleaned old backup(s)"
        fi
    fi
}

# Verify backup integrity
verify_backup() {
    log_info "Verifying backup integrity..."

    local failed=0

    # Check if all expected files exist
    local expected_files=(
        "manifest.json"
    )

    for file in "${expected_files[@]}"; do
        if [[ ! -f "$BACKUP_DIR/$file" ]]; then
            log_error "Missing expected file: $file"
            failed=1
        fi
    done

    # Verify compressed files are not corrupted
    for file in "$BACKUP_DIR"/*.gz; do
        if [[ -f "$file" ]]; then
            if ! gzip -t "$file" 2>/dev/null; then
                log_error "Corrupted compressed file: $(basename "$file")"
                failed=1
            fi
        fi
    done

    if [[ $failed -eq 0 ]]; then
        log_success "Backup integrity verification passed"
    else
        log_error "Backup integrity verification failed"
        return 1
    fi
}

# Send notification (if webhook URL is provided)
send_notification() {
    if [[ -n "${NOTIFICATION_WEBHOOK:-}" ]]; then
        log_info "Sending backup notification..."

        local payload=$(cat <<EOF
{
    "timestamp": "$(date -Iseconds)",
    "status": "success",
    "backup_path": "$BACKUP_DIR",
    "size": "$(du -sh "$BACKUP_DIR" 2>/dev/null | cut -f1)",
    "components": $(cat "$BACKUP_DIR/manifest.json" | jq '.components' 2>/dev/null || echo "{}")
}
EOF
        )

        curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$payload" \
            "$NOTIFICATION_WEBHOOK" || {
                log_warning "Failed to send notification"
            }
    fi
}

# Main execution
main() {
    log_info "Starting MCP Logging Server backup..."
    log_info "Timestamp: $TIMESTAMP"

    # Pre-flight checks
    if [[ ! -w "$BACKUP_ROOT" ]]; then
        log_error "Backup root directory is not writable: $BACKUP_ROOT"
        exit 1
    fi

    # Create backup directory
    create_backup_dir

    # Perform backups
    backup_database
    backup_config
    backup_audit
    backup_recovery
    backup_monitoring

    # Create manifest and verify
    create_manifest
    calculate_size

    if verify_backup; then
        log_success "Backup completed successfully!"
        log_info "Backup location: $BACKUP_DIR"

        # Cleanup old backups
        cleanup_old_backups

        # Send notification
        send_notification

        exit 0
    else
        log_error "Backup verification failed!"
        exit 1
    fi
}

# Run main function
main "$@"