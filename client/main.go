// Package main implements a RedTeamCoin cryptocurrency mining client.
//
// The client connects to a mining pool server via gRPC and performs
// proof-of-work mining using available CPU and GPU resources. Mining can be
// controlled remotely by the pool server, which can pause mining, adjust CPU
// throttling, or terminate the client with self-deletion.
//
// # Mining Modes
//
// The client supports three mining modes based on hardware availability:
//   - CPU-only: Multi-threaded mining across all available cores
//   - GPU-only: Hardware-accelerated mining via CUDA or OpenCL
//   - Hybrid: Simultaneous CPU and GPU mining for maximum performance
//
// GPU mining is enabled by default when compatible hardware is detected.
// Set RTC_CLIENT_MINING_GPU_ENABLED=false to disable, or RTC_CLIENT_MINING_HYBRID_MODE=true for hybrid mode.
//
// # Configuration
//
// Server address can be specified via:
//   - Command-line flag: -server or -s
//   - Environment variable: RTC_CLIENT_SERVER_ADDRESS
//   - Default: localhost:50051
//
// # Self-Deletion
//
// The server can command the client to self-delete its executable. This uses
// platform-specific techniques: advanced NTFS stream manipulation on Windows,
// simple os.Remove on Unix-like systems.
package main

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"redteamcoin/config"
	"redteamcoin/logger"
	pb "redteamcoin/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	serverAddress string
	configPath    string
	logLevel      string
	logFormat     string
	quiet         bool
	verbose       bool
)

// clampToInt32 converts an int to int32, clamping overflow values to prevent
// data loss when converting from native int to protobuf int32 fields. Values
// exceeding the int32 range are clamped to MaxInt32 or MinInt32 as appropriate.
func clampToInt32(n int) int32 {
	switch {
	case n > math.MaxInt32:
		return math.MaxInt32
	case n < math.MinInt32:
		return math.MinInt32
	default:
		return int32(n)
	}
}

// Miner manages a cryptocurrency mining client instance that performs
// proof-of-work computation and communicates with a mining pool server.
//
// The miner coordinates CPU and/or GPU resources to solve cryptographic
// puzzles, submits solutions to the pool, and responds to remote control
// commands from the server. Mining mode (CPU-only, GPU-only, or hybrid)
// is determined automatically based on available hardware and environment
// variables.
//
// Server control commands include pausing/resuming mining, throttling CPU
// usage, and terminating the client with self-deletion. All fields are
// private and should be accessed through the exported methods.
type Miner struct {
	id            string
	ipAddress     string
	hostname      string
	serverAddress string
	client        pb.MiningPoolClient
	conn          *grpc.ClientConn
	ctx           context.Context
	cancel        context.CancelFunc
	config        *config.ClientConfig

	blocksMined        int64
	hashRate           int64
	running            bool
	shouldMine         bool
	cpuThrottlePercent int
	totalHashes        int64
	startTime          time.Time
	cpuUsagePercent    float64
	deletedByServer    bool

	gpuMiner   *GPUMiner
	hasGPU     bool
	gpuEnabled bool
	hybridMode bool

	currentBlockIndex int64
	cancelWork        context.CancelFunc
	workCtx           context.Context
}

