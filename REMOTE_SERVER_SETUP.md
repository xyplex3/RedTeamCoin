# Remote Server Configuration Guide

## Overview

The RedTeamCoin client now supports connecting to remote mining pool servers. By default, it connects to `localhost:50051`, but you can configure it to connect to any remote server.

## Usage Methods

### Method 1: Command-line Flag (Recommended)

Use the `-server` or `-s` flag to specify the remote server address:

```bash
./miner -server 192.168.1.100:50051
# or shorthand
./miner -s 192.168.1.100:50051
```

### Method 2: Environment Variable

Set the `POOL_SERVER` environment variable:

```bash
export POOL_SERVER=192.168.1.100:50051
./miner
```

### Method 3: Default (localhost)

If no server address is specified, the client defaults to `localhost:50051`:

```bash
./miner
```

## Priority Order

The client resolves the server address in this order:

1. **Command-line flag** (`-server` or `-s`) - Highest priority
2. **Environment variable** (`POOL_SERVER`)
3. **Default** (`localhost:50051`) - Lowest priority

## Examples

### Connect to a remote server by IP address

```bash
./miner -server 203.0.113.42:50051
```

### Connect to a remote server by hostname

```bash
./miner -server mining-pool.example.com:50051
```

### Use environment variable for persistent configuration

```bash
export POOL_SERVER=mining-pool.example.com:50051
./miner
./miner  # Will use the same server on subsequent runs
```

### Specify custom port

```bash
./miner -server 192.168.1.100:9999
```

## Building

Build the client normally:

```bash
cd client
go build -o miner .
```

## Network Requirements

- **Outbound gRPC connectivity** to the mining pool server on the specified port (default 50051)
- **Port must be open** on the mining pool server for inbound connections
- **Firewall rules** must allow traffic between client and server

## Troubleshooting

### Connection Failed

If you see `Failed to connect to pool`, check:

1. **Server address is correct**: Verify the IP/hostname and port
2. **Server is running**: Ensure the mining pool server is active
3. **Network connectivity**: Test with `ping` or `nc` (netcat)
4. **Firewall**: Check firewall rules on both client and server
5. **Port is correct**: Default is 50051, confirm the server is listening on this port

### Example diagnostic commands

```bash
# Test connectivity
ping 192.168.1.100
nc -zv 192.168.1.100 50051

# View active connection
lsof -i :50051  # On server side
ss -an | grep 50051  # Check listening ports
```

## GPU and Hybrid Mining with Remote Server

Remote server configuration works seamlessly with GPU and hybrid mining:

```bash
# GPU mining on remote server
GPU_MINING=true ./miner -server mining-pool.example.com:50051

# Hybrid CPU+GPU mining on remote server
HYBRID_MINING=true GPU_MINING=true ./miner -s mining-pool.example.com:50051
```

## Performance Considerations

- **Latency**: Lower latency to the mining pool improves response times
- **Bandwidth**: Typically minimal bandwidth usage for mining communications
- **Network stability**: Unstable connections may cause stale work rejections
