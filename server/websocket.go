// Package main implements the RedTeamCoin mining pool server components.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// upgrader configures the WebSocket connection upgrader with permissive
// CORS settings for development. In production, CheckOrigin should be
// configured to validate allowed origins.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WebMiner represents a browser-based cryptocurrency miner connected via
// WebSocket. It tracks the miner's configuration, performance statistics,
// and connection state. All fields are protected by an internal mutex
// for thread-safe access.
type WebMiner struct {
	ID        string
	Conn      *websocket.Conn
	UserAgent string
	Threads   int
	HasGPU    bool
	Version   string
	HashRate  int64
	Hashes    int64
	Blocks    int64
	JoinedAt  time.Time
	LastSeen  time.Time
	mu        sync.Mutex
}

// WebSocketHub manages all connected web-based miners and coordinates
// message broadcasting. It maintains miner connections, handles miner
// lifecycle events (registration/disconnection), and provides work
// distribution for browser-based mining. All operations are thread-safe.
type WebSocketHub struct {
	miners    map[string]*WebMiner
	broadcast chan []byte
	register  chan *WebMiner
	unregist  chan *WebMiner
	pool      *MiningPool
	mu        sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub that coordinates web-based
// miners for the given mining pool. The hub must be started with Run
// before it can process connections.
func NewWebSocketHub(pool *MiningPool) *WebSocketHub {
	return &WebSocketHub{
		miners:    make(map[string]*WebMiner),
		broadcast: make(chan []byte, 256),
		register:  make(chan *WebMiner),
		unregist:  make(chan *WebMiner),
		pool:      pool,
	}
}

// Run starts the WebSocket hub's main event loop, processing miner
// registrations, disconnections, and broadcast messages. The context
// allows graceful shutdown of the hub. This method blocks and should
// be run in a goroutine.
func (h *WebSocketHub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Gracefully close all connections
			h.mu.Lock()
			for _, miner := range h.miners {
				if err := miner.Conn.Close(); err != nil {
					log.Printf("[WebSocket] Error closing connection for %s: %v", miner.ID, err)
				}
			}
			h.mu.Unlock()
			log.Printf("[WebSocket] Hub shutting down")
			return

		case miner := <-h.register:
			h.mu.Lock()
			h.miners[miner.ID] = miner
			h.mu.Unlock()
			log.Printf("[WebSocket] Miner registered: %s (threads: %d, gpu: %v)", miner.ID, miner.Threads, miner.HasGPU)

		case miner := <-h.unregist:
			h.mu.Lock()
			if _, ok := h.miners[miner.ID]; ok {
				delete(h.miners, miner.ID)
				if err := miner.Conn.Close(); err != nil {
					log.Printf("[WebSocket] Error closing connection for %s: %v", miner.ID, err)
				}
			}
			h.mu.Unlock()
			log.Printf("[WebSocket] Miner disconnected: %s", miner.ID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, miner := range h.miners {
				miner.mu.Lock()
				err := miner.Conn.WriteMessage(websocket.TextMessage, message)
				miner.mu.Unlock()
				if err != nil {
					log.Printf("[WebSocket] Write error for %s: %v", miner.ID, err)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// WSMessage represents the base structure for all WebSocket messages
// exchanged between the server and browser-based miners. The Type field
// determines how to interpret the Data payload.
type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// WSRegister contains miner registration information sent by browser
// clients when first connecting to the pool.
type WSRegister struct {
	MinerID   string `json:"minerId"`
	UserAgent string `json:"userAgent"`
	Threads   int    `json:"threads"`
	HasGPU    bool   `json:"hasGPU"`
	Version   string `json:"version"`
}

// WSWork represents a mining work unit sent from the server to browser
// miners. It contains all parameters needed to perform proof-of-work.
type WSWork struct {
	BlockIndex   int64  `json:"blockIndex"`
	PreviousHash string `json:"previousHash"`
	Data         string `json:"data"`
	Difficulty   int    `json:"difficulty"`
	Timestamp    int64  `json:"timestamp"`
}

// WSSubmit represents a work submission from a browser miner containing
// a potential solution (nonce and hash) for the assigned block.
type WSSubmit struct {
	MinerID    string `json:"minerId"`
	BlockIndex int64  `json:"blockIndex"`
	Nonce      int64  `json:"nonce"`
	Hash       string `json:"hash"`
}

// WSStats contains real-time mining statistics sent periodically from
// browser miners to the server for monitoring and dashboard display.
type WSStats struct {
	MinerID     string `json:"minerId"`
	HashRate    int64  `json:"hashRate"`
	TotalHashes int64  `json:"totalHashes"`
	BlocksFound int64  `json:"blocksFound"`
	Uptime      int64  `json:"uptime"`
}

// HandleWebSocket handles HTTP requests to upgrade to WebSocket
// connections for browser-based miners. It upgrades the connection
// and spawns a goroutine to handle miner messages.
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v", err)
		return
	}

	miner := &WebMiner{
		Conn:     conn,
		JoinedAt: time.Now(),
		LastSeen: time.Now(),
	}

	// Read messages
	go h.handleMiner(miner)
}

// handleMiner processes all WebSocket messages from a connected browser miner
// until the connection closes. It routes messages to appropriate handlers
// based on message type (register, getwork, submit, stats). The miner is
// automatically unregistered when this function returns.
func (h *WebSocketHub) handleMiner(miner *WebMiner) {
	defer func() {
		h.unregist <- miner
	}()

	for {
		_, message, err := miner.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Read error: %v", err)
			}
			break
		}

		miner.LastSeen = time.Now()

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[WebSocket] Invalid message: %v", err)
			continue
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "register":
			h.handleRegister(miner, msg)

		case "getwork":
			h.handleGetWork(miner)

		case "submit":
			h.handleSubmit(miner, msg)

		case "stats":
			h.handleStats(miner, msg)
		}
	}
}

