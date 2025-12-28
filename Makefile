.PHONY: all proto build run-server run-client build-cuda build-opencl build-gpu build-windows build-all-platforms build-tools build-wasm build-java serve-web clean install-gpu-deps init-config init-client-config init-server-config validate-config validate-client-config validate-server-config test

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
	@echo "Protobuf code generated"

# Build server and client (CPU only)
build: proto
	@echo "Building server..."
	go build -o bin/server ./server
	@echo "Server built: bin/server"
	@echo "Building client (CPU only)..."
	CGO_ENABLED=0 go build -o bin/client ./client
	@echo "Client built: bin/client"

# Install GPU dependencies
install-gpu-deps:
	@echo "Installing GPU dependencies..."
	@command -v nvidia-smi >/dev/null 2>&1 || echo "Warning: NVIDIA CUDA not detected. Install from https://developer.nvidia.com/cuda-downloads"
	@command -v rocm-smi >/dev/null 2>&1 || echo "Warning: AMD ROCm not detected. Install from https://rocmdocs.amd.com/en/latest/deploy/linux/index.html"
	@command -v clinfo >/dev/null 2>&1 || echo "Warning: OpenCL not detected. Install OpenCL headers and library"
	@echo "GPU dependencies check complete"

# Build with CUDA support
build-cuda: proto
	@echo "Building with CUDA support..."
	@command -v nvcc >/dev/null 2>&1 || (echo "Error: CUDA compiler (nvcc) not found. Install NVIDIA CUDA Toolkit." && exit 1)
	@echo "Compiling CUDA kernel..."
	nvcc -c -m64 -O3 client/mine.cu -o client/mine.o
	@echo "CUDA kernel compiled"
	@echo "Building client with CUDA support..."
	CGO_ENABLED=1 CGO_LDFLAGS="-L/usr/local/cuda/lib64 -lcuda -lcudart -Lbin" go build -tags cuda -o bin/client-cuda ./client
	@echo "Client built with CUDA support: bin/client-cuda"

# Build with OpenCL support
build-opencl: proto
	@echo "Building with OpenCL support..."
	@echo "Building client with OpenCL support..."
	CGO_ENABLED=1 go build -tags opencl -o bin/client ./client
	@echo "Client built with OpenCL support: bin/client"

# Build with GPU support (tries CUDA first, then OpenCL)
build-gpu: proto
	@echo "Building with GPU support..."
	@if command -v nvcc >/dev/null 2>&1; then \
		echo "CUDA detected - building with CUDA support"; \
		$(MAKE) build-cuda; \
	elif command -v rocm-smi >/dev/null 2>&1 || command -v clinfo >/dev/null 2>&1; then \
		echo "OpenCL detected - building with OpenCL support"; \
		$(MAKE) build-opencl; \
	else \
		echo "Error: No GPU acceleration tools found (CUDA, ROCm, or OpenCL). Use 'make install-gpu-deps' or 'make build' for CPU-only."; \
		exit 1; \
	fi

# Cross-compile client for Windows (CPU only)
build-windows: proto
	@echo "Building client for Windows (CPU only)..."
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/client.exe ./client
	@echo "Windows client built: bin/client.exe"

# Cross-compile client for Windows with OpenCL support
build-windows-opencl: proto
	@echo "Building client for Windows with OpenCL support..."
	@echo "Checking for MinGW-w64 cross-compiler..."
	@command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1 || (echo "Error: MinGW-w64 not found. Install with: sudo apt install mingw-w64" && exit 1)
	@echo "Checking for Windows OpenCL SDK..."
	@test -f /usr/x86_64-w64-mingw32/include/CL/cl.h || (echo "Error: Windows OpenCL headers not found. See WINDOWS_BUILD.md for setup instructions." && exit 1)
	@test -f /usr/x86_64-w64-mingw32/lib/libOpenCL.a || (echo "Error: Windows OpenCL library not found. See WINDOWS_BUILD.md for setup instructions." && exit 1)
	@mkdir -p bin
	@echo "Building Windows client with OpenCL..."
	CGO_ENABLED=1 \
	GOOS=windows \
	GOARCH=amd64 \
	CC=x86_64-w64-mingw32-gcc \
	CXX=x86_64-w64-mingw32-g++ \
	CGO_CFLAGS="-I/usr/x86_64-w64-mingw32/include -DCL_TARGET_OPENCL_VERSION=120" \
	CGO_LDFLAGS="-L/usr/x86_64-w64-mingw32/lib -lOpenCL -static-libgcc -static-libstdc++" \
	go build -tags opencl -ldflags="-s -w" -o bin/client-windows-opencl.exe ./client
	@echo "Windows OpenCL client built: bin/client-windows-opencl.exe"
	@echo ""
	@echo "Note: The client will need OpenCL.dll on the Windows system to run."
	@echo "See WINDOWS_BUILD.md for deployment instructions."

