// Package main implements the RedTeamCoin mining client.
//
// The client connects to a mining pool server via gRPC, requests work units,
// performs proof-of-work mining using CPU and/or GPU resources, and submits
// solutions back to the pool for validation and rewards.
//
// The client supports:
//   - Multi-threaded CPU mining across all available cores
//   - GPU mining via CUDA or OpenCL (if available)
//   - Hybrid mode combining CPU and GPU mining simultaneously
//   - Remote control by the pool server (pause/resume, CPU throttling)
//   - Automatic reconnection with exponential backoff
//   - Self-deletion on server command
//
// Configuration is done via command-line flags and environment variables.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	pb "redteamcoin/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultServerAddress = "localhost:50051" // Default mining pool server address
	heartbeatInterval    = 30 * time.Second  // Interval between heartbeat updates to server
)

var (
	serverAddress string
)

// Miner represents a mining client that connects to a pool server and
// performs proof-of-work mining. It manages the mining process, tracks
// statistics, and handles communication with the pool server.
//
// The miner can operate in three modes:
//   - CPU-only: Uses all available CPU cores for mining
//   - GPU-only: Uses available CUDA or OpenCL GPUs for mining
//   - Hybrid: Runs both CPU and GPU mining simultaneously
//
// All miner operations are coordinated through a context that can be
// cancelled for graceful shutdown.
type Miner struct {
	id            string              // Unique miner identifier
	ipAddress     string              // Client's outbound IP address
	hostname      string              // System hostname
	serverAddress string              // Pool server address
	client        pb.MiningPoolClient // gRPC client for pool communication
	conn          *grpc.ClientConn    // gRPC connection
	ctx           context.Context     // Cancellable context for shutdown
	cancel        context.CancelFunc  // Context cancellation function

	blocksMined        int64     // Total blocks successfully mined
	hashRate           int64     // Current hash rate (hashes per second)
	running            bool      // Whether mining is currently active
	shouldMine         bool      // Server control: whether to actively mine
	cpuThrottlePercent int       // Server control: CPU usage limit (0-100), 0 = no limit
	totalHashes        int64     // Cumulative hashes computed
	startTime          time.Time // When mining started
	cpuUsagePercent    float64   // Estimated CPU usage percentage
	deletedByServer    bool      // Whether miner was deleted by server

	// GPU mining configuration
	gpuMiner   *GPUMiner // GPU mining implementation
	hasGPU     bool      // Whether GPU hardware is detected
	gpuEnabled bool      // Whether GPU mining is enabled
	hybridMode bool      // Whether to run CPU and GPU mining together
}

// NewMiner creates a new mining client configured to connect to the specified
// server address. It automatically detects available GPU hardware and
// configures mining mode based on environment variables GPU_MINING and
// HYBRID_MINING.
func NewMiner(serverAddr string) (*Miner, error) {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	// Get IP address
	ipAddress := getOutboundIP()

	// Generate miner ID
	minerID := fmt.Sprintf("miner-%s-%d", hostname, time.Now().Unix())

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize GPU miner
	gpuMiner := NewGPUMiner()

	// Check for hybrid mode environment variable
	hybridMode := os.Getenv("HYBRID_MINING") == "true"
	gpuEnabled := os.Getenv("GPU_MINING") != "false" // Enabled by default if GPUs found

	// GPU can only be enabled if GPUs are actually detected
	hasGPU := gpuMiner.HasGPUs()
	if !hasGPU {
		gpuEnabled = false
		hybridMode = false
	}

	miner := &Miner{
		id:                 minerID,
		ipAddress:          ipAddress,
		hostname:           hostname,
		serverAddress:      serverAddr,
		ctx:                ctx,
		cancel:             cancel,
		running:            false,
		shouldMine:         true, // Start with mining enabled by default
		cpuThrottlePercent: 0,    // No throttling by default
		deletedByServer:    false,
		gpuMiner:           gpuMiner,
		hasGPU:             hasGPU,
		gpuEnabled:         gpuEnabled,
		hybridMode:         hybridMode,
	}

	return miner, nil
}

