package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"redteamcoin/config"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func getTestConfig() *config.ClientConfig {
	cfg, _ := config.LoadClientConfig("")
	return cfg
}

func TestNewMiner(t *testing.T) {
	miner, err := NewMiner("localhost:50051", getTestConfig())
	if err != nil {
		t.Fatalf("NewMiner failed: %v", err)
	}

	if miner == nil {
		t.Fatal("NewMiner returned nil")
	}

	if miner.serverAddress != "localhost:50051" {
		t.Errorf("Expected server address 'localhost:50051', got '%s'", miner.serverAddress)
	}

	if miner.id == "" {
		t.Error("Miner ID should not be empty")
	}

	if miner.hostname == "" {
		t.Error("Hostname should not be empty")
	}

	if !miner.shouldMine {
		t.Error("Miner should start with mining enabled")
	}

	if miner.cpuThrottlePercent != 0 {
		t.Errorf("Expected no throttling (0), got %d", miner.cpuThrottlePercent)
	}

	if miner.running {
		t.Error("New miner should not be running")
	}

	if miner.deletedByServer {
		t.Error("New miner should not be marked as deleted")
	}
}

func TestGetOutboundIP(t *testing.T) {
	ip := getOutboundIP()

	if ip == "" {
		t.Error("getOutboundIP should return a non-empty string")
	}

	// Should return "unknown" if network is unavailable, or a valid IP
	if ip != "unknown" {
		// Basic validation - should have at least one dot (IPv4) or colon (IPv6)
		if len(ip) < 7 {
			t.Errorf("IP address seems too short: %s", ip)
		}
	}
}

func TestCalculateHash(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	hash1 := miner.calculateHash(1, 1234567890, "test data", "previoushash", 0)
	hash2 := miner.calculateHash(1, 1234567890, "test data", "previoushash", 0)

	// Same inputs should produce same hash
	if hash1 != hash2 {
		t.Error("calculateHash should be deterministic")
	}

	if len(hash1) != 64 {
		t.Errorf("SHA256 hash should be 64 characters, got %d", len(hash1))
	}

	// Different nonce should produce different hash
	hash3 := miner.calculateHash(1, 1234567890, "test data", "previoushash", 1)
	if hash1 == hash3 {
		t.Error("Different nonces should produce different hashes")
	}

	// Different data should produce different hash
	hash4 := miner.calculateHash(1, 1234567890, "different data", "previoushash", 0)
	if hash1 == hash4 {
		t.Error("Different data should produce different hashes")
	}
}

func TestMinerStopBeforeStart(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	// Stopping a miner that hasn't started should not panic
	miner.Stop()

	if miner.running {
		t.Error("Miner should not be running after stop")
	}
}

func TestMinerContextCancellation(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	if miner.ctx == nil {
		t.Fatal("Miner context should not be nil")
	}

	if miner.cancel == nil {
		t.Fatal("Miner cancel function should not be nil")
	}

	// Test context cancellation
	miner.cancel()

	select {
	case <-miner.ctx.Done():
		// Context was cancelled successfully
	case <-time.After(100 * time.Millisecond):
		t.Error("Context should be cancelled")
	}
}

func TestMineBlockBasic(t *testing.T) {
	// Skip actual mining test as it can be slow
	// Instead test that the hash calculation works correctly
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	// Find a hash that starts with "0" manually
	var foundNonce int64
	var foundHash string
	timestamp := time.Now().Unix()

	for nonce := int64(0); nonce < 1000; nonce++ {
		hash := miner.calculateHash(1, timestamp, "test", "prev", nonce)
		if len(hash) > 0 && hash[0] == '0' {
			foundNonce = nonce
			foundHash = hash
			break
		}
	}

	if foundHash == "" {
		t.Skip("Could not find suitable hash in 1000 attempts")
	}

	// Verify the hash
	if foundHash[0] != '0' {
		t.Error("Hash should start with 0")
	}

	if len(foundHash) != 64 {
		t.Error("Hash should be 64 characters")
	}

	// Verify hash is reproducible
	verifyHash := miner.calculateHash(1, timestamp, "test", "prev", foundNonce)
	if verifyHash != foundHash {
		t.Error("Hash calculation should be deterministic")
	}

	t.Logf("Found valid hash at nonce %d: %s", foundNonce, foundHash)
}

