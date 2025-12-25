#!/usr/bin/env bash
# Mining Verification Script for Linux/macOS
# Prevents regression of CPU mining bug (hash rate stuck at 0 H/s)

set -euo pipefail

# Parse arguments
SERVER_BIN="${1:?Server binary path required}"
CLIENT_BIN="${2:?Client binary path required}"
SERVER_PORT="${3:-50051}"
API_PORT="${4:-8080}"
DURATION="${5:-30}"

echo "=================================================="
echo "Mining Verification Test (CPU Mining Bug Regression Check)"
echo "=================================================="
echo "Server Binary: $SERVER_BIN"
echo "Client Binary: $CLIENT_BIN"
echo "Server Port: $SERVER_PORT"
echo "API Port: $API_PORT"
echo "Mining Duration: ${DURATION}s"
echo ""

# Initialize result tracking
VERIFICATION_PASSED=true
HASH_RATE=0
BLOCKS_MINED=0
BLOCKCHAIN_HEIGHT=0
ERRORS=()

# Cleanup function - invoked by trap on EXIT
# shellcheck disable=SC2329
cleanup() {
	echo ""
	echo "Cleaning up..."
	if [ -n "${SERVER_PID:-}" ]; then
		kill "$SERVER_PID" 2>/dev/null || true
		wait "$SERVER_PID" 2>/dev/null || true
	fi
}
trap cleanup EXIT

# Step 1: Start server
echo "Step 1: Starting server..."
"$SERVER_BIN" >server-stdout.log 2>&1 &
SERVER_PID=$!
echo "✓ Server started (PID: $SERVER_PID)"

# Wait for server to start
sleep 5

# Verify server is running
if ! ps -p "$SERVER_PID" >/dev/null 2>&1; then
	echo "❌ Server failed to start"
	cat server-stdout.log
	exit 1
fi
echo ""

# Step 2: Run miner
echo "Step 2: Starting miner for ${DURATION} seconds..."
timeout "$DURATION" "$CLIENT_BIN" -server "localhost:$SERVER_PORT" >out.log 2>&1 || true
echo "✓ Mining test completed"
echo ""

# Step 3: Analyze miner output
echo "Step 3: Analyzing miner output..."

# Check 3.1: Verify miner registered
if grep -q "Successfully registered with pool" out.log; then
	echo "✓ Miner registered successfully"
else
	VERIFICATION_PASSED=false
	ERRORS+=("Miner failed to register with pool")
	echo "❌ Miner registration failed"
fi

# Check 3.2: Verify mining started
if grep -q "Starting mining" out.log; then
	echo "✓ Mining operations started"
else
	VERIFICATION_PASSED=false
	ERRORS+=("Mining operations never started")
	echo "❌ Mining never started"
fi

# Check 3.3: Verify hash rate is not stuck at 0 (CRITICAL BUG)
ZERO_HASH_COUNT=$(grep -c "Hash rate: 0 H/s" out.log || echo "0")
TOTAL_HASH_COUNT=$(grep -c "Hash rate:" out.log || echo "0")

if [ "$TOTAL_HASH_COUNT" -gt 0 ]; then
	if [ "$ZERO_HASH_COUNT" -eq "$TOTAL_HASH_COUNT" ]; then
		# ALL hash rate readings are 0 - this is the bug!
		VERIFICATION_PASSED=false
		ERRORS+=("REGRESSION: Hash rate stuck at 0 H/s (CPU mining bug detected)")
		HASH_RATE=0
		echo "❌ REGRESSION DETECTED: Hash rate stuck at 0 H/s (CPU mining bug)"
	else
		# Extract highest non-zero hash rate
		HASH_RATE=$(grep "Hash rate:" out.log | grep -v "0 H/s" |
			sed 's/.*Hash rate: \([0-9]*\) H\/s.*/\1/' |
			sort -n | tail -1)
		echo "✓ Hash rate working: ${HASH_RATE} H/s (max observed)"
	fi
