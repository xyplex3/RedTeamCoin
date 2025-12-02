# RedTeamCoin

A blockchain-based cryptocurrency mining pool implementation in Go, featuring a server-side blockchain and pool management system, client miners, and a web-based administration dashboard.

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

## Architecture

RedTeamCoin is a demonstration blockchain cryptocurrency mining pool system that simulates a non-Ethereum based cryptocurrency. The system consists of three main components:

1. **Blockchain Server** - Manages the blockchain and mining pool
2. **Client Miner** - Performs proof-of-work mining
3. **Web Dashboard** - Provides administration and monitoring

### Technology Stack

- **Language**: Go 1.21+
- **RPC Framework**: gRPC with Protocol Buffers
- **Web Server**: Go net/http standard library
- **Dependencies**:
  - google.golang.org/protobuf
  - google.golang.org/grpc

```
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

The blockchain implementation uses SHA-256 proof-of-work with configurable difficulty.

**Key Types:**
- `Block` - Represents a single block with index, timestamp, data, hash, previous hash, nonce, and miner ID
- `Blockchain` - Thread-safe blockchain with block validation and chain integrity checking

**Key Functions:**
- `NewBlockchain(difficulty)` - Creates blockchain with genesis block
- `AddBlock(block)` - Validates and adds a new block
- `ValidateChain()` - Validates entire blockchain integrity
- `calculateHash(block)` - Computes SHA-256 hash of block

**Difficulty**: Number of leading zeros required in block hash (default: 4)

#### 2. Mining Pool (`server/pool.go`)

Manages multiple miners, distributes work, and processes block submissions.

**Key Types:**
- `MinerRecord` - Stores miner information (ID, IP, hostname, stats)
- `PendingWork` - Tracks work assigned to miners
- `MiningPool` - Coordinates miners and work distribution

**Key Functions:**
- `RegisterMiner()` - Registers new miner with IP and hostname logging
- `GetWork()` - Assigns mining work to miners
- `SubmitWork()` - Validates and processes mined blocks
- `UpdateHeartbeat()` - Tracks miner health
- `StopMiner()` - Handles miner disconnection

**Features:**
- Work queue for pending blocks
- Automatic stale block detection
- Real-time miner statistics
- 50 RTC block reward

#### 3. gRPC Server (`server/grpc_server.go`)

Implements Protocol Buffer service for miner communication.

**gRPC Services:**
```protobuf
service MiningPool {
  rpc RegisterMiner(MinerInfo) returns (RegistrationResponse);
  rpc GetWork(WorkRequest) returns (WorkResponse);
  rpc SubmitWork(WorkSubmission) returns (SubmissionResponse);
  rpc Heartbeat(MinerStatus) returns (HeartbeatResponse);
  rpc StopMining(MinerInfo) returns (StopResponse);
}
```

**Communication Flow:**
1. Miner connects and registers with IP/hostname
2. Miner requests work
3. Miner receives block template
4. Miner computes hash with different nonces
5. Miner submits solution
6. Server validates and rewards

#### 4. Web API (`server/api.go`)

Provides HTTP REST API and HTML dashboard for administration with token-based authentication.

**Authentication:**
- Bearer token authentication via `Authorization` header
- Token can be set via `RTC_AUTH_TOKEN` environment variable
- Auto-generated secure random token if not provided
- Homepage is public, all API endpoints require authentication

**API Endpoints:**
- `GET /` - Web dashboard (auto-refreshing) - Public
- `GET /api/stats` - Pool statistics - **Authenticated**
- `GET /api/miners` - List of all miners - **Authenticated**
- `GET /api/blockchain` - Complete blockchain - **Authenticated**
- `GET /api/blocks/{index}` - Specific block - **Authenticated**
- `GET /api/validate` - Validate blockchain - **Authenticated**

**Dashboard Features:**
- Real-time pool statistics
- Active miner monitoring with IP addresses and hostnames
- Recent block history
- Auto-refresh every 5 seconds
- Token-based API authentication via URL parameter

### 5. Client Miner (`client/main.go`)

Automated mining client that connects to the pool.

**Key Features:**
- Automatic IP address detection
- Hostname logging
- Configurable mining algorithm
- Hash rate calculation
- Periodic heartbeat (every 30 seconds)
- Graceful shutdown with Ctrl+C

**Mining Process:**
1. Detect and log IP address and hostname
2. Connect to server via gRPC
3. Register with pool
4. Request work
5. Compute hashes until valid nonce found
6. Submit solution
7. Repeat

**Hash Calculation:**
```
hash = SHA256(index + timestamp + data + previousHash + nonce)
```

## Data Flow

```
Client Miner                    Server
    |                              |
    |--RegisterMiner(IP,Hostname)->|
    |<--RegistrationResponse-------|
    |                              |
    |--GetWork()------------------>|
    |<--WorkResponse(Block)--------|
    |                              |
    | [Mining: compute hashes]     |
    |                              |
    |--SubmitWork(nonce,hash)----->|
    |                              | [Validate]
    |                              | [Add to blockchain]
    |<--SubmissionResponse---------|
    |    (accepted + reward)       |
    |                              |
    |--Heartbeat(stats)----------->|
    |<--HeartbeatResponse----------|
    |                              |