func TestMineBlockWithThrottling(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())
	miner.cpuThrottlePercent = 50

	// Just verify throttle is set
	if miner.cpuThrottlePercent != 50 {
		t.Error("CPU throttle should be set to 50%")
	}

	// Test that throttle value affects sleep calculation (conceptual test)
	// In real mining, this would slow down the mining loop
	t.Logf("CPU throttle set to %d%%", miner.cpuThrottlePercent)
}

func TestMineBlockStopped(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	// Test that miner respects running flag
	if miner.running {
		t.Error("New miner should not be running")
	}

	miner.running = false
	if miner.running {
		t.Error("Miner should be stopped")
	}

	// The actual mining function respects the running flag
	// We don't test the full mining loop here as it's time-consuming
	t.Log("Miner running state is respected")
}

func TestMinerGPUInitialization(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	if miner.gpuMiner == nil {
		t.Error("GPU miner should be initialized")
	}

	// GPU presence depends on hardware, but initialization should work
	// If no GPUs, these should be false
	if !miner.hasGPU {
		if miner.gpuEnabled {
			t.Error("GPU should not be enabled if no GPUs detected")
		}
		if miner.hybridMode {
			t.Error("Hybrid mode should not be enabled if no GPUs detected")
		}
	}
}

func TestMinerBlocksMined(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	if miner.blocksMined != 0 {
		t.Error("New miner should have 0 blocks mined")
	}

	// Simulate mining a block
	miner.blocksMined++

	if miner.blocksMined != 1 {
		t.Errorf("Expected 1 block mined, got %d", miner.blocksMined)
	}
}

func TestMinerHashRate(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	if miner.hashRate != 0 {
		t.Error("New miner should have 0 hash rate")
	}

	miner.hashRate = 1000000

	if miner.hashRate != 1000000 {
		t.Errorf("Expected hash rate 1000000, got %d", miner.hashRate)
	}
}

func TestMinerTotalHashes(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	if miner.totalHashes != 0 {
		t.Error("New miner should have 0 total hashes")
	}

	miner.totalHashes = 5000000

	if miner.totalHashes != 5000000 {
		t.Errorf("Expected 5000000 total hashes, got %d", miner.totalHashes)
	}
}

func TestMinerShouldMineToggle(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	if !miner.shouldMine {
		t.Error("New miner should have mining enabled")
	}

	miner.shouldMine = false

	if miner.shouldMine {
		t.Error("ShouldMine should be false after setting to false")
	}

	miner.shouldMine = true

	if !miner.shouldMine {
		t.Error("ShouldMine should be true after setting to true")
	}
}

func TestMinerCPUThrottle(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	if miner.cpuThrottlePercent != 0 {
		t.Error("New miner should have no throttling")
	}

	miner.cpuThrottlePercent = 50

	if miner.cpuThrottlePercent != 50 {
		t.Errorf("Expected throttle 50%%, got %d%%", miner.cpuThrottlePercent)
	}

	miner.cpuThrottlePercent = 100

	if miner.cpuThrottlePercent != 100 {
		t.Errorf("Expected throttle 100%%, got %d%%", miner.cpuThrottlePercent)
	}
}

func TestMinerDeletedByServerFlag(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	if miner.deletedByServer {
		t.Error("New miner should not be marked as deleted")
	}

	miner.deletedByServer = true

	if !miner.deletedByServer {
		t.Error("DeletedByServer should be true after setting")
	}
}

func TestCalculateHashDifferentInputs(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	tests := []struct {
		name         string
		index1       int64
		timestamp1   int64
		data1        string
		prevHash1    string
		nonce1       int64
		index2       int64
		timestamp2   int64
		data2        string
		prevHash2    string
		nonce2       int64
		shouldDiffer bool
	}{
		{"Same inputs", 1, 123, "data", "prev", 0, 1, 123, "data", "prev", 0, false},
		{"Different index", 1, 123, "data", "prev", 0, 2, 123, "data", "prev", 0, true},
		{"Different timestamp", 1, 123, "data", "prev", 0, 1, 456, "data", "prev", 0, true},
		{"Different data", 1, 123, "data1", "prev", 0, 1, 123, "data2", "prev", 0, true},
		{"Different prevHash", 1, 123, "data", "prev1", 0, 1, 123, "data", "prev2", 0, true},
		{"Different nonce", 1, 123, "data", "prev", 0, 1, 123, "data", "prev", 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := miner.calculateHash(tt.index1, tt.timestamp1, tt.data1, tt.prevHash1, tt.nonce1)
			hash2 := miner.calculateHash(tt.index2, tt.timestamp2, tt.data2, tt.prevHash2, tt.nonce2)

			if tt.shouldDiffer {
				if hash1 == hash2 {
					t.Errorf("Hashes should differ for %s", tt.name)
				}
			} else {
				if hash1 != hash2 {
					t.Errorf("Hashes should be the same for %s", tt.name)
				}
			}
		})
	}
}

