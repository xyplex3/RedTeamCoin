# Building the Java Miner

## Prerequisites

You need to install Maven to build this project. Maven is not currently installed on this system.

## Installation

### Ubuntu/Debian

```bash
sudo apt update
sudo apt install maven openjdk-11-jdk
```

### macOS

```bash
brew install maven openjdk@11
```

### Windows

1. Download and install JDK 11+ from: https://adoptium.net/
2. Download Maven from: https://maven.apache.org/download.cgi
3. Extract Maven and add `bin` directory to PATH

## Building

Once Maven is installed:

```bash
cd java-client
./build.sh
```

Or manually:

```bash
cd java-client
mvn clean package
```

This will:

1. Download all dependencies (gRPC, protobuf, etc.)
2. Generate Java classes from `mining.proto`
3. Compile the Java source code
4. Package everything into a single JAR file: `target/redteamcoin-miner.jar`

## Running

After building:

```bash
# Local server
java -jar target/redteamcoin-miner.jar

# Remote server
java -jar target/redteamcoin-miner.jar -server your-server:50051
```

## Build Artifacts

After a successful build, you'll have:

- `target/redteamcoin-miner.jar` - Executable JAR (~15 MB)
- This JAR includes all dependencies and can be distributed as a single file

## Troubleshooting

### Maven not found after installation

```bash
# Verify installation
mvn --version

# If not found, add to PATH (Linux/macOS)
export PATH=$PATH:/usr/share/maven/bin
```

### Java version issues

```bash
# Check Java version (must be 11+)
java -version

# Use specific Java version
export JAVA_HOME=/usr/lib/jvm/java-11-openjdk-amd64
```

### Build fails with "cannot resolve dependencies"

```bash
# Clear Maven cache and rebuild
rm -rf ~/.m2/repository
mvn clean package
```

## Alternative: Download Pre-built JAR

If you cannot build locally, you can:

1. Build on another system with Maven
2. Transfer the JAR file (`target/redteamcoin-miner.jar`)
3. Run with just Java (no Maven needed for execution)

The JAR is self-contained and only needs Java 11+ to run.
