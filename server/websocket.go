package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// WebMiner represents a browser-based miner
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

// WebSocketHub manages all web miners
type WebSocketHub struct {
	miners    map[string]*WebMiner
	broadcast chan []byte
	register  chan *WebMiner
	unregist  chan *WebMiner
	pool      *MiningPool
	mu        sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub(pool *MiningPool) *WebSocketHub {
	return &WebSocketHub{
		miners:    make(map[string]*WebMiner),
		broadcast: make(chan []byte, 256),
		register:  make(chan *WebMiner),
		unregist:  make(chan *WebMiner),
		pool:      pool,
	}
}

// Run starts the hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case miner := <-h.register:
			h.mu.Lock()
			h.miners[miner.ID] = miner
			h.mu.Unlock()
			log.Printf("[WebSocket] Miner registered: %s (threads: %d, gpu: %v)", miner.ID, miner.Threads, miner.HasGPU)

		case miner := <-h.unregist:
			h.mu.Lock()
			if _, ok := h.miners[miner.ID]; ok {
				delete(h.miners, miner.ID)
				miner.Conn.Close()
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

// WebSocket message types
type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type WSRegister struct {
	MinerID   string `json:"minerId"`
	UserAgent string `json:"userAgent"`
	Threads   int    `json:"threads"`
	HasGPU    bool   `json:"hasGPU"`
	Version   string `json:"version"`
}

type WSWork struct {
	BlockIndex   int64  `json:"blockIndex"`
	PreviousHash string `json:"previousHash"`
	Data         string `json:"data"`
	Difficulty   int    `json:"difficulty"`
	Timestamp    int64  `json:"timestamp"`
}

type WSSubmit struct {
	MinerID    string `json:"minerId"`
	BlockIndex int64  `json:"blockIndex"`
	Nonce      int64  `json:"nonce"`
	Hash       string `json:"hash"`
}

type WSStats struct {
	MinerID     string `json:"minerId"`
	HashRate    int64  `json:"hashRate"`
	TotalHashes int64  `json:"totalHashes"`
	BlocksFound int64  `json:"blocksFound"`
	Uptime      int64  `json:"uptime"`
}

// HandleWebSocket handles WebSocket connections for web miners
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

func (h *WebSocketHub) handleGetWork(miner *WebMiner) {
	// Register web miner with the pool first if not already registered
	h.pool.RegisterMiner(miner.ID, "web", "browser", "web-client")

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

// GetWebMinersStats returns statistics for all web miners
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

// BroadcastWork sends new work to all connected web miners
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

// Work represents mining work from the pool
type Work struct {
	BlockIndex   int64
	PreviousHash string
	Data         string
	Difficulty   int
	Timestamp    int64
}
