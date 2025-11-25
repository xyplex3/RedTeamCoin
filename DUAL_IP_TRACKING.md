# Dual IP Address Tracking

## Overview

RedTeamCoin records **both** the client-reported IP address and the server-detected actual IP address for every miner. This provides comprehensive network visibility and enhanced security.

## Implementation

### What Gets Recorded

For each miner connection, the system tracks:

1. **Client-Reported IP** (`ip_address`)
   - What the client thinks its IP address is
   - Detected by the client using outbound UDP connection test
   - Sent to server during registration via protobuf

2. **Server-Detected IP** (`ip_address_actual`)
   - The actual source IP as seen by the server
   - Extracted from gRPC connection peer context
   - Cannot be spoofed by the client
   - Updated on every reconnection

### How It Works

**Client Side** (`client/main.go`):
```go
// Client detects its own IP
func getOutboundIP() string {
    conn, _ := net.Dial("udp", "8.8.8.8:80")
    defer conn.Close()
    localAddr := conn.LocalAddr().(*net.UDPAddr)
    return localAddr.IP.String()
}

// Sent during registration
resp, err := m.client.RegisterMiner(m.ctx, &pb.MinerInfo{
    MinerId:   m.id,
    IpAddress: m.ipAddress,  // Client's view
    Hostname:  m.hostname,
})
```

**Server Side** (`server/grpc_server.go`):
```go
// Server extracts actual IP from gRPC context
func getClientIP(ctx context.Context) string {
    p, _ := peer.FromContext(ctx)
    addr := p.Addr.String()
    host, _, _ := net.SplitHostPort(addr)
    return strings.Trim(host, "[]")
}

// Both IPs stored in MinerRecord
err := s.pool.RegisterMiner(
    req.MinerId,
    req.IpAddress,      // Client-reported
    req.Hostname,
    actualIP,           // Server-detected
)
```

## Use Cases

### 1. NAT/Proxy Detection

Identify miners behind NAT or proxies by comparing the two IP addresses:

```bash
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/cpu | \
     jq '.miner_stats[] | select(.ip_address != .ip_address_actual) | {
       miner: .miner_id,
       reported: .ip_address,
       actual: .ip_address_actual
     }'
```

**Output:**
```json
{
  "miner": "miner-workstation-123",
  "reported": "192.168.1.50",
  "actual": "203.0.113.45"
}
```

This indicates the miner is behind NAT (private IP internally, public IP externally).

### 2. Security and Access Control

Use the server-detected IP for trustworthy access control:

```python
import requests

def check_miner_ips(token):
    response = requests.get(
        'http://localhost:8080/api/cpu',
        headers={'Authorization': f'Bearer {token}'}
    )

    for miner in response.json()['miner_stats']:
        actual_ip = miner['ip_address_actual']

        # Whitelist check using actual IP (can't be spoofed)
        if not is_ip_whitelisted(actual_ip):
            print(f"Warning: Miner {miner['miner_id']} from "
                  f"non-whitelisted IP: {actual_ip}")
```

### 3. Network Troubleshooting

When clients report connectivity issues, compare both IPs to diagnose:

```bash
# Check if client IP matches what server sees
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/miners | \
     jq '.[] | {
       id: .ID,
       hostname: .Hostname,
       client_ip: .IPAddress,
       server_ip: .IPAddressActual,
       match: (.IPAddress == .IPAddressActual)
     }'
```

### 4. Audit Logging

Both IPs provide complete audit trail:

```json
{
  "timestamp": "2025-01-15T14:30:00Z",
  "event": "miner_registered",
  "miner_id": "miner-rig-001",
  "hostname": "mining-rig-001",
  "ip_reported": "10.0.1.100",
  "ip_actual": "203.0.113.45",
  "note": "Client behind NAT"
}
```

### 5. Geographic Analysis

Combine with IP geolocation to understand miner distribution:

```python
import requests
from ipaddress import ip_address

def analyze_miner_locations(token):
    response = requests.get(
        'http://localhost:8080/api/cpu',
        headers={'Authorization': f'Bearer {token}'}
    )

    for miner in response.json()['miner_stats']:
        actual_ip = miner['ip_address_actual']

        # Use actual IP for geolocation (more accurate)
        location = geolocate_ip(actual_ip)

        print(f"Miner: {miner['hostname']}")
        print(f"  Location: {location['city']}, {location['country']}")
        print(f"  Actual IP: {actual_ip}")

        # Check if reported IP is private
        if ip_address(miner['ip_address']).is_private:
            print(f"  Behind NAT: Yes (internal IP: {miner['ip_address']})")
```

## API Responses

### `/api/miners` Response

