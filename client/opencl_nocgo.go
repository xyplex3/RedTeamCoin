//go:build !opencl || !cgo
// +build !opencl !cgo

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

// OpenCLMiner is a stub when CGO is disabled
type OpenCLMiner struct {
	devices []GPUDevice
	running bool
}

// NewOpenCLMiner creates a new OpenCL miner stub
func NewOpenCLMiner() *OpenCLMiner {
	return &OpenCLMiner{
		devices: make([]GPUDevice, 0),
		running: false,
	}
}

// DetectDevices returns empty list when CGO disabled
func (om *OpenCLMiner) DetectDevices() []GPUDevice {
	return make([]GPUDevice, 0)
}

// HasDevices returns false when CGO disabled
func (om *OpenCLMiner) HasDevices() bool {
	return false
}

// Start is a no-op when CGO disabled
func (om *OpenCLMiner) Start() error {
	return fmt.Errorf("OpenCL mining not available (CGO disabled)")
}

// Stop is a no-op when CGO disabled
func (om *OpenCLMiner) Stop() {
	// No-op
}

// MineBlock falls back to CPU mining when CGO disabled
func (om *OpenCLMiner) MineBlock(blockIndex, timestamp int64, data, previousHash string, difficulty int, startNonce, nonceRange int64) (nonce int64, hash string, hashes int64, found bool) {
	n, h, hc := om.mineCPU(blockIndex, timestamp, data, previousHash, difficulty, startNonce, nonceRange)
	return n, h, hc, false
}

// mineCPU is the CPU fallback implementation
func (om *OpenCLMiner) mineCPU(blockIndex, timestamp int64, data, previousHash string, difficulty int, startNonce, nonceRange int64) (int64, string, int64) {
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
