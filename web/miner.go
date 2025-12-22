//go:build js && wasm
// +build js,wasm

// Package main implements a WebAssembly-based cryptocurrency miner for browsers.
//
// This package compiles to WASM and provides JavaScript bindings for browser-based
// mining. It supports multi-threaded mining using Web Workers and provides
// real-time statistics via callbacks.
//
// The miner exposes a RedTeamMiner global object in JavaScript with methods for:
//   - Starting and stopping mining
//   - Setting work parameters
//   - Configuring threads and throttling
//   - Retrieving mining statistics
//
// Example usage from JavaScript:
//
//	RedTeamMiner.start({threads: 4, throttle: 0.8});
//	RedTeamMiner.setWork({blockIndex: 1, previousHash: "...", ...});
//	const stats = RedTeamMiner.getStats();
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"sync"
	"syscall/js"
	"time"
)

// MiningWork represents a mining work unit received from the pool server.
// It contains all parameters needed to perform proof-of-work mining.
type MiningWork struct {
	BlockIndex   int64  `json:"blockIndex"`
	PreviousHash string `json:"previousHash"`
	Data         string `json:"data"`
	Difficulty   int    `json:"difficulty"`
	Timestamp    int64  `json:"timestamp"`
}

// MinerStats holds current mining statistics
type MinerStats struct {
	HashRate     int64   `json:"hashRate"`
	TotalHashes  int64   `json:"totalHashes"`
	BlocksFound  int64   `json:"blocksFound"`
	IsRunning    bool    `json:"isRunning"`
	Threads      int     `json:"threads"`
	Throttle     float64 `json:"throttle"`
	ElapsedTime  int64   `json:"elapsedTime"`
	CurrentNonce int64   `json:"currentNonce"`
}

// WebMiner manages cryptocurrency mining operations in a web browser using
// WebAssembly. It coordinates mining across multiple threads, tracks statistics,
// and provides JavaScript callbacks for results and status updates.
//
// All operations are thread-safe for concurrent access from JavaScript and
// Go goroutines.
type WebMiner struct {
	mu           sync.RWMutex
	running      bool
	threads      int
	throttle     float64
	hashRate     int64
	totalHashes  int64
	blocksFound  int64
	startTime    time.Time
	currentWork  *MiningWork
	stopChan     chan struct{}
	onResult     js.Value
	onStats      js.Value
	currentNonce int64
}

var miner *WebMiner

func main() {
	miner = &WebMiner{
		threads:  4,
		throttle: 0.8,
	}

	// Register JavaScript functions
	js.Global().Set("RedTeamMiner", js.ValueOf(map[string]interface{}{
		"start":       js.FuncOf(start),
		"stop":        js.FuncOf(stop),
		"setWork":     js.FuncOf(setWork),
		"getStats":    js.FuncOf(getStats),
		"setThreads":  js.FuncOf(setThreads),
		"setThrottle": js.FuncOf(setThrottle),
		"onResult":    js.FuncOf(onResult),
		"onStats":     js.FuncOf(onStats),
		"mine":        js.FuncOf(mineNonceRange),
		"sha256":      js.FuncOf(sha256Hash),
		"version":     js.FuncOf(version),
	}))

	// Keep the Go program running
	select {}
}

// version returns the miner version
func version(this js.Value, args []js.Value) interface{} {
	return "RedTeamMiner WASM v1.0.0"
}

// start begins mining with the given configuration
func start(this js.Value, args []js.Value) interface{} {
	miner.mu.Lock()
	defer miner.mu.Unlock()

	if miner.running {
		return map[string]interface{}{
			"success": false,
			"error":   "Miner already running",
		}
	}

	// Parse configuration if provided
	if len(args) > 0 && args[0].Type() == js.TypeObject {
		config := args[0]
		if threads := config.Get("threads"); !threads.IsUndefined() {
			miner.threads = threads.Int()
		}
		if throttle := config.Get("throttle"); !throttle.IsUndefined() {
			miner.throttle = throttle.Float()
		}
	}

	miner.running = true
	miner.startTime = time.Now()
	miner.stopChan = make(chan struct{})

	// Start stats reporter
	go miner.reportStats()

	return map[string]interface{}{
		"success": true,
		"message": "Miner started",
	}
}

// stop halts mining
func stop(this js.Value, args []js.Value) interface{} {
	miner.mu.Lock()
	defer miner.mu.Unlock()

	if !miner.running {
		return map[string]interface{}{
			"success": false,
			"error":   "Miner not running",
		}
	}

	miner.running = false
	close(miner.stopChan)

	return map[string]interface{}{
		"success":     true,
		"message":     "Miner stopped",
		"totalHashes": miner.totalHashes,
		"blocksFound": miner.blocksFound,
	}
}

// setWork sets the current mining work
func setWork(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return map[string]interface{}{
			"success": false,
			"error":   "Work object required",
		}
	}

	workObj := args[0]
	work := &MiningWork{
		BlockIndex:   int64(workObj.Get("blockIndex").Int()),
		PreviousHash: workObj.Get("previousHash").String(),
		Data:         workObj.Get("data").String(),
		Difficulty:   workObj.Get("difficulty").Int(),
		Timestamp:    int64(workObj.Get("timestamp").Int()),
	}

	miner.mu.Lock()
	miner.currentWork = work
	miner.currentNonce = 0
	miner.mu.Unlock()

	return map[string]interface{}{
		"success": true,
		"message": "Work set",
	}
}

