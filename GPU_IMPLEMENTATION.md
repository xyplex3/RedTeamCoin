# GPU Mining Implementation Guide

## Overview

This project includes complete GPU mining support for NVIDIA CUDA and OpenCL-compatible devices (AMD, Intel, etc.).

The implementation includes:

- **CUDA Kernel** (`client/mine.cu`) - Full SHA256 implementation on NVIDIA GPUs
- **OpenCL Kernel** (`client/mine.cl`) - Cross-platform GPU support
- **CGo Bindings** - Go integration with GPU libraries
- **CPU Fallback** - Automatic fallback to CPU if GPU unavailable

## Quick Start

### For CPU-Only Mining (No GPU)

```bash
make build
./bin/client -server localhost:50051
```

### For CUDA (NVIDIA GPU) Mining

```bash
make build-cuda
./bin/client -server localhost:50051
```

### For OpenCL (AMD/Intel/Other GPU) Mining

```bash
make build-opencl
./bin/client -server localhost:50051
```

### Auto-Detect GPU and Build

```bash
make build-gpu
./bin/client -server localhost:50051
```

## Installation Requirements

### CUDA (NVIDIA GPUs)

**Ubuntu/Debian:**

```bash
# Add NVIDIA package repositories
wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2004/x86_64/cuda-repo-ubuntu2004_11.8.0-1_amd64.deb
sudo dpkg -i cuda-repo-ubuntu2004_11.8.0-1_amd64.deb
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys A4B469963BF863CC

# Update and install
sudo apt update
sudo apt install cuda-toolkit nvidia-driver-545

# Add to PATH (add to ~/.bashrc)
export PATH=/usr/local/cuda/bin:$PATH
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
```

**CentOS/RHEL:**

```bash
sudo yum install cuda-toolkit
```

**Windows:**

- Download from: https://developer.nvidia.com/cuda-downloads
- Install CUDA Toolkit and add to PATH

### OpenCL (AMD/Intel/Cross-Platform)

**AMD GPUs (ROCm):**

```bash
# Ubuntu/Debian
sudo apt install rocm-opencl-runtime rocm-opencl-dev

# Or from Docker:
docker pull rocm/rocm-terminal
```

**Intel GPUs:**

```bash
sudo apt install intel-opencl-icd intel-opencl-dev
```

**Generic OpenCL:**

```bash
sudo apt install ocl-icd-libopencl1 ocl-icd-opencl-dev
clinfo  # Verify installation
```

**macOS:**

```bash
# OpenCL is built-in on macOS
brew install ocl-icd
```

## Build Commands

```bash
# Check GPU dependencies
make install-gpu-deps

# Build with CUDA (requires nvcc compiler)
make build-cuda

# Build with OpenCL
make build-opencl

# Auto-detect and build with available GPU support
make build-gpu

# Build CPU-only (no GPU)
make build

# Clean all artifacts
make clean

# Show help
make help
```

## Implementation Details

### CUDA Mining (`client/mine.cu`)

- Full SHA256 implementation in CUDA C/C++
- Parallel kernel execution with block/thread grid
- Dynamic GPU memory management
- Atomic operations for safe result writing
- Seamless integration with Go via CGo

**Compilation:**

```bash
nvcc -c -m64 -O3 client/mine.cu -o bin/mine.o
```

**Performance:**

- Single GPU can achieve 10-100x speedup over CPU
- Scales with number of CUDA cores
- Optimized memory transfers and kernel launches

### OpenCL Mining (`client/mine.cl`)

- Portable SHA256 implementation in OpenCL
- Works on NVIDIA, AMD, Intel, and other devices
- Dynamic kernel compilation at runtime
- Automatic platform and device detection
- Graceful fallback to CPU

**Supported Devices:**

- NVIDIA GPUs (via CUDA or OpenCL)
- AMD Radeon GPUs (via ROCm)
- Intel Arc GPUs (via Intel OpenCL)
- Qualcomm Adreno (mobile)
- Any OpenCL 1.2+ compatible device

**Performance:**

- Similar to CUDA on comparable hardware
- Cross-platform compatibility
- Wider hardware support