# Cross-compile client for multiple platforms (CPU-only)
build-all-platforms: proto
	@echo "Building client for multiple platforms (CPU-only)..."
	@mkdir -p bin
	@echo "Building Linux AMD64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/client-linux-amd64 ./client
	@echo "Linux AMD64 client built: bin/client-linux-amd64"
	@echo "Building Linux ARM64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/client-linux-arm64 ./client
	@echo "Linux ARM64 client built: bin/client-linux-arm64"
	@echo "Building Windows AMD64..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/client-windows-amd64.exe ./client
	@echo "Windows AMD64 client built: bin/client-windows-amd64.exe"
	@echo "Building macOS AMD64..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/client-darwin-amd64 ./client
	@echo "macOS AMD64 client built: bin/client-darwin-amd64"
	@echo "Building macOS ARM64 (Apple Silicon)..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/client-darwin-arm64 ./client
	@echo "macOS ARM64 client built: bin/client-darwin-arm64"
	@echo ""
	@echo "All platform builds complete (CPU-only)!"
	@echo "For GPU support, use platform-specific targets or build on target system."

# Build Linux client with GPU support (requires OpenCL/CUDA on build system)
build-linux-opencl: proto
	@echo "Building Linux client with OpenCL support..."
	@mkdir -p bin
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags opencl -o bin/client-linux-amd64-opencl ./client
	@echo "Linux OpenCL client built: bin/client-linux-amd64-opencl"

# Build Linux client with CUDA support (requires CUDA toolkit on build system)
build-linux-cuda: proto
	@echo "Building Linux client with CUDA support..."
	@command -v nvcc >/dev/null 2>&1 || (echo "Error: nvcc not found. Install CUDA Toolkit." && exit 1)
	@mkdir -p bin
	nvcc -c -m64 -O3 client/mine.cu -o client/mine.o
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags cuda -o bin/client-linux-amd64-cuda ./client
	@echo "Linux CUDA client built: bin/client-linux-amd64-cuda"

# Build analysis tools
build-tools:
	@echo "Building analysis tools..."
	@mkdir -p bin
	go build -o bin/generate_report ./tools/generate_report.go
	@echo "Report generator built: bin/generate_report"

# Configuration initialization targets
init-config: init-client-config init-server-config
	@echo ""
	@echo "All configuration files initialized!"
	@echo ""
	@echo "Configuration files created:"
	@echo "  - client-config.yaml (edit this for client settings)"
	@echo "  - server-config.yaml (edit this for server settings)"
	@echo ""
	@echo "Optional: Set environment variables (see .env.client.example and .env.server.example)"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Review and edit config files as needed"
	@echo "  2. Run 'make run-server' to start the server"
	@echo "  3. Run 'make run-client' to start the client"
	@echo ""

init-client-config:
	@if [ -f client-config.yaml ]; then \
		echo "client-config.yaml already exists, skipping..."; \
		echo "   (Remove it first if you want to reinitialize)"; \
	else \
		echo "Creating client-config.yaml from example..."; \
		cp client-config.example.yaml client-config.yaml; \
		echo "client-config.yaml created"; \
	fi

init-server-config:
	@if [ -f server-config.yaml ]; then \
		echo "server-config.yaml already exists, skipping..."; \
		echo "   (Remove it first if you want to reinitialize)"; \
	else \
		echo "Creating server-config.yaml from example..."; \
		cp server-config.example.yaml server-config.yaml; \
		echo "server-config.yaml created"; \
	fi

# Validate configuration files
validate-config:
	@echo "Validating configuration files..."
	@go run -tags tools ./tools/validate_config.go

validate-client-config:
	@echo "Validating client configuration..."
	@go run -tags tools ./tools/validate_config.go -client client-config.yaml

validate-server-config:
	@echo "Validating server configuration..."
	@go run -tags tools ./tools/validate_config.go -server server-config.yaml

# Run server (checks for config first)
run-server: build
	@if [ ! -f server-config.yaml ]; then \
		echo "server-config.yaml not found!"; \
		echo ""; \
		echo "Run 'make init-server-config' to create it, or:"; \
		echo "  cp server-config.example.yaml server-config.yaml"; \
		echo ""; \
		exit 1; \
	fi
	@echo "Starting RedTeamCoin server..."
	./bin/server

