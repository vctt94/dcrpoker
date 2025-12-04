#!/bin/bash
# Generate HTTP TLS certificates for the poker server
# This script generates CA, server, and client certificates for mTLS

set -e

# Default values
SERVER_HOST="${SERVER_HOST:-192.168.0.109}"
CLIENT_HOST="${CLIENT_HOST:-192.168.0.109}"
OUTPUT_DIR="${OUTPUT_DIR:-.}"

echo "Generating HTTP TLS certificates..."
echo "Server host: $SERVER_HOST"
echo "Client host: $CLIENT_HOST"
echo "Output directory: $OUTPUT_DIR"
echo ""

# Step 1: Generate CA certificate
echo "Step 1: Generating CA certificate..."
gencerts -o "Poker Server CA" -f "$OUTPUT_DIR/http-ca.cert" "$OUTPUT_DIR/http-ca.key"

# Step 2: Generate server certificate
echo "Step 2: Generating server certificate..."
gencerts -C "$OUTPUT_DIR/http-ca.cert" -K "$OUTPUT_DIR/http-ca.key" -H "$SERVER_HOST" -L -f "$OUTPUT_DIR/http.cert" "$OUTPUT_DIR/http.key"

# Step 3: Generate client certificate
echo "Step 3: Generating client certificate..."
gencerts -C "$OUTPUT_DIR/http-ca.cert" -K "$OUTPUT_DIR/http-ca.key" -H "$CLIENT_HOST" -L -f "$OUTPUT_DIR/client.cert" "$OUTPUT_DIR/client.key"

# Step 4: Convert client certificate to PKCS#12 format for browser import
echo "Step 4: Converting client certificate to PKCS#12 format..."
if command -v openssl &> /dev/null; then
    read -sp "Enter password for client.p12 (remember this for browser import): " PASSWORD
    echo ""
    openssl pkcs12 -export -out "$OUTPUT_DIR/client.p12" \
        -inkey "$OUTPUT_DIR/client.key" \
        -in "$OUTPUT_DIR/client.cert" \
        -certfile "$OUTPUT_DIR/http-ca.cert" \
        -passout pass:"$PASSWORD"
    echo "Client certificate exported to $OUTPUT_DIR/client.p12"
else
    echo "Warning: openssl not found. Skipping PKCS#12 conversion."
    echo "You can convert manually with:"
    echo "  openssl pkcs12 -export -out client.p12 -inkey client.key -in client.cert -certfile http-ca.cert"
fi

echo ""
echo "Certificate generation complete!"
echo ""
echo "Next steps:"
echo "1. Copy server certificates to your server datadir:"
echo "   cp $OUTPUT_DIR/http.cert $OUTPUT_DIR/http.key $OUTPUT_DIR/http-ca.cert /path/to/datadir/"
echo ""
echo "2. The server will automatically use these certificates if they're in the datadir"
echo "   (or configure httpcertpath, httpkeypath, httpcacertpath in your config file)"
echo ""
echo "3. Import client.p12 into your browser (see docs/http-tls-setup.md for instructions)"
echo ""
echo "4. Access the metrics endpoint at: https://$SERVER_HOST:9091/metrics"

