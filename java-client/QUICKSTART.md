# Java Client Quick Start

## TL;DR

```bash
# Install prerequisites
sudo apt install openjdk-11-jdk maven

# Build
cd java-client
mvn clean package

# Run
java -jar target/redteamcoin-miner.jar -server your-server:50051
```

## What You Get

- **Single JAR file** (~15 MB) with all dependencies
- **Cross-platform** - Works on Windows, Linux, macOS
- **CPU mining** - Multi-threaded, uses all cores
- **No compilation needed** on target systems (just JRE)

## Usage Examples

### Local mining

```bash
java -jar target/redteamcoin-miner.jar
```

### Remote server

```bash
java -jar target/redteamcoin-miner.jar -server 192.168.1.100:50051
```

### Environment variable

```bash
export POOL_SERVER=mining.example.com:50051
java -jar target/redteamcoin-miner.jar
```

### Run in background (Linux/macOS)

```bash
nohup java -jar target/redteamcoin-miner.jar > miner.log 2>&1 &
```

### Run as Windows service

```powershell
# Using NSSM (Non-Sucking Service Manager)
nssm install RedTeamCoinMiner "C:\Program Files\Java\jdk-11\bin\java.exe" `
  "-jar C:\path\to\redteamcoin-miner.jar -server server:50051"
nssm start RedTeamCoinMiner
```

## Deployment

The JAR is self-contained - just copy and run:

```bash
# Copy to target
scp target/redteamcoin-miner.jar user@target:/opt/miner/

# Run on target (only needs Java)
ssh user@target
java -jar /opt/miner/redteamcoin-miner.jar -server pool:50051
```

## Features

✅ Multi-threaded CPU mining (uses all cores)
✅ Server-side pause/resume control
✅ CPU throttling support
✅ Auto-reconnect (retries for 5 minutes)
✅ Self-delete when removed by server
✅ Heartbeat every 30 seconds
✅ Compatible with Go server

## Performance

Comparable to Go client for CPU mining:

- **Single core**: ~2 MH/s
- **8 cores**: ~16 MH/s
- **16 cores**: ~32 MH/s

## Troubleshooting

### "Java not found"

```bash
# Check installation
java -version

# Install
sudo apt install openjdk-11-jdk  # Linux
brew install openjdk@11          # macOS
```

### "Maven not found"

```bash
sudo apt install maven  # Linux
brew install maven      # macOS
```

### "Connection refused"

- Verify server is running
- Check firewall allows port 50051
- Test with: `nc -zv server 50051`

### OutOfMemoryError

```bash
# Increase heap
java -Xmx2G -jar target/redteamcoin-miner.jar
```

## Comparison: Go vs Java Client

| Feature | Go Client | Java Client |
|---------|-----------|-------------|
| CPU Mining | ✅ | ✅ |
| GPU Mining | ✅ | ❌ |
| Binary Size | ~10 MB | ~15 MB |
| Runtime | None | JRE 11+ |
| Cross-platform | ✅ | ✅ |
| Build time | Fast | Moderate |
| Deployment | Copy binary | Copy JAR |

**Use Java client when:**

- Target systems have Java installed
- Easy deployment is priority
- Cross-platform JAR is preferred
- CPU-only mining is sufficient

**Use Go client when:**

- GPU mining is needed
- No runtime dependencies wanted
- Maximum performance required
