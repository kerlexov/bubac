#!/bin/bash

# Generate self-signed certificates for development
# DO NOT use these certificates in production!

set -e

CERT_DIR="./certs"
DOMAIN="${1:-localhost}"

echo "Generating self-signed certificates for domain: $DOMAIN"

# Create certs directory
mkdir -p "$CERT_DIR"

# Generate private key
openssl genrsa -out "$CERT_DIR/server.key" 2048

# Generate certificate signing request
openssl req -new -key "$CERT_DIR/server.key" -out "$CERT_DIR/server.csr" -subj "/C=US/ST=CA/L=San Francisco/O=MCP Logging/OU=Development/CN=$DOMAIN"

# Generate self-signed certificate
openssl x509 -req -days 365 -in "$CERT_DIR/server.csr" -signkey "$CERT_DIR/server.key" -out "$CERT_DIR/server.crt" -extensions v3_req -extfile <(
cat <<EOF
[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = $DOMAIN
DNS.2 = localhost
DNS.3 = *.localhost
IP.1 = 127.0.0.1
IP.2 = ::1
EOF
)

# Clean up CSR
rm "$CERT_DIR/server.csr"

# Set appropriate permissions
chmod 600 "$CERT_DIR/server.key"
chmod 644 "$CERT_DIR/server.crt"

echo "Certificates generated successfully:"
echo "  Private key: $CERT_DIR/server.key"
echo "  Certificate: $CERT_DIR/server.crt"
echo ""
echo "To use these certificates, set the following environment variables:"
echo "  export TLS_ENABLED=true"
echo "  export TLS_CERT_PATH=$PWD/$CERT_DIR/server.crt"
echo "  export TLS_KEY_PATH=$PWD/$CERT_DIR/server.key"
echo ""
echo "⚠️  WARNING: These are self-signed certificates for development only!"
echo "   Do not use in production. Use proper certificates from a CA."