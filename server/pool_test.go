package main

import (
	"testing"
	"time"
)

func TestNewMiningPool(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	if pool == nil {
		t.Fatal("NewMiningPool returned nil")
	}

	if pool.blockchain != bc {
		t.Error("Pool blockchain reference mismatch")
	}

	if pool.blockReward != 50 {
		t.Errorf("Expected block reward 50, got %d", pool.blockReward)
	}

	if pool.miners == nil {
		t.Error("Pool miners map should be initialized")
	}

	if pool.pendingWork == nil {
		t.Error("Pool pendingWork map should be initialized")
	}
}

func TestPoolRegisterMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	ip := "192.168.1.100"
	hostname := "test-host"
	actualIP := "192.168.1.100"

	err := pool.RegisterMiner(minerID, ip, hostname, actualIP)
	if err != nil {
		t.Errorf("RegisterMiner failed: %v", err)
	}

	// Verify miner was registered
	pool.mu.RLock()
	miner, exists := pool.miners[minerID]
	pool.mu.RUnlock()

	if !exists {
		t.Fatal("Miner was not registered")
	}

	if miner.ID != minerID {
		t.Errorf("Expected miner ID %s, got %s", minerID, miner.ID)
	}

	if miner.IPAddress != ip {
		t.Errorf("Expected IP %s, got %s", ip, miner.IPAddress)
	}

	if miner.Hostname != hostname {
		t.Errorf("Expected hostname %s, got %s", hostname, miner.Hostname)
	}

	if !miner.Active {
		t.Error("Newly registered miner should be active")
	}

	if !miner.ShouldMine {
		t.Error("Newly registered miner should have mining enabled")
	}
}

func TestPoolRegisterMinerDuplicate(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	ip := "192.168.1.100"
	hostname := "test-host"
	actualIP := "192.168.1.100"

	// Register once
	err := pool.RegisterMiner(minerID, ip, hostname, actualIP)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Register again (should update, not error)
	err = pool.RegisterMiner(minerID, ip, hostname, actualIP)
	if err != nil {
		t.Errorf("Re-registration failed: %v", err)
	}

	// Should still have only one miner
	if len(pool.miners) != 1 {
		t.Errorf("Expected 1 miner, got %d", len(pool.miners))
	}
}

func TestPoolGetWork(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	block, err := pool.GetWork(minerID)
	if err != nil {
		t.Fatalf("GetWork failed: %v", err)
	}

	if block == nil {
		t.Fatal("GetWork returned nil block")
	}

	if block.Index != 1 {
		t.Errorf("Expected block index 1, got %d", block.Index)
	}

	// Verify pending work was created
	pool.mu.RLock()
	work, exists := pool.pendingWork[minerID]
	pool.mu.RUnlock()

	if !exists {
		t.Error("Pending work should be created")
	}

	if work.Block != block {
		t.Error("Pending work block mismatch")
	}
}

func TestPoolGetWorkUnregisteredMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	_, err := pool.GetWork("unregistered-miner")
	if err == nil {
		t.Error("GetWork should fail for unregistered miner")
	}
}

func TestGetWorkReturnsExistingWork(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	block1, _ := pool.GetWork(minerID)
	block2, _ := pool.GetWork(minerID)

	// Should return the same block
	if block1.Index != block2.Index {
		t.Error("GetWork should return existing pending work")
	}
}

func TestPoolSubmitWork(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	block, _ := pool.GetWork(minerID)

	// Mine the block
	for {
		block.Hash = calculateHash(block)
		if len(block.Hash) >= bc.Difficulty && block.Hash[:bc.Difficulty] == "0000" {
			break
		}
		block.Nonce++
	}

	accepted, reward, err := pool.SubmitWork(minerID, block.Index, block.Nonce, block.Hash)
	if err != nil {
		t.Fatalf("SubmitWork failed: %v", err)
	}

	if !accepted {
		t.Error("Valid work should be accepted")
	}

	if reward != 50 {
		t.Errorf("Expected reward 50, got %d", reward)
	}

	// Verify miner stats were updated
	pool.mu.RLock()
	miner := pool.miners[minerID]
	pool.mu.RUnlock()

	if miner.BlocksMined != 1 {
		t.Errorf("Expected 1 block mined, got %d", miner.BlocksMined)
	}

	// Verify blockchain was updated
	if bc.GetBlockCount() != 2 {
		t.Errorf("Expected blockchain height 2, got %d", bc.GetBlockCount())
	}
}

func TestSubmitWorkUnregisteredMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	_, _, err := pool.SubmitWork("unregistered-miner", 1, 0, "hash")
	if err == nil {
		t.Error("SubmitWork should fail for unregistered miner")
	}
}

func TestSubmitWorkNoPendingWork(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	_, _, err := pool.SubmitWork(minerID, 1, 0, "hash")
	if err == nil {
		t.Error("SubmitWork should fail when there's no pending work")
	}
}

