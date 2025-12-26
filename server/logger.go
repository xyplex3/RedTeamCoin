// Package main implements the RedTeamCoin mining pool server components.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// LogEntry represents a single event logged by the mining pool server.
// Events are timestamped and categorized by type, with optional miner
// association and additional details stored as key-value pairs.
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	MinerID   string                 `json:"miner_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Message   string                 `json:"message"`
}

// MinerLogInfo represents detailed miner state captured in log snapshots.
// This structure provides a complete view of a miner's configuration,
// status, and performance metrics at the time of logging.
type MinerLogInfo struct {
	ID                 string    `json:"id"`
	IPAddress          string    `json:"ip_address"`
	IPAddressActual    string    `json:"ip_address_actual"`
	Hostname           string    `json:"hostname"`
	RegisteredAt       time.Time `json:"registered_at"`
	LastHeartbeat      time.Time `json:"last_heartbeat"`
	Active             bool      `json:"active"`
	ShouldMine         bool      `json:"should_mine"`
	CPUThrottlePercent int32     `json:"cpu_throttle_percent"`
	BlocksMined        int64     `json:"blocks_mined"`
	HashRate           int64     `json:"hash_rate"`
	CPUUsagePercent    float64   `json:"cpu_usage_percent"`
	TotalHashes        int64     `json:"total_hashes"`
	TotalMiningTime    float64   `json:"total_mining_time_seconds"`
	GPUEnabled         bool      `json:"gpu_enabled"`
	GPUHashRate        int64     `json:"gpu_hash_rate,omitempty"`
	HybridMode         bool      `json:"hybrid_mode"`
}

// PoolSnapshot represents a complete snapshot of the mining pool's state
// at a specific point in time, including aggregate statistics and detailed
// information about all registered miners.
type PoolSnapshot struct {
	Timestamp        time.Time      `json:"timestamp"`
	TotalMiners      int            `json:"total_miners"`
	ActiveMiners     int            `json:"active_miners"`
	TotalHashRate    int64          `json:"total_hash_rate"`
	TotalBlocksMined int64          `json:"total_blocks_mined"`
	BlockchainHeight int64          `json:"blockchain_height"`
	Difficulty       int32          `json:"difficulty"`
	Miners           []MinerLogInfo `json:"miners"`
}

// LogFile represents the complete structure of a pool log file written
// to disk. It contains server metadata, event history, and a current
// snapshot of the pool state for comprehensive monitoring and analysis.
type LogFile struct {
	ServerStartTime time.Time    `json:"server_start_time"`
	ServerUptime    float64      `json:"server_uptime_seconds"`
	LastUpdate      time.Time    `json:"last_update"`
	Events          []LogEntry   `json:"events"`
	CurrentSnapshot PoolSnapshot `json:"current_snapshot"`
}

// PoolLogger handles periodic logging of mining pool state to a JSON file.
// It maintains an in-memory event buffer and periodically writes complete
// snapshots to disk for monitoring, debugging, and forensic analysis.
//
// Concurrency Safety: All methods are safe for concurrent use by multiple
// goroutines. Internal state is protected by a sync.RWMutex.
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

// NewPoolLogger creates a new pool logger that writes snapshots to the
// specified log file at the given update interval. The logger maintains
// up to 1000 recent events in memory to include in log output.
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

// LogEvent records a new event in the logger's buffer with the current
// timestamp. If the event buffer exceeds maxEvents (1000), the oldest
// events are discarded. This method is safe for concurrent use.
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

// GetPoolSnapshot creates a complete snapshot of the current mining pool
// state including all miner information and aggregate statistics. The
// snapshot reflects the pool's state at the moment this method is called.
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

// WriteLog writes the current pool state, event history, and metadata to
// the log file. It uses atomic file writes (write to temp, then rename)
// to prevent corruption. This method is safe for concurrent use.
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
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write log file: %v", err)
	}

	if err := os.Rename(tempFile, pl.logFile); err != nil {
		return fmt.Errorf("failed to rename log file: %v", err)
	}

	return nil
}

// Start begins periodic logging to disk at the configured update interval.
// It writes an initial snapshot immediately, then continues writing
// snapshots on a regular schedule. This method spawns a goroutine and
// returns immediately.
func (pl *PoolLogger) Start() {
	go func() {
		// Write initial log
		if err := pl.WriteLog(); err != nil {
			log.Printf("Error writing initial log: %v", err)
		}

		ticker := time.NewTicker(pl.updateInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := pl.WriteLog(); err != nil {
				log.Printf("Error writing log: %v", err)
			}
		}
	}()
}
