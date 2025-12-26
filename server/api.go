// Package main implements the RedTeamCoin mining pool server components.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// HTTP server timeout configurations for API and redirect servers.
const (
	apiReadTimeout       = 15 * time.Second
	apiWriteTimeout      = 15 * time.Second
	apiIdleTimeout       = 60 * time.Second
	redirectReadTimeout  = 5 * time.Second
	redirectWriteTimeout = 5 * time.Second
	redirectIdleTimeout  = 30 * time.Second
)

// APIServer provides a REST API and web dashboard for pool administration.
// It supports both HTTP and HTTPS with bearer token authentication.
//
// The server exposes endpoints for:
//   - Pool statistics and miner information
//   - Miner control (pause/resume, throttling, deletion)
//   - Blockchain inspection
//   - WebSocket updates for real-time monitoring
//
// All administrative endpoints require authentication via Bearer token.
// The server supports graceful shutdown via the Shutdown method.
type APIServer struct {
	pool           *MiningPool   // Mining pool to expose via API
	blockchain     *Blockchain   // Blockchain to query
	authToken      string        // Bearer token for authentication
	useTLS         bool          // Whether to enable HTTPS
	certFile       string        // Path to TLS certificate
	keyFile        string        // Path to TLS private key
	wsHub          *WebSocketHub // WebSocket hub for real-time updates
	ctx            context.Context
	cancel         context.CancelFunc
	server         *http.Server // Main HTTP/HTTPS server
	redirectServer *http.Server // HTTP to HTTPS redirect server
}

// NewAPIServer creates a new API server with the specified configuration.
// It initializes a WebSocket hub for real-time updates and starts it in
// a background goroutine with context-based lifecycle management. The
// authToken is required for all API requests. The input context is used as
// a parent; a derived context with cancel function is created for lifecycle
// management. The derived context will be cancelled when either the parent
// context is cancelled or Shutdown() is called.
//
// Goroutine Lifecycle: Starts 1 background goroutine (WebSocket hub)
// that runs until the context is cancelled via Shutdown().
func NewAPIServer(ctx context.Context, pool *MiningPool, blockchain *Blockchain, authToken string, useTLS bool, certFile, keyFile string) *APIServer {
	serverCtx, cancel := context.WithCancel(ctx)
	wsHub := NewWebSocketHub(pool)
	go wsHub.Run(serverCtx)

	return &APIServer{
		pool:       pool,
		blockchain: blockchain,
		authToken:  authToken,
		useTLS:     useTLS,
		certFile:   certFile,
		keyFile:    keyFile,
		wsHub:      wsHub,
		ctx:        serverCtx,
		cancel:     cancel,
	}
}

// writeJSON writes a JSON response with the given status code and value.
// It automatically sets the Content-Type header and logs any encoding errors.
func (api *APIServer) writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// sendGenericErrorPage sends a generic HTTP 503 error page that reveals no
// information about the application. This is used for authentication
// failures to prevent information disclosure about the service's true
// purpose.
func (api *APIServer) sendGenericErrorPage(w http.ResponseWriter) {
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Service Error</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #ffffff;
            color: #333333;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            margin: 0;
        }
        .error-container {
            text-align: center;
            padding: 40px;
            max-width: 500px;
        }
        h1 {
            font-size: 24px;
            margin-bottom: 20px;
            color: #666666;
        }
        p {
            font-size: 16px;
            line-height: 1.6;
            color: #888888;
        }
    </style>
</head>
<body>
    <div class="error-container">
        <h1>Service Unavailable</h1>
        <p>We apologize for the inconvenience. The service is not functioning correctly at this time.</p>
        <p>Please try again later.</p>
    </div>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("Error writing error page: %v", err)
	}
}

// authMiddleware validates the Bearer token in the Authorization header
// before allowing access to protected endpoints. Invalid or missing tokens
// result in a generic error page that reveals no application details. This
// middleware wraps HTTP handler functions to enforce authentication.
func (api *APIServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		token := r.Header.Get("Authorization")

		// Check if token matches
		if token != "Bearer "+api.authToken {
			// Send generic error page with no application details
			api.sendGenericErrorPage(w)
			return
		}

		// Token is valid, proceed to handler
		next(w, r)
	}
}