## Execution Flow

```text
Client Start
    ↓
GPU Miner Initialization
    ├─ Try CUDA Detection (nvidia-smi)
    ├─ Try OpenCL Detection (rocm-smi, clinfo)
    └─ Set GPU availability flags
    ↓
Work Loop
    ├─ Get work from pool
    ├─ If GPU available & enabled:
    │  ├─ Try GPU mining first
    │  └─ Fall back to CPU if GPU fails
    └─ If CPU-only: use CPU mining
    ↓
Result Submission
    └─ Submit hash/nonce to pool
```

## Environment Variables

```bash
# Enable/disable GPU mining (default: auto-detect)
export GPU_MINING=true
export GPU_MINING=false

# Enable hybrid CPU+GPU mining
export HYBRID_MINING=true

# CUDA-specific
export CUDA_VISIBLE_DEVICES=0,1  # Select GPUs to use
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH

# OpenCL-specific
export OCL_ICD_FILENAMES=libpocl.so  # Force specific OpenCL implementation
```

## Troubleshooting

### CUDA Build Issues

```bash
# Error: "cannot find -lcuda"
# Solution: Update LD_LIBRARY_PATH and rebuild
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
make build-cuda

# Error: "nvcc not found"
# Solution: Install CUDA Toolkit or add to PATH
export PATH=/usr/local/cuda/bin:$PATH

# Error: GPU out of memory
# Solution: Reduce nonce_range in code or use CPU fallback
```

### OpenCL Build Issues

```bash
# Error: "cannot find -lOpenCL"
# Solution: Install OpenCL development package
sudo apt install ocl-icd-opencl-dev

# Error: "No platform selected"
# Solution: Install platform-specific drivers (CUDA, ROCm, etc.)

# Verify OpenCL installation
clinfo
rocm-smi  # For AMD
nvidia-smi  # For NVIDIA
```

## Performance Monitoring

The miner reports GPU statistics in heartbeats:

```text
GPU Devices Found: 1
  - NVIDIA RTX 3080 (CUDA) - 10240 MB, 68 compute units
Mode: Hybrid (CPU + GPU)
Hash rate: 500 MH/s
```

## Hybrid Mining

Enable simultaneous CPU and GPU mining:

```bash
export HYBRID_MINING=true
./bin/client -server localhost:50051
```

This:

- Runs GPU kernel on separate nonce range
- Runs CPU mining in parallel
- Returns first solution found
- Increases total hash rate

## CPU Fallback

If GPU mining fails:

1. Kernel compilation error → fall back to CPU
2. GPU out of memory → fall back to CPU
3. GPU not available → use CPU
4. GPU disabled via env var → use CPU

No configuration needed - automatic fallback ensures mining always works.

## Testing

```bash
# Build and test with CUDA
make build-cuda
GPU_MINING=true ./bin/client -server localhost:50051

# Test with OpenCL
make build-opencl
GPU_MINING=true ./bin/client -server localhost:50051

# Test CPU fallback
GPU_MINING=false ./bin/client -server localhost:50051

# Test hybrid mode
export HYBRID_MINING=true
./bin/client -server localhost:50051
```

## Files Modified

- `client/mine.cu` - CUDA kernel implementation (NEW)
- `client/mine.cl` - OpenCL kernel implementation (NEW)
- `client/cuda.go` - CUDA bindings and fallback (MODIFIED)
- `client/opencl.go` - OpenCL bindings and fallback (MODIFIED)
- `client/gpu.go` - GPU coordination (UNCHANGED - compatible)
- `client/main.go` - GPU support integration (UNCHANGED - compatible)
- `Makefile` - GPU build targets (MODIFIED)

## References

- NVIDIA CUDA: https://developer.nvidia.com/cuda-downloads
- OpenCL: https://www.khronos.org/opencl/
- AMD ROCm: https://rocmdocs.amd.com/
- Bitcoin SHA256: https://github.com/bitcoin/bitcoin/blob/master/src/crypto/sha256.cpp
- Go CGo: https://golang.org/cmd/cgo/