```

## Prerequisites

- Go 1.21 or later
- protoc (Protocol Buffer Compiler)
- protoc-gen-go and protoc-gen-go-grpc

### Installing Prerequisites

**Ubuntu/Debian:**
```bash
# Install Go (if not already installed)
sudo apt update
sudo apt install -y golang-go

# Install protoc (Protocol Buffer Compiler)
sudo apt install -y protobuf-compiler

# Verify installations
go version
protoc --version
```

**macOS:**
```bash
# Install using Homebrew
brew install go
brew install protobuf

# Verify installations
go version
protoc --version
```

**Windows:**
- Install Go from: https://golang.org/dl/
- Install protoc from: https://github.com/protocolbuffers/protobuf/releases
- Add both to your PATH environment variable

## Installation

1. Clone the repository:
```bash
git clone https://github.com/xyplex3/RedTeamCoin.git
cd RedTeamCoin
```

2. Install required tools and dependencies:
```bash
make install-tools
make deps
```

3. Generate protobuf code and build:
```bash
make build
```

4. (Optional) Generate TLS certificates for HTTPS:
```bash
./generate_certs.sh
```

## Usage

### Running the Server

**Option 1: HTTP (Default)**
```bash
make run-server
# or
./bin/server
```

The server will start on:
- gRPC server: port **50051**
- Web dashboard: **http://localhost:8080**

**Option 2: HTTPS/TLS (Recommended)**

First, generate certificates:
```bash
./generate_certs.sh
```

Then start the server with TLS enabled:
```bash
export RTC_USE_TLS=true
make run-server
# or
RTC_USE_TLS=true ./bin/server
```

The server will start on:
- gRPC server: port **50051**
- Web dashboard: **https://localhost:8443**
- HTTP redirect: **http://localhost:8080** (redirects to HTTPS)

**Important Notes:**
- The server generates a secure authentication token displayed in the console
- With HTTPS, browsers will show a security warning (self-signed certificate) - click "Advanced" → "Proceed to localhost"
- Copy the complete URL with token from the console output

### Running a Miner

In a separate terminal, start a mining client:
```bash
make run-client
```

Or directly:
```bash
./bin/client
```

**GPU Mining:**

The client automatically detects available GPUs (NVIDIA CUDA and AMD/Intel OpenCL). To control GPU mining:

```bash
# Default: Auto-detect GPUs and use if available
./bin/client

# Disable GPU mining (CPU only)
GPU_MINING=false ./bin/client

