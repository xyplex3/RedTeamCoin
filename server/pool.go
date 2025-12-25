// Package main implements the RedTeamCoin mining pool server components.
package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// GPUDeviceInfo contains hardware information about a GPU device available
// for mining. Miners report their GPU capabilities during registration and
// heartbeats.
type GPUDeviceInfo struct {
	ID           int    // Unique device identifier
	Name         string // Device name from hardware
	Type         string // Device type: "CUDA" or "OpenCL"
	Memory       uint64 // Total device memory in bytes
	ComputeUnits int    // Number of compute units/SMs
	Available    bool   // Whether device is currently available
}

// PoolStats contains aggregated statistics about the mining pool's
// performance and state. This data is exposed via the API for monitoring.
type PoolStats struct {
	TotalMiners      int     `json:"total_miners"`
	ActiveMiners     int     `json:"active_miners"`
	TotalHashRate    int64   `json:"total_hash_rate"`
	TotalBlocksMined int64   `json:"total_blocks_mined"`
	TotalHashes      int64   `json:"total_hashes"`
	TotalMiningTime  float64 `json:"total_mining_time"`
	AvgCPUUsage      float64 `json:"avg_cpu_usage"`
	TotalCPUUsage    float64 `json:"total_cpu_usage"`
	BlockchainHeight int     `json:"blockchain_height"`
	Difficulty       int32   `json:"difficulty"`
	BlockReward      int64   `json:"block_reward"`
}

// MinerRecord tracks the state and statistics of a connected miner.
// All fields are protected by the MiningPool's mutex for thread-safe access.
//
// The server can control miner behavior through ShouldMine and
// CPUThrottlePercent fields, which are communicated during heartbeats.
type MinerRecord struct {
	ID                 string
	IPAddress          string // Client-reported IP address
	IPAddressActual    string // Server-detected actual IP address
	Hostname           string // Client-reported hostname
	RegisteredAt       time.Time
	LastHeartbeat      time.Time
	Active             bool
	ShouldMine         bool  // Server control: whether miner should mine
	CPUThrottlePercent int32 // CPU usage limit (0-100), 0 = no limit
	BlocksMined        int64
	HashRate           int64
	TotalMiningTime    time.Duration   // Total time spent mining
	CPUUsagePercent    float64         // Current CPU usage percentage
	TotalHashes        int64           // Total hashes computed
	GPUDevices         []GPUDeviceInfo // GPU devices available to this miner
	GPUHashRate        int64           // Hash rate from GPU mining
	GPUEnabled         bool            // Whether GPU mining is enabled
	HybridMode         bool            // Whether hybrid CPU+GPU mining is enabled
}

// PendingWork tracks a block assignment to a specific miner.
// Work is considered stale if not submitted within 5 minutes.
type PendingWork struct {
	MinerID    string
	Block      *Block
	AssignedAt time.Time
}