func TestSubmitWorkStaleBlock(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID1 := "miner-1"
	minerID2 := "miner-2"
	pool.RegisterMiner(minerID1, "192.168.1.100", "host1", "192.168.1.100")
	pool.RegisterMiner(minerID2, "192.168.1.101", "host2", "192.168.1.101")

	block1, _ := pool.GetWork(minerID1)
	block2, _ := pool.GetWork(minerID2)

	// Miner 1 mines and submits first
	for {
		block1.Hash = calculateHash(block1)
		if len(block1.Hash) >= bc.Difficulty && block1.Hash[:bc.Difficulty] == "0000" {
			break
		}
		block1.Nonce++
	}

	pool.SubmitWork(minerID1, block1.Index, block1.Nonce, block1.Hash)

	// Miner 2 tries to submit for the same block (now stale)
	for {
		block2.Hash = calculateHash(block2)
		if len(block2.Hash) >= bc.Difficulty && block2.Hash[:bc.Difficulty] == "0000" {
			break
		}
		block2.Nonce++
	}

	_, _, err := pool.SubmitWork(minerID2, block2.Index, block2.Nonce, block2.Hash)
	if err == nil {
		t.Error("Submitting stale block should fail")
	}
}

func TestUpdateHeartbeat(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	hashRate := int64(1000000)
	cpuUsage := 75.5
	totalHashes := int64(5000000)
	miningTime := 30 * time.Second

	err := pool.UpdateHeartbeat(minerID, hashRate, cpuUsage, totalHashes, miningTime)
	if err != nil {
		t.Fatalf("UpdateHeartbeat failed: %v", err)
	}

	pool.mu.RLock()
	miner := pool.miners[minerID]
	pool.mu.RUnlock()

	if miner.HashRate != hashRate {
		t.Errorf("Expected hash rate %d, got %d", hashRate, miner.HashRate)
	}

	if miner.CPUUsagePercent != cpuUsage {
		t.Errorf("Expected CPU usage %.2f, got %.2f", cpuUsage, miner.CPUUsagePercent)
	}

	if miner.TotalHashes != totalHashes {
		t.Errorf("Expected total hashes %d, got %d", totalHashes, miner.TotalHashes)
	}

	if miner.TotalMiningTime != miningTime {
		t.Errorf("Expected mining time %v, got %v", miningTime, miner.TotalMiningTime)
	}
}

func TestUpdateHeartbeatWithGPU(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	gpuDevices := []GPUDeviceInfo{
		{
			ID:           0,
			Name:         "NVIDIA RTX 3080",
			Type:         "CUDA",
			Memory:       10737418240,
			ComputeUnits: 68,
			Available:    true,
		},
	}

	err := pool.UpdateHeartbeatWithGPU(minerID, 1000000, 50.0, 5000000, 30*time.Second, gpuDevices, 5000000, true, true)
	if err != nil {
		t.Fatalf("UpdateHeartbeatWithGPU failed: %v", err)
	}

	pool.mu.RLock()
	miner := pool.miners[minerID]
	pool.mu.RUnlock()

	if !miner.GPUEnabled {
		t.Error("GPU should be enabled")
	}

	if !miner.HybridMode {
		t.Error("Hybrid mode should be enabled")
	}

	if len(miner.GPUDevices) != 1 {
		t.Errorf("Expected 1 GPU device, got %d", len(miner.GPUDevices))
	}

	if miner.GPUHashRate != 5000000 {
		t.Errorf("Expected GPU hash rate 5000000, got %d", miner.GPUHashRate)
	}
}

func TestStopMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	// Simulate some blocks mined
	pool.mu.Lock()
	pool.miners[minerID].BlocksMined = 5
	pool.mu.Unlock()

	blocksMined, err := pool.StopMiner(minerID)
	if err != nil {
		t.Fatalf("StopMiner failed: %v", err)
	}

	if blocksMined != 5 {
		t.Errorf("Expected 5 blocks mined, got %d", blocksMined)
	}

	pool.mu.RLock()
	miner := pool.miners[minerID]
	pool.mu.RUnlock()

	if miner.Active {
		t.Error("Miner should be inactive after stopping")
	}
}

func TestGetMiners(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	pool.RegisterMiner("miner-1", "192.168.1.100", "host1", "192.168.1.100")
	pool.RegisterMiner("miner-2", "192.168.1.101", "host2", "192.168.1.101")
	pool.RegisterMiner("miner-3", "192.168.1.102", "host3", "192.168.1.102")

	miners := pool.GetMiners()
	if len(miners) != 3 {
		t.Errorf("Expected 3 miners, got %d", len(miners))
	}
}

func TestGetActiveMinerCount(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	pool.RegisterMiner("miner-1", "192.168.1.100", "host1", "192.168.1.100")
	pool.RegisterMiner("miner-2", "192.168.1.101", "host2", "192.168.1.101")
	pool.RegisterMiner("miner-3", "192.168.1.102", "host3", "192.168.1.102")

	count := pool.GetActiveMinerCount()
	if count != 3 {
		t.Errorf("Expected 3 active miners, got %d", count)
	}

	// Make one inactive by setting old heartbeat
	pool.mu.Lock()
	pool.miners["miner-2"].LastHeartbeat = time.Now().Add(-5 * time.Minute)
	pool.mu.Unlock()

	count = pool.GetActiveMinerCount()
	if count != 2 {
		t.Errorf("Expected 2 active miners (one stale), got %d", count)
	}
}