# Enable hybrid mode (CPU + GPU simultaneously)
HYBRID_MINING=true ./bin/client
```

**Note:** GPU mining is currently in framework mode. See [GPU_MINING.md](GPU_MINING.md) for production setup instructions.

You can run multiple miners simultaneously by opening additional terminals and running the client command again.

### Web Dashboard

The server console will display a URL with the authentication token included. Copy and paste this URL into your browser, or manually navigate to:

```
http://<server IP address>:8080?token=YOUR_AUTH_TOKEN_HERE
```

Replace `YOUR_AUTH_TOKEN_HERE` with the token displayed when the server started.

The dashboard displays:
- **Pool Statistics**: Total miners, active miners, hash rate, blockchain height
- **Connected Miners**: List of all miners with IP addresses, hostnames, and stats
- **Recent Blocks**: Last 10 mined blocks with details

The dashboard auto-refreshes every 5 seconds.

### Authentication

All API endpoints (except the dashboard homepage) require authentication via Bearer token in the `Authorization` header.

**Setting a Custom Token:**
```bash
export RTC_AUTH_TOKEN="your-secret-token-here"
./bin/server
```

**Using the Auto-Generated Token:**
When you start the server without setting `RTC_AUTH_TOKEN`, a secure random token is automatically generated and displayed in the console.

### TLS/HTTPS Configuration

**Environment Variables:**
- `RTC_USE_TLS` - Set to `true` to enable HTTPS (default: `false`)
- `RTC_CERT_FILE` - Path to TLS certificate (default: `certs/server.crt`)
- `RTC_KEY_FILE` - Path to TLS private key (default: `certs/server.key`)
- `RTC_AUTH_TOKEN` - Custom authentication token (optional)

**Example with HTTPS:**
```bash
# Generate certificates (one time)
./generate_certs.sh

# Start server with HTTPS
export RTC_USE_TLS=true
export RTC_AUTH_TOKEN="my-secure-token"
./bin/server
```

**Using Custom Certificates:**
```bash
export RTC_USE_TLS=true
export RTC_CERT_FILE="/path/to/your/cert.pem"
export RTC_KEY_FILE="/path/to/your/key.pem"
./bin/server
```

## API Endpoints

The server provides the following REST API endpoints:

- `GET /` - Web dashboard (HTML) - **No authentication required**
- `GET /blocks` - View all blocks page (HTML) - **No authentication required**
- `GET /api/stats` - Pool statistics (JSON) - **Requires authentication**
- `GET /api/miners` - List of all miners (JSON) - **Requires authentication**
- `GET /api/blockchain` - Complete blockchain (JSON) - **Requires authentication**
- `GET /api/blocks/{index}` - Specific block details (JSON) - **Requires authentication**
- `GET /api/validate` - Validate blockchain integrity (JSON) - **Requires authentication**
- `GET /api/cpu` - CPU and GPU usage statistics for all miners (JSON) - **Requires authentication**
- `POST /api/miner/pause` - Pause mining for a specific miner - **Requires authentication**
- `POST /api/miner/resume` - Resume mining for a specific miner - **Requires authentication**
- `POST /api/miner/delete` - Delete a miner from the pool (client auto-terminates and self-deletes) - **Requires authentication**
- `POST /api/miner/throttle` - Set CPU throttle percentage for a specific miner - **Requires authentication**

### API Authentication Examples

**Using curl (HTTP):**
```bash
# Get pool statistics
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" http://localhost:8080/api/stats

# Get miners list
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" http://localhost:8080/api/miners

# Get blockchain
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" http://localhost:8080/api/blockchain

# Validate blockchain
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" http://localhost:8080/api/validate

# Get CPU and GPU usage statistics (total and per miner)
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" http://localhost:8080/api/cpu

# View GPU-enabled miners
curl -H "Authorization: Bearer YOUR_TOKEN_HERE" http://localhost:8080/api/cpu | \
  jq '.miner_stats[] | select(.gpu_enabled == true)'

# Pause a miner
curl -X POST -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"miner_id":"miner-hostname-1234567890"}' \
  http://localhost:8080/api/miner/pause

