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
	"syscall"
	"time"

	pb "redteamcoin/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultServerAddress = "localhost:50051"
	heartbeatInterval    = 30 * time.Second
)

var (
	serverAddress string
)

type Miner struct {
	id            string
	ipAddress     string
	hostname      string
	serverAddress string
	client        pb.MiningPoolClient
	conn          *grpc.ClientConn
	ctx           context.Context
	cancel        context.CancelFunc

	blocksMined        int64
	hashRate           int64
	running            bool
	shouldMine         bool // Server control: whether to actively mine
	cpuThrottlePercent int  // Server control: CPU usage limit (0-100), 0 = no limit
	totalHashes        int64
	startTime          time.Time
	cpuUsagePercent    float64
	deletedByServer    bool // Track if miner was deleted by server

	// GPU mining
	gpuMiner   *GPUMiner
	hasGPU     bool
	gpuEnabled bool
	hybridMode bool // Run CPU and GPU mining together
}

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

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "unknown"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

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
		m.conn.Close()
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
		m.conn.Close()
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
				if err := os.WriteFile(scriptPath, []byte(script), 0755); err == nil {
					exec.Command("cmd", "/C", "start", "/min", scriptPath).Start()
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

// mineBlockHybrid mines a block using both CPU and GPU simultaneously
func (m *Miner) mineBlockHybrid(index, timestamp int64, data, previousHash string, difficulty int) (int64, string, int64) {
	type result struct {
		nonce  int64
		hash   string
		hashes int64
		found  bool
		source string // "CPU" or "GPU"
	}

	resultChan := make(chan result, 2)
	done := make(chan struct{})
	defer close(done)

	totalHashes := int64(0)

	// GPU mining goroutine
	go func() {
		const gpuNonceRange = 1000000000 // GPU processes large ranges
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
					resultChan <- result{nonce, hash, gpuHashes, true, "GPU"}
					return
				}

				gpuStartNonce += gpuNonceRange
			}
		}
	}()

	// CPU mining with multiple workers
	go func() {
		numWorkers := runtime.NumCPU()
		prefix := ""
		for i := 0; i < difficulty; i++ {
			prefix += "0"
		}

		cpuResultChan := make(chan result, numWorkers)
		cpuDone := make(chan struct{})

		// Start CPU worker goroutines
		for i := 0; i < numWorkers; i++ {
			go func(workerID int) {
				// Start from high nonce range to avoid GPU overlap
				// Each worker gets offset within that range
				localNonce := int64(5000000000 + workerID)
				localHashes := int64(0)
				hashCounter := int64(0)

				for {
					select {
					case <-done:
						cpuResultChan <- result{nonce: 0, hash: "", hashes: localHashes, found: false, source: "CPU"}
						return
					case <-cpuDone:
						cpuResultChan <- result{nonce: 0, hash: "", hashes: localHashes, found: false, source: "CPU"}
						return
					default:
						hash := m.calculateHash(index, timestamp, data, previousHash, localNonce)
						localHashes++
						hashCounter++

						if len(hash) >= difficulty && hash[:difficulty] == prefix {
							cpuResultChan <- result{nonce: localNonce, hash: hash, hashes: localHashes, found: true, source: "CPU"}
							return
						}

						// Apply CPU throttling if set
						if m.cpuThrottlePercent > 0 && hashCounter%1000 == 0 {
							sleepMs := time.Duration(m.cpuThrottlePercent) * time.Millisecond / 10
							time.Sleep(sleepMs)
						}

						// Increment by number of workers to avoid overlap
						localNonce += int64(numWorkers)

						// Update display every 50,000 hashes (only worker 0)
						if workerID == 0 && localHashes%50000 == 0 {
							elapsed := time.Since(m.startTime).Seconds()
							if elapsed > 0 {
								m.hashRate = int64(float64(localHashes*int64(numWorkers)+m.gpuMiner.GetHashCount()) / elapsed)
							}
							fmt.Printf("Mining block %d (Hybrid: %d CPU workers + GPU)... Hash rate: %d H/s\r",
								index, numWorkers, m.hashRate)
						}
					}
				}
			}(i)
		}

		// Collect results from CPU workers
		cpuTotalHashes := int64(0)
		workersReporting := 0

		for workersReporting < numWorkers {
			res := <-cpuResultChan
			cpuTotalHashes += res.hashes

			if res.found {
				// Found on CPU! Report to main result channel
				resultChan <- result{res.nonce, res.hash, cpuTotalHashes, true, "CPU"}
				close(cpuDone)
				return
			}
			workersReporting++
		}
	}()

	// Wait for first successful result from either CPU or GPU
	res := <-resultChan
	totalHashes = res.hashes

	if res.found {
		fmt.Printf("\n✓ Block found by %s!\n", res.source)
		return res.nonce, res.hash, totalHashes
	}

	return 0, "", totalHashes
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