// MiningPool coordinates work distribution among connected miners and
// validates submitted blocks. All methods are safe for concurrent use by
// multiple goroutines.
//
// The pool automatically generates new work blocks and assigns them to miners
// on demand. When a miner successfully mines a block, the pool validates and
// adds it to the blockchain, then rewards the miner. The pool supports graceful
// shutdown via the Shutdown method.
type MiningPool struct {
	blockchain  *Blockchain
	miners      map[string]*MinerRecord
	pendingWork map[string]*PendingWork
	mu          sync.RWMutex
	workQueue   chan *Block
	blockReward int64
	logger      *PoolLogger
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewMiningPool creates a new mining pool that coordinates work distribution
// for the given blockchain. The pool starts a background goroutine to
// generate new work blocks every 30 seconds. Use Shutdown() for graceful
// termination of background goroutines.
//
// Goroutine Lifecycle: Starts 1 background goroutine (work generator)
// that runs until Shutdown() is called.
func NewMiningPool(blockchain *Blockchain) *MiningPool {
	ctx, cancel := context.WithCancel(context.Background())
	pool := &MiningPool{
		blockchain:  blockchain,
		miners:      make(map[string]*MinerRecord),
		pendingWork: make(map[string]*PendingWork),
		workQueue:   make(chan *Block, 100),
		blockReward: 50,
		logger:      nil, // Will be set after creation
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start work generator with context for lifecycle management
	go pool.generateWork(ctx)

	return pool
}

// SetLogger assigns a logger to the mining pool for event tracking and
// periodic statistics logging. This must be called before miners start
// connecting to ensure events are properly logged.
func (mp *MiningPool) SetLogger(logger *PoolLogger) {
	mp.logger = logger
}

// Shutdown gracefully shuts down the mining pool by cancelling all background
// goroutines. This stops the work generator and allows for clean shutdown of
// the pool. This method is safe for concurrent use.
func (mp *MiningPool) Shutdown() {
	mp.cancel()
}

// RegisterMiner registers a new miner in the pool or reactivates an existing
// one. It records both the client-reported IP address and the actual IP
// detected by the server. This method is safe for concurrent use.
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
		ID:                 id,
		IPAddress:          ipAddress,
		IPAddressActual:    actualIP,
		Hostname:           hostname,
		RegisteredAt:       time.Now(),
		LastHeartbeat:      time.Now(),
		Active:             true,
		ShouldMine:         true, // Mining enabled by default
		CPUThrottlePercent: 0,    // No throttling by default (0 = unlimited)
		BlocksMined:        0,
		HashRate:           0,
		TotalMiningTime:    0,
		CPUUsagePercent:    0,
		TotalHashes:        0,
	}

	fmt.Printf("Miner registered: %s (Reported IP: %s, Actual IP: %s, Hostname: %s)\n", id, ipAddress, actualIP, hostname)

	// Log the event
	if mp.logger != nil {
		mp.logger.LogEvent("miner_registered", "New miner registered", id, map[string]interface{}{
			"ip_address":        ipAddress,
			"ip_address_actual": actualIP,
			"hostname":          hostname,
		})
	}

	return nil
}

// GetWork assigns a mining work unit to the specified miner.
// It returns existing pending work if less than 5 minutes old, otherwise
// assigns new work from the queue or generates it. This method is safe for
// concurrent use.
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

// SubmitWork validates and processes a block solution submitted by a miner.
// It verifies the solution matches pending work, validates the proof-of-work,
// adds the block to the blockchain if valid, and returns the block reward.
// Stale blocks (superseded by another miner) are rejected with an error.
// Returns true and the reward amount if accepted, false otherwise. This
// method is safe for concurrent use.
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

	// Log the event
	if mp.logger != nil {
		mp.logger.LogEvent("block_mined", "Block successfully mined", minerID, map[string]interface{}{
			"block_index": blockIndex,
			"hash":        hash,
			"reward":      mp.blockReward,
			"nonce":       nonce,
		})
	}

	return true, mp.blockReward, nil
}

// UpdateHeartbeat updates a miner's last-seen timestamp and performance
// statistics. It records the current hash rate, CPU usage percentage,
// total hashes computed, and cumulative mining time. Returns an error if
// the miner is not registered. This method is safe for concurrent use.
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

// UpdateHeartbeatWithGPU updates a miner's heartbeat with both CPU and GPU
// statistics. In addition to standard heartbeat data (hash rate, CPU usage,
// total hashes, mining time), this method records GPU device information,
// GPU hash rate, and whether GPU or hybrid mining is enabled. Returns an
// error if the miner is not registered. This method is safe for concurrent
// use.
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

// StopMiner marks a miner as inactive and removes its pending work
// assignments. It returns the total number of blocks the miner successfully
// mined during its session. Returns an error if the miner is not registered.
// This method is safe for concurrent use.
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

// GetMiners returns a slice containing all registered miners in the pool,
// both active and inactive. The returned slice is a new allocation and can
// be safely modified by the caller. This method is safe for concurrent use.
func (mp *MiningPool) GetMiners() []*MinerRecord {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	miners := make([]*MinerRecord, 0, len(mp.miners))
	for _, miner := range mp.miners {
		miners = append(miners, miner)
	}
	return miners
}

// GetActiveMinerCount returns the number of miners that are currently
// active and have sent a heartbeat within the last 2 minutes. Miners that
// have not communicated recently are not counted as active. This method is
// safe for concurrent use.
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

