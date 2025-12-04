# RedTeamCoin

A blockchain-based cryptocurrency mining pool implementation in Go, featuring a server-side blockchain
and pool management system, client miners, and a web-based administration dashboard.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Quick Start](#quick-start)
- [How It Works](#how-it-works)
- [Usage](#usage)
  - [Running the Server](#running-the-server)
  - [Running a Miner](#running-a-miner)
  - [Connecting to Remote Servers](#connecting-to-remote-servers)
- [GPU Mining](#gpu-mining)
- [Configuration](#configuration)
- [API Reference](#api-reference)
- [Architecture](#architecture)
- [Development](#development)
- [Troubleshooting](#troubleshooting)
- [Documentation](#documentation)

## Overview

RedTeamCoin is a demonstration blockchain cryptocurrency mining pool system that simulates a
non-Ethereum based cryptocurrency. The system consists of three main components:

1. **Blockchain Server** - Manages the blockchain and mining pool
2. **Client Miner** - Performs proof-of-work mining
3. **Web Dashboard** - Provides administration and monitoring

## Features

- **Blockchain Implementation**: Custom proof-of-work blockchain with configurable difficulty
- **Mining Pool Server**: Manages multiple miners and work distribution via gRPC
- **Client Miner**: Automated mining client with IP address and hostname logging
- **Server-Side Miner Control**: Pause, resume, throttle CPU usage, and delete miners remotely
- **CPU Throttling**: Limit miner CPU usage from 0-100% to manage resources
- **GPU Mining Support**: CUDA and OpenCL support for GPU-accelerated mining
- **Hybrid Mining**: Simultaneous CPU and GPU mining for maximum performance
- **Web Dashboard**: Real-time monitoring of miners, statistics, and blockchain with control buttons
- **REST API Authentication**: Secure token-based authentication for API endpoints
- **Dual IP Tracking**: Records both client-reported and server-detected IP addresses
- **CPU & GPU Statistics**: Comprehensive resource usage tracking and reporting
- **Protocol Buffers**: Efficient client-server communication using protobuf/gRPC
- **Auto-Termination**: Miners automatically shut down and self-delete when removed from the server

## Quick Start

### Prerequisites

- Go 1.21 or later
- protoc (Protocol Buffer Compiler)
- protoc-gen-go and protoc-gen-go-grpc

**Ubuntu/Debian:**

```bash
sudo apt update
sudo apt install -y golang-go protobuf-compiler
```

**macOS:**

```bash
brew install go protobuf
```

**Windows:**

- Install Go from: https://golang.org/dl/
- Install protoc from: https://github.com/protocolbuffers/protobuf/releases

### Installation

```bash
# Clone the repository
git clone https://github.com/xyplex3/RedTeamCoin.git
cd RedTeamCoin

# Install tools and dependencies
make install-tools
make deps

# Build the project
make build
```

### Running

**Start the server:**

```bash
make run-server
```

The server starts on:

- gRPC: port **50051**
- Web dashboard: **http://localhost:8080**

Copy the authentication token URL from the console.

**Start a miner:**

```bash
make run-client
```

**Access the dashboard:**
Navigate to the URL displayed in the server console or:

```text
http://localhost:8080?token=YOUR_AUTH_TOKEN_HERE
```

## How It Works

1. **Server starts** and initializes the blockchain with a genesis block
2. **Miners connect** via gRPC, providing their IP address and hostname
3. **Server assigns work** to miners (blocks to be mined)
4. **Miners compute hashes** trying to find a valid nonce that meets the difficulty requirement
5. **Miners submit solutions** back to the server
6. **Server validates** and adds accepted blocks to the blockchain
7. **Web dashboard** displays real-time statistics and blockchain data

## Usage

### Running the Server

**HTTP (Default):**

```bash
./bin/server
```

- gRPC server: port **50051**
- Web dashboard: **http://localhost:8080**

**HTTPS/TLS (Recommended):**

```bash
# Generate certificates (one time)
./generate_certs.sh

# Start with TLS
RTC_USE_TLS=true ./bin/server
```

- gRPC server: port **50051**
- Web dashboard: **https://localhost:8443**
- HTTP redirect: **http://localhost:8080**

**Note:** With HTTPS, browsers will show a security warning for self-signed certificates.
Click "Advanced" → "Proceed to localhost".

### Running a Miner

**Basic usage:**

```bash
./bin/client
```

**Run multiple miners:**
Open additional terminals and run `./bin/client` in each.

### Connecting to Remote Servers

By default, clients connect to `localhost:50051`. To connect to a remote server:

**Using command-line flag (recommended):**

```bash
./bin/client -server 192.168.1.100:50051
./bin/client -s mining-pool.example.com:50051
```

**Using environment variable:**

```bash
export POOL_SERVER=192.168.1.100:50051
./bin/client
```

**Priority:** Command-line flag > Environment variable > Default (localhost:50051)

See [REMOTE_SERVER_SETUP.md](REMOTE_SERVER_SETUP.md) for detailed remote configuration.

## GPU Mining

RedTeamCoin supports GPU-accelerated mining with automatic detection and fallback.

### Build Options

```bash
make build              # CPU-only (default)
make build-gpu          # Auto-detect and build with available GPU
make build-cuda         # NVIDIA CUDA support
make build-opencl       # AMD/Intel OpenCL support
```

### Installing GPU Dependencies

**NVIDIA GPUs:**

```bash
sudo apt install cuda-toolkit
make build-cuda
```

**AMD/Intel GPUs:**

```bash
sudo apt install ocl-icd-opencl-dev
make build-opencl
```

### Running with GPU

```bash
# Auto-detect (default)
./bin/client

# Force CPU only
GPU_MINING=false ./bin/client

# Hybrid mode (CPU + GPU)
HYBRID_MINING=true ./bin/client

# GPU with remote server
GPU_MINING=true ./bin/client -server mining-pool.example.com:50051
```

### Performance Comparison

| Configuration | Hash Rate | Speedup |
|--------------|-----------|---------|
| CPU (1 core) | ~2 MH/s | Baseline |
| CPU (8 cores) | ~16 MH/s | 8x |
| GPU (RTX 3080) | ~500 MH/s | 250x |
| Hybrid (CPU+GPU) | ~620 MH/s | 310x |

See [GPU_MINING.md](GPU_MINING.md) for complete GPU mining guide.

## Configuration

### Environment Variables

**Server:**

- `RTC_USE_TLS` - Enable HTTPS (default: `false`)
- `RTC_CERT_FILE` - TLS certificate path (default: `certs/server.crt`)
- `RTC_KEY_FILE` - TLS private key path (default: `certs/server.key`)
- `RTC_AUTH_TOKEN` - Custom authentication token (auto-generated if not set)

**Client:**

- `POOL_SERVER` - Remote server address (default: `localhost:50051`)
- `GPU_MINING` - Enable/disable GPU mining (default: auto-detect)
- `HYBRID_MINING` - Enable CPU+GPU simultaneous mining (default: `false`)

### Authentication

All API endpoints (except the dashboard homepage) require Bearer token authentication.

**Auto-generated token:**
The server generates a secure token on startup and displays it in the console.

**Custom token:**

```bash
export RTC_AUTH_TOKEN="your-secret-token"
./bin/server
```

### Code Configuration

**Server (`server/main.go`):**

```go
const (
    grpcPort      = 50051
    apiPort       = 8443   // HTTPS (8080 for HTTP)
    httpPort      = 8080
    difficulty    = 4      // Mining difficulty
)
```

**Client (`client/main.go`):**

```go
const (
    serverAddress = "localhost:50051"
    heartbeatInterval = 30 * time.Second
)
```

**Pool (`server/pool.go`):**

```go
blockReward = 50  // RTC reward per block
```

### HTTPS Configuration

```bash
# Generate certificates
./generate_certs.sh

# Start with HTTPS
export RTC_USE_TLS=true
export RTC_AUTH_TOKEN="my-secure-token"
./bin/server

# Custom certificates
export RTC_CERT_FILE="/path/to/cert.pem"
export RTC_KEY_FILE="/path/to/key.pem"
./bin/server
```

See [TLS_SETUP.md](TLS_SETUP.md) for detailed TLS configuration.

## API Reference

### REST API Endpoints

**Public endpoints:**

- `GET /` - Web dashboard (HTML)
- `GET /blocks` - View all blocks page (HTML)

**Authenticated endpoints:**

- `GET /api/stats` - Pool statistics (JSON)
- `GET /api/miners` - List of all miners (JSON)
- `GET /api/blockchain` - Complete blockchain (JSON)
- `GET /api/blocks/{index}` - Specific block details (JSON)
- `GET /api/validate` - Validate blockchain integrity (JSON)
- `GET /api/cpu` - CPU and GPU usage statistics (JSON)
- `POST /api/miner/pause` - Pause mining for a specific miner
- `POST /api/miner/resume` - Resume mining for a specific miner
- `POST /api/miner/delete` - Delete a miner (auto-terminates and self-deletes)
- `POST /api/miner/throttle` - Set CPU throttle percentage (0-100%)

### Authentication Examples

**cURL:**

```bash
# Get stats
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/stats

# Get miners
curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8080/api/miners

# Pause a miner
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"miner_id":"miner-hostname-1234567890"}' \
  http://localhost:8080/api/miner/pause

# Set CPU throttle to 50%
curl -X POST -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"miner_id":"miner-hostname-1234567890","throttle_percent":50}' \
  http://localhost:8080/api/miner/throttle

# HTTPS (with self-signed cert)
curl -k -H "Authorization: Bearer YOUR_TOKEN" https://localhost:8443/api/stats
```

**JavaScript:**

```javascript
const token = 'YOUR_TOKEN_HERE';
const headers = { 'Authorization': `Bearer ${token}` };

fetch('http://localhost:8080/api/stats', { headers })
  .then(r => r.json())
  .then(data => console.log(data));
```

**Python:**

```python
import requests

headers = {'Authorization': 'Bearer YOUR_TOKEN'}
response = requests.get('http://localhost:8080/api/stats', headers=headers)
print(response.json())
```

### gRPC Services

```protobuf
service MiningPool {
  rpc RegisterMiner(MinerInfo) returns (RegistrationResponse);
  rpc GetWork(WorkRequest) returns (WorkResponse);
  rpc SubmitWork(WorkSubmission) returns (SubmissionResponse);
  rpc Heartbeat(MinerStatus) returns (HeartbeatResponse);
  rpc StopMining(MinerInfo) returns (StopResponse);
}
```

## Architecture

### Technology Stack

- **Language**: Go 1.21+
- **RPC Framework**: gRPC with Protocol Buffers
- **Web Server**: Go net/http standard library
- **Dependencies**: google.golang.org/protobuf, google.golang.org/grpc

### Project Structure

```text
RedTeamCoin/
├── server/          # Blockchain and mining pool server
│   ├── main.go      # Server entry point
│   ├── blockchain.go # Blockchain implementation
│   ├── pool.go      # Mining pool management
│   ├── grpc_server.go # gRPC service implementation
│   └── api.go       # HTTP API and web dashboard
├── client/          # Mining client
│   └── main.go      # Client miner implementation
├── proto/           # Protocol buffer definitions
│   └── mining.proto # Mining service definitions
└── Makefile         # Build automation
```

### Component Details

#### 1. Blockchain (`server/blockchain.go`)

SHA-256 proof-of-work blockchain with configurable difficulty.

**Key Types:**

- `Block` - Block with index, timestamp, data, hash, previous hash, nonce, miner ID
- `Blockchain` - Thread-safe blockchain with validation

**Key Functions:**

- `NewBlockchain(difficulty)` - Creates blockchain with genesis block
- `AddBlock(block)` - Validates and adds a block
- `ValidateChain()` - Validates blockchain integrity
- `calculateHash(block)` - Computes SHA-256 hash

**Difficulty:** Number of leading zeros in block hash (default: 4)

#### 2. Mining Pool (`server/pool.go`)

Manages miners, distributes work, processes submissions.

**Key Types:**

- `MinerRecord` - Miner info (ID, IP, hostname, stats)
- `PendingWork` - Work assignments
- `MiningPool` - Coordinates distribution

**Features:**

- Work queue
- Stale block detection
- Real-time statistics
- 50 RTC block reward

#### 3. gRPC Server (`server/grpc_server.go`)

Protocol Buffer service for miner communication.

**Flow:**

1. Miner registers with IP/hostname
2. Requests work
3. Receives block template
4. Computes hashes
5. Submits solution
6. Server validates and rewards

#### 4. Web API (`server/api.go`)

HTTP REST API and dashboard with token authentication.

**Features:**

- Real-time pool statistics
- Active miner monitoring
- Recent block history
- Auto-refresh (5 seconds)

#### 5. Client Miner (`client/main.go`)

Automated mining client.

**Features:**

- Auto IP detection
- Hostname logging
- Hash rate calculation
- Heartbeat (30s intervals)
- Graceful shutdown

**Hash:** `SHA256(index + timestamp + data + previousHash + nonce)`

### Data Flow

```text
Client Miner                    Server
    |                              |
    |--RegisterMiner(IP,Hostname)->|
    |<--RegistrationResponse-------|
    |--GetWork()------------------>|
    |<--WorkResponse(Block)--------|
    | [Mining: compute hashes]     |
    |--SubmitWork(nonce,hash)----->|
    |                              | [Validate & Add]
    |<--SubmissionResponse---------|
    |--Heartbeat(stats)----------->|
    |<--HeartbeatResponse----------|
```

### Concurrency

- **Blockchain**: Thread-safe (`sync.RWMutex`)
- **Mining Pool**: Thread-safe (`sync.RWMutex`)
- **Work Generator**: Goroutine (30s intervals)
- **Client Heartbeat**: Goroutine (30s intervals)
- **Multiple Miners**: Fully supported

### Performance

- **Difficulty 4**: ~1-10 seconds/block (single CPU)
- **Difficulty 5**: ~10-60 seconds/block
- **Difficulty 6**: ~1-10 minutes/block

Times vary by CPU performance and luck.

## Development

### Build Commands

```bash
make proto                # Generate protobuf code
make build                # Build server and client (CPU only)
make build-gpu            # Build with GPU support (auto-detect)
make build-cuda           # Build with NVIDIA CUDA
make build-opencl         # Build with AMD/Intel OpenCL
make build-windows        # Cross-compile for Windows
make build-all-platforms  # Cross-compile for all platforms
make clean                # Remove build artifacts
make deps                 # Download dependencies
make init                 # Full initialization
```

### Cross-Compilation

**Windows (CPU-only):**

```bash
make build-windows
```

Creates `bin/client.exe`

**Windows with OpenCL (GPU support):**

```bash
# Requires MinGW-w64 and Windows OpenCL SDK
make build-windows-opencl
```

Creates `bin/client-windows-opencl.exe`

See [WINDOWS_BUILD.md](WINDOWS_BUILD.md) for complete Windows build instructions including:

- Native Windows builds with GPU support
- Cross-compilation setup from Linux
- Authentication token configuration
- Troubleshooting guide

**Multiple Platforms:**

```bash
make build-all-platforms
```

Creates:

- `client-linux-amd64` - Linux 64-bit
- `client-linux-arm64` - Linux ARM64
- `client-windows-amd64.exe` - Windows 64-bit
- `client-darwin-amd64` - macOS Intel
- `client-darwin-arm64` - macOS Apple Silicon

**Note:** Cross-compiled binaries are CPU-only (CGO disabled) unless using `build-windows-opencl`.

### Analysis Tools

Generate damage assessment reports for mining impact analysis:

```bash
make build-tools
./bin/generate_report -log pool_log.json
```

**Report includes:**

- Resource consumption analysis
- Performance impact assessment
- Infrastructure damage
- Security implications
- Financial impact summary
- System-by-system analysis
- Remediation recommendations

**Cost Assumptions:**

| Parameter | Default | Notes |
|-----------|---------|-------|
| CPU Power | 150W | Full load |
| GPU Power | 250W | Full load |
| Electricity | $0.12/kWh | Adjust for region |
| Lifespan Reduction | 20-40% | From sustained mining |

**Converting Reports:**

```bash
# PDF
pandoc Report_Miner_Activity_from_<date>_to_<date>.md -o report.pdf

# HTML
pandoc Report_Miner_Activity_from_<date>_to_<date>.md -o report.html

# DOCX
pandoc Report_Miner_Activity_from_<date>_to_<date>.md -o report.docx
```

**Use Cases:**

- Post-incident analysis
- Executive briefings
- Financial justification
- Compliance documentation
- Insurance claims
- Legal evidence

See [tools/README.md](tools/README.md) for complete documentation.

## Troubleshooting

### Remote Connection Issues

**Testing connectivity:**

```bash
# Test connection
ping <server_ip>
nc -zv <server_ip> 50051

# Check server (on server side)
lsof -i :50051
ss -an | grep 50051
```

**Common issues:**

- Verify server address and port
- Ensure server is running
- Check firewall rules (client and server)
- Confirm network connectivity
- Verify port 50051 is listening

### GPU Mining Issues

| Problem | Solution |
|---------|----------|
| "cannot find -lcuda" | `export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH` |
| "nvcc not found" | `sudo apt install cuda-toolkit` |
| "No OpenCL device" | `sudo apt install ocl-icd-opencl-dev` |
| GPU slow | `export GPU_MINING=true` |
| CGo build error | `sudo apt install build-essential` |

### General Issues

**Build failures:**

- Ensure Go 1.21+ installed
- Run `make deps` to install dependencies
- Run `make install-tools` for protoc tools

**Connection refused:**

- Check server is running
- Verify port not in use: `lsof -i :50051`
- Check firewall settings

**Authentication errors:**

- Verify token matches server output
- Check `Authorization: Bearer TOKEN` header format
- Ensure token included in URL or headers

## Documentation

- [.github/README.md](.github/README.md) - GitHub Actions workflows and release process
- [WINDOWS_BUILD.md](WINDOWS_BUILD.md) - Windows build guide (native and cross-compilation)
- [GPU_MINING.md](GPU_MINING.md) - GPU mining with CUDA and OpenCL
- [REMOTE_SERVER_SETUP.md](REMOTE_SERVER_SETUP.md) - Remote server configuration
- [TLS_SETUP.md](TLS_SETUP.md) - HTTPS/TLS configuration
- [tools/README.md](tools/README.md) - Analysis tools and reports

## Security Note

This is a **demonstration/educational project** for understanding blockchain and cryptocurrency concepts.
It is not intended for production use and lacks many features required for a real cryptocurrency
(cryptographic signatures, wallets, transaction validation, etc.).

## License

This project is licensed under the GNU General Public License v3.0 (GPL-3.0).

### GNU General Public License v3.0

Copyright (C) 2024 RedTeamCoin Contributors

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

### Educational Purpose

This project is intended for educational and demonstration purposes to understand blockchain
and cryptocurrency concepts. It is not intended for production use and lacks many features
required for a real cryptocurrency (cryptographic signatures, wallets, transaction validation, etc.).
