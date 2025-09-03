# MCP Logging Server - Coolify Deployment Guide

This guide provides comprehensive instructions for deploying the MCP Logging Server on Coolify with proper security configurations.

## Prerequisites

- Coolify instance running (v4.0 or later)
- Domain names configured for API and MCP endpoints
- SSL certificates (handled automatically by Coolify)
- Go 1.23 or higher (for local development and Docker builds)

## Quick Start

1. **Clone the repository** to your Coolify server or use Git deployment
2. **Copy environment configuration**:
   ```bash
   cp .env.coolify .env
   ```
3. **Configure your domains** in `.env`
4. **Generate API keys** (see API Key Management section)
5. **Deploy using Coolify** with `docker-compose.coolify.yml`

## Environment Configuration

### Required Environment Variables

Copy `.env.coolify` to `.env` and configure the following:

```bash
# Domain Configuration (REQUIRED)
API_DOMAIN=api.mcp-logging.yourdomain.com
MCP_DOMAIN=mcp.mcp-logging.yourdomain.com

# Authentication (REQUIRED)
API_KEY_REQUIRED=true

# Rate Limiting (RECOMMENDED)
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=1000
RATE_LIMIT_BURST=100

# Data Protection (RECOMMENDED)
MASK_SENSITIVE_FIELDS=true
SENSITIVE_FIELDS=password,token,secret,key,authorization,credit_card,ssn,api_key
```

### Optional Environment Variables

```bash
# Logging Configuration
LOG_LEVEL=INFO                    # DEBUG, INFO, WARN, ERROR
LOG_FORMAT=json                   # json, text

# Buffer Configuration
BUFFER_SIZE=10000
BUFFER_FLUSH_TIMEOUT=5s
BUFFER_MAX_BATCH_SIZE=100

# Retention Configuration
RETENTION_DEFAULT_DAYS=30
RETENTION_DEBUG_DAYS=7
RETENTION_INFO_DAYS=30
RETENTION_WARN_DAYS=90
RETENTION_ERROR_DAYS=365
RETENTION_FATAL_DAYS=365

# Monitoring
METRICS_ENABLED=true
HEALTH_CHECK_ENABLED=true
AUDIT_ENABLED=true
AUDIT_LOG_RETENTION_DAYS=90
```

## API Key Management

### Generating API Keys

Use the built-in API key generator:

```bash
# Generate a new API key
docker exec -it mcp-logging-server ./apikey generate \
  --name "production-service" \
  --permissions "ingest_logs,metrics" \
  --rate-limit 5000

# Generate admin key with expiration
docker exec -it mcp-logging-server ./apikey generate \
  --name "admin-key" \
  --permissions "admin" \
  --rate-limit 10000 \
  --expires "2025-12-31T23:59:59Z"

# Generate read-only monitoring key
docker exec -it mcp-logging-server ./apikey generate \
  --name "monitoring-readonly" \
  --permissions "query_logs,metrics" \
  --rate-limit 1000
```

### Managing API Keys

```bash
# List all API keys
docker exec -it mcp-logging-server ./apikey list

# Revoke an API key
docker exec -it mcp-logging-server ./apikey revoke --key "key-hash-here"

# Update API key permissions
docker exec -it mcp-logging-server ./apikey update \
  --key "key-hash-here" \
  --permissions "ingest_logs,metrics,query_logs"
```

### API Key Configuration File

The system uses `/app/config/api-keys.yaml` for API key storage. This file is automatically managed by the API key commands, but you can also manually configure it:

```yaml
require_auth: true
api_keys:
  "your-hashed-api-key-here":
    name: "service-name"
    permissions:
      - "ingest_logs"
      - "metrics"
    rate_limit: 5000
    created_at: "2024-01-15T10:30:00Z"
    is_active: true
```

## Security Configuration

### TLS/HTTPS

TLS is handled automatically by Coolify's reverse proxy (Traefik). The application runs on HTTP internally and Coolify terminates TLS.

**Important**: Never set `TLS_ENABLED=true` when deploying on Coolify.

### Security Headers

