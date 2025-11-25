package main

import (
	"fmt"
	"sync"
	"time"
)

// GPUDeviceInfo represents information about a GPU device
type GPUDeviceInfo struct {
	ID           int
	Name         string
	Type         string // "CUDA" or "OpenCL"
	Memory       uint64
	ComputeUnits int
	Available    bool
}

// MinerRecord represents information about a connected miner
type MinerRecord struct {
	ID                 string
	IPAddress          string        // Client-reported IP address
	IPAddressActual    string        // Server-detected actual IP address
	Hostname           string        // Client-reported hostname
	RegisteredAt       time.Time
	LastHeartbeat      time.Time
	Active             bool
	BlocksMined        int64
	HashRate           int64
	TotalMiningTime    time.Duration      // Total time spent mining
	CPUUsagePercent    float64            // Current CPU usage percentage
	TotalHashes        int64              // Total hashes computed
	GPUDevices         []GPUDeviceInfo    // GPU devices available to this miner
	GPUHashRate        int64              // Hash rate from GPU mining
	GPUEnabled         bool               // Whether GPU mining is enabled
	HybridMode         bool               // Whether hybrid CPU+GPU mining is enabled
}

// PendingWork represents work assigned to a miner
type PendingWork struct {
	MinerID       string
	Block         *Block
	AssignedAt    time.Time
}

// MiningPool manages the pool of miners and work distribution
type MiningPool struct {
	blockchain    *Blockchain
	miners        map[string]*MinerRecord
	pendingWork   map[string]*PendingWork
	mu            sync.RWMutex
	workQueue     chan *Block
	blockReward   int64
}

// NewMiningPool creates a new mining pool
func NewMiningPool(blockchain *Blockchain) *MiningPool {
	pool := &MiningPool{
		blockchain:  blockchain,
		miners:      make(map[string]*MinerRecord),
		pendingWork: make(map[string]*PendingWork),
		workQueue:   make(chan *Block, 100),
		blockReward: 50,
	}

	// Start work generator
	go pool.generateWork()

	return pool
}

// RegisterMiner registers a new miner in the pool
func (mp *MiningPool) RegisterMiner(id, ipAddress, hostname, actualIP string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if _, exists := mp.miners[id]; exists {
		// Update existing miner
		mp.miners[id].Active = true
		mp.miners[id].LastHeartbeat = time.Now()
		mp.miners[id].IPAddressActual = actualIP // Update actual IP in case it changed
		return nil
	}

	mp.miners[id] = &MinerRecord{
		ID:              id,
		IPAddress:       ipAddress,
		IPAddressActual: actualIP,
		Hostname:        hostname,
		RegisteredAt:    time.Now(),
		LastHeartbeat:   time.Now(),
		Active:          true,
		BlocksMined:     0,
		HashRate:        0,
		TotalMiningTime: 0,
		CPUUsagePercent: 0,
		TotalHashes:     0,
	}

	fmt.Printf("Miner registered: %s (Reported IP: %s, Actual IP: %s, Hostname: %s)\n", id, ipAddress, actualIP, hostname)
	return nil
}

// GetWork assigns work to a miner
func (mp *MiningPool) GetWork(minerID string) (*Block, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	miner, exists := mp.miners[minerID]
	if !exists {
		return nil, fmt.Errorf("miner not registered")
	}

	miner.LastHeartbeat = time.Now()

	// Check if miner already has pending work
	if work, hasPending := mp.pendingWork[minerID]; hasPending {
		// If work is less than 5 minutes old, return it
		if time.Since(work.AssignedAt) < 5*time.Minute {
			return work.Block, nil
		}
	}

	// Get new work from queue
	select {
	case block := <-mp.workQueue:
		mp.pendingWork[minerID] = &PendingWork{
			MinerID:    minerID,
			Block:      block,
			AssignedAt: time.Now(),
		}
		return block, nil
	default:
		// No work available, create new work
		latest := mp.blockchain.GetLatestBlock()
		newBlock := &Block{
			Index:        latest.Index + 1,
			Timestamp:    time.Now().Unix(),
			Data:         fmt.Sprintf("Block data %d", latest.Index+1),
			PreviousHash: latest.Hash,
			Nonce:        0,
		}

		mp.pendingWork[minerID] = &PendingWork{
			MinerID:    minerID,
			Block:      newBlock,
			AssignedAt: time.Now(),
		}
		return newBlock, nil
	}
}