// getOutboundIP determines the client's outbound IP address by establishing
// a UDP connection to a public DNS server. It returns "unknown" if the
// IP cannot be determined.
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "unknown"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// Connect establishes a gRPC connection to the mining pool server and
// registers the miner. It displays connection information including detected
// GPU hardware. Returns an error if connection or registration fails.
func (m *Miner) Connect() error {
	fmt.Printf("Connecting to mining pool at %s...\n", m.serverAddress)

	conn, err := grpc.Dial(m.serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	m.conn = conn
	m.client = pb.NewMiningPoolClient(conn)

	// Register with the pool
	fmt.Printf("Registering miner...\n")
	fmt.Printf("  Miner ID:   %s\n", m.id)
	fmt.Printf("  IP Address: %s\n", m.ipAddress)
	fmt.Printf("  Hostname:   %s\n", m.hostname)

	// Display GPU information
	switch {
	case m.hasGPU && m.gpuEnabled:
		devices := m.gpuMiner.GetDevices()
		fmt.Printf("  GPUs Found: %d\n", len(devices))
		for _, dev := range devices {
			fmt.Printf("    - %s (%s) - %d MB, %d compute units\n",
				dev.Name, dev.Type, dev.Memory/1024/1024, dev.ComputeUnits)
		}
		if m.hybridMode {
			fmt.Printf("  Mode:       Hybrid (CPU + GPU)\n")
		} else {
			fmt.Printf("  Mode:       GPU only\n")
		}
	case m.hasGPU && !m.gpuEnabled:
		fmt.Printf("  GPUs:       Detected but disabled (set GPU_MINING=true to enable)\n")
	default:
		fmt.Printf("  GPUs:       None detected - using CPU only\n")
	}

	resp, err := m.client.RegisterMiner(m.ctx, &pb.MinerInfo{
		MinerId:   m.id,
		IpAddress: m.ipAddress,
		Hostname:  m.hostname,
		Timestamp: time.Now().Unix(),
	})

	if err != nil {
		return fmt.Errorf("failed to register: %v", err)
	}

	if !resp.Success {
		return fmt.Errorf("registration failed: %s", resp.Message)
	}

	fmt.Printf("✓ Successfully registered with pool: %s\n\n", resp.Message)
	return nil
}

// Start begins the mining process, initializing GPU miners if available
// and starting background goroutines for heartbeats and CPU monitoring.
// This method blocks until mining is stopped.
func (m *Miner) Start() {
	m.running = true
	m.startTime = time.Now()
	m.totalHashes = 0

	// Start GPU miner if available
	if m.hasGPU && m.gpuEnabled {
		if err := m.gpuMiner.Start(); err != nil {
			log.Printf("Warning: Failed to start GPU miner: %v", err)
		}
	}

	// Start heartbeat
	go m.sendHeartbeat()

	// Start CPU monitoring
	go m.monitorCPU()

	// Start mining
	m.mine()
}

// Stop gracefully shuts down the mining process, stopping GPU miners and
// notifying the pool server unless the miner was deleted by the server.
// It closes the gRPC connection and cancels the miner's context.
func (m *Miner) Stop() {
	if !m.running {
		return
	}

	m.running = false
	fmt.Println("\nStopping miner...")

	// Stop GPU miner if running
	if m.hasGPU && m.gpuEnabled {
		m.gpuMiner.Stop()
	}

	// Notify server (only if not deleted by server)
	if !m.deletedByServer {
		resp, err := m.client.StopMining(m.ctx, &pb.MinerInfo{
			MinerId:   m.id,
			IpAddress: m.ipAddress,
			Hostname:  m.hostname,
			Timestamp: time.Now().Unix(),
		})

		if err != nil {
			log.Printf("Error stopping miner: %v", err)
		} else {
			fmt.Printf("Miner stopped. Total blocks mined: %d\n", resp.TotalBlocksMined)
		}
	}

	m.cancel()
	if m.conn != nil {
		if err := m.conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
	}
}

// selfDelete removes the client executable from disk
func (m *Miner) selfDelete() {
	// Get the path to the current executable
	exePath, err := os.Executable()
	if err != nil {
		log.Printf("Failed to get executable path: %v", err)
		return
	}

	fmt.Printf("Deleting executable: %s\n", exePath)

	// Close all file handles and prepare for deletion
	if m.conn != nil {
		if err := m.conn.Close(); err != nil {
			log.Printf("Error closing connection before deletion: %v", err)
		}
	}

	// Schedule deletion after a short delay to allow cleanup
	go func() {
		time.Sleep(500 * time.Millisecond)

		// On Unix-like systems, we can delete the file while it's running
		// On Windows, we need to use a script
		if err := os.Remove(exePath); err != nil {
			// If direct deletion fails (Windows), create a script to delete after exit
			if runtime.GOOS == "windows" {
				scriptPath := exePath + "_delete.bat"
				script := fmt.Sprintf("@echo off\ntimeout /t 1 /nobreak >nul\ndel /f /q \"%s\"\ndel /f /q \"%%~f0\"", exePath)
				if err := os.WriteFile(scriptPath, []byte(script), 0600); err == nil {
					if err := exec.Command("cmd", "/C", "start", "/min", scriptPath).Start(); err != nil {
						log.Printf("Failed to start delete script: %v", err)
					}
				}
			} else {
				log.Printf("Failed to delete executable: %v", err)
			}
		} else {
			fmt.Println("Executable deleted successfully")
		}
	}()
}

func (m *Miner) mine() {
	fmt.Println("Starting mining...")
	fmt.Println("Press Ctrl+C to stop mining")

	startTime := time.Now()
	totalHashes := int64(0)

	for m.running {
		// Check if server wants us to mine
		if !m.shouldMine {
			time.Sleep(5 * time.Second)
			continue
		}

		// Get work from pool
		workResp, err := m.client.GetWork(m.ctx, &pb.WorkRequest{
			MinerId: m.id,
		})

		if err != nil {
			log.Printf("Error getting work: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		fmt.Printf("Received work for block %d (difficulty: %d)\n", workResp.BlockIndex, workResp.Difficulty)

		// Mine the block - use hybrid mining if GPU available and enabled
		var nonce int64
		var hash string
		var hashes int64

		switch {
		case m.hasGPU && m.gpuEnabled && m.hybridMode:
			// Hybrid: CPU + GPU mining simultaneously
			nonce, hash, hashes = m.mineBlockHybrid(
				workResp.BlockIndex,
				workResp.Timestamp,
				workResp.Data,
				workResp.PreviousHash,
				int(workResp.Difficulty),
			)
		case m.hasGPU && m.gpuEnabled:
			// GPU only
			nonce, hash, hashes = m.mineBlockGPU(
				workResp.BlockIndex,
				workResp.Timestamp,
				workResp.Data,
				workResp.PreviousHash,
				int(workResp.Difficulty),
			)
		default:
			// CPU only
			nonce, hash, hashes = m.mineBlock(
				workResp.BlockIndex,
				workResp.Timestamp,
				workResp.Data,
				workResp.PreviousHash,
				int(workResp.Difficulty),
			)
		}

		totalHashes += hashes
		m.totalHashes += hashes

		if !m.running {
			break
		}

		// Calculate hash rate
		elapsed := time.Since(startTime).Seconds()
		if elapsed > 0 {
			m.hashRate = int64(float64(totalHashes) / elapsed)
		}

		// Submit the solution
		submitResp, err := m.client.SubmitWork(m.ctx, &pb.WorkSubmission{
			MinerId:    m.id,
			BlockIndex: workResp.BlockIndex,
			Nonce:      nonce,
			Hash:       hash,
		})

		if err != nil {
			log.Printf("Error submitting work: %v", err)
			continue
		}

		if submitResp.Accepted {
			m.blocksMined++
			fmt.Printf("✓ BLOCK MINED! Block %d accepted! Reward: %d RTC (Total blocks: %d, Hash rate: %d H/s)\n\n",
				workResp.BlockIndex, submitResp.Reward, m.blocksMined, m.hashRate)
		} else {
			fmt.Printf("✗ Block %d rejected: %s\n\n", workResp.BlockIndex, submitResp.Message)
		}
	}
}

func (m *Miner) mineBlock(index, timestamp int64, data, previousHash string, difficulty int) (int64, string, int64) {
	prefix := ""
	for i := 0; i < difficulty; i++ {
		prefix += "0"
	}

	// Use all available CPU cores
	numWorkers := runtime.NumCPU()
	fmt.Printf("Starting %d worker threads for CPU mining...\n", numWorkers)

	type result struct {
		nonce  int64
		hash   string
		hashes int64
		found  bool
	}

	resultChan := make(chan result, numWorkers)
	done := make(chan struct{})

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			// Each worker gets its own nonce range offset
			// Worker 0: 0, numWorkers, 2*numWorkers, ...
			// Worker 1: 1, numWorkers+1, 2*numWorkers+1, ...
			localNonce := int64(workerID)
			localHashes := int64(0)
			hashCounter := int64(0)

			for {
				select {
				case <-done:
					// Send final hash count even if not found
					resultChan <- result{nonce: 0, hash: "", hashes: localHashes, found: false}
					return
				default:
					hash := m.calculateHash(index, timestamp, data, previousHash, localNonce)
					localHashes++
					hashCounter++

					if len(hash) >= difficulty && hash[:difficulty] == prefix {
						resultChan <- result{nonce: localNonce, hash: hash, hashes: localHashes, found: true}
						return
					}

					// Apply CPU throttling if set
					if m.cpuThrottlePercent > 0 && hashCounter%1000 == 0 {
						sleepMs := time.Duration(m.cpuThrottlePercent) * time.Millisecond / 10
						time.Sleep(sleepMs)
					}

					// Increment by number of workers to avoid overlap
					localNonce += int64(numWorkers)

					// Update display every 100,000 hashes (only worker 0)
					if workerID == 0 && localHashes%100000 == 0 {
						fmt.Printf("Mining block %d... Nonce: %d, Hash rate: %d H/s\r",
							index, localNonce, m.hashRate)
					}
				}
			}
		}(i)
	}

	// Wait for first successful result or stop signal
	totalHashes := int64(0)
	workersReporting := 0
	var foundResult result

	for workersReporting < numWorkers {
		res := <-resultChan
		totalHashes += res.hashes

		if res.found && foundResult.hash == "" {
			// Found a solution! Signal all workers to stop
			foundResult = res
			close(done)
		} else {
			workersReporting++
		}
	}

	// If we found a result, return it
	if foundResult.hash != "" {
		return foundResult.nonce, foundResult.hash, totalHashes
	}

	return 0, "", totalHashes
}

func (m *Miner) calculateHash(index, timestamp int64, data, previousHash string, nonce int64) string {
	record := strconv.FormatInt(index, 10) +
		strconv.FormatInt(timestamp, 10) +
		data +
		previousHash +
		strconv.FormatInt(nonce, 10)

	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func (m *Miner) monitorCPU() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !m.running {
				return
			}

			// Estimate CPU usage based on hash rate
			// This is a simple estimation - actual CPU usage would require OS-specific calls
			// For demonstration: assume each hash uses a small amount of CPU time
			// Typical CPU can do millions of hashes per second
			// We'll estimate based on activity level
			if m.hashRate > 0 {
				// Rough estimation: higher hash rate = higher CPU usage
				// Cap at 100%
				estimated := float64(m.hashRate) / 1000000.0 * 100.0
				if estimated > 100.0 {
					estimated = 100.0
				}
				m.cpuUsagePercent = estimated
			} else {
				m.cpuUsagePercent = 0.0
			}

		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Miner) sendHeartbeat() {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !m.running {
				return
			}

			miningTime := time.Since(m.startTime)

			// Prepare GPU device information
			var gpuDevices []*pb.GPUDevice
			var gpuHashRate int64
			if m.hasGPU && m.gpuEnabled {
				devices := m.gpuMiner.GetDevices()
				for _, dev := range devices {
					gpuDevices = append(gpuDevices, &pb.GPUDevice{
						Id:           int32(dev.ID),
						Name:         dev.Name,
						Type:         dev.Type,
						Memory:       dev.Memory,
						ComputeUnits: int32(dev.ComputeUnits),
						Available:    dev.Available,
					})
				}
				gpuHashRate = m.gpuMiner.GetHashCount()
			}

			resp, err := m.client.Heartbeat(m.ctx, &pb.MinerStatus{
				MinerId:           m.id,
				HashRate:          m.hashRate,
				BlocksMined:       m.blocksMined,
				CpuUsagePercent:   m.cpuUsagePercent,
				TotalHashes:       m.totalHashes,
				MiningTimeSeconds: int64(miningTime.Seconds()),
				GpuDevices:        gpuDevices,
				GpuHashRate:       gpuHashRate,
				GpuEnabled:        m.gpuEnabled,
				HybridMode:        m.hybridMode,
			})

			if err != nil {
				log.Printf("Error sending heartbeat: %v", err)
			} else {
				// Check if miner was deleted from the server
				if !resp.Active {
					fmt.Println("\n" + resp.Message)
					fmt.Println("Shutting down miner...")
					m.deletedByServer = true
					m.running = false

					// Delete the executable
					m.selfDelete()

					m.cancel()
					return
				}

				// Update shouldMine based on server response
				if m.shouldMine != resp.ShouldMine {
					m.shouldMine = resp.ShouldMine
					if m.shouldMine {
						fmt.Println("Server resumed mining")
					} else {
						fmt.Println("Server paused mining")
					}
				}

				// Update CPU throttle based on server response
				if m.cpuThrottlePercent != int(resp.CpuThrottlePercent) {
					m.cpuThrottlePercent = int(resp.CpuThrottlePercent)
					if m.cpuThrottlePercent == 0 {
						fmt.Println("Server removed CPU throttle (unlimited)")
					} else {
						fmt.Printf("Server set CPU throttle to %d%%\n", m.cpuThrottlePercent)
					}
				}
			}

		case <-m.ctx.Done():
			return
		}
	}
}

