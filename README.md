# RedTeamCoin

[![License](https://img.shields.io/github/license/xyplex3/RedTeamCoin?label=License&style=flat&color=blue&logo=github)](https://github.com/xyplex3/RedTeamCoin/blob/main/LICENSE)
[![Pre-Commit](https://github.com/xyplex3/RedTeamCoin/actions/workflows/pre-commit.yaml/badge.svg)](https://github.com/xyplex3/RedTeamCoin/actions/workflows/pre-commit.yaml)
[![Security Scanning](https://github.com/xyplex3/RedTeamCoin/actions/workflows/security.yaml/badge.svg)](https://github.com/xyplex3/RedTeamCoin/actions/workflows/security.yaml)
[![Release](https://github.com/xyplex3/RedTeamCoin/actions/workflows/goreleaser.yaml/badge.svg)](https://github.com/xyplex3/RedTeamCoin/actions/workflows/goreleaser.yaml)
[![Test & Build Verification](https://github.com/xyplex3/RedTeamCoin/actions/workflows/test-and-build.yaml/badge.svg)](https://github.com/xyplex3/RedTeamCoin/actions/workflows/test-and-build.yaml)
[![Windows Miner Test](https://github.com/xyplex3/RedTeamCoin/actions/workflows/windows-miner-test.yaml/badge.svg)](https://github.com/xyplex3/RedTeamCoin/actions/workflows/windows-miner-test.yaml

RedTeamCoin is a blockchain-based cryptocurrency mining pool implementation designed for authorized security testing and red team operations. Built in Go with Java client support, it simulates real-world cryptomining attacks to help organizations assess their detection capabilities and quantify potential damage from threat actor mining operations.

This tool enables security teams to safely and legally demonstrate cryptomining attack scenarios on corporate systems, generate comprehensive impact reports, and validate security controls—all within a controlled environment using an isolated, non-public blockchain.

**Created by:**
- Peter Greko (Xyplex2), Luciano Krigun, and Jayson Grace (@l50).

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
- [Contributing](#contributing)
- [Troubleshooting](#troubleshooting)
- [Documentation](#documentation)
- [License](#license)
- [Acknowledgments](#acknowledgments)

## Overview

**Why use RedTeamCoin?**

- Demonstrate the real-world impact of cryptojacking attacks
- Test and validate security monitoring and detection systems
- Generate quantifiable damage assessments for executive reporting
- Safely simulate mining operations without connecting to public blockchains

**System Components:**

- **Mining Pool Server** - Manages blockchain and coordinates work distribution
- **Client Miners** - CPU/GPU mining clients (Go and Java implementations)
- **Web Dashboard** - Real-time monitoring, control, and statistics
- **Analysis Tools** - Generate comprehensive damage assessment reports

## Features

- **Blockchain Implementation**: Custom proof-of-work blockchain with
  configurable difficulty
- **Mining Pool Server**: Manages multiple miners and work distribution via gRPC
- **Client Miners**: Automated mining clients (Go and Java) with IP address and hostname logging
- **Java Standalone Miner**: GUI-enabled desktop miner with visual interface
- **Java gRPC Client**: Headless gRPC client for servers and automation
- **Server-Side Miner Control**: Pause, resume, throttle CPU usage, and delete
  miners remotely
- **CPU Throttling**: Limit miner CPU usage from 0-100% to manage resources
- **GPU Mining Support**: CUDA and OpenCL support for GPU-accelerated mining
- **Hybrid Mining**: Simultaneous CPU and GPU mining for maximum performance
- **Web Dashboard**: Real-time monitoring of miners, statistics, and blockchain
  with control buttons
- **REST API Authentication**: Secure token-based authentication for API endpoints
- **Dual IP Tracking**: Records both client-reported and server-detected IP addresses
- **CPU & GPU Statistics**: Comprehensive resource usage tracking and reporting
- **Protocol Buffers**: Efficient client-server communication using protobuf/gRPC
- **Auto-Termination**: Miners automatically shut down and self-delete when
  removed from the server

## Quick Start

Get RedTeamCoin running in under 5 minutes:

### Prerequisites

- **Go** 1.21 or later
- **Protocol Buffer Compiler** (protoc)

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

- Install Go from https://golang.org/dl/
- Install protoc from https://github.com/protocolbuffers/protobuf/releases

### Installation

```bash
# Clone the repository
git clone https://github.com/xyplex3/RedTeamCoin.git
cd RedTeamCoin

# Install dependencies
make install-tools
make deps

# Build server and client
make build
```

### Verification

Verify the build completed successfully:

```bash
ls -lh bin/
# Expected output: server and client binaries (1-5 MB each)

./bin/server --help
# Expected output: Usage information for the server
```

### Running Your First Mining Pool

**1. Start the server:**

```bash
make run-server
```

You should see output like:

```text
RedTeamCoin Mining Pool Server
Authentication Token: abc123def456...
Dashboard URL: http://localhost:8080?token=abc123def456...
gRPC server listening on port 50051
```

**2. Start a miner** (in a new terminal):

```bash
make run-client
```

You should see mining activity:

```text
Connected to mining pool at localhost:50051
Registered as miner: miner-hostname-1234567890
Mining block #1... Hash rate: 2.5 MH/s
Block found! Nonce: 98765, Hash: 000000ab1cd...
```

**3. View the dashboard:**

Open the URL from step 1 in your browser, or navigate to:

```text
http://localhost:8080?token=YOUR_AUTH_TOKEN_HERE
```

You'll see real-time statistics including active miners, hash rates, and mined blocks.

**Note:** The server listens on all network interfaces - replace `localhost` with your server's IP for remote access.

## How It Works

The mining pool operates using a proof-of-work consensus mechanism:

1. Server initializes blockchain with genesis block
2. Miners connect via gRPC and register with IP/hostname
3. Server distributes work assignments (block templates)
4. Miners compute SHA-256 hashes to find valid nonces
5. Miners submit solutions when difficulty target is met
6. Server validates and adds blocks to the blockchain
7. Dashboard displays real-time statistics and miner activity

Each block requires finding a hash with a specific number of leading zeros
(configurable difficulty). Miners receive 50 RTC per block mined.

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
    |                              | [Validate proof-of-work
    |                              |  and add block to chain]
    |<--SubmissionResponse---------|
    |--Heartbeat(stats)----------->|
    |<--HeartbeatResponse----------|
```

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

**Note:** With HTTPS, browsers will show a security warning for self-signed
certificates. Click "Advanced" → "Proceed to localhost".
**NOTE:** The web dashboard and gRPC server will run on all interfaces of the server and can be used in place of localhost.

### Running a Miner

#### Go Client (Native Binary)

**Basic usage:**

```bash
./bin/client
```

**Run multiple miners:**
Open additional terminals and run `./bin/client` in each.

#### Java Miners

RedTeamCoin provides two Java miner implementations:

##### Java gRPC Client (Headless)

Production-ready gRPC client for servers and automation.

**Prerequisites:**

- Java 11 or later
- Maven (for building only)

**Build using Makefile:**

```bash
make build-java-client
```

**Or build manually:**

```bash
cd java-client
mvn clean package
```

**Run:**

```bash
# Connect to localhost
java -jar bin/redteamcoin-miner-client.jar

# Connect to remote server
java -jar bin/redteamcoin-miner-client.jar -server 192.168.1.100:50051
```

**Features:**

- gRPC protocol (matches Go client)
- Headless operation (no GUI)
- Server control (pause/resume/throttle)
- Auto-reconnection
- Self-deletion on server command

##### Java Standalone Miner (GUI)

Desktop miner with graphical interface.

**Prerequisites:**

- Java 21 or later
- Maven (for building only)

**Build using Makefile:**

```bash
make build-java-standalone
```

**Or build manually:**

```bash
cd java-standalone
mvn clean package
```

**Run:**

```bash
# GUI mode (default)
java -jar bin/redteamcoin-miner-standalone.jar

# CLI mode
java -jar bin/redteamcoin-miner-standalone.jar --pool localhost:50051
```

**Features:**

- Desktop GUI interface
- Visual statistics display
- Simple JSON/Socket protocol
- Embedded in applications

##### Build All Java Miners

```bash
make build-java-all
```

**Benefits of Java Miners:**

- Single JAR file, easy to distribute
- Cross-platform (Windows, Linux, macOS)
- No compilation needed on target systems
- Just requires Java Runtime (JRE)

See [java-client/README.md](java-client/README.md) for complete Java gRPC client documentation.

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

## GPU Mining

RedTeamCoin supports GPU-accelerated mining for NVIDIA (CUDA) and AMD/Intel
(OpenCL) GPUs. GPU mining is 100-400x faster and significantly more energy
efficient than CPU mining.

### GPU Installation

**NVIDIA CUDA:**

```bash
sudo apt install cuda-toolkit
make build-cuda
```

**AMD/Intel OpenCL:**

```bash
sudo apt install ocl-icd-opencl-dev
make build-opencl
```

**Auto-detect:**

```bash
make build-gpu  # Automatically detects and builds for available GPU
```

### GPU Usage

```bash
# Auto-detect GPU (default)
./bin/client

# Force CPU only
GPU_MINING=false ./bin/client

# Hybrid mode (CPU + GPU)
HYBRID_MINING=true ./bin/client

# GPU with remote server
./bin/client -server mining-pool.example.com:50051
```

### Performance Comparison

| Configuration    | Hash Rate | Speedup  | Efficiency (MH/W) |
| ---------------- | --------- | -------- | ----------------- |
| CPU (1 core)     | ~2 MH/s   | Baseline | ~0.02             |
| CPU (8 cores)    | ~16 MH/s  | 8x       | ~0.02             |
| GPU (RTX 3080)   | ~500 MH/s | 250x     | ~2.0              |
| GPU (RTX 3090)   | ~600 MH/s | 300x     | ~2.5              |
| GPU (AMD MI250)  | ~800 MH/s | 400x     | ~3.0              |
| Hybrid (CPU+GPU) | ~620 MH/s | 310x     | ~2.0              |

**Note:** GPU mining is 100-150x more energy efficient than CPU mining.

### Testing GPU Functionality

```bash
# Test GPU detection
./bin/client 2>&1 | grep -i "gpu\|cuda\|opencl"

# Force CPU mining (verify fallback works)
GPU_MINING=false ./bin/client

# Force GPU mining (if available)
GPU_MINING=true ./bin/client

# Test hybrid CPU+GPU mode
HYBRID_MINING=true ./bin/client
```

See [docs/GPU_MINING.md](docs/GPU_MINING.md) for complete GPU mining guide.

## Configuration

### Environment Variables

**Server Configuration:**

| Variable         | Description                 | Required | Default            |
| ---------------- | --------------------------- | -------- | ------------------ |
| `RTC_USE_TLS`    | Enable HTTPS/TLS            | No       | `false`            |
| `RTC_CERT_FILE`  | TLS certificate path        | No       | `certs/server.crt` |
| `RTC_KEY_FILE`   | TLS private key path        | No       | `certs/server.key` |
| `RTC_AUTH_TOKEN` | Custom authentication token | No       | Auto-generated     |

**Client Configuration:**

| Variable        | Description                        | Required | Default           |
| --------------- | ---------------------------------- | -------- | ----------------- |
| `POOL_SERVER`   | Remote server address              | No       | `localhost:50051` |
| `GPU_MINING`    | Enable/disable GPU mining          | No       | Auto-detect       |
| `HYBRID_MINING` | Enable CPU+GPU simultaneous mining | No       | `false`           |

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
    difficulty    = 6      // Mining difficulty
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

### Performance

Expected mining times vary by CPU performance and luck:

- **Difficulty 4**: ~1-10 seconds/block (single CPU)
- **Difficulty 5**: ~10-60 seconds/block
- **Difficulty 6**: ~1-10 minutes/block

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

See [docs/TLS_SETUP.md](docs/TLS_SETUP.md) for detailed TLS configuration.

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
const token = "YOUR_TOKEN_HERE";
const headers = { Authorization: `Bearer ${token}` };

fetch("http://localhost:8080/api/stats", { headers })
  .then((r) => r.json())
  .then((data) => console.log(data));
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

### Testing

RedTeamCoin includes comprehensive unit tests for both server and client components.

**Run all tests:**

```bash
make test                 # Run all tests (server + client)
```

**Run specific test suites:**

```bash
# Server tests
cd server && go test -v

# Client tests
cd client && go test -v -short

# With coverage
cd server && go test -cover
cd client && go test -cover
```

**Test Coverage:**

- **Server**: 76 tests, 66.3% coverage

  - `blockchain_test.go`: 15 tests - Blockchain validation, hash calculation, concurrent access
  - `pool_test.go`: 26 tests - Miner management, work distribution, statistics
  - `grpc_server_test.go`: 17 tests - gRPC endpoints, miner control, heartbeats
  - `api_test.go`: 18 tests - REST API, authentication, miner operations

- **Client**: 41 tests, 16.2% coverage
  - `main_test.go`: 24 tests - Mining logic, hash calculation, state management
  - `gpu_test.go`: 18 tests - GPU detection, device management, statistics

**Total**: 117 unit tests covering core functionality

**What's tested:**

- Blockchain validation and integrity
- Block mining and proof-of-work
- Mining pool work distribution
- Miner registration and management
- Server-side miner control (pause/resume/delete/throttle)
- gRPC communication protocols
- REST API endpoints and authentication
- GPU device detection and initialization
- Hash rate calculation and statistics
- Concurrent operations and thread safety
- Error handling and edge cases

**CI/CD Testing:**

All tests run automatically on:

- Pull requests
- Commits to main branch
- Release builds

See [.github/workflows/build-verification.yaml](.github/workflows/build-verification.yaml) for CI configuration.

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

See [docs/WINDOWS_BUILD.md](docs/WINDOWS_BUILD.md) for complete Windows build instructions including:

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

| Parameter          | Default   | Notes                 |
| ------------------ | --------- | --------------------- |
| CPU Power          | 150W      | Full load             |
| GPU Power          | 250W      | Full load             |
| Electricity        | $0.12/kWh | Adjust for region     |
| Lifespan Reduction | 20-40%    | From sustained mining |

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

| Problem              | Solution                                                        |
| -------------------- | --------------------------------------------------------------- |
| "cannot find -lcuda" | `export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH` |
| "nvcc not found"     | `sudo apt install cuda-toolkit`                                 |
| "No OpenCL device"   | `sudo apt install ocl-icd-opencl-dev`                           |
| GPU slow             | `export GPU_MINING=true`                                        |
| CGo build error      | `sudo apt install build-essential`                              |

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

### User Guides

- [docs/GPU_MINING.md](docs/GPU_MINING.md) - GPU mining with CUDA and OpenCL
- [docs/WINDOWS_BUILD.md](docs/WINDOWS_BUILD.md) - Windows build guide (native
  and cross-compilation)
- [docs/TLS_SETUP.md](docs/TLS_SETUP.md) - HTTPS/TLS configuration
- [java-client/README.md](java-client/README.md) - Java gRPC client (headless miner)
- [tools/README.md](tools/README.md) - Analysis tools and damage reports

### Technical References

- [docs/WORKFLOWS.md](docs/WORKFLOWS.md) - CI/CD workflows and development processes

### Examples

- [docs/examples/](docs/examples/) - Sample reports and usage examples

## Contributing

We welcome contributions! Here's how to get started:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/amazing-feature`
3. Make your changes and add tests
4. Run the test suite: `make test`
5. Commit your changes: `git commit -m 'Add amazing feature'`
6. Push to the branch: `git push origin feature/amazing-feature`
7. Open a Pull Request

### Development Setup

```bash
# Clone your fork
git clone https://github.com/YOUR_USERNAME/RedTeamCoin.git
cd RedTeamCoin

# Install dependencies
make install-tools
make deps

# Run tests
make test

# Build and test locally
make build
make run-server  # Terminal 1
make run-client  # Terminal 2
```

### Code Guidelines

- Follow Go standard formatting (`gofmt`)
- Add unit tests for new functionality
- Update documentation for user-facing changes
- Ensure all CI checks pass before submitting PR

## Security Note

This is a **demonstration/educational project** for authorized security testing
and red team operations. It is intended for:

- Authorized penetration testing engagements
- Security control validation
- Detection capability assessment
- Educational and training purposes

This tool is not intended for production use as a real cryptocurrency and
lacks many features required for a production system (cryptographic signatures,
wallets, transaction validation, network consensus, etc.).

**Use only with explicit authorization on systems you own or have permission to test.**

## License

This project is licensed under the GNU General Public License v3.0 - see the
[LICENSE](LICENSE) file for details.

## Acknowledgments

**Created by:**

- Peter Greko
- Luciano Krigun
- Jayson Grace ([@l50](https://github.com/l50))