// SubmitWork processes a submitted solution
func (mp *MiningPool) SubmitWork(minerID string, blockIndex int64, nonce int64, hash string) (bool, int64, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	miner, exists := mp.miners[minerID]
	if !exists {
		return false, 0, fmt.Errorf("miner not registered")
	}

	// Get pending work
	work, hasPending := mp.pendingWork[minerID]
	if !hasPending {
		return false, 0, fmt.Errorf("no pending work for miner")
	}

	if work.Block.Index != blockIndex {
		return false, 0, fmt.Errorf("block index mismatch")
	}

	// Verify the solution
	work.Block.Nonce = nonce
	work.Block.Hash = hash
	work.Block.MinedBy = minerID

	// Validate the block
	latest := mp.blockchain.GetLatestBlock()
	if blockIndex != latest.Index+1 {
		// Block is stale, another miner was faster
		delete(mp.pendingWork, minerID)
		return false, 0, fmt.Errorf("block is stale")
	}

	// Add block to blockchain
	err := mp.blockchain.AddBlock(work.Block)
	if err != nil {
		delete(mp.pendingWork, minerID)
		return false, 0, err
	}

	// Update miner stats
	miner.BlocksMined++
	miner.LastHeartbeat = time.Now()

	// Clear pending work
	delete(mp.pendingWork, minerID)

	// Clear other miners' pending work for same block height
	for mid, pw := range mp.pendingWork {
		if pw.Block.Index == blockIndex {
			delete(mp.pendingWork, mid)
		}
	}

	fmt.Printf("Block %d mined by %s (Hash: %s)\n", blockIndex, minerID, hash)
	return true, mp.blockReward, nil
}

// UpdateHeartbeat updates miner heartbeat and statistics
func (mp *MiningPool) UpdateHeartbeat(minerID string, hashRate int64, cpuUsage float64, totalHashes int64, miningTime time.Duration) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	miner, exists := mp.miners[minerID]
	if !exists {
		return fmt.Errorf("miner not registered")
	}

	miner.LastHeartbeat = time.Now()
	miner.HashRate = hashRate
	miner.CPUUsagePercent = cpuUsage
	miner.TotalHashes = totalHashes
	miner.TotalMiningTime = miningTime
	return nil
}

// UpdateHeartbeatWithGPU updates miner heartbeat with GPU statistics
func (mp *MiningPool) UpdateHeartbeatWithGPU(minerID string, hashRate int64, cpuUsage float64, totalHashes int64, miningTime time.Duration, gpuDevices []GPUDeviceInfo, gpuHashRate int64, gpuEnabled bool, hybridMode bool) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	miner, exists := mp.miners[minerID]
	if !exists {
		return fmt.Errorf("miner not registered")
	}

	miner.LastHeartbeat = time.Now()
	miner.HashRate = hashRate
	miner.CPUUsagePercent = cpuUsage
	miner.TotalHashes = totalHashes
	miner.TotalMiningTime = miningTime
	miner.GPUDevices = gpuDevices
	miner.GPUHashRate = gpuHashRate
	miner.GPUEnabled = gpuEnabled
	miner.HybridMode = hybridMode
	return nil
}

// StopMiner stops a miner
func (mp *MiningPool) StopMiner(minerID string) (int64, error) {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	miner, exists := mp.miners[minerID]
	if !exists {
		return 0, fmt.Errorf("miner not registered")
	}

	miner.Active = false
	blocksMined := miner.BlocksMined

	// Remove pending work
	delete(mp.pendingWork, minerID)

	fmt.Printf("Miner stopped: %s (Total blocks mined: %d)\n", minerID, blocksMined)
	return blocksMined, nil
}