Security headers are configured via Traefik labels in the Docker Compose file:

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Strict-Transport-Security: max-age=31536000; includeSubDomains`
- `Content-Security-Policy: default-src 'self'`
- `Referrer-Policy: strict-origin-when-cross-origin`

### Rate Limiting

Rate limiting is implemented at both the application and reverse proxy levels:

**Application Level** (recommended):
```bash
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=1000
RATE_LIMIT_BURST=100
```

**Traefik Level** (additional protection):
- Configured via Docker labels in `docker-compose.coolify.yml`
- Average: 100 requests/second
- Burst: 200 requests

### Data Protection

Configure field masking for sensitive data:

```bash
MASK_SENSITIVE_FIELDS=true
SENSITIVE_FIELDS=password,token,secret,key,authorization,credit_card,ssn,api_key
```

Supported protection methods:
- **Masking**: Replace with `***`
- **Hashing**: SHA-256 hash of the value
- **Dropping**: Remove field entirely

## Volume Mounts and Persistence

The deployment uses named volumes for data persistence:

- `mcp_logging_data`: SQLite database and log storage
- `mcp_logging_config`: Configuration files (API keys, etc.)
- `mcp_logging_recovery`: Recovery data for crash scenarios
- `mcp_logging_audit`: Audit logs for security events

### Backup Strategy

```bash
# Backup data volume
docker run --rm -v mcp_logging_data:/data -v $(pwd):/backup alpine \
  tar czf /backup/mcp-logging-data-$(date +%Y%m%d).tar.gz -C /data .

# Backup config volume
docker run --rm -v mcp_logging_config:/config -v $(pwd):/backup alpine \
  tar czf /backup/mcp-logging-config-$(date +%Y%m%d).tar.gz -C /config .
```

## Health Checks and Monitoring

### Health Check Endpoints

- **Application Health**: `GET /health`
- **Metrics**: `GET /metrics` (Prometheus format)
- **API Status**: `GET /v1/status`

### Coolify Integration

The deployment includes Coolify-specific labels for:
- Automatic health checking
- Service discovery
- Load balancing
- SSL certificate management

### Monitoring Setup

1. **Health Checks**: Configured automatically via Docker Compose
2. **Metrics Collection**: Prometheus-compatible metrics at `/metrics`
3. **Log Aggregation**: Application logs are available via `docker logs`
4. **Audit Logging**: Security events logged to audit volume

## Deployment Steps

### 1. Prepare Configuration

```bash
# Clone repository
git clone <repository-url>
cd mcp-logging-server

# Check Go version compatibility
./scripts/check-go-version.sh

# Determine correct build architecture for your server
./scripts/determine-architecture.sh

# Copy and configure environment
cp .env.coolify .env
# Edit .env with your domain names and settings
```

### 2. Deploy on Coolify

1. **Create New Application** in Coolify
2. **Select Docker Compose** deployment type
3. **Point to** `docker-compose.coolify.yml`
4. **Configure Environment Variables** from your `.env` file
5. **Set Domain Names** for both API and MCP endpoints
6. **Deploy**

### 3. Post-Deployment Setup

```bash
# Generate initial API keys
docker exec -it mcp-logging-server ./apikey generate \
  --name "initial-admin" \
  --permissions "admin" \
  --rate-limit 10000

# Verify health
curl https://api.mcp-logging.yourdomain.com/health

# Test MCP endpoint
curl https://mcp.mcp-logging.yourdomain.com/mcp/health
```

## Troubleshooting

### Architecture-Specific Builds

The Dockerfile supports cross-platform builds. You can either use the automated build script or manually specify architecture parameters:

#### Automated Build (Recommended)
```bash
# Automatic architecture detection and build
./scripts/build-docker.sh

# Build with custom tag
./scripts/build-docker.sh --tag=my-registry.com/mcp-server

# Build without cache
./scripts/build-docker.sh --no-cache

# Force specific architecture
./scripts/build-docker.sh --arch=amd64  # For Hetzner x86_64
./scripts/build-docker.sh --arch=arm64  # For AWS Graviton
```

#### Manual Build
```bash
# For Hetzner (x86_64) servers:
docker build --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux -t mcp-logging-server .