// Start begins serving HTTP or HTTPS traffic on the specified port. If TLS
// is enabled and httpPort is non-zero, also starts an HTTP redirect server.
// This method blocks until the server encounters an error or is shut down.
// All administrative endpoints require Bearer token authentication.
func (api *APIServer) Start(port int, httpPort int) error {
	mux := http.NewServeMux()

	// Register handlers with authentication middleware
	mux.HandleFunc("/api/stats", api.authMiddleware(api.handleStats))
	mux.HandleFunc("/api/miners", api.authMiddleware(api.handleMiners))
	mux.HandleFunc("/api/blockchain", api.authMiddleware(api.handleBlockchain))
	mux.HandleFunc("/api/blocks/", api.authMiddleware(api.handleBlock))
	mux.HandleFunc("/api/validate", api.authMiddleware(api.handleValidate))
	mux.HandleFunc("/api/cpu", api.authMiddleware(api.handleCPUStats))
	mux.HandleFunc("/api/miner/pause", api.authMiddleware(api.handlePauseMiner))
	mux.HandleFunc("/api/miner/resume", api.authMiddleware(api.handleResumeMiner))
	mux.HandleFunc("/api/miner/delete", api.authMiddleware(api.handleDeleteMiner))
	mux.HandleFunc("/api/miner/throttle", api.authMiddleware(api.handleThrottleMiner))

	// WebSocket endpoint for web miners (no auth required)
	mux.HandleFunc("/ws", api.wsHub.HandleWebSocket)

	// Public endpoint - no authentication required
	mux.HandleFunc("/", api.handleIndex)

	addr := fmt.Sprintf(":%d", port)

	// Configure HTTP server with timeouts to prevent resource exhaustion
	api.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  apiReadTimeout,
		WriteTimeout: apiWriteTimeout,
		IdleTimeout:  apiIdleTimeout,
	}

	if api.useTLS {
		// Start HTTP to HTTPS redirect server in background goroutine if httpPort is provided
		// Lifecycle: Runs until Shutdown() is called
		if httpPort > 0 {
			go api.startHTTPRedirect(httpPort, port)
		}

		// Start HTTPS server
		protocol := "https"
		fmt.Printf("Starting API server on %s://localhost%s\n", protocol, addr)
		fmt.Printf("TLS enabled - using certificates:\n")
		fmt.Printf("  Certificate: %s\n", api.certFile)
		fmt.Printf("  Private Key: %s\n", api.keyFile)
		fmt.Printf("API authentication enabled - token required in Authorization header\n")

		return api.server.ListenAndServeTLS(api.certFile, api.keyFile)
	}

	// Start HTTP server
	fmt.Printf("Starting API server on http://localhost%s\n", addr)
	fmt.Printf("API authentication enabled - token required in Authorization header\n")
	fmt.Printf("WARNING: TLS is disabled - connections are not encrypted\n")
	return api.server.ListenAndServe()
}

// Shutdown gracefully shuts down the API server and all background goroutines.
// It stops the main HTTP/HTTPS server, redirect server (if running), and the
// WebSocket hub. The shutdown timeout is controlled by the provided context.
func (api *APIServer) Shutdown(ctx context.Context) error {
	// Cancel all background goroutines
	api.cancel()

	var shutdownErrs []error

	// Shutdown main server
	if api.server != nil {
		if err := api.server.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
			shutdownErrs = append(shutdownErrs, fmt.Errorf("main server shutdown: %w", err))
		}
	}

	// Shutdown redirect server if it exists
	if api.redirectServer != nil {
		if err := api.redirectServer.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
			shutdownErrs = append(shutdownErrs, fmt.Errorf("redirect server shutdown: %w", err))
		}
	}

	if len(shutdownErrs) > 0 {
		// Return first error, log the rest
		for i, err := range shutdownErrs {
			if i > 0 {
				log.Printf("Additional shutdown error: %v", err)
			}
		}
		return shutdownErrs[0]
	}

	return nil
}

// startHTTPRedirect starts an HTTP server that issues 301 permanent
// redirects to the HTTPS equivalent URL. This runs in a goroutine and
// handles port normalization for default HTTP (80) and HTTPS (443) ports.
func (api *APIServer) startHTTPRedirect(httpPort, httpsPort int) {
	redirect := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpsURL := fmt.Sprintf("https://%s:%d%s", r.Host, httpsPort, r.RequestURI)
		// Remove port from host if it's the default HTTP port
		if httpPort == 80 {
			httpsURL = fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)
		}
		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})

	httpAddr := fmt.Sprintf(":%d", httpPort)
	fmt.Printf("Starting HTTP->HTTPS redirect server on http://localhost%s\n", httpAddr)

	api.redirectServer = &http.Server{
		Addr:         httpAddr,
		Handler:      redirect,
		ReadTimeout:  redirectReadTimeout,
		WriteTimeout: redirectWriteTimeout,
		IdleTimeout:  redirectIdleTimeout,
	}

	if err := api.redirectServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP redirect server error: %v", err)
	}
}

