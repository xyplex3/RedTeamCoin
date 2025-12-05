//go:build cuda && cgo
// +build cuda,cgo

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"unsafe"
)

// #cgo linux CFLAGS: -I/usr/local/cuda/include -I/usr/include
// #cgo linux LDFLAGS: ${SRCDIR}/mine_cuda.o -L/usr/local/cuda/lib64 -L/usr/lib/x86_64-linux-gnu -lcuda -lcudart -lstdc++
// #cgo windows CFLAGS: -IC:/Program\ Files/NVIDIA\ GPU\ Computing\ Toolkit/CUDA/v12.0/include
// #cgo windows LDFLAGS: ${SRCDIR}/mine_cuda.o -LC:/Program\ Files/NVIDIA\ GPU\ Computing\ Toolkit/CUDA/v12.0/lib/x64 -lcuda -lcudart
// #cgo darwin CFLAGS: -I/usr/local/cuda/include
// #cgo darwin LDFLAGS: ${SRCDIR}/mine_cuda.o -L/usr/local/cuda/lib -lcuda -lcudart -lstdc++
// #include <stdint.h>
// #include <stdbool.h>
// #include <cuda_runtime.h>
// extern void cuda_mine(
//     const uint8_t* block_data,
//     int data_len,
//     int difficulty,
//     uint64_t start_nonce,
//     uint64_t nonce_range,
//     uint64_t* result_nonce,
//     uint8_t* result_hash,
//     _Bool* found
// );
import "C"

// CUDAMiner handles NVIDIA GPU mining via CUDA
type CUDAMiner struct {
	devices []GPUDevice
	running bool
	mu      sync.Mutex
}

// NewCUDAMiner creates a new CUDA miner
func NewCUDAMiner() *CUDAMiner {
	return &CUDAMiner{
		devices: make([]GPUDevice, 0),
		running: false,
	}
}

// DetectDevices detects NVIDIA CUDA-capable GPUs
func (cm *CUDAMiner) DetectDevices() []GPUDevice {
	devices := make([]GPUDevice, 0)

	log.Println("Detecting NVIDIA CUDA devices...")

	// Try to detect NVIDIA GPUs using nvidia-smi
	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,memory.total", "--format=csv,noheader,nounits")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("CUDA not available - no NVIDIA GPUs detected or nvidia-smi not installed")
		return devices
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for i, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, ",")
		if len(parts) < 3 {
			continue
		}

		name := strings.TrimSpace(parts[1])
		memoryStr := strings.TrimSpace(parts[2])

		var memory uint64
		_, err := fmt.Sscanf(memoryStr, "%d", &memory)
		if err != nil {
			memory = 2048 // Default to 2GB if parsing fails
		}
		memory = memory * 1024 * 1024 // Convert MB to bytes

		device := GPUDevice{
			ID:           i,
			Name:         fmt.Sprintf("NVIDIA %s", name),
			Type:         "CUDA",
			Memory:       memory,
			ComputeUnits: 128, // Approximate value
			Available:    true,
		}

		devices = append(devices, device)
		log.Printf("Found CUDA GPU: %s (ID: %d, Memory: %d MB)\n", name, i, memory/1024/1024)
	}

	cm.devices = devices
	return devices
}

// checkCUDAAvailability checks if CUDA is available
func (cm *CUDAMiner) checkCUDAAvailability() bool {
	cmd := exec.Command("which", "nvidia-smi")
	err := cmd.Run()
	return err == nil
}

// HasDevices returns true if CUDA devices are available
func (cm *CUDAMiner) HasDevices() bool {
	return len(cm.devices) > 0
}

// Start begins CUDA mining
func (cm *CUDAMiner) Start() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.running {
		return fmt.Errorf("CUDA miner already running")
	}

	if len(cm.devices) == 0 {
		return fmt.Errorf("no CUDA devices available")
	}

	cm.running = true
	log.Println("CUDA miner started")
	return nil
}

// Stop halts CUDA mining
func (cm *CUDAMiner) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.running {
		return
	}

	cm.running = false
	log.Println("CUDA miner stopped")
}

