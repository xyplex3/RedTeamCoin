# CPU Statistics API Documentation

## Overview

The `/api/cpu` endpoint provides detailed CPU usage statistics for all miners connected to the RedTeamCoin mining pool. This includes both aggregate statistics and per-miner breakdowns.

## Endpoint

```
GET /api/cpu
```

**Authentication:** Required (Bearer token)

## Response Format

The endpoint returns a JSON object containing:
- **Aggregate statistics** for all miners combined
- **Per-miner statistics** array with detailed information for each miner

### Response Structure

```json
{
  "total_miners": 5,
  "active_miners": 3,
  "total_cpu_usage_percent": 215.5,
  "average_cpu_usage_percent": 71.83,
  "total_hashes": 15432890,
  "total_mining_hours": 12.5,
  "total_mining_seconds": 45000.0,
  "total_hash_rate": 342500,
  "miner_stats": [
    {
      "miner_id": "miner-hostname-1234567890",
      "ip_address": "192.168.1.100",
      "ip_address_actual": "192.168.1.100",
      "hostname": "mining-rig-01",
      "cpu_usage_percent": 85.3,
      "total_hashes": 5432100,
      "mining_time_hours": 4.2,
      "mining_time_seconds": 15120.0,
      "hash_rate": 125000,
      "active": true,
      "registered_at": "2025-01-15T10:30:00Z"
    },
    ...
  ]
}
```

## Field Descriptions

### Aggregate Fields

| Field | Type | Description |
|-------|------|-------------|
| `total_miners` | integer | Total number of miners ever registered |
| `active_miners` | integer | Number of currently active miners |
| `total_cpu_usage_percent` | float | Sum of CPU usage from all active miners |
| `average_cpu_usage_percent` | float | Average CPU usage across active miners |
| `total_hashes` | integer | Total hashes computed by all miners |
| `total_mining_hours` | float | Total mining time in hours (all miners) |
| `total_mining_seconds` | float | Total mining time in seconds (all miners) |
| `total_hash_rate` | integer | Combined hash rate of all active miners (H/s) |

### Per-Miner Fields

| Field | Type | Description |
|-------|------|-------------|
| `miner_id` | string | Unique identifier for the miner |
| `ip_address` | string | **Client-reported** IP address |
| `ip_address_actual` | string | **Server-detected** actual IP address from connection |
| `hostname` | string | Client-reported hostname of the mining machine |
| `cpu_usage_percent` | float | Current CPU usage percentage (0-100) |
| `total_hashes` | integer | Total hashes computed by this miner |
| `mining_time_hours` | float | Total time mining in hours |
| `mining_time_seconds` | float | Total time mining in seconds |
| `hash_rate` | integer | Current hash rate in hashes/second |
| `active` | boolean | Whether miner is currently active |
| `registered_at` | string | ISO 8601 timestamp of registration |

## Usage Examples

### curl (HTTP)

```bash
# Get CPU statistics
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8080/api/cpu | jq .

# Get total CPU usage
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8080/api/cpu | jq '.total_cpu_usage_percent'

# Get average CPU usage
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8080/api/cpu | jq '.average_cpu_usage_percent'

# List active miners with CPU > 50%
curl -H "Authorization: Bearer YOUR_TOKEN" \
     http://localhost:8080/api/cpu | \
     jq '.miner_stats[] | select(.active and .cpu_usage_percent > 50)'
```

### curl (HTTPS)

```bash
curl -k -H "Authorization: Bearer YOUR_TOKEN" \
     https://localhost:8443/api/cpu | jq .
```

### Python

```python
import requests
import json

TOKEN = "your-token-here"
API_URL = "http://localhost:8080"

headers = {
    'Authorization': f'Bearer {TOKEN}',
    'Content-Type': 'application/json'
}

# Get CPU statistics
response = requests.get(f'{API_URL}/api/cpu', headers=headers)
data = response.json()

# Print summary
print(f"Total Miners: {data['total_miners']}")
print(f"Active Miners: {data['active_miners']}")
print(f"Total CPU Usage: {data['total_cpu_usage_percent']:.2f}%")
print(f"Average CPU Usage: {data['average_cpu_usage_percent']:.2f}%")
print(f"Total Hashes: {data['total_hashes']:,}")
print(f"Total Mining Time: {data['total_mining_hours']:.2f} hours")

# Print per-miner details
print("\n=== Per-Miner Statistics ===")
for miner in data['miner_stats']:
    print(f"\nMiner: {miner['miner_id']}")
    print(f"  Hostname: {miner['hostname']} ({miner['ip_address']})")
    print(f"  CPU Usage: {miner['cpu_usage_percent']:.2f}%")
    print(f"  Hash Rate: {miner['hash_rate']:,} H/s")
    print(f"  Total Hashes: {miner['total_hashes']:,}")
    print(f"  Mining Time: {miner['mining_time_hours']:.2f} hours")
    print(f"  Active: {miner['active']}")
```

### JavaScript/Node.js

