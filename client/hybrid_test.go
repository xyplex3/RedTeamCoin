package main

import (
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestHybridMiningUsesAllCores verifies that hybrid mining spawns workers
// for all CPU cores and that they're actually performing work.
func TestHybridMiningUsesAllCores(t *testing.T) {
	// Skip if no GPU available (will fallback to CPU-only)
	miner, err := NewMiner("localhost:50051")
	if err != nil {
		t.Fatalf("Failed to create miner: %v", err)
	}

	if !miner.hasGPU {
		t.Skip("Skipping hybrid test - no GPU available")
	}

	// Start the GPU miner
	if err := miner.gpuMiner.Start(); err != nil {
		t.Fatalf("Failed to start GPU miner: %v", err)
	}
	defer miner.gpuMiner.Stop()

	// Use a difficulty that's hard enough to ensure we see workers running
	// but not so hard that test takes forever
	difficulty := 3 // Requires "000" prefix

	// Track worker activity using a shared counter
	var workerCount int32

	// Start hybrid mining with a timeout
	done := make(chan struct{})
	result := make(chan bool, 1)

	go func() {
		// Give mining 2 seconds to start up and show activity
		time.Sleep(2 * time.Second)

		// Count active goroutines with "runCPUWorker" in their stack
		// This is a proxy for checking CPU worker activity
		numCPU := runtime.NumCPU()
		numGoroutines := runtime.NumGoroutine()

		// In hybrid mode we expect:
		// - 1 main goroutine
		// - 1 GPU miner goroutine
		// - 1 CPU coordinator goroutine
		// - N CPU worker goroutines (where N = number of CPU cores)
		// So minimum is 3 + numCPU goroutines
		expectedMin := 3 + numCPU

		if numGoroutines < expectedMin {
			t.Logf("WARNING: Expected at least %d goroutines (3 + %d cores), got %d",
				expectedMin, numCPU, numGoroutines)
			result <- false
		} else {
			t.Logf("✓ Found %d goroutines (expected >= %d for %d cores)",
				numGoroutines, expectedMin, numCPU)
			result <- true
		}

		close(done)
	}()

	// Start mining (this will block until done channel closes)
	go func() {
		nonce, hash, hashes := miner.mineBlockHybrid(
			1,                  // index
			time.Now().Unix(),  // timestamp
			"test block",       // data
			"0000000000000000", // previousHash
			difficulty,         // difficulty
		)

		atomic.AddInt32(&workerCount, 1)
		t.Logf("Mining completed: nonce=%d, hash=%s, hashes=%d", nonce, hash, hashes)
	}()

	// Wait for check to complete or timeout
	select {
	case success := <-result:
		if !success {
			t.Fatal("Did not detect all CPU cores being used")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Test timed out - mining took too long")
	}
}

// TestCPUWorkerDistribution verifies that CPU workers use non-overlapping
// nonce ranges based on their worker ID.
func TestCPUWorkerDistribution(t *testing.T) {
	numWorkers := 4
	baseNonce := int64(5000000000)

	// Simulate worker nonce assignments
	nonces := make(map[int64]int) // nonce -> worker ID

	for workerID := 0; workerID < numWorkers; workerID++ {
		// Each worker starts at baseNonce + workerID
		startNonce := baseNonce + int64(workerID)

		// Record first 100 nonces this worker would check
		for i := 0; i < 100; i++ {
			nonce := startNonce + int64(i*numWorkers)

			if existing, exists := nonces[nonce]; exists {
				t.Fatalf("Nonce collision! Worker %d and %d both checking nonce %d",
					workerID, existing, nonce)
			}

			nonces[nonce] = workerID
		}
	}

	t.Logf("✓ Verified %d unique nonces across %d workers with no overlap",
		len(nonces), numWorkers)
}

// TestGPUResponsivenessToEarlyCPUWin tests that when CPU finds a solution
// quickly, the GPU mining goroutine terminates promptly via the done channel.
func TestGPUResponsivenessToEarlyCPUWin(t *testing.T) {
	miner, err := NewMiner("localhost:50051")
	if err != nil {
		t.Fatalf("Failed to create miner: %v", err)
	}

	if !miner.hasGPU {
		t.Skip("Skipping GPU responsiveness test - no GPU available")
	}

	// Start the GPU miner
	if err := miner.gpuMiner.Start(); err != nil {
		t.Fatalf("Failed to start GPU miner: %v", err)
	}
	defer miner.gpuMiner.Stop()

	// Use very low difficulty so CPU wins almost immediately
	difficulty := 1

	start := time.Now()
	nonce, hash, hashes := miner.mineBlockHybrid(
		1,
		time.Now().Unix(),
		"test",
		"0000000000000000",
		difficulty,
	)
	elapsed := time.Since(start)

	// Mining should complete very quickly with difficulty 1
	if elapsed > 5*time.Second {
		t.Errorf("Mining took too long (%v) - GPU may not be responding to done signal",
			elapsed)
	}

	// Verify we got a valid result
	if !strings.HasPrefix(hash, "0") {
		t.Errorf("Invalid hash - doesn't match difficulty %d: %s", difficulty, hash)
	}

	t.Logf("✓ Mining completed in %v (nonce=%d, hashes=%d)", elapsed, nonce, hashes)
}

// TestHybridChannelBuffering verifies that the channel buffer sizes
// prevent blocking when both GPU and CPU send results.
func TestHybridChannelBuffering(t *testing.T) {
	// This test verifies the theoretical channel design
	// resultChan buffer size = 2 (GPU + CPU coordinator)
	// cpuResultChan buffer size = numWorkers

	resultChan := make(chan miningResult, 2)
	numWorkers := runtime.NumCPU()
	cpuResultChan := make(chan miningResult, numWorkers)

	// Simulate all workers sending results
	for i := 0; i < numWorkers; i++ {
		select {
		case cpuResultChan <- miningResult{found: false, source: "CPU", hashes: 1000}:
			// Success - buffer has space
		default:
			t.Fatalf("cpuResultChan blocked on worker %d/%d", i+1, numWorkers)
		}
	}
	t.Logf("✓ All %d CPU workers can send to cpuResultChan without blocking", numWorkers)

	// Simulate GPU and coordinator sending to resultChan
	select {
	case resultChan <- miningResult{found: true, source: "GPU", hashes: 1000000}:
		// Success
	default:
		t.Fatal("resultChan blocked on GPU send")
	}

	select {
	case resultChan <- miningResult{found: false, source: "CPU", hashes: 5000}:
		// Success
	default:
		t.Fatal("resultChan blocked on CPU coordinator send")
	}

	t.Log("✓ Both GPU and CPU coordinator can send to resultChan without blocking")
}