fi

# Check 3.4: Verify blocks were mined
BLOCKS_MINED=$(grep -c "BLOCK MINED!" out.log || echo "0")

if [ "$BLOCKS_MINED" -eq 0 ]; then
	VERIFICATION_PASSED=false
	ERRORS+=("No blocks were mined during test period")
	echo "❌ No blocks mined"
else
	echo "✓ Blocks mined: $BLOCKS_MINED"
fi
echo ""

# Step 4: Query API for blockchain stats
echo "Step 4: Querying API for blockchain stats..."

# Extract API token from server log
if API_TOKEN=$(grep "Token:" server-stdout.log | head -1 | awk '{print $2}'); then
	echo "✓ API token extracted"

	# Query blockchain with token
	if BLOCKCHAIN_JSON=$(curl -s -H "Authorization: Bearer $API_TOKEN" \
		"http://localhost:$API_PORT/api/blockchain" 2>/dev/null); then

		# Count blocks (including genesis)
		BLOCKCHAIN_HEIGHT=$(echo "$BLOCKCHAIN_JSON" | jq '. | length' 2>/dev/null || echo "0")
		echo "✓ Blockchain height: $BLOCKCHAIN_HEIGHT blocks (including genesis)"

		# Verify blockchain grew
		if [ "$BLOCKCHAIN_HEIGHT" -le 1 ] && [ "$BLOCKS_MINED" -gt 0 ]; then
			VERIFICATION_PASSED=false
			ERRORS+=("Blockchain did not grow despite blocks being mined")
			echo "❌ Blockchain inconsistency detected"
		fi

		# Verify latest block hash meets difficulty
		if [ "$BLOCKCHAIN_HEIGHT" -gt 1 ]; then
			LATEST_HASH=$(echo "$BLOCKCHAIN_JSON" | jq -r '.[-1].Hash' 2>/dev/null || echo "")
			if echo "$LATEST_HASH" | grep -q "^000000"; then
				echo "✓ Latest block hash meets difficulty: $LATEST_HASH"
			else
				VERIFICATION_PASSED=false
				ERRORS+=("Latest block hash does not meet difficulty requirement")
				echo "❌ Block hash invalid: $LATEST_HASH"
			fi
		fi
	else
		echo "⚠ Could not query blockchain API (non-critical)"
	fi
else
	echo "⚠ Could not extract API token, skipping blockchain verification"
fi
echo ""

# Step 5: Write results
echo "=================================================="
if [ "$VERIFICATION_PASSED" = true ]; then
	echo "✅ MINING VERIFICATION PASSED"
else
	echo "❌ MINING VERIFICATION FAILED"
	echo ""
	echo "Errors:"
	for error in "${ERRORS[@]}"; do
		echo "  - $error"
	done
fi
echo "=================================================="
echo ""

# Create JSON result
cat >mining-verification.json <<EOF
{
  "success": $VERIFICATION_PASSED,
  "hash_rate": $HASH_RATE,
  "blocks_mined": $BLOCKS_MINED,
  "blockchain_height": $BLOCKCHAIN_HEIGHT,
  "mining_duration_seconds": $DURATION,
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "platform": "$(uname -s | tr '[:upper:]' '[:lower:]')",
  "errors": $(printf '%s\n' "${ERRORS[@]}" | jq -R . | jq -s .)
}
EOF

# Set GitHub Actions outputs
{
	echo "hash-rate=$HASH_RATE"
	echo "blocks-mined=$BLOCKS_MINED"
	echo "blockchain-height=$BLOCKCHAIN_HEIGHT"
	echo "verification-passed=$VERIFICATION_PASSED"
} >>"$GITHUB_OUTPUT"

# Exit with appropriate code
if [ "$VERIFICATION_PASSED" != true ]; then
	exit 1
fi

exit 0