// GetMiners returns all registered miners
func (mp *MiningPool) GetMiners() []*MinerRecord {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	miners := make([]*MinerRecord, 0, len(mp.miners))
	for _, miner := range mp.miners {
		miners = append(miners, miner)
	}
	return miners
}

// GetActiveMinerCount returns the number of active miners
func (mp *MiningPool) GetActiveMinerCount() int {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	count := 0
	for _, miner := range mp.miners {
		if miner.Active && time.Since(miner.LastHeartbeat) < 2*time.Minute {
			count++
		}
	}
	return count
}

// generateWork continuously generates new work blocks
func (mp *MiningPool) generateWork() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		latest := mp.blockchain.GetLatestBlock()
		newBlock := &Block{
			Index:        latest.Index + 1,
			Timestamp:    time.Now().Unix(),
			Data:         fmt.Sprintf("Block data %d", latest.Index+1),
			PreviousHash: latest.Hash,
			Nonce:        0,
		}

		// Try to add to queue, don't block if full
		select {
		case mp.workQueue <- newBlock:
		default:
		}
	}
}

// GetPoolStats returns pool statistics
func (mp *MiningPool) GetPoolStats() map[string]interface{} {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	totalHashRate := int64(0)
	activeMiners := 0
	totalBlocksMined := int64(0)
	totalCPUUsage := 0.0
	totalHashes := int64(0)
	totalMiningTime := time.Duration(0)

	for _, miner := range mp.miners {
		if miner.Active && time.Since(miner.LastHeartbeat) < 2*time.Minute {
			activeMiners++
			totalHashRate += miner.HashRate
			totalCPUUsage += miner.CPUUsagePercent
		}
		totalBlocksMined += miner.BlocksMined
		totalHashes += miner.TotalHashes
		totalMiningTime += miner.TotalMiningTime
	}

	// Calculate average CPU usage for active miners
	avgCPU := 0.0
	if activeMiners > 0 {
		avgCPU = totalCPUUsage / float64(activeMiners)
	}

	return map[string]interface{}{
		"total_miners":        len(mp.miners),
		"active_miners":       activeMiners,
		"total_hash_rate":     totalHashRate,
		"total_blocks_mined":  totalBlocksMined,
		"total_hashes":        totalHashes,
		"total_mining_time":   totalMiningTime.Seconds(),
		"avg_cpu_usage":       avgCPU,
		"total_cpu_usage":     totalCPUUsage,
		"blockchain_height":   mp.blockchain.GetBlockCount(),
		"difficulty":          mp.blockchain.Difficulty,
		"block_reward":        mp.blockReward,
	}
}

// CPUStats represents CPU usage statistics for a miner
// GPUDeviceStats represents GPU device statistics for API
type GPUDeviceStats struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	MemoryMB     uint64 `json:"memory_mb"`
	ComputeUnits int    `json:"compute_units"`
	Available    bool   `json:"available"`
}

type CPUStats struct {
	MinerID         string            `json:"miner_id"`
	IPAddress       string            `json:"ip_address"`
	IPAddressActual string            `json:"ip_address_actual"`
	Hostname        string            `json:"hostname"`
	CPUUsagePercent float64           `json:"cpu_usage_percent"`
	TotalHashes     int64             `json:"total_hashes"`
	MiningTimeHours float64           `json:"mining_time_hours"`
	MiningTimeSec   float64           `json:"mining_time_seconds"`
	HashRate        int64             `json:"hash_rate"`
	Active          bool              `json:"active"`
	RegisteredAt    string            `json:"registered_at"`
	GPUDevices      []GPUDeviceStats  `json:"gpu_devices,omitempty"`
	GPUHashRate     int64             `json:"gpu_hash_rate,omitempty"`
	GPUEnabled      bool              `json:"gpu_enabled"`
	HybridMode      bool              `json:"hybrid_mode"`
}