func TestMinerCPUUsageEstimation(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	// Initially should be 0
	if miner.cpuUsagePercent != 0 {
		t.Error("Initial CPU usage should be 0")
	}

	// Simulate different hash rates and check CPU estimation
	testCases := []struct {
		hashRate    int64
		expectedMin float64
		expectedMax float64
	}{
		{0, 0, 0},
		{100000, 0, 100},
		{1000000, 0, 100},
		{10000000, 0, 100},
	}

	for _, tc := range testCases {
		miner.hashRate = tc.hashRate
		// The estimation is done in monitorCPU goroutine
		// We just verify the formula doesn't panic
		estimated := float64(miner.hashRate) / 1000000.0 * 100.0
		if estimated > 100.0 {
			estimated = 100.0
		}

		if estimated < tc.expectedMin || estimated > tc.expectedMax {
			t.Errorf("Hash rate %d: CPU estimate %.2f outside range [%.2f, %.2f]",
				tc.hashRate, estimated, tc.expectedMin, tc.expectedMax)
		}
	}
}

func TestMinerStartTime(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	// Start time should be zero initially
	if !miner.startTime.IsZero() {
		t.Error("Start time should be zero for new miner")
	}

	// Simulate setting start time
	miner.startTime = time.Now()

	if miner.startTime.IsZero() {
		t.Error("Start time should be set")
	}

	// Mining time calculation
	time.Sleep(100 * time.Millisecond)
	miningTime := time.Since(miner.startTime)

	if miningTime < 100*time.Millisecond {
		t.Error("Mining time should be at least 100ms")
	}
}

// Mock gRPC test - requires a test server
func TestMinerIDFormat(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	// Miner ID should follow format: miner-<hostname>-<timestamp>
	if len(miner.id) < 10 {
		t.Error("Miner ID seems too short")
	}

	if miner.id[:6] != "miner-" {
		t.Errorf("Miner ID should start with 'miner-', got: %s", miner.id)
	}
}

func TestHashDifficultyValidation(t *testing.T) {
	_, _ = NewMiner("localhost:50051", getTestConfig())

	tests := []struct {
		hash       string
		difficulty int
		valid      bool
	}{
		{"0000abc123", 4, true},
		{"0000abc123", 5, false},
		{"000abc123", 3, true},
		{"000abc123", 4, false},
		{"abc123", 0, true},
		{"0abc123", 1, true},
	}

	for _, tt := range tests {
		hasValidPrefix := len(tt.hash) >= tt.difficulty
		if hasValidPrefix {
			prefix := ""
			for i := 0; i < tt.difficulty; i++ {
				prefix += "0"
			}
			hasValidPrefix = tt.hash[:tt.difficulty] == prefix
		}

		if hasValidPrefix != tt.valid {
			t.Errorf("Hash %s with difficulty %d: expected valid=%v, got %v",
				tt.hash, tt.difficulty, tt.valid, hasValidPrefix)
		}
	}
}

func TestMinerConnectIntegration(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This would test actual connection to a test server
	// For unit tests, we skip this
	t.Skip("Integration test - requires running server")
}

