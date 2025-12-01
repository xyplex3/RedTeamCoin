.PHONY: all proto build run-server run-client clean

# Ensure Go bin is in PATH
export PATH := $(PATH):$(HOME)/go/bin

all: proto build

# Install required tools
install-tools:
	@echo "Installing protoc-gen-go and protoc-gen-go-grpc..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	@mkdir -p proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/mining.proto
	@echo "✓ Protobuf code generated"

# Build server and client
build: proto
	@echo "Building server..."
	go build -o bin/server ./server
	@echo "✓ Server built: bin/server"
	@echo "Building client..."
	go build -o bin/client ./client
	@echo "✓ Client built: bin/client"

# Run server
run-server: build
	@echo "Starting RedTeamCoin server..."
	./bin/server

# Run client
run-client: build
	@echo "Starting RedTeamCoin miner..."
	./bin/client

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f proto/*.pb.go
	@echo "✓ Clean complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	@echo "✓ Dependencies downloaded"

# Initialize the project
init: install-tools deps proto
	@echo "✓ Project initialized"
