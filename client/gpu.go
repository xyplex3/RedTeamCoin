// Package main implements the RedTeamCoin mining client.
package main

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

// GPUDevice represents a detected GPU device available for mining.
// Devices can be either CUDA (NVIDIA) or OpenCL (AMD/Intel) compatible.
type GPUDevice struct {
	ID           int    // Unique device identifier
	Name         string // Device name from hardware
	Type         string // Device type: "CUDA" or "OpenCL"
	Memory       uint64 // Total device memory in bytes
	ComputeUnits int    // Number of compute units/SMs/CUs
	Available    bool   // Whether device is currently available for use
}

// GPUMiner coordinates GPU-based cryptocurrency mining across multiple
// devices. It supports both CUDA (NVIDIA) and OpenCL (AMD/Intel) devices
// and can utilize all detected GPUs simultaneously.
//
// The miner automatically detects available devices during initialization
// and manages their lifecycle. All operations are thread-safe.
type GPUMiner struct {
	devices     []GPUDevice  // All detected GPU devices
	running     bool         // Whether mining is active
	mu          sync.Mutex   // Protects concurrent access
	hashCount   int64        // Atomic hash counter
	cudaMiner   *CUDAMiner   // CUDA mining implementation
	openCLMiner *OpenCLMiner // OpenCL mining implementation
}

// NewGPUMiner creates a new GPU miner and automatically detects all available
// CUDA and OpenCL devices. The miner is initialized in a stopped state and
// must be started with Start before mining begins.
func NewGPUMiner() *GPUMiner {
	gm := &GPUMiner{
		devices:   make([]GPUDevice, 0),
		running:   false,
		hashCount: 0,
	}

	// Detect CUDA devices
	gm.cudaMiner = NewCUDAMiner()
	cudaDevices := gm.cudaMiner.DetectDevices()
	gm.devices = append(gm.devices, cudaDevices...)

	// Detect OpenCL devices
	gm.openCLMiner = NewOpenCLMiner()
	openCLDevices := gm.openCLMiner.DetectDevices()
	gm.devices = append(gm.devices, openCLDevices...)

	return gm
}

// GetDevices returns a slice of all detected GPU devices including both
// CUDA and OpenCL devices. The returned slice is safe to read concurrently
// with mining operations. This method is safe for concurrent use.
func (gm *GPUMiner) GetDevices() []GPUDevice {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	return gm.devices
}

// HasGPUs reports whether any GPU devices (CUDA or OpenCL) were detected
// during initialization. Returns false if no compatible GPUs are available.
func (gm *GPUMiner) HasGPUs() bool {
	return len(gm.devices) > 0
}

// Start initializes and begins GPU mining on all detected devices. It
// starts both CUDA and OpenCL miners if their respective devices are
// available. Returns an error if already running or if no GPUs are detected.
// Individual GPU initialization errors are logged but don't prevent startup.
func (gm *GPUMiner) Start() error {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if gm.running {
		return fmt.Errorf("GPU miner already running")
	}

	if !gm.HasGPUs() {
		return fmt.Errorf("no GPUs detected")
	}

	gm.running = true

	// Start CUDA miner if available
	if gm.cudaMiner.HasDevices() {
		if err := gm.cudaMiner.Start(); err != nil {
			log.Printf("Failed to start CUDA miner: %v", err)
		}
	}

	// Start OpenCL miner if available
	if gm.openCLMiner.HasDevices() {
		if err := gm.openCLMiner.Start(); err != nil {
			log.Printf("Failed to start OpenCL miner: %v", err)
		}
	}

	return nil
}

// Stop halts all GPU mining operations and releases GPU resources. This
// method is safe to call multiple times and safe for concurrent use.
func (gm *GPUMiner) Stop() {
	gm.mu.Lock()
	defer gm.mu.Unlock()

	if !gm.running {
		return
	}

	gm.running = false

	if gm.cudaMiner != nil {
		gm.cudaMiner.Stop()
	}

	if gm.openCLMiner != nil {
		gm.openCLMiner.Stop()
	}
}

// MineBlock attempts to find a valid nonce for the given block parameters
// using GPU acceleration. It tries CUDA first, then OpenCL if CUDA fails
// or is unavailable. Returns the found nonce, hash, total hashes computed,
// and whether a valid solution was found within the nonce range.
func (gm *GPUMiner) MineBlock(blockIndex, timestamp int64, data, previousHash string, difficulty int, startNonce, nonceRange int64) (nonce int64, hash string, hashes int64, found bool) {
	// Try CUDA first if available
	if gm.cudaMiner != nil && gm.cudaMiner.HasDevices() {
		nonce, hash, hashes, found = gm.cudaMiner.MineBlock(blockIndex, timestamp, data, previousHash, difficulty, startNonce, nonceRange)
		if found {
			atomic.AddInt64(&gm.hashCount, hashes)
			return
		}
		atomic.AddInt64(&gm.hashCount, hashes)
	}

	// Try OpenCL if CUDA didn't find solution
	if gm.openCLMiner != nil && gm.openCLMiner.HasDevices() {
		nonce, hash, hashes, found = gm.openCLMiner.MineBlock(blockIndex, timestamp, data, previousHash, difficulty, startNonce, nonceRange)
		atomic.AddInt64(&gm.hashCount, hashes)
		return
	}

	return 0, "", 0, false
}

// GetHashCount returns the cumulative number of hashes computed by all
// GPU devices since mining started. This counter is updated atomically
// and is safe to read concurrently.
func (gm *GPUMiner) GetHashCount() int64 {
	return atomic.LoadInt64(&gm.hashCount)
}

// GetStats returns a map containing GPU mining statistics including device
// count, running status, total hashes, and per-device information (name,
// type, memory, compute units, availability). Useful for monitoring and
// reporting.
func (gm *GPUMiner) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["devices"] = len(gm.devices)
	stats["running"] = gm.running
	stats["total_hashes"] = atomic.LoadInt64(&gm.hashCount)

	deviceStats := make([]map[string]interface{}, 0)
	for _, device := range gm.devices {
		deviceStats = append(deviceStats, map[string]interface{}{
			"id":            device.ID,
			"name":          device.Name,
			"type":          device.Type,
			"memory_mb":     device.Memory / 1024 / 1024,
			"compute_units": device.ComputeUnits,
			"available":     device.Available,
		})
	}
	stats["device_list"] = deviceStats

	return stats
}