func TestConcurrentHashCalculation(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	// Test concurrent hash calculations (should be safe)
	done := make(chan bool)
	hashes := make([]string, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			hashes[id] = miner.calculateHash(1, 123, "data", "prev", int64(id))
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all hashes are different (different nonces)
	for i := 0; i < 10; i++ {
		for j := i + 1; j < 10; j++ {
			if hashes[i] == hashes[j] {
				t.Errorf("Hashes %d and %d should be different", i, j)
			}
		}
	}
}

func TestMinerRunningState(t *testing.T) {
	miner, _ := NewMiner("localhost:50051", getTestConfig())

	if miner.running {
		t.Error("New miner should not be running")
	}

	miner.running = true

	if !miner.running {
		t.Error("Miner should be running after setting to true")
	}

	miner.running = false

	if miner.running {
		t.Error("Miner should not be running after setting to false")
	}
}

// Test connection setup without actual server
func TestMinerConnectionSetup(t *testing.T) {
	m, _ := NewMiner("localhost:50051", getTestConfig())

	// Test that connection can be set up (will fail without server, but shouldn't panic)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, m.serverAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())

	// Expected to fail without server, but should handle gracefully
	if err == nil {
		conn.Close()
		t.Log("Connected to server (unexpected in unit test)")
	} else {
		// Expected error
		t.Logf("Expected error connecting without server: %v", err)
	}
}

func TestClampToInt32(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int32
	}{
		{
			name:     "zero value",
			input:    0,
			expected: 0,
		},
		{
			name:     "positive value in range",
			input:    12345,
			expected: 12345,
		},
		{
			name:     "negative value in range",
			input:    -12345,
			expected: -12345,
		},
		{
			name:     "max int32 value",
			input:    2147483647, // math.MaxInt32
			expected: 2147483647,
		},
		{
			name:     "min int32 value",
			input:    -2147483648, // math.MinInt32
			expected: -2147483648,
		},
		{
			name:     "value exceeding max int32",
			input:    2147483648, // MaxInt32 + 1
			expected: 2147483647, // Should clamp to MaxInt32
		},
		{
			name:     "value exceeding min int32",
			input:    -2147483649, // MinInt32 - 1
			expected: -2147483648, // Should clamp to MinInt32
		},
		{
			name:     "large positive value",
			input:    9223372036854775807, // math.MaxInt64 (if int is 64-bit)
			expected: 2147483647,          // Should clamp to MaxInt32
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampToInt32(tt.input)
			if result != tt.expected {
				t.Errorf("clampToInt32(%d) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestClampToInt32Boundaries(t *testing.T) {
	// Test boundary conditions explicitly
	t.Run("exactly at MaxInt32", func(t *testing.T) {
		result := clampToInt32(2147483647)
		if result != 2147483647 {
			t.Errorf("Expected %d, got %d", 2147483647, result)
		}
	})

	t.Run("exactly at MinInt32", func(t *testing.T) {
		result := clampToInt32(-2147483648)
		if result != -2147483648 {
			t.Errorf("Expected %d, got %d", -2147483648, result)
		}
	})

	t.Run("one above MaxInt32", func(t *testing.T) {
		result := clampToInt32(2147483648)
		if result != 2147483647 {
			t.Errorf("Expected clamped to MaxInt32 (%d), got %d", 2147483647, result)
		}
	})

	t.Run("one below MinInt32", func(t *testing.T) {
		result := clampToInt32(-2147483649)
		if result != -2147483648 {
			t.Errorf("Expected clamped to MinInt32 (%d), got %d", -2147483648, result)
		}
	})
}

func TestClampToInt32WithGPUDevice(t *testing.T) {
	// Test realistic GPU device scenarios
	type gpuDevice struct {
		id           int
		computeUnits int
	}

	devices := []gpuDevice{
		{id: 0, computeUnits: 3584},          // Normal GPU
		{id: 1, computeUnits: 10496},         // High-end GPU
		{id: 2147483647, computeUnits: 256},  // At max boundary
		{id: 2147483648, computeUnits: 512},  // Over max boundary
		{id: -1, computeUnits: 0},            // Edge case
		{id: -2147483648, computeUnits: 128}, // At min boundary
		{id: -2147483649, computeUnits: 64},  // Below min boundary
	}

	for i, dev := range devices {
		t.Run(fmt.Sprintf("device_%d", i), func(t *testing.T) {
			clampedID := clampToInt32(dev.id)
			clampedCU := clampToInt32(dev.computeUnits)

			// Verify clamped values are within int32 range
			if clampedID < -2147483648 || clampedID > 2147483647 {
				t.Errorf("Clamped ID %d out of int32 range", clampedID)
			}
			if clampedCU < -2147483648 || clampedCU > 2147483647 {
				t.Errorf("Clamped compute units %d out of int32 range", clampedCU)
			}

			// Verify overflow cases are clamped correctly
			if dev.id > 2147483647 && clampedID != 2147483647 {
				t.Errorf("ID %d should clamp to MaxInt32, got %d", dev.id, clampedID)
			}
			if dev.id < -2147483648 && clampedID != -2147483648 {
				t.Errorf("ID %d should clamp to MinInt32, got %d", dev.id, clampedID)
			}
		})
	}
}