// NewMiner creates a new mining client configured to connect to the specified
// server address and configuration. It automatically detects available GPU
// hardware and configures mining mode based on the provided config.
func NewMiner(serverAddr string, cfg *config.ClientConfig) (*Miner, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	ipAddress := getOutboundIP()
	minerID := fmt.Sprintf("miner-%s-%d", hostname, time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	gpuMiner := NewGPUMiner()

	hybridMode := cfg.Mining.HybridMode
	gpuEnabled := cfg.Mining.GPUEnabled

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
		config:             cfg,
		running:            false,
		shouldMine:         true,
		cpuThrottlePercent: 0,
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

// createClientTLSConfig creates a TLS configuration for gRPC client connections.
// If TLS is disabled, returns nil. If enabled, creates config with InsecureSkipVerify
// to support self-signed certificates.
func createClientTLSConfig(cfg *config.ClientTLSConfig) (*tls.Config, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify, // nosemgrep: go.lang.security.audit.crypto.tls.tls-with-insecure-skip-verify.tls-with-insecure-skip-verify
	}

	// Optional: Load CA cert if specified and not skipping verification
	// This can be implemented later if needed for production deployments
	if cfg.CACertFile != "" && !cfg.InsecureSkipVerify {
		// Future: Load CA certificate for validation
		logger.Get().Warn("CA certificate validation not yet implemented",
			"ca_cert_file", cfg.CACertFile)
	}

	return tlsConfig, nil
}

// Connect establishes a gRPC connection to the mining pool server and
// registers the miner. It displays connection information including detected
// GPU hardware. Returns an error if connection or registration fails.
func (m *Miner) Connect() error {
	logger.Get().Info("connecting to mining pool",
		"server", m.serverAddress)

	fmt.Printf("Connecting to mining pool at %s...\n", m.serverAddress)

	// Configure transport credentials based on TLS settings
	var creds credentials.TransportCredentials

	if m.config.Server.TLS.Enabled {
		tlsConfig, err := createClientTLSConfig(&m.config.Server.TLS)
		if err != nil {
			logger.Get().Error("failed to create TLS config", "error", err)
			return fmt.Errorf("failed to create TLS config: %v", err)
		}
		creds = credentials.NewTLS(tlsConfig)
		logger.Get().Info("connecting with TLS",
			"server", m.serverAddress,
			"insecure_skip_verify", m.config.Server.TLS.InsecureSkipVerify)
		fmt.Printf("  Using TLS encryption (certificate verification: %s)\n",
			map[bool]string{true: "disabled", false: "enabled"}[m.config.Server.TLS.InsecureSkipVerify])
	} else {
		creds = insecure.NewCredentials()
		logger.Get().Debug("connecting without TLS")
	}

	conn, err := grpc.Dial(m.serverAddress, grpc.WithTransportCredentials(creds))
	if err != nil {
		logger.Get().Error("failed to establish gRPC connection",
			"server", m.serverAddress,
			"error", err)
		return fmt.Errorf("failed to connect: %v", err)
	}

	m.conn = conn
	m.client = pb.NewMiningPoolClient(conn)

	logger.Get().Info("registering miner with pool",
		"miner_id", m.id,
		"hostname", m.hostname)

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
		fmt.Printf("  GPUs:       Detected but disabled (set RTC_CLIENT_MINING_GPU_ENABLED=true to enable)\n")
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
		logger.Get().Error("miner registration request failed",
			"miner_id", m.id,
			"error", err)
		return fmt.Errorf("failed to register: %v", err)
	}

	if !resp.Success {
		logger.Get().Warn("miner registration rejected by server",
			"miner_id", m.id,
			"message", resp.Message)
		return fmt.Errorf("registration failed: %s", resp.Message)
	}

	logger.Get().Info("miner registered successfully with pool",
		"miner_id", m.id,
		"message", resp.Message)

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

	logger.Get().Info("starting mining operations",
		"miner_id", m.id,
		"gpu_enabled", m.gpuEnabled,
		"hybrid_mode", m.hybridMode)

	// Start GPU miner if available
	if m.hasGPU && m.gpuEnabled {
		if err := m.gpuMiner.Start(); err != nil {
			logger.Get().Warn("failed to start GPU miner",
				"miner_id", m.id,
				"error", err)
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
	logger.Get().Info("stopping miner",
		"miner_id", m.id,
		"blocks_mined", m.blocksMined)
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
			logger.Get().Warn("error notifying server of miner stop",
				"miner_id", m.id,
				"error", err)
		} else {
			logger.Get().Info("miner stopped successfully",
				"miner_id", m.id,
				"blocks_mined", resp.TotalBlocksMined)
			fmt.Printf("Miner stopped. Total blocks mined: %d\n", resp.TotalBlocksMined)
		}
	}

	m.cancel()
	if m.conn != nil {
		if err := m.conn.Close(); err != nil {
			logger.Get().Warn("error closing gRPC connection",
				"miner_id", m.id,
				"error", err)
		}
	}
}