# Resume a miner
curl -X POST -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"miner_id":"miner-hostname-1234567890"}' \
  http://localhost:8080/api/miner/resume

# Delete a miner (client will auto-terminate and delete its executable)
curl -X POST -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"miner_id":"miner-hostname-1234567890"}' \
  http://localhost:8080/api/miner/delete

# Set CPU throttle (0-100%, 0 = no limit)
curl -X POST -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{"miner_id":"miner-hostname-1234567890","throttle_percent":50}' \
  http://localhost:8080/api/miner/throttle
```

**Using curl (HTTPS with self-signed certificate):**
```bash
# Use -k flag to accept self-signed certificates
curl -k -H "Authorization: Bearer YOUR_TOKEN_HERE" https://localhost:8443/api/stats
curl -k -H "Authorization: Bearer YOUR_TOKEN_HERE" https://localhost:8443/api/miners
curl -k -H "Authorization: Bearer YOUR_TOKEN_HERE" https://localhost:8443/api/blockchain
```

**Using JavaScript/Fetch:**
```javascript
const token = 'YOUR_TOKEN_HERE';
const headers = { 'Authorization': `Bearer ${token}` };

fetch('http://localhost:8080/api/stats', { headers })
  .then(r => r.json())
  .then(data => console.log(data));
```

**Using Python requests:**
```python
import requests

token = 'YOUR_TOKEN_HERE'
headers = {'Authorization': f'Bearer {token}'}

