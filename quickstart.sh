#!/bin/bash

echo "=== RedTeamCoin Quick Start ==="
echo

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed."
    echo "Please install Protocol Buffers compiler:"
    echo "  Ubuntu/Debian: sudo apt-get install -y protobuf-compiler"
    echo "  macOS: brew install protobuf"
    echo "  Or download from: https://github.com/protocolbuffers/protobuf/releases"
    exit 1
fi

echo "✓ protoc found"

# Install Go tools
echo "Installing Go protobuf plugins..."
make install-tools

# Download dependencies
echo "Downloading Go dependencies..."
make deps

# Generate protobuf code and build
echo "Building project..."
make build

echo
echo "✓ Build complete!"
echo
echo "To start the server:"
echo "  make run-server"
echo "  or: ./bin/server"
echo
echo "To start a miner (in a new terminal):"
echo "  make run-client"
echo "  or: ./bin/client"
echo
echo "Web dashboard will be available at: http://localhost:8080"
echo