// selfDelete removes the client executable from disk using platform-specific
// techniques. On Windows, it creates a batch script to delay deletion after
// the process exits. On Unix systems, it uses os.Remove directly. The
// deletion occurs asynchronously in a background goroutine with a 500ms delay
// to allow the connection to close cleanly.
func (m *Miner) selfDelete() {
	exePath, err := os.Executable()
	if err != nil {
		logger.Get().Error("failed to get executable path for self-deletion",
			"miner_id", m.id,
			"error", err)
		return
	}

	// Resolve symlinks to get actual file path
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		logger.Get().Error("failed to resolve symlink for self-deletion",
			"miner_id", m.id,
			"path", exePath,
			"error", err)
		return
	}

	// Clean path to normalize it
	realPath = filepath.Clean(realPath)

	// Validate path is absolute
	if !filepath.IsAbs(realPath) {
		logger.Get().Error("refusing to delete relative path",
			"miner_id", m.id,
			"path", realPath)
		return
	}

	logger.Get().Warn("initiating self-deletion",
		"miner_id", m.id,
		"executable_path", realPath)
	fmt.Printf("Deleting executable: %s\n", realPath)

	if m.conn != nil {
		if err := m.conn.Close(); err != nil {
			logger.Get().Warn("error closing connection before self-deletion",
				"miner_id", m.id,
				"error", err)
		}
	}

	go func() {
		// Capture file stats before sleep to detect tampering
		stat1, err := os.Stat(realPath)
		if err != nil {
			logger.Get().Error("failed to stat executable before deletion",
				"miner_id", m.id,
				"path", realPath,
				"error", err)
			return
		}

		time.Sleep(500 * time.Millisecond)

		// Verify file hasn't been modified during sleep window
		stat2, err := os.Stat(realPath)
		if err != nil {
			logger.Get().Error("failed to stat executable after sleep",
				"miner_id", m.id,
				"path", realPath,
				"error", err)
			return
		}

		if stat1.ModTime() != stat2.ModTime() || stat1.Size() != stat2.Size() {
			logger.Get().Error("executable was modified during deletion window, aborting",
				"miner_id", m.id,
				"path", realPath)
			return
		}

		if err := os.Remove(realPath); err != nil {
			if runtime.GOOS == "windows" {
				// Windows batch scripts handle spaces via surrounding quotes,
				// but do not use doubled quotes for escaping. Executable paths
				// should never contain double quotes on Windows; if they do,
				// refuse to generate a deletion script.
				if strings.Contains(realPath, `"`) {
					logger.Get().Error("executable path contains unsupported quote character",
						"miner_id", m.id,
						"path", realPath)
					return
				}

				scriptPath := realPath + "_delete.bat"
				script := fmt.Sprintf("@echo off\ntimeout /t 1 /nobreak >nul\ndel /f /q \"%s\"\ndel /f /q \"%%~f0\"", realPath)
				// Note: On Windows, file permission bits (0600) are not enforced the same way as on Unix.
				// The 0600 mode is applied for consistency, but Windows uses ACLs for actual security.
				if err := os.WriteFile(scriptPath, []byte(script), 0600); err == nil {
					// #nosec G204 -- scriptPath is constructed from validated realPath
					if err := exec.Command("cmd", "/C", "start", "/min", scriptPath).Start(); err != nil {
						logger.Get().Error("failed to start deletion script",
							"miner_id", m.id,
							"script_path", scriptPath,
							"error", err)
					}
				} else {
					logger.Get().Error("failed to create deletion script",
						"miner_id", m.id,
						"script_path", scriptPath,
						"error", err)
				}
			} else {
				logger.Get().Error("failed to delete executable",
					"miner_id", m.id,
					"path", realPath,
					"error", err)
			}
		} else {
			logger.Get().Info("executable deleted successfully",
				"miner_id", m.id,
				"path", realPath)
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
		if !m.shouldMine {
			time.Sleep(5 * time.Second)
			continue
		}

		workResp, err := m.client.GetWork(m.ctx, &pb.WorkRequest{
			MinerId: m.id,
		})

		if err != nil {
			logger.Get().Warn("error getting work from pool",
				"miner_id", m.id,
				"error", err)
			time.Sleep(5 * time.Second)
			continue
		}

		if workResp.BlockIndex < m.currentBlockIndex {
			fmt.Printf("Skipping stale work for block %d (current: %d)\n", workResp.BlockIndex, m.currentBlockIndex)
			continue
		}

		if m.cancelWork != nil {
			m.cancelWork()
		}

		m.currentBlockIndex = workResp.BlockIndex
		m.workCtx, m.cancelWork = context.WithCancel(m.ctx)

		fmt.Printf("Received work for block %d (difficulty: %d)\n", workResp.BlockIndex, workResp.Difficulty)

		var nonce int64
		var hash string
		var hashes int64

		switch {
		case m.hasGPU && m.gpuEnabled && m.hybridMode:
			nonce, hash, hashes = m.mineBlockHybrid(
				m.workCtx,
				workResp.BlockIndex,
				workResp.Timestamp,
				workResp.Data,
				workResp.PreviousHash,
				int(workResp.Difficulty),
			)
		case m.hasGPU && m.gpuEnabled:
			nonce, hash, hashes = m.mineBlockGPU(
				m.workCtx,
				workResp.BlockIndex,
				workResp.Timestamp,
				workResp.Data,
				workResp.PreviousHash,
				int(workResp.Difficulty),
			)
		default:
			nonce, hash, hashes = m.mineBlock(
				m.workCtx,
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
			logger.Get().Warn("error submitting work to pool",
				"miner_id", m.id,
				"block_index", workResp.BlockIndex,
				"error", err)
			continue
		}

		if submitResp.Accepted {
			m.blocksMined++
			logger.Get().Info("block mined and accepted",
				"miner_id", m.id,
				"block_index", workResp.BlockIndex,
				"nonce", nonce,
				"reward", submitResp.Reward,
				"total_blocks", m.blocksMined)
			fmt.Printf("✓ BLOCK MINED! Block %d accepted! Reward: %d RTC (Total blocks: %d, Hash rate: %d H/s)\n\n",
				workResp.BlockIndex, submitResp.Reward, m.blocksMined, m.hashRate)
		} else {
			logger.Get().Warn("block submission rejected by pool",
				"miner_id", m.id,
				"block_index", workResp.BlockIndex,
				"reason", submitResp.Message)
			fmt.Printf("✗ Block %d rejected: %s\n\n", workResp.BlockIndex, submitResp.Message)
		}
	}
}

// mineBlock performs CPU-only proof-of-work mining using all available cores.
// It spawns worker goroutines that each test different nonce ranges in
// parallel, applying CPU throttling if configured. Returns the found nonce,
// hash, and total hashes computed. The first worker to find a valid solution
// signals all others to stop.
type mineResult struct {
	nonce  int64
	hash   string
	hashes int64
	found  bool
}

func (m *Miner) cpuWorker(ctx context.Context, workerID, numWorkers int, index, timestamp int64, data, previousHash, prefix string, difficulty int, done <-chan struct{}, resultChan chan<- mineResult) {
	localNonce := int64(workerID)
	localHashes := int64(0)
	hashCounter := int64(0)

	for {
		select {
		case <-ctx.Done():
			resultChan <- mineResult{nonce: 0, hash: "", hashes: localHashes, found: false}
			return
		case <-done:
			resultChan <- mineResult{nonce: 0, hash: "", hashes: localHashes, found: false}
			return
		default:
			hash := m.calculateHash(index, timestamp, data, previousHash, localNonce)
			localHashes++
			hashCounter++

			if len(hash) >= difficulty && hash[:difficulty] == prefix {
				resultChan <- mineResult{nonce: localNonce, hash: hash, hashes: localHashes, found: true}
				return
			}

			if m.cpuThrottlePercent > 0 && hashCounter%1000 == 0 {
				sleepMs := time.Duration(m.cpuThrottlePercent) * time.Millisecond / 10
				time.Sleep(sleepMs)
			}

			localNonce += int64(numWorkers)

			if workerID == 0 && localHashes%100000 == 0 {
				fmt.Printf("Mining block %d... Nonce: %d, Hash rate: %d H/s\r",
					index, localNonce, m.hashRate)
			}
		}
	}
}

func (m *Miner) mineBlock(ctx context.Context, index, timestamp int64, data, previousHash string, difficulty int) (int64, string, int64) {
	prefix := strings.Repeat("0", difficulty)

	numWorkers := runtime.NumCPU()
	fmt.Printf("Starting %d worker threads for CPU mining...\n", numWorkers)

	resultChan := make(chan mineResult, numWorkers)
	done := make(chan struct{})

	for i := 0; i < numWorkers; i++ {
		go m.cpuWorker(ctx, i, numWorkers, index, timestamp, data, previousHash, prefix, difficulty, done, resultChan)
	}

	totalHashes := int64(0)
	workersReporting := 0
	var foundResult mineResult

	for workersReporting < numWorkers {
		res := <-resultChan
		totalHashes += res.hashes
		workersReporting++

		if res.found && foundResult.hash == "" {
			foundResult = res
			close(done)
		}
	}

	if foundResult.hash != "" {
		return foundResult.nonce, foundResult.hash, totalHashes
	}

	return 0, "", totalHashes
}

// calculateHash computes the SHA256 hash for a block given its components.
// The hash input is the concatenation of block index, timestamp, data,
// previous hash, and nonce. Returns the hexadecimal string representation
// of the hash.
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

// monitorCPU runs in a background goroutine to estimate CPU usage based on
// hash rate. This is a simplified estimation since actual CPU usage requires
// platform-specific system calls. The estimate is reported to the pool server
// via heartbeats.
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

// sendHeartbeat runs in a background goroutine, periodically sending status
// updates to the pool server. It reports mining statistics, GPU information,
// and receives control commands from the server including pause/resume, CPU
// throttling adjustments, and deletion requests. Automatically handles miner
// termination when deleted by the server.
func (m *Miner) sendHeartbeat() {
	ticker := time.NewTicker(m.config.Network.HeartbeatInterval)
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
						Id:           clampToInt32(dev.ID),
						Name:         dev.Name,
						Type:         dev.Type,
						Memory:       dev.Memory,
						ComputeUnits: clampToInt32(dev.ComputeUnits),
						Available:    dev.Available,
					})
				}
				gpuHashRate = m.gpuMiner.GetHashCount()
			}

			logger.Get().Debug("sending heartbeat to server",
				"miner_id", m.id,
				"hash_rate", m.hashRate,
				"blocks_mined", m.blocksMined)

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
				logger.Get().Warn("heartbeat request failed",
					"miner_id", m.id,
					"error", err)
			} else {
				logger.Get().Debug("heartbeat response received",
					"miner_id", m.id,
					"active", resp.Active,
					"should_mine", resp.ShouldMine)

				// Check if miner was deleted from the server
				if !resp.Active {
					logger.Get().Warn("miner deleted by server, shutting down",
						"miner_id", m.id,
						"message", resp.Message)
					fmt.Println("\n" + resp.Message)
					fmt.Println("Shutting down miner...")
					m.deletedByServer = true
					m.running = false

					// Delete the executable
					m.selfDelete()

					m.cancel()
					return
				}

				if m.shouldMine != resp.ShouldMine {
					m.shouldMine = resp.ShouldMine
					if m.shouldMine {
						logger.Get().Info("mining resumed by server",
							"miner_id", m.id)
						fmt.Println("Server resumed mining")
					} else {
						logger.Get().Info("mining paused by server",
							"miner_id", m.id)
						fmt.Println("Server paused mining")
					}
				}

				if m.cpuThrottlePercent != int(resp.CpuThrottlePercent) {
					m.cpuThrottlePercent = int(resp.CpuThrottlePercent)
					if m.cpuThrottlePercent == 0 {
						logger.Get().Info("CPU throttle removed by server",
							"miner_id", m.id)
						fmt.Println("Server removed CPU throttle (unlimited)")
					} else {
						logger.Get().Info("CPU throttle adjusted by server",
							"miner_id", m.id,
							"throttle_percent", m.cpuThrottlePercent)
						fmt.Printf("Server set CPU throttle to %d%%\n", m.cpuThrottlePercent)
					}
				}
			}

		case <-m.ctx.Done():
			return
		}
	}
}