// mineBlockGPU mines a block using GPU only
func (m *Miner) mineBlockGPU(index, timestamp int64, data, previousHash string, difficulty int) (int64, string, int64) {
	// Use GPU miner for the entire nonce range
	const nonceRange = 1000000000 // 1 billion nonces per GPU batch

	startNonce := int64(0)
	totalHashes := int64(0)

	for m.running {
		nonce, hash, hashes, found := m.gpuMiner.MineBlock(
			index, timestamp, data, previousHash, difficulty,
			startNonce, nonceRange,
		)

		totalHashes += hashes

		if found {
			return nonce, hash, totalHashes
		}

		startNonce += nonceRange

		// Update display
		elapsed := time.Since(m.startTime).Seconds()
		if elapsed > 0 {
			m.hashRate = int64(float64(totalHashes) / elapsed)
		}
		fmt.Printf("Mining block %d (GPU)... Nonce: %d, Hash rate: %d H/s\r",
			index, startNonce, m.hashRate)
	}

	return 0, "", totalHashes
}

// miningResult represents the result from a mining operation
type miningResult struct {
	nonce  int64
	hash   string
	hashes int64
	found  bool
	source string // "CPU" or "GPU"
}

// runGPUMiner runs GPU mining in a goroutine
func (m *Miner) runGPUMiner(index, timestamp int64, data, previousHash string, difficulty int, done <-chan struct{}, resultChan chan<- miningResult) {
	const gpuNonceRange = 1000000000
	gpuStartNonce := int64(0)
	gpuHashes := int64(0)

	for {
		select {
		case <-done:
			return
		default:
			nonce, hash, hashes, found := m.gpuMiner.MineBlock(
				index, timestamp, data, previousHash, difficulty,
				gpuStartNonce, gpuNonceRange,
			)
			gpuHashes += hashes

			if found {
				resultChan <- miningResult{nonce, hash, gpuHashes, true, "GPU"}
				return
			}

			gpuStartNonce += gpuNonceRange
		}
	}
}

