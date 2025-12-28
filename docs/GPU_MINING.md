# GPU Mining Guide

## Overview

RedTeamCoin supports GPU-accelerated mining using both NVIDIA CUDA and AMD/Intel OpenCL.
The client can automatically detect available GPUs and mine using:

1. **GPU-only mode** - All mining on GPU hardware
2. **Hybrid mode** - Simultaneous CPU + GPU mining for maximum performance
3. **CPU-only mode** - Fallback when no GPUs are detected

## Features

- **CUDA Support** - NVIDIA GPUs (GeForce, Quadro, Tesla)
- **OpenCL Support** - AMD GPUs, Intel integrated graphics, and other OpenCL-capable devices
- **Hybrid Mining** - Parallel CPU and GPU mining with work distribution
- **Auto-Detection** - Automatic GPU discovery and configuration
- **Statistics Tracking** - Separate hash rate tracking for CPU and GPU
- **API Integration** - GPU information exposed via REST API

## Current Implementation Status

### Framework (Current State)

The current implementation provides a **framework** for GPU mining with:

- Complete architecture and code structure
- GPU detection interfaces
- Mining orchestration logic
- Hybrid CPU+GPU work distribution
- Server-side GPU statistics tracking
- API endpoints for GPU information

### What's Needed for Production

To enable actual GPU mining, you need to:

1. **Install GPU Runtime**
   - CUDA Toolkit (for NVIDIA GPUs)
   - OpenCL Runtime (for AMD/Intel GPUs)

2. **Enable CGo and GPU Bindings**
   - Modify build process to use CGo
   - Link against CUDA or OpenCL libraries
   - Implement actual GPU kernels

3. **Implement SHA256 GPU Kernels**
   - Write CUDA kernel for NVIDIA GPUs
   - Write OpenCL kernel for AMD/Intel GPUs

See the **Production Implementation** section below for details.

## Quick Start

### Default Behavior (Framework Mode)

By default, the client will:

1. Attempt to detect GPUs (currently returns empty list)
2. Fall back to CPU-only mining
3. Log GPU detection status

```bash
cd client
go build
./client
```

Output:

```text
=== RedTeamCoin Miner ===

Connecting to mining pool at localhost:50051...
Registering miner...
  Miner ID:   miner-hostname-1234567890
  IP Address: 192.168.1.100
  Hostname:   hostname
  GPUs:       None detected - using CPU only
Successfully registered with pool: Miner registered successfully

Starting mining...
```

### Environment Variables

Control GPU mining behavior:

```bash
# Disable GPU mining (use CPU only)
export RTC_CLIENT_MINING_GPU_ENABLED=false

# Enable hybrid mode (CPU + GPU simultaneously)
export RTC_CLIENT_MINING_HYBRID_MODE=true

# Run the client
./client
```

## Configuration

### GPU Mining Modes

#### 1. CPU Only (Default - Current)

```bash
# No GPUs detected or GPU disabled
export RTC_CLIENT_MINING_GPU_ENABLED=false
./client
```

#### 2. GPU Only (Future - After Production Setup)

```bash
# GPUs detected and GPU enabled (default)
./client
```

#### 3. Hybrid Mode (Future - After Production Setup)

```bash
# Simultaneous CPU and GPU mining
export RTC_CLIENT_MINING_HYBRID_MODE=true
./client
```

### Work Distribution

In hybrid mode, work is distributed as follows:

- **GPU**: Processes nonce ranges 0 - 5,000,000,000 (5 billion)
  - Large batches (1 billion nonces per kernel launch)
  - Parallel processing on thousands of GPU cores

- **CPU**: Processes nonce ranges 5,000,000,000+ (offset to avoid overlap)
  - Smaller batches (100 million nonces)
  - Sequential processing on CPU cores

First to find a valid block wins and stops the other.

## API Integration

### GPU Statistics Endpoint

The `/api/cpu` endpoint now includes GPU information:

```bash
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/cpu | jq .
```

**Response with GPU data:**