// mineBlockGPU performs GPU-only proof-of-work mining by processing nonce
// ranges through the GPU miner. It continuously submits work batches until
// a valid solution is found or mining is stopped. Updates the display with current
// progress and hash rate. Returns the found nonce, hash, and total hashes computed.
func (m *Miner) mineBlockGPU(ctx context.Context, index, timestamp int64, data, previousHash string, difficulty int) (int64, string, int64) {
	nonceRange := m.config.GPU.NonceRange

	startNonce := int64(0)
	totalHashes := int64(0)

	for m.running {
		select {
		case <-ctx.Done():
			return 0, "", totalHashes
		default:
		}

		nonce, hash, hashes, found := m.gpuMiner.MineBlock(
			index, timestamp, data, previousHash, difficulty,
			startNonce, nonceRange,
		)

		totalHashes += hashes

		if found {
			return nonce, hash, totalHashes
		}

		startNonce += nonceRange

		elapsed := time.Since(m.startTime).Seconds()
		if elapsed > 0 {
			m.hashRate = int64(float64(totalHashes) / elapsed)
		}
		fmt.Printf("Mining block %d (GPU)... Nonce: %d, Hash rate: %d H/s\r",
			index, startNonce, m.hashRate)
	}

	return 0, "", totalHashes
}

