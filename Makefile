.PHONY: all proto build run-server run-client build-cuda build-opencl build-gpu build-windows build-all-platforms build-tools clean install-gpu-deps

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

# Build server and client (CPU only)
build: proto
	@echo "Building server..."
	go build -o bin/server ./server
	@echo "✓ Server built: bin/server"
	@echo "Building client (CPU only)..."
	CGO_ENABLED=0 go build -o bin/client ./client
	@echo "✓ Client built: bin/client"

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
	nvcc -c -m64 -O3 client/mine.cu -o bin/mine.o
	@echo "✓ CUDA kernel compiled"
	@echo "Building client with CUDA support..."
	CGO_ENABLED=1 CGO_LDFLAGS="-L/usr/local/cuda/lib64 -lcuda -lcudart -Lbin" go build -tags cuda -o bin/client ./client
	@echo "✓ Client built with CUDA support: bin/client"

# Build with OpenCL support
build-opencl: proto
	@echo "Building with OpenCL support..."
	@echo "Building client with OpenCL support..."
	CGO_ENABLED=1 go build -tags opencl -o bin/client ./client
	@echo "✓ Client built with OpenCL support: bin/client"

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
	@echo "✓ Windows client built: bin/client.exe"

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
	@echo "✓ Windows OpenCL client built: bin/client-windows-opencl.exe"
	@echo ""
	@echo "Note: The client will need OpenCL.dll on the Windows system to run."
	@echo "See WINDOWS_BUILD.md for deployment instructions."

# Cross-compile client for multiple platforms
build-all-platforms: proto
	@echo "Building client for multiple platforms..."
	@mkdir -p bin
	@echo "Building Linux AMD64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/client-linux-amd64 ./client
	@echo "✓ Linux AMD64 client built: bin/client-linux-amd64"
	@echo "Building Linux ARM64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/client-linux-arm64 ./client
	@echo "✓ Linux ARM64 client built: bin/client-linux-arm64"
	@echo "Building Windows AMD64..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o bin/client-windows-amd64.exe ./client
	@echo "✓ Windows AMD64 client built: bin/client-windows-amd64.exe"
	@echo "Building macOS AMD64..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o bin/client-darwin-amd64 ./client
	@echo "✓ macOS AMD64 client built: bin/client-darwin-amd64"
	@echo "Building macOS ARM64 (Apple Silicon)..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o bin/client-darwin-arm64 ./client
	@echo "✓ macOS ARM64 client built: bin/client-darwin-arm64"
	@echo ""
	@echo "All platform builds complete!"

# Build analysis tools
build-tools:
	@echo "Building analysis tools..."
	@mkdir -p bin
	go build -o bin/generate_report ./tools/generate_report.go
	@echo "✓ Report generator built: bin/generate_report"

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
	rm -f client/mine.o
	@echo "✓ Clean complete (removed all binaries and generated files)"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	@echo "✓ Dependencies downloaded"

# Initialize the project
init: install-tools deps proto
	@echo "✓ Project initialized"

# Help message
help:
	@echo "RedTeamCoin Makefile targets:"
	@echo ""
	@echo "  make build              - Build server and client (CPU only)"
	@echo "  make build-cuda         - Build with NVIDIA CUDA GPU support"
	@echo "  make build-opencl       - Build with OpenCL GPU support (AMD, Intel, etc.)"
	@echo "  make build-gpu          - Build with GPU support (auto-detects CUDA or OpenCL)"
	@echo "  make build-windows      - Cross-compile client for Windows (CPU only)"
	@echo "  make build-windows-opencl - Cross-compile client for Windows with OpenCL support"
	@echo "  make build-all-platforms - Cross-compile client for all platforms (Linux, Windows, macOS)"
	@echo "  make build-tools        - Build analysis tools (damage report generator)"
	@echo "  make install-gpu-deps   - Check and report GPU dependencies"
	@echo "  make run-server         - Start the mining pool server"
	@echo "  make run-client         - Start the mining client"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make deps               - Download Go dependencies"
	@echo "  make proto              - Generate protobuf code"
	@echo "  make init               - Initialize project"
	@echo "  make help               - Show this help message"