// runCPUWorker runs a single CPU mining worker
func (m *Miner) runCPUWorker(index, timestamp int64, data, previousHash, prefix string, difficulty, workerID, numWorkers int, done, cpuDone <-chan struct{}, resultChan chan<- miningResult) {
	localNonce := int64(5000000000 + workerID)
	localHashes := int64(0)
	hashCounter := int64(0)

	for {
		select {
		case <-done:
			resultChan <- miningResult{nonce: 0, hash: "", hashes: localHashes, found: false, source: "CPU"}
			return
		case <-cpuDone:
			resultChan <- miningResult{nonce: 0, hash: "", hashes: localHashes, found: false, source: "CPU"}
			return
		default:
			hash := m.calculateHash(index, timestamp, data, previousHash, localNonce)
			localHashes++
			hashCounter++

			if len(hash) >= difficulty && hash[:difficulty] == prefix {
				resultChan <- miningResult{nonce: localNonce, hash: hash, hashes: localHashes, found: true, source: "CPU"}
				return
			}

			m.applyCPUThrottling(hashCounter)
			localNonce += int64(numWorkers)
			m.updateHashRateDisplay(workerID, index, localHashes, numWorkers)
		}
	}
}

// applyCPUThrottling applies CPU throttling if configured
func (m *Miner) applyCPUThrottling(hashCounter int64) {
	if m.cpuThrottlePercent > 0 && hashCounter%1000 == 0 {
		sleepMs := time.Duration(m.cpuThrottlePercent) * time.Millisecond / 10
		time.Sleep(sleepMs)
	}
}

