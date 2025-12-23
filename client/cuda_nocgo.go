//go:build !cuda || !cgo
// +build !cuda !cgo

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

// CUDAMiner is a stub when CGO is disabled
type CUDAMiner struct {
	devices []GPUDevice
	running bool
}

// NewCUDAMiner creates a new CUDA miner stub
func NewCUDAMiner() *CUDAMiner {
	return &CUDAMiner{
		devices: make([]GPUDevice, 0),
		running: false,
	}
}

// DetectDevices returns empty list when CGO disabled
func (cm *CUDAMiner) DetectDevices() []GPUDevice {
	return make([]GPUDevice, 0)
}

// HasDevices returns false when CGO disabled
func (cm *CUDAMiner) HasDevices() bool {
	return false
}

// Start is a no-op when CGO disabled
func (cm *CUDAMiner) Start() error {
	return fmt.Errorf("CUDA mining not available (CGO disabled)")
}

// Stop is a no-op when CGO disabled
func (cm *CUDAMiner) Stop() {
	// No-op
}

// MineBlock falls back to CPU mining when CGO disabled
func (cm *CUDAMiner) MineBlock(blockIndex, timestamp int64, data, previousHash string, difficulty int, startNonce, nonceRange int64) (nonce int64, hash string, hashes int64, found bool) {
	n, h, hc := cm.mineCPU(blockIndex, timestamp, data, previousHash, difficulty, startNonce, nonceRange)
	return n, h, hc, false
}

// mineCPU is the CPU fallback implementation
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