// handleRegister processes miner registration messages from browser clients,
// extracting miner ID, user agent, thread count, GPU capability, and version
// information. It registers the miner with the hub and sends back confirmation
// along with initial mining work.
func (h *WebSocketHub) handleRegister(miner *WebMiner, msg map[string]interface{}) {
	if id, ok := msg["minerId"].(string); ok {
		miner.ID = id
	}
	if ua, ok := msg["userAgent"].(string); ok {
		miner.UserAgent = ua
	}
	if threads, ok := msg["threads"].(float64); ok {
		miner.Threads = int(threads)
	}
	if gpu, ok := msg["hasGPU"].(bool); ok {
		miner.HasGPU = gpu
	}
	if ver, ok := msg["version"].(string); ok {
		miner.Version = ver
	}

	h.register <- miner

	// Send confirmation
	h.sendToMiner(miner, map[string]interface{}{
		"type":    "registered",
		"success": true,
		"message": "Welcome to RedTeamCoin pool",
	})

	// Send initial work
	h.handleGetWork(miner)
}

// handleGetWork assigns a new mining work unit to a browser miner. It
// registers the miner with the pool if not already registered, retrieves
// a work block, and sends it to the miner via WebSocket. Sends an error
// message if work cannot be retrieved.
func (h *WebSocketHub) handleGetWork(miner *WebMiner) {
	// Register web miner with the pool first if not already registered
	if err := h.pool.RegisterMiner(miner.ID, "web", "browser", "web-client"); err != nil {
		log.Printf("[WebSocket] Error registering miner %s: %v", miner.ID, err)
	}

	// Get work from the pool
	block, err := h.pool.GetWork(miner.ID)
	if err != nil || block == nil {
		h.sendToMiner(miner, map[string]interface{}{
			"type":    "error",
			"message": "No work available: " + err.Error(),
		})
		return
	}

	h.sendToMiner(miner, map[string]interface{}{
		"type": "work",
		"work": map[string]interface{}{
			"blockIndex":   block.Index,
			"previousHash": block.PreviousHash,
			"data":         block.Data,
			"difficulty":   4, // Default difficulty
			"timestamp":    block.Timestamp,
		},
	})
}