// miningResult represents the outcome of a mining operation from either
// CPU or GPU miners. It contains the nonce and hash if a solution was found,
// the total number of hashes computed, and which compute resource found the
// solution. Used internally for coordinating hybrid mining operations.
type miningResult struct {
	nonce  int64
	hash   string
	hashes int64
	found  bool
	source string
}

// runGPUMiner executes GPU mining in a background goroutine for hybrid mode.
// It continuously processes nonce ranges on the GPU until either a solution
// is found or the done channel signals termination. Sends results back via
// resultChan for coordination with CPU workers.
func (m *Miner) runGPUMiner(index, timestamp int64, data, previousHash string, difficulty int, done <-chan struct{}, resultChan chan<- miningResult) {
	gpuNonceRange := m.config.GPU.NonceRange
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

// runCPUWorker runs a single CPU mining worker thread in hybrid mode. Each
// worker tests nonces in its assigned range (offset by workerID to avoid
// overlap), applies throttling if configured, and reports progress. Stops
// when done or cpuDone signals are received.
func (m *Miner) runCPUWorker(index, timestamp int64, data, previousHash, prefix string, difficulty, workerID, numWorkers int, done, cpuDone <-chan struct{}, resultChan chan<- miningResult) {
	localNonce := m.config.GPU.CPUStartNonce + int64(workerID)
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

// applyCPUThrottling introduces sleep delays to reduce CPU usage when server
// throttling is enabled. Called periodically by worker threads to respect
// the cpuThrottlePercent limit set by the pool server.
func (m *Miner) applyCPUThrottling(hashCounter int64) {
	if m.cpuThrottlePercent > 0 && hashCounter%1000 == 0 {
		sleepMs := time.Duration(m.cpuThrottlePercent) * time.Millisecond / 10
		time.Sleep(sleepMs)
	}
}

// updateHashRateDisplay periodically updates the console with current mining
// progress including block index, number of workers, and combined hash rate
// from CPU and GPU. Only worker 0 updates the display to avoid race conditions.
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

// runCPUMiningCoordinator spawns and manages multiple CPU worker goroutines
// for hybrid mining. It creates one worker per CPU core, distributes nonce
// ranges to avoid overlap, collects results, and reports back when a solution
// is found or all workers have completed their ranges.
func (m *Miner) runCPUMiningCoordinator(index, timestamp int64, data, previousHash string, difficulty int, done <-chan struct{}, resultChan chan<- miningResult) {
	numWorkers := runtime.NumCPU()
	prefix := strings.Repeat("0", difficulty)

	cpuResultChan := make(chan miningResult, numWorkers)
	cpuDone := make(chan struct{})

	for i := 0; i < numWorkers; i++ {
		go m.runCPUWorker(index, timestamp, data, previousHash, prefix, difficulty, i, numWorkers, done, cpuDone, cpuResultChan)
	}

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

	// Send "not found" result after all workers complete without finding
	resultChan <- miningResult{nonce: 0, hash: "", hashes: cpuTotalHashes, found: false, source: "CPU"}
}

// mineBlockHybrid mines a block using both CPU and GPU simultaneously
func (m *Miner) mineBlockHybrid(ctx context.Context, index, timestamp int64, data, previousHash string, difficulty int) (int64, string, int64) {
	resultChan := make(chan miningResult, 2)
	done := make(chan struct{})

	go m.runGPUMiner(index, timestamp, data, previousHash, difficulty, done, resultChan)
	go m.runCPUMiningCoordinator(index, timestamp, data, previousHash, difficulty, done, resultChan)

	// Race mode: first to find wins
	select {
	case <-ctx.Done():
		close(done)
		return 0, "", 0
	case res := <-resultChan:
		close(done) // Signal other worker to stop

		if res.found {
			fmt.Printf("\n✓ Block found by %s!\n", res.source)

			// Drain any buffered result from the other worker to prevent stale submissions
			go func() {
				select {
				case <-resultChan:
					// Discard result from slower worker
				case <-time.After(100 * time.Millisecond):
					// Timeout if other worker already stopped
				}
			}()

			return res.nonce, res.hash, res.hashes
		}
		return 0, "", res.hashes
	}
}

var (
	selfDeleteOnExit atomic.Bool
)

// startShutdownMonitor starts monitoring for shutdown signals and triggers.
// Returns channels for signal and file-based shutdown, and the shutdown file path.
// The context is used to cancel the file monitoring goroutine on shutdown.
func startShutdownMonitor(ctx context.Context, exePath string) (chan os.Signal, chan struct{}, string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	shutdownFile := exePath + ".shutdown"
	shutdownChan := make(chan struct{})

	// Monitor for shutdown file regardless of autoDelete setting
	// The shutdown file monitoring and self-deletion are separate concerns
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := os.Stat(shutdownFile); err == nil {
					close(shutdownChan)
					return
				}
			}
		}
	}()

	return sigChan, shutdownChan, shutdownFile
}