```json
{
  "total_miners": 2,
  "active_miners": 2,
  "total_cpu_usage_percent": 85.5,
  "average_cpu_usage_percent": 42.75,
  "total_hashes": 15432890,
  "total_mining_hours": 12.5,
  "total_mining_seconds": 45000.0,
  "total_hash_rate": 342500,
  "total_gpu_hash_rate": 1500000000,
  "gpu_enabled_miners": 1,
  "hybrid_miners": 1,
  "miner_stats": [
    {
      "miner_id": "miner-gpu-rig-001",
      "ip_address": "192.168.1.100",
      "ip_address_actual": "192.168.1.100",
      "hostname": "gpu-mining-rig",
      "cpu_usage_percent": 25.3,
      "total_hashes": 8500000,
      "mining_time_hours": 6.5,
      "mining_time_seconds": 23400.0,
      "hash_rate": 150000,
      "active": true,
      "registered_at": "2025-01-15T10:30:00Z",
      "gpu_devices": [
        {
          "id": 0,
          "name": "NVIDIA GeForce RTX 3080",
          "type": "CUDA",
          "memory_mb": 10240,
          "compute_units": 68,
          "available": true
        }
      ],
      "gpu_hash_rate": 750000000,
      "gpu_enabled": true,
      "hybrid_mode": true
    }
  ]
}
```

### New API Fields

**Aggregate Statistics:**

- `total_gpu_hash_rate` - Combined GPU hash rate across all miners
- `gpu_enabled_miners` - Number of miners with GPU mining enabled
- `hybrid_miners` - Number of miners running hybrid CPU+GPU mode

**Per-Miner Statistics:**

- `gpu_devices[]` - Array of GPU devices available to the miner
  - `id` - GPU device ID
  - `name` - GPU model name
  - `type` - "CUDA" or "OpenCL"
  - `memory_mb` - GPU memory in megabytes
  - `compute_units` - Number of compute units (CUDA cores, OpenCL compute units)
  - `available` - Whether the GPU is available for mining
- `gpu_hash_rate` - Current GPU hash rate
- `gpu_enabled` - Whether GPU mining is enabled
- `hybrid_mode` - Whether hybrid CPU+GPU mining is active

## Production Implementation

### Prerequisites

**For NVIDIA CUDA:**

1. NVIDIA GPU with CUDA support (compute capability 3.5+)
2. CUDA Toolkit 11.0 or later
3. NVIDIA drivers

**For AMD/Intel OpenCL:**

1. AMD GPU or Intel integrated graphics
2. OpenCL runtime
3. Appropriate GPU drivers

### Installation Steps

#### 1. Install CUDA Toolkit (NVIDIA)

**Ubuntu/Debian:**

```bash
# Add NVIDIA package repository
wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.0-1_all.deb
sudo dpkg -i cuda-keyring_1.0-1_all.deb
sudo apt-get update

# Install CUDA
sudo apt-get install -y cuda

# Add to PATH
echo 'export PATH=/usr/local/cuda/bin:$PATH' >> ~/.bashrc
echo 'export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH' >> ~/.bashrc
source ~/.bashrc
```

**Verify installation:**

```bash
nvcc --version
nvidia-smi
```

#### 2. Install OpenCL Runtime (AMD/Intel)

**Ubuntu/Debian:**

```bash
# For AMD GPUs
sudo apt-get install -y mesa-opencl-icd

# For Intel GPUs
sudo apt-get install -y intel-opencl-icd

# OpenCL headers and tools
sudo apt-get install -y opencl-headers clinfo

# Verify installation
clinfo
```

#### 3. Install Go Build Dependencies

```bash
# Install CGo dependencies
sudo apt-get install -y build-essential
```

#### 4. Implement GPU Kernels

**CUDA Kernel (client/kernels/sha256_cuda.cu):**

See detailed implementation in `client/cuda.go` comments (lines 154-216).

Key components:

- SHA256 hash computation on GPU
- Difficulty checking
- Atomic result storage
- Kernel launch configuration

**OpenCL Kernel (client/kernels/sha256_opencl.cl):**

See detailed implementation in `client/opencl.go` comments (lines 149-248).

Key components:

- Platform and device selection
- Kernel compilation
- Buffer management
- Work group configuration

#### 5. Update Build Process

**Makefile additions:**

```makefile
# Build client with CUDA support
client-cuda:
 cd client && \
 nvcc -c kernels/sha256_cuda.cu -o sha256_cuda.o && \
 CGO_ENABLED=1 go build -tags cuda -o client-cuda

# Build client with OpenCL support
client-opencl:
 cd client && \
 CGO_ENABLED=1 CGO_LDFLAGS="-lOpenCL" go build -tags opencl -o client-opencl

# Build with both CUDA and OpenCL
client-gpu:
 cd client && \
 nvcc -c kernels/sha256_cuda.cu -o sha256_cuda.o && \
 CGO_ENABLED=1 CGO_LDFLAGS="-lOpenCL -L/usr/local/cuda/lib64 -lcuda -lcudart" \
 go build -tags "cuda opencl" -o client-gpu
```