// updateHashRateDisplay updates the mining hash rate display
func (m *Miner) updateHashRateDisplay(workerID int, index, localHashes int64, numWorkers int) {
	if workerID == 0 && localHashes%50000 == 0 {
		elapsed := time.Since(m.startTime).Seconds()
		if elapsed > 0 {
			m.hashRate = int64(float64(localHashes*int64(numWorkers)+m.gpuMiner.GetHashCount()) / elapsed)
		}
		fmt.Printf("Mining block %d (Hybrid: %d CPU workers + GPU)... Hash rate: %d H/s\r",
			index, numWorkers, m.hashRate)
	}
}

// runCPUMiningCoordinator coordinates multiple CPU mining workers
func (m *Miner) runCPUMiningCoordinator(index, timestamp int64, data, previousHash string, difficulty int, done <-chan struct{}, resultChan chan<- miningResult) {
	numWorkers := runtime.NumCPU()
	prefix := strings.Repeat("0", difficulty)

	cpuResultChan := make(chan miningResult, numWorkers)
	cpuDone := make(chan struct{})

	// Start CPU worker goroutines
	for i := 0; i < numWorkers; i++ {
		go m.runCPUWorker(index, timestamp, data, previousHash, prefix, difficulty, i, numWorkers, done, cpuDone, cpuResultChan)
	}

	// Collect results from CPU workers
	cpuTotalHashes := int64(0)
	workersReporting := 0

	for workersReporting < numWorkers {
		res := <-cpuResultChan
		cpuTotalHashes += res.hashes

		if res.found {
			resultChan <- miningResult{res.nonce, res.hash, cpuTotalHashes, true, "CPU"}
			close(cpuDone)
			return
		}
		workersReporting++
	}
}

