#!/bin/bash

# Generate self-signed TLS certificates for RedTeamCoin server

CERT_DIR="certs"
CERT_FILE="${CERT_DIR}/server.crt"
KEY_FILE="${CERT_DIR}/server.key"

echo "=== RedTeamCoin TLS Certificate Generator ==="
echo

# Create certs directory if it doesn't exist
mkdir -p ${CERT_DIR}

# Check if certificates already exist
if [ -f "${CERT_FILE}" ] && [ -f "${KEY_FILE}" ]; then
	echo "Certificates already exist in ${CERT_DIR}/"
	echo "Delete them first if you want to regenerate."
	exit 0
fi

# Generate self-signed certificate
echo "Generating self-signed TLS certificate..."
if openssl req -x509 -newkey rsa:4096 -nodes \
	-keyout ${KEY_FILE} \
	-out ${CERT_FILE} \
	-days 365 \
	-subj "/C=US/ST=State/L=City/O=RedTeamCoin/OU=Mining/CN=localhost" \
	-addext "subjectAltName=DNS:localhost,IP:127.0.0.1"; then
	echo
	echo "✓ Certificates generated successfully!"
	echo
	echo "Certificate: ${CERT_FILE}"
	echo "Private Key: ${KEY_FILE}"
	echo
	echo "Note: This is a self-signed certificate for development/testing."
	echo "Browsers will show a security warning - this is expected."
	echo
	echo "To use HTTPS, start the server with:"
	echo "  export RTC_USE_TLS=true"
	echo "  make run-server"
	echo
else
	echo
	echo "✗ Failed to generate certificates"
	echo "Please ensure openssl is installed"
	exit 1
fi
