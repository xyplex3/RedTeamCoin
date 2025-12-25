#!/usr/bin/env pwsh
# Mining Verification Script for Windows
# Prevents regression of CPU mining bug (hash rate stuck at 0 H/s)

param(
    [Parameter(Mandatory=$true)]
    [string]$ServerBin,

    [Parameter(Mandatory=$true)]
    [string]$ClientBin,

    [int]$ServerPort = 50051,
    [int]$ApiPort = 8080,
    [int]$Duration = 30
)

$ErrorActionPreference = "Stop"

Write-Host "=================================================="
Write-Host "Mining Verification Test (CPU Mining Bug Regression Check)"
Write-Host "=================================================="
Write-Host "Server Binary: $ServerBin"
Write-Host "Client Binary: $ClientBin"
Write-Host "Server Port: $ServerPort"
Write-Host "API Port: $ApiPort"
Write-Host "Mining Duration: ${Duration}s"
Write-Host ""

# Initialize result object
$result = @{
    success = $true
    hash_rate = 0
    blocks_mined = 0
    blockchain_height = 0
    mining_duration_seconds = $Duration
    timestamp = (Get-Date).ToUniversalTime().ToString("o")
    platform = "windows"
    errors = @()
}

try {
    # Convert paths to absolute
    $ServerBinAbs = (Resolve-Path $ServerBin).Path
    $ClientBinAbs = (Resolve-Path $ClientBin).Path

    # Step 1: Start server in background
    Write-Host "Step 1: Starting server..."
    $serverJob = Start-Job -ScriptBlock {
        Set-Location $using:PWD
        & $using:ServerBinAbs 2>&1 | Tee-Object -FilePath "server-stdout.log"
    }

    # Wait for server to start
    Start-Sleep -Seconds 5

    # Verify server is running
    if (-not (Get-Job -Id $serverJob.Id -ErrorAction SilentlyContinue) -or
        (Get-Job -Id $serverJob.Id).State -ne 'Running') {
        throw "Server job is not running"
    }
    Write-Host "✓ Server started (Job ID: $($serverJob.Id))"
    Write-Host ""

    # Step 2: Run miner for specified duration
    Write-Host "Step 2: Starting miner for ${Duration} seconds..."
    $minerProc = Start-Process -FilePath $ClientBinAbs `
        -ArgumentList "--server", "127.0.0.1:$ServerPort" `
        -PassThru -NoNewWindow `
        -RedirectStandardOutput "out.log" `
        -RedirectStandardError "err.log"

    Write-Host "✓ Miner started (PID: $($minerProc.Id))"

    # Wait for mining duration
    $timeout = $minerProc | Wait-Process -Timeout $Duration -ErrorAction SilentlyContinue -PassThru

    # Stop miner if still running
    if (-not $minerProc.HasExited) {
        Write-Host "Stopping miner after ${Duration} seconds..."
        Stop-Process -Id $minerProc.Id -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 2
    }
    Write-Host ""

    # Step 3: Parse miner output
    Write-Host "Step 3: Analyzing miner output..."
    $minerOutput = Get-Content "out.log" -Raw -ErrorAction SilentlyContinue

    # Check 3.1: Verify miner registered
    if ($minerOutput -match "Successfully registered with pool") {
        Write-Host "✓ Miner registered successfully"
    } else {
        $result.success = $false
        $result.errors += "Miner failed to register with pool"
        Write-Host "❌ Miner registration failed"
    }

    # Check 3.2: Verify mining started
    if ($minerOutput -match "Starting mining") {
        Write-Host "✓ Mining operations started"
    } else {
        $result.success = $false
        $result.errors += "Mining operations never started"
        Write-Host "❌ Mining never started"
    }

    # Check 3.3: Verify hash rate is not stuck at 0 (CRITICAL BUG)
    $zeroHashRateCount = ([regex]::Matches($minerOutput, "Hash rate: 0 H/s")).Count
    $totalHashRateCount = ([regex]::Matches($minerOutput, "Hash rate: \d+ H/s")).Count

    if ($totalHashRateCount -gt 0) {
        if ($zeroHashRateCount -eq $totalHashRateCount) {
            # ALL hash rate readings are 0 - this is the bug!
            $result.success = $false
            $result.errors += "REGRESSION: Hash rate stuck at 0 H/s (CPU mining bug detected)"
            $result.hash_rate = 0
            Write-Host "❌ REGRESSION DETECTED: Hash rate stuck at 0 H/s (CPU mining bug)" -ForegroundColor Red
        } else {
            # Extract highest non-zero hash rate
            $hashRates = [regex]::Matches($minerOutput, "Hash rate: (\d+) H/s") |
                       ForEach-Object { [int]$_.Groups[1].Value } |
                       Where-Object { $_ -gt 0 }

            if ($hashRates.Count -gt 0) {
                $result.hash_rate = ($hashRates | Measure-Object -Maximum).Maximum
                Write-Host "✓ Hash rate working: $($result.hash_rate) H/s (max observed)" -ForegroundColor Green
            }
        }
    }

    # Check 3.4: Verify blocks were mined
    $blocksMinedCount = ([regex]::Matches($minerOutput, "BLOCK MINED!")).Count
    $result.blocks_mined = $blocksMinedCount

    if ($blocksMinedCount -eq 0) {
        $result.success = $false
        $result.errors += "No blocks were mined during test period"
        Write-Host "❌ No blocks mined" -ForegroundColor Red
    } else {
        Write-Host "✓ Blocks mined: $blocksMinedCount" -ForegroundColor Green
    }
    Write-Host ""

    # Step 4: Query API for final stats
    Write-Host "Step 4: Querying API for blockchain stats..."

    try {
        # Extract API token from server log
        $serverLog = Get-Content "server-stdout.log" -Raw
        if ($serverLog -match "Token: ([a-f0-9]{64})") {
            $apiToken = $matches[1]
            Write-Host "✓ API token extracted"

            # Query blockchain
            $headers = @{ "Authorization" = "Bearer $apiToken" }
            $blockchain = Invoke-RestMethod -Uri "http://127.0.0.1:$ApiPort/api/blockchain" -Headers $headers
            $result.blockchain_height = $blockchain.Count

            Write-Host "✓ Blockchain height: $($result.blockchain_height) blocks (including genesis)"

            # Verify blockchain grew (should be > 1 if blocks were mined)
            if ($result.blockchain_height -le 1 -and $blocksMinedCount -gt 0) {
                $result.success = $false
                $result.errors += "Blockchain did not grow despite blocks being mined"
                Write-Host "❌ Blockchain inconsistency detected"
            }

            # Verify latest block hash meets difficulty
            if ($blockchain.Count -gt 1) {
                $latestBlock = $blockchain[-1]
                if ($latestBlock.Hash -match "^000000") {
                    Write-Host "✓ Latest block hash meets difficulty: $($latestBlock.Hash)"
                } else {
                    $result.success = $false
                    $result.errors += "Latest block hash does not meet difficulty requirement"
                    Write-Host "❌ Block hash invalid: $($latestBlock.Hash)"
                }
            }
        } else {
            Write-Host "⚠ Could not extract API token, skipping blockchain verification"
        }
    } catch {
        Write-Host "⚠ API query failed (non-critical): $_"
        # Don't fail the test if API is unavailable
    }
    Write-Host ""

} catch {
    $result.success = $false
    $result.errors += "Unexpected error: $_"
    Write-Host "❌ Test failed with error: $_" -ForegroundColor Red
} finally {
    # Cleanup: Stop server
    if ($serverJob) {
        Stop-Job -Id $serverJob.Id -ErrorAction SilentlyContinue
        Remove-Job -Id $serverJob.Id -Force -ErrorAction SilentlyContinue
    }
}

# Step 5: Write results
Write-Host "=================================================="
if ($result.success) {
    Write-Host "✅ MINING VERIFICATION PASSED" -ForegroundColor Green
} else {
    Write-Host "❌ MINING VERIFICATION FAILED" -ForegroundColor Red
    Write-Host ""
    Write-Host "Errors:"
    $result.errors | ForEach-Object { Write-Host "  - $_" }
}
Write-Host "=================================================="
Write-Host ""

# Save JSON result for artifact upload
$result | ConvertTo-Json -Depth 10 | Out-File "mining-verification.json" -Encoding UTF8

# Set GitHub Actions outputs
$verificationStatus = if ($result.success) { "true" } else { "false" }
Add-Content -Path $env:GITHUB_OUTPUT -Value "hash-rate=$($result.hash_rate)"
Add-Content -Path $env:GITHUB_OUTPUT -Value "blocks-mined=$($result.blocks_mined)"
Add-Content -Path $env:GITHUB_OUTPUT -Value "blockchain-height=$($result.blockchain_height)"
Add-Content -Path $env:GITHUB_OUTPUT -Value "verification-passed=$verificationStatus"

# Exit with appropriate code
if (-not $result.success) {
    exit 1
}

exit 0
