//go:build !cuda || !cgo
// +build !cuda !cgo

// Package main provides CUDA mining stubs when CGO or CUDA tags are disabled.
//
// This file is compiled when either the cuda or cgo build tags are not set,
// providing no-op implementations of CUDA mining functions that fall back to
// CPU mining. This ensures the program compiles and runs without CUDA support.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

// CUDAMiner is a stub implementation used when CUDA support is not compiled
// in. All methods return errors or empty results, ensuring the program
// compiles without CGO/CUDA but cannot perform GPU mining.
type CUDAMiner struct {
	devices []GPUDevice
	running bool
}

// NewCUDAMiner creates a new CUDA miner stub that provides no-op
// implementations of all CUDA mining methods.
func NewCUDAMiner() *CUDAMiner {
	return &CUDAMiner{
		devices: make([]GPUDevice, 0),
		running: false,
	}
}

// DetectDevices returns an empty slice, indicating no CUDA devices are
// available when built without CGO/CUDA support.
func (cm *CUDAMiner) DetectDevices() []GPUDevice {
	return make([]GPUDevice, 0)
}

// HasDevices reports whether any CUDA devices were detected. Always returns
// false when built without CGO/CUDA support.
func (cm *CUDAMiner) HasDevices() bool {
	return false
}

// Start attempts to start CUDA mining but always returns an error indicating
// CUDA is unavailable when built without CGO/CUDA support.
func (cm *CUDAMiner) Start() error {
	return fmt.Errorf("CUDA mining not available (CGO disabled)")
}

// Stop is a no-op when built without CGO/CUDA support, as there is no GPU
// mining to halt.
func (cm *CUDAMiner) Stop() {
}

// MineBlock falls back to CPU-based proof-of-work mining when built without
// CGO/CUDA support. Returns the nonce, hash, hash count, and always returns
// false for found (indicating GPU mining was not attempted).
func (cm *CUDAMiner) MineBlock(blockIndex, timestamp int64, data, previousHash string, difficulty int, startNonce, nonceRange int64) (nonce int64, hash string, hashes int64, found bool) {
	n, h, hc := cm.mineCPU(blockIndex, timestamp, data, previousHash, difficulty, startNonce, nonceRange)
	return n, h, hc, false
}

// mineCPU provides CPU-based proof-of-work mining as the sole mining method
// when CUDA is unavailable. It iterates through the nonce range using SHA256
// hashing until a solution matching the difficulty is found or the range is
// exhausted. Returns the nonce, hash, and total hashes computed.
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