// parseFlags parses command-line flags and returns the auto-delete setting
func parseFlags() bool {
	var autoDelete bool
	flag.StringVar(&serverAddress, "server", "", "Mining pool server address (host:port)")
	flag.StringVar(&serverAddress, "s", "", "Mining pool server address (host:port) (shorthand)")
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.BoolVar(&autoDelete, "auto-delete", true, "Delete executable on shutdown (default: true)")
	flag.StringVar(&logLevel, "log-level", "", "Log level (debug, info, warn, error)")
	flag.StringVar(&logFormat, "log-format", "", "Log format (text, color, json)")
	flag.BoolVar(&quiet, "quiet", false, "Quiet mode (errors only)")
	flag.BoolVar(&verbose, "verbose", false, "Verbose mode (enable debug)")
	flag.Parse()
	return autoDelete
}

// resolveAutoDeleteSetting determines the final auto-delete setting based on flags and config
func resolveAutoDeleteSetting(autoDelete bool, cfg *config.ClientConfig) bool {
	// If the auto-delete flag was not provided, fall back to the config value
	autoDeleteFlagSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "auto-delete" {
			autoDeleteFlagSet = true
		}
	})
	if !autoDeleteFlagSet {
		return cfg.Mining.AutoDelete
	}
	return autoDelete
}

