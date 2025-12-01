package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// APIServer provides HTTP API for administration
type APIServer struct {
	pool       *MiningPool
	blockchain *Blockchain
	authToken  string
	useTLS     bool
	certFile   string
	keyFile    string
}

// NewAPIServer creates a new API server
func NewAPIServer(pool *MiningPool, blockchain *Blockchain, authToken string, useTLS bool, certFile, keyFile string) *APIServer {
	return &APIServer{
		pool:       pool,
		blockchain: blockchain,
		authToken:  authToken,
		useTLS:     useTLS,
		certFile:   certFile,
		keyFile:    keyFile,
	}
}

// authMiddleware checks for valid authentication token
func (api *APIServer) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		token := r.Header.Get("Authorization")

		// Check if token matches
		if token != "Bearer "+api.authToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Unauthorized - Invalid or missing authentication token",
			})
			return
		}

		// Token is valid, proceed to handler
		next(w, r)
	}
}

// Start starts the API server
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

	// Public endpoint - no authentication required
	mux.HandleFunc("/", api.handleIndex)

	addr := fmt.Sprintf(":%d", port)

	if api.useTLS {
		// Start HTTP to HTTPS redirect server if httpPort is provided
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

		return http.ListenAndServeTLS(addr, api.certFile, api.keyFile, mux)
	}

	// Start HTTP server
	fmt.Printf("Starting API server on http://localhost%s\n", addr)
	fmt.Printf("API authentication enabled - token required in Authorization header\n")
	fmt.Printf("WARNING: TLS is disabled - connections are not encrypted\n")
	return http.ListenAndServe(addr, mux)
}

// startHTTPRedirect starts an HTTP server that redirects to HTTPS
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

	if err := http.ListenAndServe(httpAddr, redirect); err != nil {
		log.Printf("HTTP redirect server error: %v", err)
	}
}

// handleIndex provides a simple HTML dashboard
func (api *APIServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
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

        function updateMiners(miners) {
            let html = '<table><tr><th>Miner ID</th><th>IP Address</th><th>Hostname</th><th>Status</th><th>Mining</th><th>CPU Throttle</th><th>Blocks Mined</th><th>Hash Rate</th><th>Last Heartbeat</th><th>Actions</th></tr>';
            miners.forEach(miner => {
                const status = miner.Active ? '<span class="active">Active</span>' : '<span class="inactive">Inactive</span>';
                const miningStatus = miner.ShouldMine ? '<span class="active">Mining</span>' : '<span class="inactive">Paused</span>';
                const throttleDisplay = miner.CPUThrottlePercent === 0 ? 'None' : miner.CPUThrottlePercent + '%';
                const lastSeen = new Date(miner.LastHeartbeat).toLocaleString();
                const pauseResumeBtn = miner.ShouldMine
                    ? '<button onclick="controlMiner(\'pause\', \'' + miner.ID + '\')">Pause</button>'
                    : '<button onclick="controlMiner(\'resume\', \'' + miner.ID + '\')">Resume</button>';
                const throttleBtn = '<button onclick="setThrottle(\'' + miner.ID + '\')">Throttle</button>';
                const deleteBtn = '<button onclick="if(confirm(\'Delete miner ' + miner.ID + '?\')) controlMiner(\'delete\', \'' + miner.ID + '\')">Delete</button>';
                html += '<tr><td>' + miner.ID + '</td><td>' + miner.IPAddress + '</td><td>' + miner.Hostname + '</td><td>' + status + '</td><td>' + miningStatus + '</td><td>' + throttleDisplay + '</td><td>' + miner.BlocksMined + '</td><td>' + miner.HashRate + ' H/s</td><td>' + lastSeen + '</td><td>' + pauseResumeBtn + ' ' + throttleBtn + ' ' + deleteBtn + '</td></tr>';
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
	w.Write([]byte(html))
}

// handleStats returns pool statistics
func (api *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	stats := api.pool.GetPoolStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleMiners returns list of all miners
func (api *APIServer) handleMiners(w http.ResponseWriter, r *http.Request) {
	miners := api.pool.GetMiners()

	type MinerResponse struct {
		ID                 string    `json:"ID"`
		IPAddress          string    `json:"IPAddress"`
		IPAddressActual    string    `json:"IPAddressActual"`
		Hostname           string    `json:"Hostname"`
		RegisteredAt       time.Time `json:"RegisteredAt"`
		LastHeartbeat      time.Time `json:"LastHeartbeat"`
		Active             bool      `json:"Active"`
		ShouldMine         bool      `json:"ShouldMine"`
		CPUThrottlePercent int       `json:"CPUThrottlePercent"`
		BlocksMined        int64     `json:"BlocksMined"`
		HashRate           int64     `json:"HashRate"`
	}

	response := make([]MinerResponse, len(miners))
	for i, miner := range miners {
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
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleBlockchain returns the entire blockchain
func (api *APIServer) handleBlockchain(w http.ResponseWriter, r *http.Request) {
	blocks := api.blockchain.GetBlockchain()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blocks)
}

// handleBlock returns a specific block
func (api *APIServer) handleBlock(w http.ResponseWriter, r *http.Request) {
	// Extract block index from URL
	var index int
	fmt.Sscanf(r.URL.Path, "/api/blocks/%d", &index)

	blocks := api.blockchain.GetBlockchain()
	if index >= 0 && index < len(blocks) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(blocks[index])
	} else {
		http.Error(w, "Block not found", http.StatusNotFound)
	}
}

// handleValidate validates the blockchain
func (api *APIServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	valid := api.blockchain.ValidateChain()
	response := map[string]interface{}{
		"valid": valid,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCPUStats returns CPU usage statistics for all miners
func (api *APIServer) handleCPUStats(w http.ResponseWriter, r *http.Request) {
	stats := api.pool.GetCPUStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handlePauseMiner pauses mining for a specific miner
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"message":  "Miner paused",
		"miner_id": req.MinerID,
	})
}

// handleResumeMiner resumes mining for a specific miner
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"message":  "Miner resumed",
		"miner_id": req.MinerID,
	})
}

// handleDeleteMiner deletes a miner from the pool
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"message":  "Miner deleted",
		"miner_id": req.MinerID,
	})
}

// handleThrottleMiner sets the CPU throttle percentage for a miner
func (api *APIServer) handleThrottleMiner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MinerID          string `json:"miner_id"`
		ThrottlePercent  int    `json:"throttle_percent"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := api.pool.SetCPUThrottle(req.MinerID, req.ThrottlePercent); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":           "success",
		"message":          fmt.Sprintf("CPU throttle set to %d%%", req.ThrottlePercent),
		"miner_id":         req.MinerID,
		"throttle_percent": req.ThrottlePercent,
	})
}