```json
[
  {
    "ID": "miner-hostname-1234567890",
    "IPAddress": "192.168.1.100",
    "IPAddressActual": "203.0.113.45",
    "Hostname": "mining-rig-01",
    "RegisteredAt": "2025-01-15T10:30:00Z",
    "LastHeartbeat": "2025-01-15T14:30:00Z",
    "Active": true,
    "BlocksMined": 15,
    "HashRate": 125000
  }
]
```

### `/api/cpu` Response

```json
{
  "total_miners": 3,
  "active_miners": 2,
  "miner_stats": [
    {
      "miner_id": "miner-rig-001",
      "ip_address": "192.168.1.100",
      "ip_address_actual": "203.0.113.45",
      "hostname": "mining-rig-01",
      "cpu_usage_percent": 85.3,
      "active": true
    }
  ]
}
```

## Common Scenarios

### Local Testing
```
Client IP:  127.0.0.1
Actual IP:  127.0.0.1
Status:     Match - local connection
```

### Home Network (NAT)
```
Client IP:  192.168.1.50
Actual IP:  203.0.113.45
Status:     NAT detected - private to public translation
```

### Corporate Network (Proxy)
```
Client IP:  10.10.10.50
Actual IP:  198.51.100.1
Status:     Corporate proxy/NAT
```

### VPN Connection
```
Client IP:  10.8.0.5
Actual IP:  198.51.100.10
Status:     VPN tunnel detected
```

### Cloud/Data Center
```
Client IP:  172.31.45.100
Actual IP:  54.123.45.67
Status:     Cloud instance (AWS/GCP/Azure)
```

## Security Considerations

### Trust Model

- **DO NOT TRUST** `ip_address` (client-reported) for security decisions
- **ALWAYS USE** `ip_address_actual` (server-detected) for:
  - Access control lists (ACLs)
  - Rate limiting
  - Geofencing
  - Security logging

### Example: IP-Based Access Control

```go
// BAD - Don't do this
if miner.IPAddress == "192.168.1.100" {
    allowAccess()  // Client can lie about this!
}

// GOOD - Do this instead
if miner.IPAddressActual == "203.0.113.45" {
    allowAccess()  // Server-verified, trustworthy
}
```

### Detection of IP Spoofing Attempts

Monitor for suspicious patterns:

```python
def detect_spoofing_attempts(miners):
    for miner in miners:
        # Client claims public IP but server sees different public IP
        if (not is_private_ip(miner['ip_address']) and
            not is_private_ip(miner['ip_address_actual']) and
            miner['ip_address'] != miner['ip_address_actual']):

            print(f"⚠️  Possible spoofing attempt:")
            print(f"   Miner: {miner['miner_id']}")
            print(f"   Claims: {miner['ip_address']}")
            print(f"   Actually: {miner['ip_address_actual']}")
```

## Technical Implementation

### Data Structure

```go
type MinerRecord struct {
    ID                 string
    IPAddress          string        // Client-reported
    IPAddressActual    string        // Server-detected
    Hostname           string
    RegisteredAt       time.Time
    LastHeartbeat      time.Time
    Active             bool
    BlocksMined        int64
    HashRate           int64
    TotalMiningTime    time.Duration
    CPUUsagePercent    float64
    TotalHashes        int64
}
```

### Registration Flow

1. Client detects its own IP via outbound connection test
2. Client sends registration request with its IP and hostname
3. Server receives request via gRPC
4. Server extracts actual source IP from peer context
5. Server stores both IPs in MinerRecord
6. Both IPs included in all API responses

### IP Extraction Methods

**Client Side:**
- Uses `net.Dial("udp", "8.8.8.8:80")` to detect outbound IP
- Gets local address from UDP connection
- Works reliably for most network configurations

**Server Side:**
- Uses `peer.FromContext(ctx)` to get gRPC peer info
- Extracts IP from `Addr.String()` (format: "IP:port")
- Handles both IPv4 and IPv6 addresses
- Strips port number and brackets

## Monitoring and Alerts

### Dashboard Query Examples

**Find all miners behind NAT:**
```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/cpu | \
  jq '[.miner_stats[] | select(.ip_address != .ip_address_actual)] | length'
```

**List public vs private IP miners:**
```bash
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/cpu | \
  jq '.miner_stats[] | {
    miner: .miner_id,
    type: (if (.ip_address | startswith("192.168.") or startswith("10.") or startswith("172."))
           then "private" else "public" end),
    actual: .ip_address_actual
  }'
```

## See Also

- [CPU Statistics API](CPU_STATS_API.md) - Full API documentation
- [README](README.md) - General project documentation
- [Architecture](ARCHITECTURE.md) - System design details