response = requests.get('http://localhost:8080/api/stats', headers=headers)
print(response.json())
```

## gRPC Services

The mining pool provides the following gRPC services:

- `RegisterMiner` - Register a new miner with IP and hostname
- `GetWork` - Request mining work from the pool
- `SubmitWork` - Submit a mined block
- `Heartbeat` - Send miner status updates
- `StopMining` - Gracefully stop mining

## Configuration

Default configuration in the code:

- **Blockchain Difficulty**: 4 (leading zeros required in block hash)
- **Block Reward**: 50 RTC
- **gRPC Port**: 50051
- **API Port**: 8080
- **Heartbeat Interval**: 30 seconds

To modify these, edit the constants in:
- `server/main.go` - Server configuration
- `client/main.go` - Client configuration

## Development

### Build Commands

```bash
make proto                # Generate protobuf code only
make build                # Build server and client (CPU only)
make build-windows        # Cross-compile client for Windows
make build-all-platforms  # Cross-compile client for all platforms
make clean                # Remove build artifacts
make deps                 # Download Go dependencies
make init                 # Full project initialization
```

### Cross-Compilation for Windows

To build the client for Windows from Linux/macOS:

```bash
make build-windows
```

This creates `bin/client.exe` which can be transferred to Windows systems.

### Cross-Compilation for Multiple Platforms

To build clients for all supported platforms:

```bash
make build-all-platforms
```

This creates the following binaries in the `bin/` directory:
- `client-linux-amd64` - Linux 64-bit (Intel/AMD)
- `client-linux-arm64` - Linux ARM64 (Raspberry Pi, AWS Graviton, etc.)
- `client-windows-amd64.exe` - Windows 64-bit
- `client-darwin-amd64` - macOS 64-bit (Intel)
- `client-darwin-arm64` - macOS ARM64 (Apple Silicon M1/M2/M3)

**Note**: Cross-compiled binaries are CPU-only and do not include GPU mining support (CGO is disabled for cross-compilation).

### Analysis Tools

RedTeamCoin includes a damage assessment report generator for analyzing the impact of cryptocurrency mining on systems:

```bash
make build-tools
./bin/generate_report -log pool_log.json
```

This generates a comprehensive markdown report (`Report_Miner_Activity_from_<date>_to_<date>.md`) that includes:

- **Resource Consumption Analysis** - CPU/GPU utilization, electricity costs, network impact
- **Performance Impact Assessment** - System degradation levels and productivity loss estimates
- **Infrastructure Damage** - Hardware wear, thermal stress, lifespan reduction
- **Security Implications** - Breach indicators, persistence mechanisms, exposure risks
- **Financial Impact Summary** - Direct and indirect costs with detailed breakdowns
- **System-by-System Analysis** - Individual impact metrics for each compromised system
- **Recommendations** - Immediate, short-term, and long-term remediation steps

See [tools/README.md](tools/README.md) for complete documentation on the report generator.

### Project Structure

- **Blockchain**: Implements a simple proof-of-work blockchain with SHA-256 hashing
- **Mining Pool**: Manages work distribution, miner registration, and block validation
- **Client Miner**: Connects to pool, receives work, mines blocks, and submits solutions
- **gRPC Communication**: All miner-server communication uses Protocol Buffers
- **Web API**: HTTP API built with Go's standard `net/http` package

## How It Works

1. **Server starts** and initializes the blockchain with a genesis block
2. **Miners connect** via gRPC, providing their IP address and hostname
3. **Server assigns work** to miners (blocks to be mined)
4. **Miners compute hashes** trying to find a valid nonce that meets the difficulty requirement
5. **Miners submit solutions** back to the server
6. **Server validates** and adds accepted blocks to the blockchain
7. **Web dashboard** displays real-time statistics and blockchain data

## Documentation

- [GPU_MINING.md](GPU_MINING.md) - Complete guide to GPU mining with CUDA and OpenCL
- [CPU_STATS_API.md](CPU_STATS_API.md) - Detailed API documentation for CPU and GPU statistics
- [DUAL_IP_TRACKING.md](DUAL_IP_TRACKING.md) - Dual IP address tracking documentation
- [TLS_SETUP.md](TLS_SETUP.md) - HTTPS/TLS configuration guide
- [ARCHITECTURE.md](ARCHITECTURE.md) - System architecture and design details
- [tools/README.md](tools/README.md) - Analysis tools and damage assessment report generator

## Security Note

This is a **demonstration/educational project** for understanding blockchain and cryptocurrency concepts. It is not intended for production use and lacks many features required for a real cryptocurrency (cryptographic signatures, wallets, transaction validation, etc.).

## License

This project is for educational purposes.

## Configuration

### Server Configuration (`server/main.go`)
```go
const (
    grpcPort      = 50051  // gRPC port for miners
    apiPort       = 8443   // HTTPS port (8080 for HTTP fallback)
    httpPort      = 8080   // HTTP redirect port (when TLS enabled)
    difficulty    = 4      // Mining difficulty
    defaultCertFile = "certs/server.crt"
    defaultKeyFile  = "certs/server.key"
)
```

**Environment Variables:**
- `RTC_USE_TLS=true` - Enable HTTPS/TLS (default: false)
- `RTC_CERT_FILE` - TLS certificate path (default: certs/server.crt)
- `RTC_KEY_FILE` - TLS key path (default: certs/server.key)
- `RTC_AUTH_TOKEN` - Custom auth token (optional, auto-generated if not set)

### Client Configuration (`client/main.go`)
```go
const (
    serverAddress = "localhost:50051"
    heartbeatInterval = 30 * time.Second
)
```

### Pool Configuration (`server/pool.go`)
```go
blockReward = 50  // RTC reward per block
```

## Threading and Concurrency

- **Blockchain**: Thread-safe with `sync.RWMutex`
- **Mining Pool**: Thread-safe with `sync.RWMutex`
- **Work Generator**: Runs in goroutine, generates work every 30 seconds
- **Client Heartbeat**: Runs in goroutine, sends heartbeat every 30 seconds
- **Multiple Miners**: Fully supported, each in separate process

## Performance Characteristics

- **Difficulty 4**: ~1-10 seconds per block (single CPU)
- **Difficulty 5**: ~10-60 seconds per block
- **Difficulty 6**: ~1-10 minutes per block

Actual times vary based on CPU performance and luck.
