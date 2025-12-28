//go:build cuda && cgo
// +build cuda,cgo

// Package main provides NVIDIA CUDA-accelerated proof-of-work mining.
// Compiled only with cuda and cgo build tags. Falls back to CPU on GPU errors.
//
// Build: Install CUDA Toolkit, compile kernel (nvcc -c -m64 -O3 client/mine.cu),
// then CGO_ENABLED=1 go build -tags cuda -o bin/client ./client
//
// GPU detection uses nvidia-smi. Performance: 10-100x faster than CPU mining.
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
// extern int cuda_mine(
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

// CUDAMiner handles NVIDIA GPU mining via CUDA. It manages GPU device
// detection, initialization, and proof-of-work computation using NVIDIA
// CUDA kernels. Falls back to CPU mining if CUDA is unavailable.
type CUDAMiner struct {
	devices []GPUDevice
	running bool
	mu      sync.Mutex
}

// NewCUDAMiner creates a new CUDA miner instance in a stopped state.
// Device detection must be performed by calling DetectDevices before
// mining can begin.
func NewCUDAMiner() *CUDAMiner {
	return &CUDAMiner{
		devices: make([]GPUDevice, 0),
		running: false,
	}
}

// DetectDevices detects NVIDIA CUDA-capable GPUs using nvidia-smi. It
// queries GPU index, name, and memory information, returning a slice of
// detected devices. Returns an empty slice if nvidia-smi is unavailable
// or no NVIDIA GPUs are found.
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

// checkCUDAAvailability reports whether the nvidia-smi command is
// available on the system, indicating CUDA driver installation.
func (cm *CUDAMiner) checkCUDAAvailability() bool {
	cmd := exec.Command("which", "nvidia-smi")
	err := cmd.Run()
	return err == nil
}

// HasDevices reports whether any CUDA devices were detected. This should
// be checked before calling Start or MineBlock.
func (cm *CUDAMiner) HasDevices() bool {
	return len(cm.devices) > 0
}

// Start marks the CUDA miner as running, enabling it to accept mining
// requests. Returns an error if already running or if no CUDA devices
// are available.
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

// Stop halts CUDA mining operations and marks the miner as stopped. Safe
// to call multiple times.
func (cm *CUDAMiner) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.running {
		return
	}

	cm.running = false
	log.Println("CUDA miner stopped")
}

// MineBlock attempts to find a valid block nonce using CUDA GPU
// acceleration. It tries GPU mining first via CUDA kernel; if that fails
// or is unavailable, falls back to CPU mining. Returns the nonce, hash,
// hash count, and whether a valid solution was found.
func (cm *CUDAMiner) MineBlock(blockIndex, timestamp int64, data, previousHash string, difficulty int, startNonce, nonceRange int64) (nonce int64, hash string, hashes int64, found bool) {
	if !cm.running || len(cm.devices) == 0 {
		return 0, "", 0, false
	}

	// Prepare block data (all components except nonce)
	blockData := fmt.Sprintf("%d%d%s%s", blockIndex, timestamp, data, previousHash)
	blockBytes := []byte(blockData)

	// Try GPU mining first
	gpuNonce, gpuHash, found, err := cm.tryGPUMining(blockBytes, difficulty, startNonce, nonceRange)
	if err != nil {
		// GPU error - fallback to CPU mining
		log.Printf("GPU mining error, falling back to CPU: %v", err)
		n, h, hc := cm.mineCPU(blockIndex, timestamp, data, previousHash, difficulty, startNonce, nonceRange)
		return n, h, hc, h != ""
	}

	// GPU searched successfully - return result even if not found
	return gpuNonce, gpuHash, nonceRange, found
}

// tryGPUMining attempts to mine using the CUDA kernel by calling the
// C/CUDA interface. It prepares block data, invokes the GPU kernel, and
// returns results. Returns nonce, hash, found status, and error.
func (cm *CUDAMiner) tryGPUMining(blockData []byte, difficulty int, startNonce, nonceRange int64) (int64, string, bool, error) {
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
	ret := C.cuda_mine(
		blockDataPtr,
		blockDataLen,
		difficultyC,
		startNonceC,
		nonceRangeC,
		resultNoncePtr,
		resultHashPtr,
		foundPtr,
	)

	if ret != 0 {
		return 0, "", false, fmt.Errorf("CUDA mining failed with error code %d", ret)
	}

	if foundFlag {
		hashStr := hex.EncodeToString(resultHash)
		return int64(resultNonce), hashStr, true, nil
	}

	return 0, "", false, nil
}

// mineCPU provides CPU-based proof-of-work mining as a fallback when CUDA
// is unavailable or fails. It iterates through the nonce range using SHA256
// hashing until a valid solution is found or the range is exhausted.
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
