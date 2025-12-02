.PHONY: all proto build run-server run-client build-cuda build-opencl build-gpu clean install-gpu-deps

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
	@echo "✓ Clean complete"

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
	@echo "  make build           - Build server and client (CPU only)"
	@echo "  make build-cuda      - Build with NVIDIA CUDA GPU support"
	@echo "  make build-opencl    - Build with OpenCL GPU support (AMD, Intel, etc.)"
	@echo "  make build-gpu       - Build with GPU support (auto-detects CUDA or OpenCL)"
	@echo "  make install-gpu-deps - Check and report GPU dependencies"
	@echo "  make run-server      - Start the mining pool server"
	@echo "  make run-client      - Start the mining client"
	@echo "  make clean           - Remove build artifacts"
	@echo "  make deps            - Download Go dependencies"
	@echo "  make proto           - Generate protobuf code"
	@echo "  make init            - Initialize project"
	@echo "  make help            - Show this help message"
