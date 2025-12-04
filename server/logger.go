package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	MinerID   string                 `json:"miner_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Message   string                 `json:"message"`
}

// MinerLogInfo represents miner information in the log
type MinerLogInfo struct {
	ID                 string    `json:"id"`
	IPAddress          string    `json:"ip_address"`
	IPAddressActual    string    `json:"ip_address_actual"`
	Hostname           string    `json:"hostname"`
	RegisteredAt       time.Time `json:"registered_at"`
	LastHeartbeat      time.Time `json:"last_heartbeat"`
	Active             bool      `json:"active"`
	ShouldMine         bool      `json:"should_mine"`
	CPUThrottlePercent int       `json:"cpu_throttle_percent"`
	BlocksMined        int64     `json:"blocks_mined"`
	HashRate           int64     `json:"hash_rate"`
	CPUUsagePercent    float64   `json:"cpu_usage_percent"`
	TotalHashes        int64     `json:"total_hashes"`
	TotalMiningTime    float64   `json:"total_mining_time_seconds"`
	GPUEnabled         bool      `json:"gpu_enabled"`
	GPUHashRate        int64     `json:"gpu_hash_rate,omitempty"`
	HybridMode         bool      `json:"hybrid_mode"`
}

// PoolSnapshot represents a snapshot of the pool state
type PoolSnapshot struct {
	Timestamp        time.Time      `json:"timestamp"`
	TotalMiners      int            `json:"total_miners"`
	ActiveMiners     int            `json:"active_miners"`
	TotalHashRate    int64          `json:"total_hash_rate"`
	TotalBlocksMined int64          `json:"total_blocks_mined"`
	BlockchainHeight int64          `json:"blockchain_height"`
	Difficulty       int            `json:"difficulty"`
	Miners           []MinerLogInfo `json:"miners"`
}

// LogFile represents the complete log file structure
type LogFile struct {
	ServerStartTime time.Time    `json:"server_start_time"`
	ServerUptime    float64      `json:"server_uptime_seconds"`
	LastUpdate      time.Time    `json:"last_update"`
	Events          []LogEntry   `json:"events"`
	CurrentSnapshot PoolSnapshot `json:"current_snapshot"`
}

// PoolLogger handles periodic logging of pool state to JSON file
type PoolLogger struct {
	pool           *MiningPool
	blockchain     *Blockchain
	logFile        string
	updateInterval time.Duration
	startTime      time.Time
	events         []LogEntry
	mu             sync.RWMutex
	maxEvents      int // Maximum number of events to keep in memory
}

// NewPoolLogger creates a new pool logger
func NewPoolLogger(pool *MiningPool, blockchain *Blockchain, logFile string, updateInterval time.Duration) *PoolLogger {
	return &PoolLogger{
		pool:           pool,
		blockchain:     blockchain,
		logFile:        logFile,
		updateInterval: updateInterval,
		startTime:      time.Now(),
		events:         make([]LogEntry, 0),
		maxEvents:      1000, // Keep last 1000 events
	}
}

// LogEvent adds an event to the log
func (pl *PoolLogger) LogEvent(eventType, message string, minerID string, details map[string]interface{}) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now(),
		EventType: eventType,
		MinerID:   minerID,
		Message:   message,
		Details:   details,
	}

	pl.events = append(pl.events, entry)

	// Keep only the last maxEvents
	if len(pl.events) > pl.maxEvents {
		pl.events = pl.events[len(pl.events)-pl.maxEvents:]
	}
}

// GetPoolSnapshot creates a snapshot of the current pool state
func (pl *PoolLogger) GetPoolSnapshot() PoolSnapshot {
	miners := pl.pool.GetMiners()
	stats := pl.pool.GetPoolStats()

	minerInfos := make([]MinerLogInfo, 0, len(miners))
	for _, miner := range miners {
		minerInfos = append(minerInfos, MinerLogInfo{
			ID:                 miner.ID,
			IPAddress:          miner.IPAddress,
			IPAddressActual:    miner.IPAddressActual,
			Hostname:           miner.Hostname,
			RegisteredAt:       miner.RegisteredAt,
			LastHeartbeat:      miner.LastHeartbeat,
			Active:             miner.Active && time.Since(miner.LastHeartbeat) < 2*time.Minute,
			ShouldMine:         miner.ShouldMine,
			CPUThrottlePercent: miner.CPUThrottlePercent,
			BlocksMined:        miner.BlocksMined,
			HashRate:           miner.HashRate,
			CPUUsagePercent:    miner.CPUUsagePercent,
			TotalHashes:        miner.TotalHashes,
			TotalMiningTime:    miner.TotalMiningTime.Seconds(),
			GPUEnabled:         miner.GPUEnabled,
			GPUHashRate:        miner.GPUHashRate,
			HybridMode:         miner.HybridMode,
		})
	}

	return PoolSnapshot{
		Timestamp:        time.Now(),
		TotalMiners:      stats.TotalMiners,
		ActiveMiners:     stats.ActiveMiners,
		TotalHashRate:    stats.TotalHashRate,
		TotalBlocksMined: stats.TotalBlocksMined,
		BlockchainHeight: int64(stats.BlockchainHeight),
		Difficulty:       stats.Difficulty,
		Miners:           minerInfos,
	}
}

// WriteLog writes the current state to the log file
func (pl *PoolLogger) WriteLog() error {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	logData := LogFile{
		ServerStartTime: pl.startTime,
		ServerUptime:    time.Since(pl.startTime).Seconds(),
		LastUpdate:      time.Now(),
		Events:          pl.events,
		CurrentSnapshot: pl.GetPoolSnapshot(),
	}

	data, err := json.MarshalIndent(logData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal log data: %v", err)
	}

	// Write to temporary file first, then rename (atomic operation)
	tempFile := pl.logFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write log file: %v", err)
	}

	if err := os.Rename(tempFile, pl.logFile); err != nil {
		return fmt.Errorf("failed to rename log file: %v", err)
	}

	return nil
}

// Start begins the periodic logging
func (pl *PoolLogger) Start() {
	go func() {
		// Write initial log
		pl.WriteLog()

		ticker := time.NewTicker(pl.updateInterval)
		defer ticker.Stop()

		for range ticker.C {
			pl.WriteLog()
		}
	}()
}