// generateWork continuously creates new work blocks every 30 seconds and
// adds them to the work queue. This function runs in a background goroutine
// that respects context cancellation for graceful shutdown. It uses a
// non-blocking send to avoid stalling if the queue is full.
func (mp *MiningPool) generateWork(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
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
}

// GetPoolStats returns aggregated statistics for the entire mining pool,
// including total and active miner counts, combined hash rates, total blocks
// mined, average CPU usage, blockchain height, difficulty, and block reward.
// Active miners are those with heartbeats within the last 2 minutes. This
// method is safe for concurrent use.
func (mp *MiningPool) GetPoolStats() PoolStats {
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

	return PoolStats{
		TotalMiners:      len(mp.miners),
		ActiveMiners:     activeMiners,
		TotalHashRate:    totalHashRate,
		TotalBlocksMined: totalBlocksMined,
		TotalHashes:      totalHashes,
		TotalMiningTime:  totalMiningTime.Seconds(),
		AvgCPUUsage:      avgCPU,
		TotalCPUUsage:    totalCPUUsage,
		BlockchainHeight: mp.blockchain.GetBlockCount(),
		Difficulty:       mp.blockchain.Difficulty,
		BlockReward:      mp.blockReward,
	}
}

// GPUDeviceStats represents GPU device statistics for API responses,
// providing a JSON-serializable view of GPU hardware information including
// device identification, memory, and compute capabilities.
type GPUDeviceStats struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	MemoryMB     uint64 `json:"memory_mb"`
	ComputeUnits int    `json:"compute_units"`
	Available    bool   `json:"available"`
}

// CPUStats represents detailed performance and resource usage statistics
// for a single miner, including CPU usage, hash rates, mining time, and
// GPU information if applicable. This structure is used for API responses
// and monitoring dashboards.
type CPUStats struct {
	MinerID         string           `json:"miner_id"`
	IPAddress       string           `json:"ip_address"`
	IPAddressActual string           `json:"ip_address_actual"`
	Hostname        string           `json:"hostname"`
	CPUUsagePercent float64          `json:"cpu_usage_percent"`
	TotalHashes     int64            `json:"total_hashes"`
	MiningTimeHours float64          `json:"mining_time_hours"`
	MiningTimeSec   float64          `json:"mining_time_seconds"`
	HashRate        int64            `json:"hash_rate"`
	Active          bool             `json:"active"`
	RegisteredAt    string           `json:"registered_at"`
	GPUDevices      []GPUDeviceStats `json:"gpu_devices,omitempty"`
	GPUHashRate     int64            `json:"gpu_hash_rate,omitempty"`
	GPUEnabled      bool             `json:"gpu_enabled"`
	HybridMode      bool             `json:"hybrid_mode"`
}

// TotalCPUStats represents aggregate CPU and GPU statistics
type TotalCPUStats struct {
	TotalMiners      int        `json:"total_miners"`
	ActiveMiners     int        `json:"active_miners"`
	TotalCPUUsage    float64    `json:"total_cpu_usage_percent"`
	AverageCPUUsage  float64    `json:"average_cpu_usage_percent"`
	TotalHashes      int64      `json:"total_hashes"`
	TotalMiningHours float64    `json:"total_mining_hours"`
	TotalMiningTime  float64    `json:"total_mining_seconds"`
	TotalHashRate    int64      `json:"total_hash_rate"`
	TotalGPUHashRate int64      `json:"total_gpu_hash_rate"`
	GPUEnabledMiners int        `json:"gpu_enabled_miners"`
	HybridMiners     int        `json:"hybrid_miners"`
	MinerStats       []CPUStats `json:"miner_stats"`
}

// PauseMiner sets the ShouldMine flag to false for the specified miner,
// instructing it to stop mining operations on its next heartbeat check.
// The miner remains registered in the pool and can be resumed later.
// Returns an error if the miner ID is not found.
func (mp *MiningPool) PauseMiner(minerID string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	miner, exists := mp.miners[minerID]
	if !exists {
		return fmt.Errorf("miner not found")
	}

	miner.ShouldMine = false
	fmt.Printf("Miner paused: %s\n", minerID)

	// Log the event
	if mp.logger != nil {
		mp.logger.LogEvent("miner_paused", "Miner mining paused by server", minerID, nil)
	}

	return nil
}

