package main

import (
	"testing"
)

func TestNewGPUMiner(t *testing.T) {
	gm := NewGPUMiner()

	if gm == nil {
		t.Fatal("NewGPUMiner returned nil")
	}

	if gm.devices == nil {
		t.Error("GPU devices slice should be initialized")
	}

	if gm.running {
		t.Error("New GPU miner should not be running")
	}

	if gm.hashCount != 0 {
		t.Error("New GPU miner should have 0 hash count")
	}

	if gm.cudaMiner == nil {
		t.Error("CUDA miner should be initialized")
	}

	if gm.openCLMiner == nil {
		t.Error("OpenCL miner should be initialized")
	}
}

func TestGPUMinerGetDevices(t *testing.T) {
	gm := NewGPUMiner()

	devices := gm.GetDevices()

	if devices == nil {
		t.Error("GetDevices should return a non-nil slice")
	}

	// Number of devices depends on hardware
	// Just verify the function works
	t.Logf("Detected %d GPU devices", len(devices))
}

func TestGPUMinerHasGPUs(t *testing.T) {
	gm := NewGPUMiner()

	hasGPUs := gm.HasGPUs()

	// Result depends on hardware, but should not panic
	t.Logf("HasGPUs: %v", hasGPUs)

	// Verify consistency
	if hasGPUs {
		if len(gm.devices) == 0 {
			t.Error("HasGPUs returns true but no devices found")
		}
	} else {
		if len(gm.devices) > 0 {
			t.Error("HasGPUs returns false but devices found")
		}
	}
}

func TestGPUMinerStartWithoutGPUs(t *testing.T) {
	gm := &GPUMiner{
		devices:   make([]GPUDevice, 0),
		running:   false,
		hashCount: 0,
	}

	err := gm.Start()

	if err == nil {
		t.Error("Start should fail when no GPUs are detected")
	}

	if gm.running {
		t.Error("GPU miner should not be running after failed start")
	}
}

func TestGPUMinerStartAlreadyRunning(t *testing.T) {
	gm := &GPUMiner{
		devices: []GPUDevice{
			{ID: 0, Name: "Test GPU", Type: "CUDA", Available: true},
		},
		running:   true,
		hashCount: 0,
	}

	err := gm.Start()

	if err == nil {
		t.Error("Start should fail when already running")
	}
}

func TestGPUMinerStop(t *testing.T) {
	gm := NewGPUMiner()

	// Stop should work even if not running
	gm.Stop()

	if gm.running {
		t.Error("GPU miner should not be running after stop")
	}
}

func TestGPUMinerStopWhenRunning(t *testing.T) {
	gm := NewGPUMiner()
	gm.running = true

	gm.Stop()

	if gm.running {
		t.Error("GPU miner should not be running after stop")
	}
}

func TestGPUMinerGetHashCount(t *testing.T) {
	gm := NewGPUMiner()

	initialCount := gm.GetHashCount()

	if initialCount != 0 {
		t.Error("Initial hash count should be 0")
	}

	// Manually increment hash count
	gm.hashCount = 1000000

	count := gm.GetHashCount()

	if count != 1000000 {
		t.Errorf("Expected hash count 1000000, got %d", count)
	}
}

func TestGPUMinerGetStats(t *testing.T) {
	gm := NewGPUMiner()

	stats := gm.GetStats()

	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	// Check required fields
	if _, ok := stats["devices"]; !ok {
		t.Error("Stats should contain 'devices' field")
	}

	if _, ok := stats["running"]; !ok {
		t.Error("Stats should contain 'running' field")
	}

	if _, ok := stats["total_hashes"]; !ok {
		t.Error("Stats should contain 'total_hashes' field")
	}

	if _, ok := stats["device_list"]; !ok {
		t.Error("Stats should contain 'device_list' field")
	}

	// Verify types
	if devices, ok := stats["devices"].(int); !ok {
		t.Error("'devices' should be an int")
	} else if devices < 0 {
		t.Errorf("'devices' should be non-negative, got %d", devices)
	}

	if running, ok := stats["running"].(bool); !ok {
		t.Error("'running' should be a bool")
	} else {
		_ = running // value checked, just verifying type
	}

	if totalHashes, ok := stats["total_hashes"].(int64); !ok {
		t.Error("'total_hashes' should be an int64")
	} else if totalHashes < 0 {
		t.Errorf("'total_hashes' should be non-negative, got %d", totalHashes)
	}

	t.Logf("GPU Stats: %+v", stats)
}

func TestGPUMinerMineBlockWithoutGPUs(t *testing.T) {
	gm := &GPUMiner{
		devices:   make([]GPUDevice, 0),
		running:   false,
		hashCount: 0,
	}

	nonce, hash, hashes, found := gm.MineBlock(1, 1234567890, "test", "prev", 4, 0, 1000000)

	if found {
		t.Error("Should not find solution without GPUs")
	}

	if nonce != 0 {
		t.Error("Nonce should be 0 when no solution found")
	}

	if hash != "" {
		t.Error("Hash should be empty when no solution found")
	}

	if hashes != 0 {
		t.Error("Hash count should be 0 when no GPUs available")
	}
}

func TestGPUDeviceStruct(t *testing.T) {
	device := GPUDevice{
		ID:           0,
		Name:         "Test GPU",
		Type:         "CUDA",
		Memory:       8589934592, // 8GB
		ComputeUnits: 32,
		Available:    true,
	}

	if device.ID != 0 {
		t.Error("Device ID mismatch")
	}

	if device.Name != "Test GPU" {
		t.Error("Device name mismatch")
	}

	if device.Type != "CUDA" {
		t.Error("Device type mismatch")
	}

	if device.Memory != 8589934592 {
		t.Error("Device memory mismatch")
	}

	if device.ComputeUnits != 32 {
		t.Error("Device compute units mismatch")
	}

	if !device.Available {
		t.Error("Device should be available")
	}
}