# Run client (checks for config first)
run-client: build
	@if [ ! -f client-config.yaml ]; then \
		echo "client-config.yaml not found!"; \
		echo ""; \
		echo "Run 'make init-client-config' to create it, or:"; \
		echo "  cp client-config.example.yaml client-config.yaml"; \
		echo ""; \
		exit 1; \
	fi
	@echo "Starting RedTeamCoin miner..."
	./bin/client

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f proto/*.pb.go
	rm -f client/mine.o
	rm -f web-wasm/miner.wasm
	rm -rf java-standalone/target/
	rm -rf java-client/target/
	@echo "Clean complete (removed all binaries and generated files)"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	@echo "Dependencies downloaded"

# Build WebAssembly miner
build-wasm:
	@echo "Building WebAssembly miner..."
	@mkdir -p web-wasm
	GOOS=js GOARCH=wasm go build -o web-wasm/miner.wasm ./web-wasm/
	@echo "WebAssembly miner built: web-wasm/miner.wasm"
	@echo "Files needed for web deployment:"
	@echo "  - web-wasm/miner.wasm"
	@echo "  - web-wasm/wasm_exec.js"
	@echo "  - web-wasm/miner.js"
	@echo "  - web-wasm/worker.js"
	@echo "  - web-wasm/index.html"
	@echo "  - web-wasm/sha256.wgsl (for WebGPU)"

# Build Java standalone miner (GUI-enabled desktop application)
build-java-standalone:
	@echo "Building Java standalone miner (with GUI)..."
	@command -v mvn >/dev/null 2>&1 || (echo "Error: Maven not found. Install with: sudo apt install maven" && exit 1)
	cd java-standalone && mvn clean package -q
	@mkdir -p bin
	cp java-standalone/target/redteamcoin-miner-1.0.0.jar bin/redteamcoin-miner-standalone.jar
	@echo "Java standalone miner built: bin/redteamcoin-miner-standalone.jar"
	@echo "Run with GUI: java -jar bin/redteamcoin-miner-standalone.jar"
	@echo "Run CLI mode: java -jar bin/redteamcoin-miner-standalone.jar --pool localhost:50051"

# Build Java client miner (gRPC headless client)
build-java-client:
	@echo "Building Java gRPC client miner..."
	@command -v mvn >/dev/null 2>&1 || (echo "Error: Maven not found. Install with: sudo apt install maven" && exit 1)
	cd java-client && mvn clean package -q
	@mkdir -p bin
	cp java-client/target/redteamcoin-miner.jar bin/redteamcoin-miner-client.jar
	@echo "Java client miner built: bin/redteamcoin-miner-client.jar"
	@echo "Run with: java -jar bin/redteamcoin-miner-client.jar -server localhost:50051"

# Build all Java miners
build-java-all: build-java-standalone build-java-client
	@echo "All Java miners built successfully"

# Legacy alias for backward compatibility
build-java: build-java-standalone
	@echo "Note: 'build-java' now builds the standalone version. Use 'build-java-all' to build both."

# Serve web miner locally for testing
serve-web: build-wasm
	@echo "Starting local web server on http://localhost:8080"
	@echo "Press Ctrl+C to stop"
	cd web-wasm && python3 -m http.server 8080

# Initialize the project
init: install-tools deps proto
	@echo "Project initialized"

# Help message
help:
	@echo "RedTeamCoin Makefile targets:"
	@echo ""
	@echo "Configuration:"
	@echo "  make init-config        - Initialize all configuration files"
	@echo "  make init-client-config - Initialize client configuration only"
	@echo "  make init-server-config - Initialize server configuration only"
	@echo "  make validate-config    - Validate all configuration files"
	@echo "  make validate-client-config - Validate client configuration only"
	@echo "  make validate-server-config - Validate server configuration only"
	@echo ""
	@echo "Building:"
	@echo "  make build              - Build server and client (CPU only)"
	@echo "  make build-cuda         - Build with NVIDIA CUDA GPU support (native)"
	@echo "  make build-opencl       - Build with OpenCL GPU support (native)"
	@echo "  make build-gpu          - Build with GPU support (auto-detects CUDA or OpenCL)"
	@echo "  make build-linux-opencl - Build Linux client with OpenCL support"
	@echo "  make build-linux-cuda   - Build Linux client with CUDA support"
	@echo "  make build-windows      - Cross-compile client for Windows (CPU only)"
	@echo "  make build-windows-opencl - Cross-compile client for Windows with OpenCL support"
	@echo "  make build-all-platforms - Cross-compile client for all platforms (CPU-only)"
	@echo "  make build-wasm         - Build WebAssembly miner for browsers"
	@echo "  make build-java-standalone - Build Java standalone miner (with GUI)"
	@echo "  make build-java-client  - Build Java gRPC client miner (headless)"
	@echo "  make build-java-all     - Build all Java miners"
	@echo "  make build-java         - Build Java standalone miner (legacy alias)"
	@echo "  make serve-web          - Start local web server for testing web miner"
	@echo "  make build-tools        - Build analysis tools (damage report generator)"
	@echo "  make install-gpu-deps   - Check and report GPU dependencies"
	@echo ""
	@echo "Running:"
	@echo "  make run-server         - Start the mining pool server"
	@echo "  make run-client         - Start the mining client"
	@echo ""
	@echo "Development:"
	@echo "  make test               - Run all tests"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make deps               - Download Go dependencies"
	@echo "  make proto              - Generate protobuf code"
	@echo "  make init               - Initialize project (tools + deps + proto)"
	@echo ""
	@echo "Utilities:"
	@echo "  make help               - Show this help message"
