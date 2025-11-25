# RedTeamCoin Architecture

## Overview

RedTeamCoin is a demonstration blockchain cryptocurrency mining pool system that simulates a non-Ethereum based cryptocurrency. The system consists of three main components:

1. **Blockchain Server** - Manages the blockchain and mining pool
2. **Client Miner** - Performs proof-of-work mining
3. **Web Dashboard** - Provides administration and monitoring

## Technology Stack

- **Language**: Go 1.21+
- **RPC Framework**: gRPC with Protocol Buffers
- **Web Server**: Go net/http standard library
- **Dependencies**:
  - google.golang.org/protobuf
  - google.golang.org/grpc

## Component Details

### 1. Blockchain (`server/blockchain.go`)

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

### 2. Mining Pool (`server/pool.go`)

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

### 3. gRPC Server (`server/grpc_server.go`)

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

### 4. Web API (`server/api.go`)

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

**Authentication Middleware:**
```go
func (api *APIServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
    // Validates Bearer token in Authorization header
    // Returns 401 Unauthorized if invalid/missing
}
```

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

## Security Considerations

This is a **demonstration project** and lacks production-ready security features:

**Implemented Security:**
- REST API authentication via Bearer tokens
- Secure random token generation (64 hex characters)
- Environment variable support for token management
- 401 Unauthorized responses for invalid/missing tokens
- HTTPS/TLS support with configurable certificates
- HTTP to HTTPS automatic redirect
- Self-signed certificate generation script
- TLS certificate validation on startup

**Missing Features:**
- No cryptographic signatures
- No wallet system
- No transaction validation
- No peer-to-peer networking
- No consensus mechanism beyond single-server
- No double-spend protection
- No Sybil attack prevention
- No rate limiting
- No token expiration/refresh mechanism
- No certificate rotation
- No mutual TLS (mTLS) for client authentication

**Use Cases:**
- Educational blockchain learning
- Development testing
- Proof-of-work concept demonstration
- Red team training exercises
- API security demonstration

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

## Extension Points

To extend this project:

1. **Add Transactions**: Implement transaction validation and merkle trees
2. **Add Wallets**: Implement public/private key cryptography
3. **Add Networking**: Implement peer-to-peer blockchain distribution
4. **Add Consensus**: Implement consensus mechanism (longest chain rule)
5. **Add Persistence**: Store blockchain to disk (currently in-memory)
6. **Add Smart Contracts**: Add programmable transaction logic
7. **Adjust Difficulty**: Implement dynamic difficulty adjustment
8. **Add Mempool**: Queue pending transactions

## Testing

To test the system:

1. Start the server: `make run-server`
2. Open dashboard: http://localhost:8080
3. Start multiple miners in different terminals: `make run-client`
4. Watch blocks being mined in real-time
5. Monitor statistics on the dashboard

## Troubleshooting

**Problem**: Miner can't connect to server
- Ensure server is running
- Check firewall settings
- Verify ports 50051 and 8080 are available

**Problem**: Protobuf compilation errors
- Run `make install-tools` to install protoc plugins
- Ensure protoc is installed system-wide

**Problem**: Slow mining
- Reduce difficulty in `server/main.go`
- Run multiple miners simultaneously

**Problem**: Blocks rejected as stale
- This is normal when multiple miners compete
- Only one miner wins per block
- Rejected miners automatically request new work
