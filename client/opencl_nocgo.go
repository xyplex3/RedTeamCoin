//go:build !opencl || !cgo
// +build !opencl !cgo

// Package main provides OpenCL mining stubs when CGO or OpenCL tags are
// disabled.
//
// This file is compiled when either the opencl or cgo build tags are not set,
// providing no-op implementations of OpenCL mining functions that fall back
// to CPU mining. This ensures the program compiles and runs without OpenCL
// support.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

// OpenCLMiner is a stub implementation used when OpenCL support is not
// compiled in. All methods return errors or empty results, ensuring the
// program compiles without CGO/OpenCL but cannot perform GPU mining.
type OpenCLMiner struct {
	devices []GPUDevice
	running bool
}

// NewOpenCLMiner creates a new OpenCL miner stub that provides no-op
// implementations of all OpenCL mining methods.
func NewOpenCLMiner() *OpenCLMiner {
	return &OpenCLMiner{
		devices: make([]GPUDevice, 0),
		running: false,
	}
}

// DetectDevices returns an empty slice, indicating no OpenCL devices are
// available when built without CGO/OpenCL support.
func (om *OpenCLMiner) DetectDevices() []GPUDevice {
	return make([]GPUDevice, 0)
}

// HasDevices reports whether any OpenCL devices were detected. Always returns
// false when built without CGO/OpenCL support.
func (om *OpenCLMiner) HasDevices() bool {
	return false
}

// Start attempts to start OpenCL mining but always returns an error
// indicating OpenCL is unavailable when built without CGO/OpenCL support.
func (om *OpenCLMiner) Start() error {
	return fmt.Errorf("OpenCL mining not available (CGO disabled)")
}

// Stop is a no-op when built without CGO/OpenCL support, as there is no GPU
// mining to halt.
func (om *OpenCLMiner) Stop() {
}

// MineBlock falls back to CPU-based proof-of-work mining when built without
// CGO/OpenCL support. Returns the nonce, hash, hash count, and always returns
// false for found (indicating GPU mining was not attempted).
func (om *OpenCLMiner) MineBlock(blockIndex, timestamp int64, data, previousHash string, difficulty int, startNonce, nonceRange int64) (nonce int64, hash string, hashes int64, found bool) {
	n, h, hc := om.mineCPU(blockIndex, timestamp, data, previousHash, difficulty, startNonce, nonceRange)
	return n, h, hc, false
}

// mineCPU provides CPU-based proof-of-work mining as the sole mining method
// when OpenCL is unavailable. It iterates through the nonce range using
// SHA256 hashing until a solution matching the difficulty is found or the
// range is exhausted. Returns the nonce, hash, and total hashes computed.
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