#### 6. Enable CGo Bindings

Update `client/cuda.go` and `client/opencl.go` to include CGo directives:

**client/cuda.go:**

```go
// +build cuda

package main

// #cgo LDFLAGS: -L/usr/local/cuda/lib64 -lcuda -lcudart
// #include <cuda.h>
// #include <cuda_runtime.h>
import "C"

import (
    "unsafe"
)

// Actual CUDA implementation with CGo
```

**client/opencl.go:**

```go
// +build opencl

package main

// #cgo LDFLAGS: -lOpenCL
// #include <CL/cl.h>
import "C"

import (
    "unsafe"
)

// Actual OpenCL implementation with CGo
```

### Performance Expectations

**Typical Hash Rates:**

| Hardware | Expected Hash Rate | Speedup vs CPU |
|----------|-------------------|----------------|
| CPU (Intel i7) | 1-10 MH/s | 1x baseline |
| NVIDIA GTX 1660 | 100-200 MH/s | 10-20x |
| NVIDIA RTX 3080 | 500-1000 MH/s | 50-100x |
| NVIDIA RTX 4090 | 1500-2500 MH/s | 150-250x |
| AMD RX 6800 XT | 400-800 MH/s | 40-80x |

**Hybrid Mode Benefits:**

- 5-10% additional hash rate from CPU contribution
- Better hardware utilization
- Increased chance of finding blocks

## Troubleshooting

### No GPUs Detected

**Symptom:**

```text
GPUs: None detected - using CPU only
```

**Solutions:**

1. **Check GPU drivers:**

   ```bash
   # For NVIDIA
   nvidia-smi

   # For AMD/OpenCL
   clinfo
   ```

2. **Verify GPU support:**

   ```bash
   # Check CUDA-capable devices
   nvidia-smi -L

   # Check OpenCL platforms
   clinfo | grep "Platform Name"
   ```

3. **Ensure proper installation:**
   - CUDA Toolkit installed correctly
   - OpenCL runtime available
   - GPU drivers up to date

### GPU Mining Not Working

**Current State:**
GPU mining is in framework mode. To enable actual GPU mining:

1. Follow **Production Implementation** steps above
2. Implement actual GPU kernels
3. Build with CGo enabled
4. Link against CUDA/OpenCL libraries

**Future Debugging:**

1. **Check CUDA availability:**

   ```go
   // In production code
   deviceCount := C.cudaGetDeviceCount(&count)
   if deviceCount != C.cudaSuccess {
       log.Printf("CUDA error: %v", deviceCount)
   }
   ```

2. **Check OpenCL availability:**

   ```go
   // In production code
   status := C.clGetPlatformIDs(0, nil, &platformCount)
   if status != C.CL_SUCCESS {
       log.Printf("OpenCL error: %v", status)
   }
   ```

3. **Enable debug logging:**

   ```bash
   export DEBUG=1
   ./client
   ```

### Performance Issues

**Low GPU utilization:**

- Increase batch size (nonce range per kernel)
- Adjust work group size
- Check GPU memory limitations

**Hybrid mode slower than GPU-only:**

- CPU overhead may exceed benefit
- Try GPU-only mode: `export RTC_CLIENT_MINING_HYBRID_MODE=false`

**Thermal throttling:**

- Monitor GPU temperature
- Improve cooling
- Reduce mining intensity

## Architecture Details

### Mining Flow

```text
Client Startup
    ↓
Detect GPUs (CUDA, OpenCL)
    ↓
Initialize GPU Miners
    ↓
Connect to Pool
    ↓
Register (send GPU info)
    ↓
┌─────────────────┐
│  Mining Loop    │
│                 │
│ ┌─────────────┐ │
│ │ Get Work    │ │
│ └──────┬──────┘ │
│        ↓        │
│   GPU Mode?     │
│    ↙     ↘      │
│  Yes      No    │
│   ↓       ↓     │
│ GPU   CPU Only  │
│Mining  Mining   │
│   ↓       ↓     │
│ Hybrid?         │
│ ↙    ↘          │
│Yes    No        │
│ ↓      ↓        │
│CPU+GPU GPU      │
│ ↓      ↓    ↓   │
│ Submit Work     │
│ └──────┬────────┘
│        ↓        │
│    Heartbeat    │
│  (send stats)   │
└─────────────────┘
```

