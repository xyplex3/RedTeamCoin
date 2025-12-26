# Verify Mining Action

Prevents the CPU mining bug (Issue #8) from coming back.

## Why this exists

We had a nasty bug where mining looked like it was working but wasn't:

- Hash rate stuck at 0 H/s
- Nonces incrementing but no blocks found
- Everything seemed fine until you actually checked the results

This action runs a quick mining test to make sure:

- Miner connects to the pool
- Mining actually starts
- Hash rate isn't stuck at zero
- Blocks are being found
- Blockchain grows
- Block hashes are valid

## Usage

```yaml
steps:
  - name: Checkout code
    uses: actions/checkout@v6

  - name: Build binaries
    run: |
      make proto
      make build

  - name: Verify mining works
    uses: ./.github/actions/verify-mining
    with:
      server-bin: ./bin/server
      client-bin: ./bin/client
      server-port: '50051'
      api-port: '8080'
      mining-duration: '30'

  - name: Upload verification artifacts
    if: always()
    uses: actions/upload-artifact@v4
    with:
      name: mining-verification-results
      path: |
        mining-verification.json
        server-stdout.log
        out.log
      retention-days: 7
```

## Inputs

| Input | Description | Required | Default |
|-------|-------------|----------|---------|
| `server-bin` | Path to server binary | Yes | - |
| `client-bin` | Path to client binary | Yes | - |
| `server-port` | Server gRPC port | No | `50051` |
| `api-port` | API server port | No | `8080` |
| `mining-duration` | Mining duration in seconds | No | `30` |

## Outputs

| Output | Description |
|--------|-------------|
| `hash-rate` | Measured hash rate in H/s |
| `blocks-mined` | Number of blocks mined |
| `blockchain-height` | Final blockchain height |
| `verification-passed` | Whether verification passed (true/false) |

## Using the outputs

```yaml
- name: Verify mining
  id: mining
  uses: ./.github/actions/verify-mining
  with:
    server-bin: ./bin/server
    client-bin: ./bin/client

- name: Check results
  run: |
    echo "Hash rate: ${{ steps.mining.outputs.hash-rate }} H/s"
    echo "Blocks mined: ${{ steps.mining.outputs.blocks-mined }}"
    echo "Passed: ${{ steps.mining.outputs.verification-passed }}"
```

## Output files

Creates `mining-verification.json` with the results:

```json
{
  "success": true,
  "hash_rate": 3825000,
  "blocks_mined": 4,
  "blockchain_height": 5,
  "mining_duration_seconds": 30,
  "timestamp": "2025-12-25T21:30:00Z",
  "platform": "windows",
  "errors": []
}
```

## Platform Support

Works on Windows (PowerShell), Linux, and macOS (bash).

## When it fails

Fails if any of these happen:

- Hash rate stuck at 0 H/s (the bug we're trying to catch)
- No blocks mined during the test
- Miner can't register with pool
- Mining doesn't start
- Blockchain height doesn't increase
- Block hashes don't meet difficulty

## Testing locally

Windows:

```powershell
.\.github\actions\verify-mining\verify.ps1 `
  -ServerBin "bin\server.exe" `
  -ClientBin "bin\client.exe" `
  -Duration 30
```

Linux/macOS:

```bash
./.github/actions/verify-mining/verify.sh \
  "bin/server" \
  "bin/client" \
  50051 \
  8080 \
  30
```