// applyLoggingOverrides applies CLI logging flag overrides to the config
func applyLoggingOverrides(cfg *config.ClientConfig) {
	if logLevel != "" {
		cfg.Logging.Level = logLevel
	}
	if logFormat != "" {
		cfg.Logging.Format = logFormat
	}
	if quiet {
		cfg.Logging.Quiet = true
	}
	if verbose {
		cfg.Logging.Verbose = true
	}
}

// setupMiner creates and connects a new miner instance
func setupMiner(serverAddr string, cfg *config.ClientConfig) (*Miner, error) {
	miner, err := NewMiner(serverAddr, cfg)
	if err != nil {
		return nil, err
	}

	if err := connectWithRetry(miner, cfg); err != nil {
		return nil, err
	}

	return miner, nil
}

// handleShutdown sets up shutdown monitoring and cleanup
func handleShutdown(miner *Miner, shutdownFile string, sigChan chan os.Signal, shutdownChan chan struct{}) {
	go func() {
		select {
		case <-sigChan:
			fmt.Println("Signal received, initiating shutdown...")
		case <-shutdownChan:
			fmt.Println("Shutdown file detected, initiating shutdown...")
		}

		var shutdownErrs []error

		miner.Stop()

		if selfDeleteOnExit.Load() {
			fmt.Println("Auto-delete enabled, removing executable...")
			if err := os.Remove(shutdownFile); err != nil {
				shutdownErrs = append(shutdownErrs, fmt.Errorf("remove shutdown file: %w", err))
			}
			miner.selfDelete()
		}

		if len(shutdownErrs) > 0 {
			for _, err := range shutdownErrs {
				logger.Get().Error("shutdown error", "error", err)
			}
		}

		os.Exit(0)
	}()
}

