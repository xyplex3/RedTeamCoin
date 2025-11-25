#!/bin/bash

# Example script for testing RedTeamCoin API with authentication

# Set your authentication token here
# You can get this from the server console when it starts
TOKEN="${RTC_AUTH_TOKEN:-your-token-here}"

# API URL - change to https://localhost:8443 if using TLS
# Use -k flag with curl to accept self-signed certificates
USE_TLS="${RTC_USE_TLS:-false}"

if [ "$USE_TLS" = "true" ]; then
    API_URL="https://localhost:8443"
    CURL_OPTS="-k"  # Accept self-signed certificates
else
    API_URL="http://localhost:8080"
    CURL_OPTS=""
fi

echo "=== RedTeamCoin API Testing ==="
echo "API URL: $API_URL"
echo "TLS: $USE_TLS"
echo "Using token: ${TOKEN:0:20}..."
echo

# Test stats endpoint
echo "1. Getting pool statistics..."
curl $CURL_OPTS -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     ${API_URL}/api/stats | jq .
echo

# Test miners endpoint
echo "2. Getting miners list..."
curl $CURL_OPTS -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     ${API_URL}/api/miners | jq .
echo

# Test blockchain endpoint
echo "3. Getting blockchain (first 3 blocks)..."
curl $CURL_OPTS -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     ${API_URL}/api/blockchain | jq '.[0:3]'
echo

# Test validate endpoint
echo "4. Validating blockchain..."
curl $CURL_OPTS -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     ${API_URL}/api/validate | jq .
echo

# Test specific block
echo "5. Getting block 0 (genesis block)..."
curl $CURL_OPTS -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     ${API_URL}/api/blocks/0 | jq .
echo

# Test unauthorized access (should fail)
echo "6. Testing unauthorized access (should fail)..."
curl $CURL_OPTS -H "Authorization: Bearer invalid-token" \
     -H "Content-Type: application/json" \
     ${API_URL}/api/stats
echo
echo

echo "=== Testing complete ==="