// handleSubmit processes a block solution submission from a browser miner.
// It validates the solution with the pool, increments the miner's block
// count if accepted, sends back acceptance status and reward information,
// and automatically assigns new work if the submission was accepted.
func (h *WebSocketHub) handleSubmit(miner *WebMiner, msg map[string]interface{}) {
	blockIndex := int64(msg["blockIndex"].(float64))
	nonce := int64(msg["nonce"].(float64))
	hash := msg["hash"].(string)

	// Verify and submit to pool
	accepted, reward, err := h.pool.SubmitWork(miner.ID, blockIndex, nonce, hash)

	message := "Block submitted"
	if err != nil {
		message = err.Error()
	} else if accepted {
		message = fmt.Sprintf("Block accepted! Reward: %d", reward)
	}

	if accepted {
		miner.Blocks++
		log.Printf("[WebSocket] Block accepted from %s: index=%d, nonce=%d, reward=%d", miner.ID, blockIndex, nonce, reward)
	}

	h.sendToMiner(miner, map[string]interface{}{
		"type":     "accepted",
		"accepted": accepted,
		"message":  message,
		"reward":   reward,
	})

	// Send new work after submission
	if accepted {
		h.handleGetWork(miner)
	}
}

// handleStats processes periodic statistics updates from browser miners,
// updating the miner's hash rate, total hashes computed, and blocks found.
// These statistics are used for monitoring and dashboard display.
func (h *WebSocketHub) handleStats(miner *WebMiner, msg map[string]interface{}) {
	if hashRate, ok := msg["hashRate"].(float64); ok {
		miner.HashRate = int64(hashRate)
	}
	if hashes, ok := msg["totalHashes"].(float64); ok {
		miner.Hashes = int64(hashes)
	}
	if blocks, ok := msg["blocksFound"].(float64); ok {
		miner.Blocks = int64(blocks)
	}
}

// sendToMiner sends a JSON-encoded message to a specific browser miner
// via WebSocket. It marshals the message to JSON, acquires the miner's
// connection lock, and writes the message. Logs any marshaling or sending
// errors.
func (h *WebSocketHub) sendToMiner(miner *WebMiner, msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[WebSocket] Marshal error: %v", err)
		return
	}

	miner.mu.Lock()
	defer miner.mu.Unlock()

	if err := miner.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("[WebSocket] Send error to %s: %v", miner.ID, err)
	}
}

// GetWebMinersStats returns current statistics for all connected
// web-based miners including hash rates, uptime, and block counts.
// This method is safe for concurrent access.
func (h *WebSocketHub) GetWebMinersStats() []map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := make([]map[string]interface{}, 0, len(h.miners))
	for _, miner := range h.miners {
		stats = append(stats, map[string]interface{}{
			"id":        miner.ID,
			"threads":   miner.Threads,
			"hasGPU":    miner.HasGPU,
			"hashRate":  miner.HashRate,
			"hashes":    miner.Hashes,
			"blocks":    miner.Blocks,
			"uptime":    time.Since(miner.JoinedAt).Seconds(),
			"lastSeen":  miner.LastSeen.Format(time.RFC3339),
			"userAgent": miner.UserAgent,
		})
	}
	return stats
}

// BroadcastWork sends new mining work to all connected web-based miners
// simultaneously. This is useful when the pool wants to reassign work
// or notify miners of new opportunities.
func (h *WebSocketHub) BroadcastWork(work *WSWork) {
	msg, err := json.Marshal(map[string]interface{}{
		"type": "work",
		"work": work,
	})
	if err != nil {
		log.Printf("[WebSocket] Broadcast marshal error: %v", err)
		return
	}
	h.broadcast <- msg
}

// Work represents a mining work unit from the pool containing all
// parameters needed for miners to perform proof-of-work computation.
type Work struct {
	BlockIndex   int64
	PreviousHash string
	Data         string
	Difficulty   int
	Timestamp    int64
}