// getStats returns current mining statistics
func getStats(this js.Value, args []js.Value) interface{} {
	miner.mu.RLock()
	defer miner.mu.RUnlock()

	elapsed := int64(0)
	if miner.running {
		elapsed = int64(time.Since(miner.startTime).Seconds())
	}

	return map[string]interface{}{
		"hashRate":     miner.hashRate,
		"totalHashes":  miner.totalHashes,
		"blocksFound":  miner.blocksFound,
		"isRunning":    miner.running,
		"threads":      miner.threads,
		"throttle":     miner.throttle,
		"elapsedTime":  elapsed,
		"currentNonce": miner.currentNonce,
	}
}

// setThreads sets the number of mining threads
func setThreads(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	miner.mu.Lock()
	miner.threads = args[0].Int()
	miner.mu.Unlock()
	return true
}

// setThrottle sets CPU throttle (0.0 - 1.0)
func setThrottle(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	miner.mu.Lock()
	miner.throttle = args[0].Float()
	miner.mu.Unlock()
	return true
}

// onResult registers a callback for mining results
func onResult(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 || args[0].Type() != js.TypeFunction {
		return false
	}
	miner.mu.Lock()
	miner.onResult = args[0]
	miner.mu.Unlock()
	return true
}

// onStats registers a callback for stats updates
func onStats(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 || args[0].Type() != js.TypeFunction {
		return false
	}
	miner.mu.Lock()
	miner.onStats = args[0]
	miner.mu.Unlock()
	return true
}

// mineNonceRange mines a specific nonce range (called from Web Workers)
func mineNonceRange(this js.Value, args []js.Value) interface{} {
	if len(args) < 5 {
		return map[string]interface{}{
			"found": false,
			"error": "Required: blockIndex, timestamp, data, previousHash, difficulty",
		}
	}

	blockIndex := int64(args[0].Int())
	timestamp := int64(args[1].Int())
	data := args[2].String()
	previousHash := args[3].String()
	difficulty := args[4].Int()
	startNonce := int64(0)
	endNonce := int64(1000000)

	if len(args) > 5 {
		startNonce = int64(args[5].Int())
	}
	if len(args) > 6 {
		endNonce = int64(args[6].Int())
	}

	// Create difficulty prefix
	prefix := ""
	for i := 0; i < difficulty; i++ {
		prefix += "0"
	}

	hashCount := int64(0)
	startTime := time.Now()

	for nonce := startNonce; nonce < endNonce; nonce++ {
		// Build block string
		record := strconv.FormatInt(blockIndex, 10) +
			strconv.FormatInt(timestamp, 10) +
			data +
			previousHash +
			strconv.FormatInt(nonce, 10)

		// Compute SHA256
		h := sha256.New()
		h.Write([]byte(record))
		hashed := h.Sum(nil)
		hashStr := hex.EncodeToString(hashed)

		hashCount++

		// Update miner stats
		miner.mu.Lock()
		miner.totalHashes++
		miner.currentNonce = nonce
		miner.mu.Unlock()

		// Check if meets difficulty
		if len(hashStr) >= difficulty && hashStr[:difficulty] == prefix {
			elapsed := time.Since(startTime).Seconds()
			hashRate := int64(0)
			if elapsed > 0 {
				hashRate = int64(float64(hashCount) / elapsed)
			}

			miner.mu.Lock()
			miner.blocksFound++
			miner.hashRate = hashRate
			miner.mu.Unlock()

			return map[string]interface{}{
				"found":    true,
				"nonce":    nonce,
				"hash":     hashStr,
				"hashes":   hashCount,
				"hashRate": hashRate,
			}
		}
	}

	elapsed := time.Since(startTime).Seconds()
	hashRate := int64(0)
	if elapsed > 0 {
		hashRate = int64(float64(hashCount) / elapsed)
	}

	miner.mu.Lock()
	miner.hashRate = hashRate
	miner.mu.Unlock()

	return map[string]interface{}{
		"found":    false,
		"hashes":   hashCount,
		"hashRate": hashRate,
	}
}

// sha256Hash computes SHA256 hash of input
func sha256Hash(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return ""
	}
	input := args[0].String()
	h := sha256.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// reportStats periodically reports stats to JavaScript callback
func (m *WebMiner) reportStats() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.mu.RLock()
			if !m.onStats.IsUndefined() && m.onStats.Type() == js.TypeFunction {
				stats := MinerStats{
					HashRate:     m.hashRate,
					TotalHashes:  m.totalHashes,
					BlocksFound:  m.blocksFound,
					IsRunning:    m.running,
					Threads:      m.threads,
					Throttle:     m.throttle,
					ElapsedTime:  int64(time.Since(m.startTime).Seconds()),
					CurrentNonce: m.currentNonce,
				}
				statsJSON, _ := json.Marshal(stats)
				m.onStats.Invoke(string(statsJSON))
			}
			m.mu.RUnlock()
		}
	}
}
