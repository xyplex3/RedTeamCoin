#!/bin/bash

# RedTeamCoin Java Miner Build Script

set -e

echo "=== Building RedTeamCoin Java Miner ==="
echo

# Check for Java
if ! command -v java &>/dev/null; then
	echo "Error: Java not found. Please install Java 11 or later."
	echo "  Ubuntu/Debian: sudo apt install openjdk-11-jdk"
	echo "  macOS: brew install openjdk@11"
	exit 1
fi

# Check for Maven
if ! command -v mvn &>/dev/null; then
	echo "Error: Maven not found. Please install Maven."
	echo "  Ubuntu/Debian: sudo apt install maven"
	echo "  macOS: brew install maven"
	exit 1
fi

# Check Java version
JAVA_VERSION=$(java -version 2>&1 | awk -F '"' '/version/ {print $2}' | cut -d'.' -f1)
if [ "$JAVA_VERSION" -lt 11 ]; then
	echo "Error: Java 11 or later required (found Java $JAVA_VERSION)"
	exit 1
fi

echo "Java version: $(java -version 2>&1 | head -n 1)"
echo "Maven version: $(mvn -version | head -n 1)"
echo

# Build the project
echo "Building project..."
mvn clean package

echo
echo "=== Build Complete ==="
echo
echo "JAR file created at: target/redteamcoin-miner.jar"
echo
echo "To run:"
echo "  java -jar target/redteamcoin-miner.jar"
echo
echo "To connect to remote server:"
echo "  java -jar target/redteamcoin-miner.jar -server <host>:50051"
echo