### Hybrid Mining Work Distribution

```text
Nonce Space: 0 to 2^64

GPU Thread Pool                CPU Thread
┌──────────────┐              ┌──────────┐
│  0 - 1B      │              │ 5B - 5.1B│
│  1B - 2B     │              │ 5.1B-5.2B│
│  2B - 3B     │   Offset →   │ 5.2B-5.3B│
│  3B - 4B     │              │   ...    │
│  4B - 5B     │              │          │
└──────────────┘              └──────────┘

First to find valid hash wins → Signal stop → Submit
```

### Data Flow

```text
Client                     Server                    API
┌──────┐                  ┌──────┐                 ┌─────┐
│ GPU  │                  │ Pool │                 │/cpu │
│Miner │                  │      │                 │     │
│      │                  │      │                 │     │
│Detect│──Register────────→Store │                 │     │
│GPUs  │   (GPU info)     │GPU   │                 │     │
│      │                  │data  │                 │     │
│Mine  │                  │      │                 │     │
│      │──Heartbeat───────→Update│                 │     │
│      │   (GPU stats)    │stats │                 │     │
│      │                  │      │                 │     │
│      │                  │      │←──GET /api/cpu──│User │
│      │                  │      │                 │     │
│      │                  │      │──JSON with──────→     │
│      │                  │      │  GPU stats      │     │
└──────┘                  └──────┘                 └─────┘
```

## Code References

### Client Files

- `client/main.go:42-47` - GPU miner fields in Miner struct
- `client/main.go:65-82` - GPU miner initialization
- `client/main.go:115-132` - GPU device display
- `client/main.go:159-163` - GPU miner startup
- `client/main.go:234-261` - Mining mode selection (GPU/Hybrid/CPU)
- `client/main.go:407-438` - GPU-only mining function
- `client/main.go:440-541` - Hybrid CPU+GPU mining function
- `client/gpu.go` - GPU miner orchestration layer
- `client/cuda.go` - CUDA implementation framework
- `client/opencl.go` - OpenCL implementation framework

### Server Files

- `server/pool.go:9-17` - GPUDeviceInfo struct
- `server/pool.go:33-36` - GPU fields in MinerRecord
- `server/pool.go:229-249` - UpdateHeartbeatWithGPU method
- `server/pool.go:365-373` - GPUDeviceStats struct
- `server/pool.go:387-390` - GPU fields in CPUStats
- `server/pool.go:403-406` - GPU aggregate stats in TotalCPUStats
- `server/pool.go:447-458` - GPU device stats conversion
- `server/grpc_server.go:115-138` - GPU stats handling in Heartbeat

### Protocol Files

- `proto/mining.proto:69-77` - GPUDevice message
- `proto/mining.proto:87-90` - GPU fields in MinerStatus
- `proto/mining_pb.go` - Generated protobuf types with GPU support

## Future Enhancements

1. **Multi-GPU Support**
   - Distribute work across multiple GPUs
   - Per-GPU statistics and monitoring

2. **GPU Pool Selection**
   - Prefer specific GPU types
   - Load balancing across GPUs

3. **Advanced Kernel Optimization**
   - Optimized SHA256 implementations
   - Memory coalescing
   - Shared memory usage

4. **Power Management**
   - GPU power limiting
   - Temperature-based throttling
   - Efficiency modes

5. **Overclocking Support**
   - GPU clock adjustment
   - Memory clock tuning
   - Voltage control (advanced)

## See Also

- [README.md](README.md) - Main project documentation
- [ARCHITECTURE.md](ARCHITECTURE.md) - System architecture
- [CPU_STATS_API.md](CPU_STATS_API.md) - API documentation
- [DUAL_IP_TRACKING.md](DUAL_IP_TRACKING.md) - IP tracking documentation

## References

- [CUDA Programming Guide](https://docs.nvidia.com/cuda/cuda-c-programming-guide/)
- [OpenCL Programming Guide](https://www.khronos.org/opencl/)
- [Bitcoin SHA256 Mining](https://en.bitcoin.it/wiki/Block_hashing_algorithm)
- [GPU Mining Optimization](https://github.com/bitcoin/bitcoin/tree/master/src/crypto)