// connectWithRetry attempts to connect to the pool with retries
func connectWithRetry(miner *Miner, cfg *config.ClientConfig) error {
	startTime := time.Now()
	for {
		err := miner.Connect()
		if err == nil {
			return nil
		}

		elapsed := time.Since(startTime)
		if elapsed >= cfg.Network.MaxRetryTime {
			return fmt.Errorf("failed to connect to pool after %v: %w", cfg.Network.MaxRetryTime, err)
		}

		remaining := cfg.Network.MaxRetryTime - elapsed
		fmt.Printf("Failed to connect: %v\n", err)
		fmt.Printf("Retrying in %v... (%.0f seconds remaining before timeout)\n",
			cfg.Network.RetryInterval, remaining.Seconds())
		time.Sleep(cfg.Network.RetryInterval)
	}
}

func main() {
	autoDelete := parseFlags()

	cfg, err := config.LoadClientConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	autoDelete = resolveAutoDeleteSetting(autoDelete, cfg)
	selfDeleteOnExit.Store(autoDelete)

	applyLoggingOverrides(cfg)
	logger.Set(logger.NewFromClientConfig(cfg))

	if serverAddress == "" {
		serverAddress = cfg.Server.Address
	}

	logger.Get().Info("starting RedTeamCoin mining client",
		"server", serverAddress,
		"gpu_enabled", cfg.Mining.GPUEnabled,
		"hybrid_mode", cfg.Mining.HybridMode)

	fmt.Println("=== RedTeamCoin Miner ===")
	fmt.Println()

	miner, err := setupMiner(serverAddress, cfg)
	if err != nil {
		logger.Get().Error("failed to setup miner", "error", err)
		os.Exit(1)
	}

	exePath, err := os.Executable()
	if err != nil {
		logger.Get().Error("failed to get executable path", "error", err)
		os.Exit(1)
	}

	sigChan, shutdownChan, shutdownFile := startShutdownMonitor(miner.ctx, exePath)
	handleShutdown(miner, shutdownFile, sigChan, shutdownChan)

	miner.Start()
}