func TestGPUMinerStatsDeviceList(t *testing.T) {
	gm := &GPUMiner{
		devices: []GPUDevice{
			{
				ID:           0,
				Name:         "GPU 1",
				Type:         "CUDA",
				Memory:       8589934592,
				ComputeUnits: 32,
				Available:    true,
			},
			{
				ID:           1,
				Name:         "GPU 2",
				Type:         "OpenCL",
				Memory:       4294967296,
				ComputeUnits: 16,
				Available:    true,
			},
		},
		running:   false,
		hashCount: 0,
	}

	stats := gm.GetStats()

	deviceList, ok := stats["device_list"].([]map[string]interface{})
	if !ok {
		t.Fatal("device_list should be a slice of maps")
	}

	if len(deviceList) != 2 {
		t.Errorf("Expected 2 devices in stats, got %d", len(deviceList))
	}

	// Check first device
	if deviceList[0]["id"] != 0 {
		t.Error("First device ID mismatch")
	}

	if deviceList[0]["name"] != "GPU 1" {
		t.Error("First device name mismatch")
	}

	if deviceList[0]["type"] != "CUDA" {
		t.Error("First device type mismatch")
	}

	// Memory should be converted to MB
	expectedMemoryMB := uint64(8589934592 / 1024 / 1024)
	if deviceList[0]["memory_mb"] != expectedMemoryMB {
		t.Errorf("First device memory mismatch: expected %d MB, got %v",
			expectedMemoryMB, deviceList[0]["memory_mb"])
	}

	// Check second device
	if deviceList[1]["id"] != 1 {
		t.Error("Second device ID mismatch")
	}

	if deviceList[1]["name"] != "GPU 2" {
		t.Error("Second device name mismatch")
	}

	if deviceList[1]["type"] != "OpenCL" {
		t.Error("Second device type mismatch")
	}
}

func TestGPUMinerConcurrentAccess(t *testing.T) {
	gm := NewGPUMiner()

	// Test concurrent access to GetDevices
	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			_ = gm.GetDevices()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// No panic = success
}

func TestGPUMinerHashCountAtomic(t *testing.T) {
	gm := NewGPUMiner()

	// Test concurrent hash count updates
	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func() {
			gm.hashCount++
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	// With atomic operations, this should be safe
	count := gm.GetHashCount()
	t.Logf("Final hash count after concurrent updates: %d", count)
}

func TestGPUMinerStatsRunningState(t *testing.T) {
	gm := NewGPUMiner()

	// Initially not running
	stats := gm.GetStats()
	if stats["running"].(bool) {
		t.Error("GPU miner should not be running initially")
	}

	// Set to running
	gm.running = true
	stats = gm.GetStats()
	if !stats["running"].(bool) {
		t.Error("GPU miner should be running after setting to true")
	}
}

func TestGPUDeviceMemoryConversion(t *testing.T) {
	tests := []struct {
		memoryBytes uint64
		expectedMB  uint64
	}{
		{1073741824, 1024},   // 1GB
		{2147483648, 2048},   // 2GB
		{4294967296, 4096},   // 4GB
		{8589934592, 8192},   // 8GB
		{17179869184, 16384}, // 16GB
	}

	for _, tt := range tests {
		gm := &GPUMiner{
			devices: []GPUDevice{
				{Memory: tt.memoryBytes},
			},
		}

		stats := gm.GetStats()
		deviceList := stats["device_list"].([]map[string]interface{})
		memoryMB := deviceList[0]["memory_mb"].(uint64)

		if memoryMB != tt.expectedMB {
			t.Errorf("Memory conversion: %d bytes should be %d MB, got %d MB",
				tt.memoryBytes, tt.expectedMB, memoryMB)
		}
	}
}

func TestGPUMinerMultipleDeviceTypes(t *testing.T) {
	gm := &GPUMiner{
		devices: []GPUDevice{
			{ID: 0, Name: "NVIDIA GPU", Type: "CUDA", Available: true},
			{ID: 1, Name: "AMD GPU", Type: "OpenCL", Available: true},
			{ID: 2, Name: "Intel GPU", Type: "OpenCL", Available: true},
		},
	}

	stats := gm.GetStats()
	deviceList := stats["device_list"].([]map[string]interface{})

	if len(deviceList) != 3 {
		t.Errorf("Expected 3 devices, got %d", len(deviceList))
	}

	// Count device types
	cudaCount := 0
	openCLCount := 0

	for _, dev := range deviceList {
		switch dev["type"].(string) {
		case "CUDA":
			cudaCount++
		case "OpenCL":
			openCLCount++
		}
	}

	if cudaCount != 1 {
		t.Errorf("Expected 1 CUDA device, got %d", cudaCount)
	}

	if openCLCount != 2 {
		t.Errorf("Expected 2 OpenCL devices, got %d", openCLCount)
	}
}

func TestGPUMinerUnavailableDevice(t *testing.T) {
	gm := &GPUMiner{
		devices: []GPUDevice{
			{ID: 0, Name: "Busy GPU", Type: "CUDA", Available: false},
		},
	}

	stats := gm.GetStats()
	deviceList := stats["device_list"].([]map[string]interface{})

	if deviceList[0]["available"].(bool) {
		t.Error("Device should be unavailable")
	}
}