// ResumeMiner sets the ShouldMine flag to true for the specified miner,
// instructing it to resume mining operations on its next heartbeat check.
// This reverses the effect of PauseMiner. Returns an error if the miner
// ID is not found.
func (mp *MiningPool) ResumeMiner(minerID string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	miner, exists := mp.miners[minerID]
	if !exists {
		return fmt.Errorf("miner not found")
	}

	miner.ShouldMine = true
	fmt.Printf("Miner resumed: %s\n", minerID)

	// Log the event
	if mp.logger != nil {
		mp.logger.LogEvent("miner_resumed", "Miner mining resumed by server", minerID, nil)
	}

	return nil
}

// DeleteMiner permanently removes a miner from the pool, clearing all its
// statistics and pending work assignments. The miner will need to
// re-register to join the pool again. Returns an error if the miner ID
// is not found.
func (mp *MiningPool) DeleteMiner(minerID string) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	miner, exists := mp.miners[minerID]
	if !exists {
		return fmt.Errorf("miner not found")
	}

	// Remove pending work
	delete(mp.pendingWork, minerID)

	// Log the event before deletion
	if mp.logger != nil {
		mp.logger.LogEvent("miner_deleted", "Miner deleted from pool", minerID, map[string]interface{}{
			"total_blocks_mined": miner.BlocksMined,
			"total_hashes":       miner.TotalHashes,
			"ip_address":         miner.IPAddress,
			"hostname":           miner.Hostname,
		})
	}

	// Remove miner
	delete(mp.miners, minerID)

	fmt.Printf("Miner deleted: %s (Total blocks mined: %d)\n", minerID, miner.BlocksMined)
	return nil
}

// GetMinerStatus returns the ShouldMine flag value for the specified
// miner, indicating whether the miner should currently be mining. Returns
// true if mining is enabled, false if paused. Returns an error if the
// miner ID is not found.
func (mp *MiningPool) GetMinerStatus(minerID string) (bool, error) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	miner, exists := mp.miners[minerID]
	if !exists {
		return false, fmt.Errorf("miner not found")
	}

	return miner.ShouldMine, nil
}

// SetCPUThrottle configures CPU usage limits for the specified miner.
// The throttlePercent parameter must be between 0 and 100, where 0 means
// no CPU throttling (unlimited usage) and higher values increase
// throttling intensity. The new throttle setting takes effect on the
// miner's next heartbeat check. Returns an error if the miner is not
// found or if throttlePercent is outside the valid range.
func (mp *MiningPool) SetCPUThrottle(minerID string, throttlePercent int32) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	miner, exists := mp.miners[minerID]
	if !exists {
		return fmt.Errorf("miner not found")
	}

	// Validate throttle percentage
	if throttlePercent < 0 || throttlePercent > 100 {
		return fmt.Errorf("throttle percentage must be between 0 and 100")
	}

	miner.CPUThrottlePercent = throttlePercent
	fmt.Printf("Miner %s CPU throttle set to %d%%\n", minerID, throttlePercent)

	// Log the event
	if mp.logger != nil {
		mp.logger.LogEvent("miner_throttled", "CPU throttle set for miner", minerID, map[string]interface{}{
			"throttle_percent": throttlePercent,
		})
	}

	return nil
}

// GetCPUThrottle returns the current CPU throttle setting (0-100) for
// the specified miner, where 0 indicates no throttling. Returns an error
// if the miner ID is not found.
func (mp *MiningPool) GetCPUThrottle(minerID string) (int32, error) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()

	miner, exists := mp.miners[minerID]
	if !exists {
		return 0, fmt.Errorf("miner not found")
	}

	return miner.CPUThrottlePercent, nil
}

// GetCPUStats returns comprehensive CPU and GPU usage statistics for all
// miners in the pool. It aggregates data including total miners, active
// count, hash rates, CPU usage, and GPU statistics. The returned structure
// includes both aggregate totals and per-miner breakdowns. This method is
// safe for concurrent use.
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