// MineBlock mines a block using CUDA GPU
func (cm *CUDAMiner) MineBlock(blockIndex, timestamp int64, data, previousHash string, difficulty int, startNonce, nonceRange int64) (nonce int64, hash string, hashes int64, found bool) {
	if !cm.running || len(cm.devices) == 0 {
		return 0, "", 0, false
	}

	// Prepare block data (all components except nonce)
	blockData := fmt.Sprintf("%d%d%s%s", blockIndex, timestamp, data, previousHash)
	blockBytes := []byte(blockData)

	// Try GPU mining first
	gpuNonce, gpuHash, found := cm.tryGPUMining(blockBytes, difficulty, startNonce, nonceRange)
	if found {
		return gpuNonce, gpuHash, nonceRange, true
	}

	// Fallback to CPU mining if GPU not available or kernel compilation failed
	n, h, hc := cm.mineCPU(blockIndex, timestamp, data, previousHash, difficulty, startNonce, nonceRange)
	return n, h, hc, h != ""
}

// tryGPUMining attempts to mine using CUDA
func (cm *CUDAMiner) tryGPUMining(blockData []byte, difficulty int, startNonce, nonceRange int64) (int64, string, bool) {
	// Allocate result buffers
	var resultNonce uint64
	resultHash := make([]byte, 32)
	foundFlag := false

	// Prepare C data
	blockDataPtr := (*C.uint8_t)(unsafe.Pointer(&blockData[0]))
	blockDataLen := C.int(len(blockData))
	difficultyC := C.int(difficulty)
	startNonceC := C.uint64_t(startNonce)
	nonceRangeC := C.uint64_t(nonceRange)
	resultNoncePtr := (*C.uint64_t)(unsafe.Pointer(&resultNonce))
	resultHashPtr := (*C.uint8_t)(unsafe.Pointer(&resultHash[0]))
	foundPtr := (*C._Bool)(unsafe.Pointer(&foundFlag))

	// Call CUDA kernel
	C.cuda_mine(
		blockDataPtr,
		blockDataLen,
		difficultyC,
		startNonceC,
		nonceRangeC,
		resultNoncePtr,
		resultHashPtr,
		foundPtr,
	)

	if foundFlag {
		hashStr := hex.EncodeToString(resultHash)
		return int64(resultNonce), hashStr, true
	}

	return 0, "", false
}

// mineCPU falls back to CPU mining
func (cm *CUDAMiner) mineCPU(blockIndex, timestamp int64, data, previousHash string, difficulty int, startNonce, nonceRange int64) (int64, string, int64) {
	prefix := ""
	for i := 0; i < difficulty; i++ {
		prefix += "0"
	}

	hashes := int64(0)
	for n := startNonce; n < startNonce+nonceRange; n++ {
		record := strconv.FormatInt(blockIndex, 10) +
			strconv.FormatInt(timestamp, 10) +
			data +
			previousHash +
			strconv.FormatInt(n, 10)

		h := sha256.New()
		h.Write([]byte(record))
		hashed := h.Sum(nil)
		hashStr := hex.EncodeToString(hashed)

		hashes++

		if len(hashStr) >= difficulty && hashStr[:difficulty] == prefix {
			return n, hashStr, hashes
		}
	}

	return 0, "", hashes
}

/*
CUDA Implementation Notes:

The CUDA kernel (mine.cu) implements:
1. Full SHA256 computation on GPU
2. Parallel nonce testing with kernel grid/block configuration
3. Device memory management and data transfer
4. Atomic operations for result synchronization

To build and use:

1. Install NVIDIA CUDA Toolkit:
   - Download from: https://developer.nvidia.com/cuda-downloads
   - Install dependencies: sudo apt-get install nvidia-cuda-toolkit

2. Compile the CUDA kernel:
   - nvcc -c -m64 -O3 client/mine.cu -o client/mine.o

3. Build with CUDA support:
   - CGO_ENABLED=1 go build -tags cuda -o bin/client ./client

4. GPU detection and fallback:
   - If CUDA not available or kernel fails, CPU fallback is used
   - Performance: GPU mining can be 10-100x faster than CPU for large nonce ranges
   - Energy efficient: Offloads computation to GPU, reducing CPU usage

The mining function automatically:
- Tries GPU mining first
- Falls back to CPU if GPU unavailable
- Reports hash count for performance monitoring
*/