```javascript
const axios = require('axios');

const TOKEN = 'your-token-here';
const API_URL = 'http://localhost:8080';

const headers = {
    'Authorization': `Bearer ${TOKEN}`,
    'Content-Type': 'application/json'
};

async function getCPUStats() {
    try {
        const response = await axios.get(`${API_URL}/api/cpu`, { headers });
        const data = response.data;

        console.log(`Total Miners: ${data.total_miners}`);
        console.log(`Active Miners: ${data.active_miners}`);
        console.log(`Average CPU: ${data.average_cpu_usage_percent.toFixed(2)}%`);
        console.log(`Total Mining Time: ${data.total_mining_hours.toFixed(2)} hours`);

        // Filter and display high CPU miners
        const highCPU = data.miner_stats.filter(m =>
            m.active && m.cpu_usage_percent > 70
        );

        console.log(`\nHigh CPU Miners (>70%):`);
        highCPU.forEach(miner => {
            console.log(`  ${miner.hostname}: ${miner.cpu_usage_percent.toFixed(2)}%`);
        });

    } catch (error) {
        console.error('Error:', error.message);
    }
}

getCPUStats();
```

## Monitoring Use Cases

### 1. Track Total Pool CPU Consumption

Monitor the aggregate CPU usage across all miners to understand total resource consumption:

```bash
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/cpu | \
     jq '{total_cpu: .total_cpu_usage_percent, avg_cpu: .average_cpu_usage_percent}'
```

### 2. Identify High CPU Miners

Find miners using excessive CPU resources:

```bash
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/cpu | \
     jq '.miner_stats[] | select(.cpu_usage_percent > 90) | {id: .miner_id, cpu: .cpu_usage_percent}'
```

### 3. Calculate Total Mining Hours

Get total time all miners have spent mining:

```bash
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/cpu | \
     jq '.total_mining_hours'
```

### 4. Per-Miner Efficiency Analysis

Analyze hash rate vs CPU usage for each miner:

```bash
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/cpu | \
     jq '.miner_stats[] | {
       miner: .miner_id,
       efficiency: (.hash_rate / .cpu_usage_percent)
     }'
```

### 5. Monitor Mining Time Trends

Track how long each miner has been active:

```bash
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/cpu | \
     jq '.miner_stats | sort_by(.mining_time_hours) | reverse | .[0:5] | .[] | {
       hostname: .hostname,
       hours: .mining_time_hours
     }'
```

## Notes

### IP Address Tracking

The system records **both** IP addresses for each miner:

1. **`ip_address`** (Client-Reported)
   - The IP address the client detects for itself
   - Sent by the client during registration
   - May differ from actual IP if client is behind NAT/proxy
   - Useful for understanding client's network perspective

2. **`ip_address_actual`** (Server-Detected)
   - The actual source IP address as seen by the server
   - Extracted from the gRPC connection context
   - More trustworthy and cannot be spoofed by the client
   - Shows the real IP the server is communicating with

**Use Cases:**
- **Detect NAT/Proxy**: Compare both IPs to identify clients behind NAT
- **Security**: Use `ip_address_actual` for IP-based access control
- **Debugging**: Client-reported IP helps troubleshoot network issues
- **Audit**: Track both for comprehensive connection logging

**Example Scenarios:**

```json
{
  "ip_address": "192.168.1.100",
  "ip_address_actual": "203.0.113.45"
}
```
*Client is behind NAT - internal IP differs from public IP*

```json
{
  "ip_address": "127.0.0.1",
  "ip_address_actual": "127.0.0.1"
}
```
*Local testing - both IPs match*

### CPU Usage Calculation

The client miner estimates CPU usage based on hash rate activity. This is a simplified estimation:

```
CPU Usage ≈ (Hash Rate / 1,000,000) * 100%
```

Capped at 100%. For more accurate CPU measurements, OS-specific system calls would be required.

### Active vs Inactive Miners

A miner is considered **active** if:
- `Active` flag is `true`
- Last heartbeat was received within the last 2 minutes

Inactive miners still appear in the statistics but don't contribute to aggregate totals for active metrics.

### Time Calculations

- **`total_mining_hours`**: Cumulative time since miner started (in hours)
- **`total_mining_seconds`**: Cumulative time since miner started (in seconds)
- Time continues accumulating while the miner is running
- Time persists across reconnections (tracked by miner ID)

## Integration Examples

### Prometheus Metrics

Convert CPU stats to Prometheus format:

```python
def to_prometheus_metrics(data):
    metrics = []

    # Pool-level metrics
    metrics.append(f'rtc_pool_total_miners {data["total_miners"]}')
    metrics.append(f'rtc_pool_active_miners {data["active_miners"]}')
    metrics.append(f'rtc_pool_avg_cpu {data["average_cpu_usage_percent"]}')
    metrics.append(f'rtc_pool_total_hashes {data["total_hashes"]}')

    # Per-miner metrics
    for miner in data["miner_stats"]:
        labels = f'{{miner_id="{miner["miner_id"]}",hostname="{miner["hostname"]}"}}'
        metrics.append(f'rtc_miner_cpu_usage{labels} {miner["cpu_usage_percent"]}')
        metrics.append(f'rtc_miner_hash_rate{labels} {miner["hash_rate"]}')
        metrics.append(f'rtc_miner_total_hashes{labels} {miner["total_hashes"]}')

    return '\n'.join(metrics)
```

### Grafana Dashboard

Use the API to feed a Grafana dashboard showing:
- Real-time CPU usage per miner
- Total pool mining time
- Hash rate efficiency trends
- Historical hash counts

## Error Responses

### 401 Unauthorized

```json
{
  "error": "Unauthorized - Invalid or missing authentication token"
}
```

**Solution**: Provide valid Bearer token in `Authorization` header.

### 500 Internal Server Error

Contact server administrator if this occurs.

## See Also

- [Main README](README.md) - General API documentation
- [TLS Setup Guide](TLS_SETUP.md) - HTTPS configuration
- [Architecture Documentation](ARCHITECTURE.md) - System design details