# For AWS Graviton/ARM64 servers:
docker build --build-arg TARGETARCH=arm64 --build-arg TARGETOS=linux -t mcp-logging-server .

# For automatic detection:
./scripts/determine-architecture.sh
```

### Common Issues

1. **Go Version Compatibility Error**
   ```bash
   # Run the version check script
   ./scripts/check-go-version.sh

   # If Go version is too old, upgrade to Go 1.23+
   # Download from: https://golang.org/dl/
   ```

2. **ARM64 Assembly Errors on x86_64**
   ```bash
   # This happens when building for wrong architecture
   # Use the automated build script (recommended):
   ./scripts/build-docker.sh

   # Or manually specify architecture for Hetzner x86_64 servers:
   docker build --build-arg TARGETARCH=amd64 --build-arg TARGETOS=linux -t mcp-logging-server .

   # Check your server architecture:
   uname -m  # Should show x86_64 for Hetzner
   ./scripts/determine-architecture.sh  # For detailed info
   ```

3. **GCC Assembly Instruction Errors**
   ```bash
   # If you see errors like "no such instruction: stp x29,x30,[sp,"
   # This means you're trying to build ARM64 code on x86_64
   # Solution: Use correct TARGETARCH
   docker build --build-arg TARGETARCH=amd64 -t mcp-logging-server .
   ```

4. **Container Won't Start**
   - Check environment variables in Coolify
   - Verify volume mounts are created
   - Check container logs: `docker logs mcp-logging-server`

2. **SSL Certificate Issues**
   - Ensure domains are properly configured in Coolify
   - Check Traefik logs for certificate generation
   - Verify DNS records point to Coolify server

3. **API Key Authentication Failing**
   - Verify API keys are properly generated
   - Check `/app/config/api-keys.yaml` exists in container
   - Ensure `API_KEY_REQUIRED=true` is set

4. **Rate Limiting Too Aggressive**
   - Adjust `RATE_LIMIT_REQUESTS_PER_MINUTE` and `RATE_LIMIT_BURST`
   - Check application logs for rate limit violations
   - Consider per-API-key rate limits

### Debug Commands

```bash
# Check container status
docker ps | grep mcp-logging

# View application logs
docker logs -f mcp-logging-server

# Check configuration
docker exec -it mcp-logging-server cat /app/config/config.yaml

# Test internal health check
docker exec -it mcp-logging-server wget -qO- http://localhost:9080/health

# Check API key configuration
docker exec -it mcp-logging-server ./apikey list
```

### Performance Tuning

For high-volume deployments:

```bash
# Increase buffer sizes
BUFFER_SIZE=50000
BUFFER_MAX_BATCH_SIZE=500

# Adjust rate limits
RATE_LIMIT_REQUESTS_PER_MINUTE=5000
RATE_LIMIT_BURST=1000

# Optimize retention
RETENTION_DEBUG_DAYS=3
RETENTION_INFO_DAYS=14
```

## Security Best Practices

1. **API Key Management**
   - Use strong, unique API keys for each service
   - Implement key rotation policies
   - Monitor API key usage via audit logs

2. **Network Security**
   - Use Coolify's built-in firewall rules
   - Restrict access to management endpoints
   - Monitor for unusual traffic patterns

3. **Data Protection**
   - Enable field masking for sensitive data
   - Regular backup of configuration and data
   - Implement audit log monitoring

4. **Updates and Maintenance**
   - Regular container image updates
   - Monitor security advisories
   - Test updates in staging environment

## Support and Maintenance

### Regular Maintenance Tasks

1. **Weekly**: Check health status and logs
2. **Monthly**: Review API key usage and rotate if needed
3. **Quarterly**: Update container images and review security settings
4. **Annually**: Review and update retention policies

### Scaling Considerations

For high-volume deployments, consider:
- Multiple container instances behind load balancer
- External database (PostgreSQL) instead of SQLite
- Dedicated monitoring and alerting setup
- Log archival to external storage

## Contact and Support

For deployment issues or questions:
- Check application logs first
- Review this documentation
- Consult Coolify documentation for platform-specific issues