// mineBlockHybrid mines a block using both CPU and GPU simultaneously
func (m *Miner) mineBlockHybrid(index, timestamp int64, data, previousHash string, difficulty int) (int64, string, int64) {
	resultChan := make(chan miningResult, 2)
	done := make(chan struct{})
	defer close(done)

	go m.runGPUMiner(index, timestamp, data, previousHash, difficulty, done, resultChan)
	go m.runCPUMiningCoordinator(index, timestamp, data, previousHash, difficulty, done, resultChan)

	res := <-resultChan

	if res.found {
		fmt.Printf("\n✓ Block found by %s!\n", res.source)
		return res.nonce, res.hash, res.hashes
	}

	return 0, "", res.hashes
}

func main() {
	// Parse command-line flags
	flag.StringVar(&serverAddress, "server", "", "Mining pool server address (host:port)")
	flag.StringVar(&serverAddress, "s", "", "Mining pool server address (host:port) (shorthand)")
	flag.Parse()

	// Check environment variable as fallback
	if serverAddress == "" {
		if envServer := os.Getenv("POOL_SERVER"); envServer != "" {
			serverAddress = envServer
		} else {
			serverAddress = defaultServerAddress
		}
	}

	fmt.Println("=== RedTeamCoin Miner ===")
	fmt.Println()

	miner, err := NewMiner(serverAddress)
	if err != nil {
		log.Fatalf("Failed to create miner: %v", err)
	}

	// Connection retry logic: try to connect for up to 5 minutes
	const (
		retryInterval = 10 * time.Second
		maxRetryTime  = 5 * time.Minute
	)

	startTime := time.Now()
	connected := false

	for !connected {
		err = miner.Connect()
		if err == nil {
			connected = true
			break
		}

		// Check if we've exceeded the maximum retry time
		elapsed := time.Since(startTime)
		if elapsed >= maxRetryTime {
			log.Fatalf("Failed to connect to pool after %v: %v", maxRetryTime, err)
		}

		// Calculate remaining time
		remaining := maxRetryTime - elapsed
		fmt.Printf("Failed to connect: %v\n", err)
		fmt.Printf("Retrying in %v... (%.0f seconds remaining before timeout)\n",
			retryInterval, remaining.Seconds())
		time.Sleep(retryInterval)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		miner.Stop()
	}()

	// Start mining (this blocks until mining stops)
	miner.Start()

	// Wait a moment for cleanup
	time.Sleep(1 * time.Second)
	fmt.Println("Miner terminated.")
}