// handleIndex serves the main web dashboard with real-time pool statistics,
// miner management, and blockchain visualization. It accepts authentication
// via Bearer token in the Authorization header or as a query parameter.
// Unauthenticated requests receive a generic error page.
func (api *APIServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Check authentication - accept token from either header or query parameter
	token := r.Header.Get("Authorization")
	if token == "" {
		// Check for token in query parameter
		queryToken := r.URL.Query().Get("token")
		if queryToken != "" {
			token = "Bearer " + queryToken
		}
	}

	// If token doesn't match, show generic error page
	if token != "Bearer "+api.authToken {
		api.sendGenericErrorPage(w)
		return
	}

	html := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>RedTeamCoin - Mining Pool Dashboard</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 20px;
            background-color: #1a1a1a;
            color: #e0e0e0;
        }
        h1 { color: #ff6b6b; }
        h2 { color: #4ecdc4; border-bottom: 2px solid #4ecdc4; padding-bottom: 5px; }
        .container { max-width: 1200px; margin: 0 auto; }
        .stats, .miners, .blockchain {
            background-color: #2a2a2a;
            padding: 20px;
            margin: 20px 0;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.3);
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 10px;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border-bottom: 1px solid #444;
        }
        th {
            background-color: #333;
            color: #4ecdc4;
            font-weight: bold;
        }
        tr:hover { background-color: #333; }
        .stat-item {
            display: inline-block;
            margin: 10px 20px 10px 0;
            padding: 10px 15px;
            background-color: #333;
            border-radius: 5px;
        }
        .stat-label {
            font-weight: bold;
            color: #4ecdc4;
        }
        .stat-value {
            font-size: 1.2em;
            color: #ff6b6b;
        }
        .active { color: #51cf66; font-weight: bold; }
        .inactive { color: #ff6b6b; font-weight: bold; }
        button {
            background-color: #4ecdc4;
            color: #1a1a1a;
            border: none;
            padding: 10px 20px;
            border-radius: 5px;
            cursor: pointer;
            font-size: 14px;
            font-weight: bold;
            width: 90px;
            margin-right: 2px;
            margin-top: 2px;
        }
        button:hover { background-color: #45b8ac; }
    </style>
    <script>
        // Get auth token from URL parameter
        const urlParams = new URLSearchParams(window.location.search);
        const authToken = urlParams.get('token') || '';

        async function loadData() {
            try {
                const headers = authToken ? { 'Authorization': 'Bearer ' + authToken } : {};

                const stats = await fetch('/api/stats', { headers }).then(r => {
                    if (!r.ok) throw new Error('Authentication failed');
                    return r.json();
                });
                const miners = await fetch('/api/miners', { headers }).then(r => r.json());
                const blockchain = await fetch('/api/blockchain', { headers }).then(r => r.json());

                updateStats(stats);
                updateMiners(miners);
                updateBlockchain(blockchain);
            } catch (error) {
                console.error('Error loading data:', error);
                if (error.message === 'Authentication failed') {
                    document.getElementById('stats').innerHTML = '<p style="color: #ff6b6b;">Authentication required. Please provide a valid token.</p>';
                }
            }
        }

        function updateStats(stats) {
            document.getElementById('stats').innerHTML =
                '<div class="stat-item"><span class="stat-label">Total Miners:</span> <span class="stat-value">' + stats.total_miners + '</span></div>' +
                '<div class="stat-item"><span class="stat-label">Active Miners:</span> <span class="stat-value">' + stats.active_miners + '</span></div>' +
                '<div class="stat-item"><span class="stat-label">Total Hash Rate:</span> <span class="stat-value">' + stats.total_hash_rate + ' H/s</span></div>' +
                '<div class="stat-item"><span class="stat-label">Blockchain Height:</span> <span class="stat-value">' + stats.blockchain_height + '</span></div>' +
                '<div class="stat-item"><span class="stat-label">Difficulty:</span> <span class="stat-value">' + stats.difficulty + '</span></div>' +
                '<div class="stat-item"><span class="stat-label">Block Reward:</span> <span class="stat-value">' + stats.block_reward + ' RTC</span></div>';
        }

        async function controlMiner(action, minerID) {
            try {
                const headers = {
                    'Authorization': 'Bearer ' + authToken,
                    'Content-Type': 'application/json'
                };

                const response = await fetch('/api/miner/' + action, {
                    method: 'POST',
                    headers: headers,
                    body: JSON.stringify({ miner_id: minerID })
                });

                const result = await response.json();
                if (response.ok) {
                    alert(result.message);
                    loadData(); // Refresh the data
                } else {
                    alert('Error: ' + (result.error || result.message));
                }
            } catch (error) {
                alert('Error: ' + error.message);
            }
        }

        async function setThrottle(minerID) {
            const throttleValue = prompt('Enter CPU throttle percentage (0-100):\n0 = No limit\n100 = Maximum throttle', '0');
            if (throttleValue === null) return; // Cancelled

            const throttle = parseInt(throttleValue);
            if (isNaN(throttle) || throttle < 0 || throttle > 100) {
                alert('Invalid throttle value. Must be between 0 and 100.');
                return;
            }

            try {
                const headers = {
                    'Authorization': 'Bearer ' + authToken,
                    'Content-Type': 'application/json'
                };

                const response = await fetch('/api/miner/throttle', {
                    method: 'POST',
                    headers: headers,
                    body: JSON.stringify({
                        miner_id: minerID,
                        throttle_percent: throttle
                    })
                });

                const result = await response.json();
                if (response.ok) {
                    alert(result.message);
                    loadData(); // Refresh the data
                } else {
                    alert('Error: ' + (result.error || result.message));
                }
            } catch (error) {
                alert('Error: ' + error.message);
            }
        }

        function getMiningType(miner) {
            if (!miner.GPUEnabled) {
                return 'CPU';
            }

            // Check GPU devices to determine type
            if (miner.GPUDevices && miner.GPUDevices.length > 0) {
                const gpuTypes = miner.GPUDevices.map(gpu => gpu.Type).filter((v, i, a) => a.indexOf(v) === i);
                const typeStr = gpuTypes.join('/');
                return miner.HybridMode ? 'CPU+' + typeStr : typeStr;
            }

            return miner.HybridMode ? 'CPU+GPU' : 'GPU';
        }

        function updateMiners(miners) {
            let html = '<table><tr><th>Miner ID</th><th>IP Address</th><th>Hostname</th><th>Mining Type</th><th>Status</th><th>Mining</th><th>CPU Throttle</th><th>Blocks Mined</th><th>Hash Rate</th><th>Last Heartbeat</th><th>Actions</th></tr>';
            miners.forEach(miner => {
                const status = miner.Active ? '<span class="active">Active</span>' : '<span class="inactive">Inactive</span>';
                const miningStatus = miner.ShouldMine ? '<span class="active">Mining</span>' : '<span class="inactive">Paused</span>';
                const miningType = getMiningType(miner);
                const throttleDisplay = miner.CPUThrottlePercent === 0 ? 'None' : miner.CPUThrottlePercent + '%';
                const lastSeen = new Date(miner.LastHeartbeat).toLocaleString();
                const pauseResumeBtn = miner.ShouldMine
                    ? '<button onclick="controlMiner(\'pause\', \'' + miner.ID + '\')">Pause</button>'
                    : '<button onclick="controlMiner(\'resume\', \'' + miner.ID + '\')">Resume</button>';
                const throttleBtn = '<button onclick="setThrottle(\'' + miner.ID + '\')">Throttle</button>';
                const deleteBtn = '<button onclick="if(confirm(\'Delete miner ' + miner.ID + '?\')) controlMiner(\'delete\', \'' + miner.ID + '\')">Delete</button>';
                html += '<tr><td>' + miner.ID + '</td><td>' + miner.IPAddress + '</td><td>' + miner.Hostname + '</td><td><span class="stat-label">' + miningType + '</span></td><td>' + status + '</td><td>' + miningStatus + '</td><td>' + throttleDisplay + '</td><td>' + miner.BlocksMined + '</td><td>' + miner.HashRate + ' H/s</td><td>' + lastSeen + '</td><td>' + pauseResumeBtn + ' ' + throttleBtn + ' ' + deleteBtn + '</td></tr>';
            });
            html += '</table>';
            document.getElementById('miners').innerHTML = html;
        }

        function updateBlockchain(blocks) {
            let html = '<table><tr><th>Index</th><th>Timestamp</th><th>Data</th><th>Hash</th><th>Previous Hash</th><th>Nonce</th><th>Mined By</th></tr>';
            blocks.slice().reverse().slice(0, 10).forEach(block => {
                const timestamp = new Date(block.Timestamp * 1000).toLocaleString();
                html += '<tr><td>' + block.Index + '</td><td>' + timestamp + '</td><td>' + block.Data + '</td><td>' + block.Hash.substring(0, 16) + '...</td><td>' + block.PreviousHash.substring(0, 16) + '...</td><td>' + block.Nonce + '</td><td>' + (block.MinedBy || 'Genesis') + '</td></tr>';
            });
            html += '</table>';
            document.getElementById('blockchain').innerHTML = html;
        }

        setInterval(loadData, 5000);
        window.onload = loadData;
    </script>
</head>
<body>
    <div class="container">
        <h1>⛏️ RedTeamCoin Mining Pool Dashboard</h1>

        <div class="stats">
            <h2>Pool Statistics</h2>
            <div id="stats">Loading...</div>
        </div>

        <div class="miners">
            <h2>Connected Miners</h2>
            <div id="miners">Loading...</div>
        </div>

        <div class="blockchain">
            <h2>Recent Blocks (Last 10)</h2>
            <div id="blockchain">Loading...</div>
        </div>
    </div>
</body>
</html>
`
	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write([]byte(html)); err != nil {
		log.Printf("Error writing index page: %v", err)
	}
}

// handleStats returns JSON-encoded pool statistics including total and
// active miner counts, hash rates, blocks mined, blockchain height,
// difficulty, and block reward.
func (api *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := api.pool.GetPoolStats()
	api.writeJSON(w, http.StatusOK, stats)
}

// handleMiners returns a JSON array of all registered miners with their
// status, performance metrics, IP addresses, GPU information, and server
// control settings. Includes both active and inactive miners.
func (api *APIServer) handleMiners(w http.ResponseWriter, r *http.Request) {
	miners := api.pool.GetMiners()

	type GPUDeviceResponse struct {
		ID           int    `json:"ID"`
		Name         string `json:"Name"`
		Type         string `json:"Type"`
		Memory       uint64 `json:"Memory"`
		ComputeUnits int    `json:"ComputeUnits"`
		Available    bool   `json:"Available"`
	}

	type MinerResponse struct {
		ID                 string              `json:"ID"`
		IPAddress          string              `json:"IPAddress"`
		IPAddressActual    string              `json:"IPAddressActual"`
		Hostname           string              `json:"Hostname"`
		RegisteredAt       time.Time           `json:"RegisteredAt"`
		LastHeartbeat      time.Time           `json:"LastHeartbeat"`
		Active             bool                `json:"Active"`
		ShouldMine         bool                `json:"ShouldMine"`
		CPUThrottlePercent int32               `json:"CPUThrottlePercent"`
		BlocksMined        int64               `json:"BlocksMined"`
		HashRate           int64               `json:"HashRate"`
		GPUDevices         []GPUDeviceResponse `json:"GPUDevices,omitempty"`
		GPUHashRate        int64               `json:"GPUHashRate,omitempty"`
		GPUEnabled         bool                `json:"GPUEnabled"`
		HybridMode         bool                `json:"HybridMode"`
	}

	response := make([]MinerResponse, len(miners))
	for i, miner := range miners {
		// Convert GPU devices
		gpuDevices := make([]GPUDeviceResponse, len(miner.GPUDevices))
		for j, gpu := range miner.GPUDevices {
			gpuDevices[j] = GPUDeviceResponse{
				ID:           gpu.ID,
				Name:         gpu.Name,
				Type:         gpu.Type,
				Memory:       gpu.Memory,
				ComputeUnits: gpu.ComputeUnits,
				Available:    gpu.Available,
			}
		}

		response[i] = MinerResponse{
			ID:                 miner.ID,
			IPAddress:          miner.IPAddress,
			IPAddressActual:    miner.IPAddressActual,
			Hostname:           miner.Hostname,
			RegisteredAt:       miner.RegisteredAt,
			LastHeartbeat:      miner.LastHeartbeat,
			Active:             miner.Active,
			ShouldMine:         miner.ShouldMine,
			CPUThrottlePercent: miner.CPUThrottlePercent,
			BlocksMined:        miner.BlocksMined,
			HashRate:           miner.HashRate,
			GPUDevices:         gpuDevices,
			GPUHashRate:        miner.GPUHashRate,
			GPUEnabled:         miner.GPUEnabled,
			HybridMode:         miner.HybridMode,
		}
	}

	api.writeJSON(w, http.StatusOK, response)
}

// handleBlockchain returns a JSON array containing all blocks in the
// blockchain from genesis to the latest block, including full block
// details (index, timestamp, data, hashes, nonce, and miner ID).
func (api *APIServer) handleBlockchain(w http.ResponseWriter, r *http.Request) {
	blocks := api.blockchain.GetBlockchain()
	api.writeJSON(w, http.StatusOK, blocks)
}

// handleBlock returns JSON details for a single block specified by index
// in the URL path (/api/blocks/{index}). Returns 404 if the block index
// is out of range.
func (api *APIServer) handleBlock(w http.ResponseWriter, r *http.Request) {
	// Extract block index from URL
	var index int
	if n, err := fmt.Sscanf(r.URL.Path, "/api/blocks/%d", &index); err != nil || n != 1 {
		http.Error(w, "Invalid block index", http.StatusBadRequest)
		return
	}

	blocks := api.blockchain.GetBlockchain()
	if index >= 0 && index < len(blocks) {
		api.writeJSON(w, http.StatusOK, blocks[index])
	} else {
		http.Error(w, "Block not found", http.StatusNotFound)
	}
}

// handleValidate performs a full blockchain validation check, verifying
// all block linkages and proof-of-work requirements. Returns JSON with a
// boolean "valid" field indicating whether the chain is valid.
func (api *APIServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	valid := api.blockchain.ValidateChain()
	response := map[string]interface{}{
		"valid": valid,
	}
	api.writeJSON(w, http.StatusOK, response)
}

// handleCPUStats returns detailed JSON statistics for all miners including
// per-miner CPU usage, hash rates, mining time, GPU information, and
// aggregate totals. This provides comprehensive resource usage data for
// monitoring and analysis.
func (api *APIServer) handleCPUStats(w http.ResponseWriter, r *http.Request) {
	stats := api.pool.GetCPUStats()
	api.writeJSON(w, http.StatusOK, stats)
}

// handlePauseMiner remotely pauses mining on a specific miner by setting
// its ShouldMine flag to false. The miner will stop mining on its next
// heartbeat check. Requires POST with JSON body containing "miner_id".
// Returns 404 if miner not found.
func (api *APIServer) handlePauseMiner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MinerID string `json:"miner_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := api.pool.PauseMiner(req.MinerID); err != nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]string{
		"status":   "success",
		"message":  "Miner paused",
		"miner_id": req.MinerID,
	})
}

// handleResumeMiner remotely resumes mining on a paused miner by setting
// its ShouldMine flag to true. The miner will restart mining on its next
// heartbeat check. Requires POST with JSON body containing "miner_id".
// Returns 404 if miner not found.
func (api *APIServer) handleResumeMiner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MinerID string `json:"miner_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := api.pool.ResumeMiner(req.MinerID); err != nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]string{
		"status":   "success",
		"message":  "Miner resumed",
		"miner_id": req.MinerID,
	})
}

// handleDeleteMiner removes a miner from the pool entirely. On the next
// heartbeat, the miner will receive Active=false and trigger self-deletion
// of its executable. Requires POST with JSON body containing "miner_id".
// Returns 404 if miner not found.
func (api *APIServer) handleDeleteMiner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MinerID string `json:"miner_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := api.pool.DeleteMiner(req.MinerID); err != nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]string{
		"status":   "success",
		"message":  "Miner deleted",
		"miner_id": req.MinerID,
	})
}

// handleThrottleMiner sets CPU usage limits for a specific miner. The
// throttle_percent value (0-100) controls CPU consumption, where 0 means
// unlimited and 100 means maximum throttling. Requires POST with JSON body
// containing "miner_id" and "throttle_percent". Returns 400 if percentage
// is out of range, 404 if miner not found.
func (api *APIServer) handleThrottleMiner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MinerID         string `json:"miner_id"`
		ThrottlePercent int32  `json:"throttle_percent"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := api.pool.SetCPUThrottle(req.MinerID, req.ThrottlePercent); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":           "success",
		"message":          fmt.Sprintf("CPU throttle set to %d%%", req.ThrottlePercent),
		"miner_id":         req.MinerID,
		"throttle_percent": req.ThrottlePercent,
	})
}