// TotalCPUStats represents aggregate CPU and GPU statistics
type TotalCPUStats struct {
	TotalMiners       int         `json:"total_miners"`
	ActiveMiners      int         `json:"active_miners"`
	TotalCPUUsage     float64     `json:"total_cpu_usage_percent"`
	AverageCPUUsage   float64     `json:"average_cpu_usage_percent"`
	TotalHashes       int64       `json:"total_hashes"`
	TotalMiningHours  float64     `json:"total_mining_hours"`
	TotalMiningTime   float64     `json:"total_mining_seconds"`
	TotalHashRate     int64       `json:"total_hash_rate"`
	TotalGPUHashRate  int64       `json:"total_gpu_hash_rate"`
	GPUEnabledMiners  int         `json:"gpu_enabled_miners"`
	HybridMiners      int         `json:"hybrid_miners"`
	MinerStats        []CPUStats  `json:"miner_stats"`
}

// GetCPUStats returns detailed CPU usage statistics
func (mp *MiningPool) GetCPUStats() *TotalCPUStats {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	stats := &TotalCPUStats{
		MinerStats: make([]CPUStats, 0),
	}

	totalCPU := 0.0
	totalHashes := int64(0)
	totalTime := time.Duration(0)
	totalHashRate := int64(0)
	totalGPUHashRate := int64(0)
	activeCount := 0
	gpuEnabledCount := 0
	hybridCount := 0

	for _, miner := range mp.miners {
		isActive := miner.Active && time.Since(miner.LastHeartbeat) < 2*time.Minute

		if isActive {
			activeCount++
			totalCPU += miner.CPUUsagePercent
			totalHashRate += miner.HashRate
			totalGPUHashRate += miner.GPUHashRate

			if miner.GPUEnabled {
				gpuEnabledCount++
			}
			if miner.HybridMode {
				hybridCount++
			}
		}

		totalHashes += miner.TotalHashes
		totalTime += miner.TotalMiningTime

		// Convert GPU devices to stats format
		var gpuDeviceStats []GPUDeviceStats
		for _, dev := range miner.GPUDevices {
			gpuDeviceStats = append(gpuDeviceStats, GPUDeviceStats{
				ID:           dev.ID,
				Name:         dev.Name,
				Type:         dev.Type,
				MemoryMB:     dev.Memory / 1024 / 1024,
				ComputeUnits: dev.ComputeUnits,
				Available:    dev.Available,
			})
		}

		minerStat := CPUStats{
			MinerID:         miner.ID,
			IPAddress:       miner.IPAddress,
			IPAddressActual: miner.IPAddressActual,
			Hostname:        miner.Hostname,
			CPUUsagePercent: miner.CPUUsagePercent,
			TotalHashes:     miner.TotalHashes,
			MiningTimeHours: miner.TotalMiningTime.Hours(),
			MiningTimeSec:   miner.TotalMiningTime.Seconds(),
			HashRate:        miner.HashRate,
			Active:          isActive,
			RegisteredAt:    miner.RegisteredAt.Format(time.RFC3339),
			GPUDevices:      gpuDeviceStats,
			GPUHashRate:     miner.GPUHashRate,
			GPUEnabled:      miner.GPUEnabled,
			HybridMode:      miner.HybridMode,
		}

		stats.MinerStats = append(stats.MinerStats, minerStat)
	}

	stats.TotalMiners = len(mp.miners)
	stats.ActiveMiners = activeCount
	stats.TotalCPUUsage = totalCPU
	stats.TotalHashes = totalHashes
	stats.TotalMiningHours = totalTime.Hours()
	stats.TotalMiningTime = totalTime.Seconds()
	stats.TotalHashRate = totalHashRate
	stats.TotalGPUHashRate = totalGPUHashRate
	stats.GPUEnabledMiners = gpuEnabledCount
	stats.HybridMiners = hybridCount

	if activeCount > 0 {
		stats.AverageCPUUsage = totalCPU / float64(activeCount)
	}

	return stats
}
