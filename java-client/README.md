# RedTeamCoin Java Miner

A cross-platform Java client miner for RedTeamCoin mining pool. Packaged as a single executable JAR file.

## Features

- **CPU Mining** - Multi-threaded SHA-256 mining using all available cores
- **Cross-Platform** - Single JAR runs on Windows, Linux, macOS
- **gRPC Communication** - Compatible with RedTeamCoin Go server
- **Server Controls** - Pause/resume, CPU throttling, remote deletion
- **Auto-Reconnect** - Retries connection for 5 minutes on failure
- **Self-Delete** - Removes itself when deleted by server
- **Heartbeat** - 30-second status updates to server

## Prerequisites

- **Java 11 or later** (JRE for running, JDK for building)
- **Maven 3.6+** (for building only)

### Installing Java

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install openjdk-11-jdk maven
```

**macOS:**
```bash
brew install openjdk@11 maven
```

**Windows:**
- Download JDK from: https://adoptium.net/
- Download Maven from: https://maven.apache.org/download.cgi

## Building

```bash
cd java-client

# Compile and package JAR
mvn clean package

# The JAR will be created at: target/redteamcoin-miner.jar
```

## Running

### Basic Usage

```bash
# Connect to localhost
java -jar target/redteamcoin-miner.jar

# Connect to remote server
java -jar target/redteamcoin-miner.jar -server 192.168.1.100:50051
java -jar target/redteamcoin-miner.jar -s mining-pool.example.com:50051
```

### Using Environment Variable

```bash
export POOL_SERVER=192.168.1.100:50051
java -jar target/redteamcoin-miner.jar
```

### Command-Line Options

- `-server <address>` or `-s <address>` - Server address (host:port)

**Priority:** Command-line flag > Environment variable > Default (localhost:50051)

## Deployment

The JAR file is completely self-contained and can be distributed as a single file:

```bash
# Copy to target system
scp target/redteamcoin-miner.jar user@target:/path/to/miner.jar

# Run on target
java -jar /path/to/miner.jar -server your-server:50051
```

## Performance

- Uses all available CPU cores automatically
- Multi-threaded SHA-256 hashing
- Comparable performance to Go client
- Hash rate: ~2-16 MH/s depending on CPU

## Server Features

The Java miner supports all server-side controls:

- **Pause/Resume** - Server can pause mining remotely
- **CPU Throttling** - Server can limit CPU usage (0-100%)
- **Delete Miner** - Server can trigger auto-termination and self-deletion
- **Statistics** - Reports hash rate, blocks mined, CPU usage

## Architecture

```
MinerClient.java (main class)
├── Connection (gRPC)
├── Registration (sends IP, hostname)
├── Mining Loop
│   ├── Get Work (from server)
│   ├── Multi-threaded Mining (all CPU cores)
│   └── Submit Work (solution to server)
├── Heartbeat (30s interval)
└── Self-Delete (when removed by server)
```

## Troubleshooting

### Build Errors

**Problem:** Maven not found
```bash
# Install Maven
sudo apt install maven  # Linux
brew install maven      # macOS
```

**Problem:** Java version too old
```bash
# Check version
java -version

# Should be 11 or higher
```

### Runtime Errors

**Problem:** Connection refused
- Verify server is running
- Check server address and port
- Ensure firewall allows port 50051

**Problem:** OutOfMemoryError
```bash
# Increase heap size
java -Xmx2G -jar target/redteamcoin-miner.jar
```

## Comparison with Go Client

| Feature           | Go Client | Java Client |
|-------------------|-----------|-------------|
| CPU Mining        | ✅        | ✅          |
| GPU Mining        | ✅        | ❌ (CPU only) |
| Multi-threading   | ✅        | ✅          |
| Cross-platform    | ✅        | ✅          |
| Single binary     | ✅        | ✅ (JAR)    |
| No runtime needed | ✅        | ❌ (needs JRE) |
| File size         | ~10 MB    | ~15 MB      |
| Performance       | Baseline  | Similar     |

## Development

### Project Structure

```
java-client/
├── pom.xml                           # Maven build configuration
├── src/main/
│   ├── proto/
│   │   └── mining.proto              # Protocol Buffer definition
│   └── java/com/redteamcoin/miner/
│       └── MinerClient.java          # Main miner implementation
└── target/
    └── redteamcoin-miner.jar         # Built JAR (after mvn package)
```

### Modifying the Code

1. Edit `src/main/java/com/redteamcoin/miner/MinerClient.java`
2. Rebuild: `mvn clean package`
3. Test: `java -jar target/redteamcoin-miner.jar`

### Adding Dependencies

Edit `pom.xml` and add to `<dependencies>` section, then rebuild.

## Security Note

This is a demonstration/educational project. The Java client:
- Connects to mining pools via insecure channels (no TLS by default)
- Can self-delete when instructed by server
- Is intended for authorized testing environments only

## License

This project is licensed under the GNU General Public License v3.0 (GPL-3.0).

See the main RedTeamCoin LICENSE file for details.