func TestGetPoolStats(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	pool.RegisterMiner("miner-1", "192.168.1.100", "host1", "192.168.1.100")
	pool.UpdateHeartbeat("miner-1", 1000000, 50.0, 5000000, 30*time.Second)

	stats := pool.GetPoolStats()

	if stats.TotalMiners != 1 {
		t.Errorf("Expected 1 total miner, got %d", stats.TotalMiners)
	}

	if stats.ActiveMiners != 1 {
		t.Errorf("Expected 1 active miner, got %d", stats.ActiveMiners)
	}

	if stats.TotalHashRate != 1000000 {
		t.Errorf("Expected total hash rate 1000000, got %d", stats.TotalHashRate)
	}

	if stats.BlockReward != 50 {
		t.Errorf("Expected block reward 50, got %d", stats.BlockReward)
	}
}

func TestPauseMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	err := pool.PauseMiner(minerID)
	if err != nil {
		t.Fatalf("PauseMiner failed: %v", err)
	}

	shouldMine, _ := pool.GetMinerStatus(minerID)
	if shouldMine {
		t.Error("Miner should not be mining after pause")
	}
}

func TestResumeMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	pool.PauseMiner(minerID)
	err := pool.ResumeMiner(minerID)
	if err != nil {
		t.Fatalf("ResumeMiner failed: %v", err)
	}

	shouldMine, _ := pool.GetMinerStatus(minerID)
	if !shouldMine {
		t.Error("Miner should be mining after resume")
	}
}

func TestDeleteMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	err := pool.DeleteMiner(minerID)
	if err != nil {
		t.Fatalf("DeleteMiner failed: %v", err)
	}

	pool.mu.RLock()
	_, exists := pool.miners[minerID]
	pool.mu.RUnlock()

	if exists {
		t.Error("Miner should be deleted from pool")
	}

	// Verify can't get status of deleted miner
	_, err = pool.GetMinerStatus(minerID)
	if err == nil {
		t.Error("GetMinerStatus should fail for deleted miner")
	}
}

func TestSetCPUThrottle(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	err := pool.SetCPUThrottle(minerID, 50)
	if err != nil {
		t.Fatalf("SetCPUThrottle failed: %v", err)
	}

	throttle, _ := pool.GetCPUThrottle(minerID)
	if throttle != 50 {
		t.Errorf("Expected throttle 50, got %d", throttle)
	}
}

func TestSetCPUThrottleInvalidValue(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	err := pool.SetCPUThrottle(minerID, 150)
	if err == nil {
		t.Error("SetCPUThrottle should fail for value > 100")
	}

	err = pool.SetCPUThrottle(minerID, -10)
	if err == nil {
		t.Error("SetCPUThrottle should fail for negative value")
	}
}

func TestGetCPUStats(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	pool.RegisterMiner("miner-1", "192.168.1.100", "host1", "192.168.1.100")
	pool.UpdateHeartbeat("miner-1", 1000000, 50.0, 5000000, 30*time.Second)

	gpuDevices := []GPUDeviceInfo{
		{
			ID:           0,
			Name:         "Test GPU",
			Type:         "CUDA",
			Memory:       8589934592,
			ComputeUnits: 32,
			Available:    true,
		},
	}
	pool.UpdateHeartbeatWithGPU("miner-1", 1000000, 50.0, 5000000, 30*time.Second, gpuDevices, 2000000, true, true)

	stats := pool.GetCPUStats()

	if stats.TotalMiners != 1 {
		t.Errorf("Expected 1 total miner, got %d", stats.TotalMiners)
	}

	if stats.ActiveMiners != 1 {
		t.Errorf("Expected 1 active miner, got %d", stats.ActiveMiners)
	}

	if stats.GPUEnabledMiners != 1 {
		t.Errorf("Expected 1 GPU-enabled miner, got %d", stats.GPUEnabledMiners)
	}

	if stats.HybridMiners != 1 {
		t.Errorf("Expected 1 hybrid miner, got %d", stats.HybridMiners)
	}

	if len(stats.MinerStats) != 1 {
		t.Errorf("Expected 1 miner stat entry, got %d", len(stats.MinerStats))
	}

	if len(stats.MinerStats[0].GPUDevices) != 1 {
		t.Errorf("Expected 1 GPU device in stats, got %d", len(stats.MinerStats[0].GPUDevices))
	}
}

func TestConcurrentMinerOperations(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)

	// Register multiple miners concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			minerID := "miner-" + string(rune('0'+id))
			pool.RegisterMiner(minerID, "192.168.1.100", "host", "192.168.1.100")
			pool.UpdateHeartbeat(minerID, 1000000, 50.0, 5000000, 30*time.Second)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all operations completed successfully
	miners := pool.GetMiners()
	if len(miners) < 1 {
		t.Error("Concurrent operations should register at least some miners")
	}
}
