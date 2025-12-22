package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAPIServer(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	authToken := "test-token"

	api := NewAPIServer(pool, bc, authToken, false, "", "")

	if api == nil {
		t.Fatal("NewAPIServer returned nil")
	}

	if api.pool != pool {
		t.Error("API pool reference mismatch")
	}

	if api.blockchain != bc {
		t.Error("API blockchain reference mismatch")
	}

	if api.authToken != authToken {
		t.Error("API auth token mismatch")
	}
}

func TestAuthMiddleware(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	handler := api.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	tests := []struct {
		name           string
		token          string
		expectedStatus int
	}{
		{"Valid token", "Bearer test-token", http.StatusOK},
		{"Invalid token", "Bearer wrong-token", http.StatusServiceUnavailable},
		{"No token", "", http.StatusServiceUnavailable},
		{"Malformed token", "test-token", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}

			w := httptest.NewRecorder()
			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestHandleStats(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	pool.RegisterMiner("miner-1", "192.168.1.100", "host1", "192.168.1.100")
	pool.UpdateHeartbeat("miner-1", 1000000, 50.0, 5000000, 30*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.handleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var stats PoolStats
	err := json.NewDecoder(w.Body).Decode(&stats)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if stats.TotalMiners != 1 {
		t.Errorf("Expected 1 total miner, got %d", stats.TotalMiners)
	}

	if stats.BlockReward != 50 {
		t.Errorf("Expected block reward 50, got %d", stats.BlockReward)
	}
}

func TestHandleMiners(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	pool.RegisterMiner("miner-1", "192.168.1.100", "host1", "192.168.1.100")
	pool.RegisterMiner("miner-2", "192.168.1.101", "host2", "192.168.1.101")

	req := httptest.NewRequest(http.MethodGet, "/api/miners", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.handleMiners(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var miners []map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&miners)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(miners) != 2 {
		t.Errorf("Expected 2 miners, got %d", len(miners))
	}
}

func TestHandleBlockchain(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	req := httptest.NewRequest(http.MethodGet, "/api/blockchain", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.handleBlockchain(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var blocks []*Block
	err := json.NewDecoder(w.Body).Decode(&blocks)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(blocks) != 1 {
		t.Errorf("Expected 1 block (genesis), got %d", len(blocks))
	}

	if blocks[0].Index != 0 {
		t.Errorf("Expected genesis block index 0, got %d", blocks[0].Index)
	}
}

func TestHandleBlock(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	// Test valid block
	req := httptest.NewRequest(http.MethodGet, "/api/blocks/0", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.handleBlock(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var block Block
	err := json.NewDecoder(w.Body).Decode(&block)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if block.Index != 0 {
		t.Errorf("Expected block index 0, got %d", block.Index)
	}
}

func TestHandleBlockInvalid(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	// Test invalid block index
	req := httptest.NewRequest(http.MethodGet, "/api/blocks/999", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.handleBlock(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleValidate(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	req := httptest.NewRequest(http.MethodGet, "/api/validate", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.handleValidate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	valid, ok := result["valid"].(bool)
	if !ok {
		t.Fatal("Response should contain 'valid' boolean field")
	}

	if !valid {
		t.Error("New blockchain should be valid")
	}
}

func TestHandleCPUStats(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	pool.RegisterMiner("miner-1", "192.168.1.100", "host1", "192.168.1.100")
	pool.UpdateHeartbeat("miner-1", 1000000, 50.0, 5000000, 30*time.Second)

	req := httptest.NewRequest(http.MethodGet, "/api/cpu", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.handleCPUStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var stats TotalCPUStats
	err := json.NewDecoder(w.Body).Decode(&stats)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if stats.TotalMiners != 1 {
		t.Errorf("Expected 1 total miner, got %d", stats.TotalMiners)
	}

	if len(stats.MinerStats) != 1 {
		t.Errorf("Expected 1 miner stat, got %d", len(stats.MinerStats))
	}
}

func TestHandlePauseMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	reqBody := map[string]string{"miner_id": minerID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/miner/pause", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handlePauseMiner(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)

	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got '%s'", result["status"])
	}

	// Verify miner is paused
	shouldMine, _ := pool.GetMinerStatus(minerID)
	if shouldMine {
		t.Error("Miner should be paused")
	}
}

func TestHandlePauseMinerNotFound(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	reqBody := map[string]string{"miner_id": "nonexistent"}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/miner/pause", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handlePauseMiner(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestHandleResumeMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")
	pool.PauseMiner(minerID)

	reqBody := map[string]string{"miner_id": minerID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/miner/resume", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleResumeMiner(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)

	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got '%s'", result["status"])
	}

	// Verify miner is resumed
	shouldMine, _ := pool.GetMinerStatus(minerID)
	if !shouldMine {
		t.Error("Miner should be mining after resume")
	}
}

func TestHandleDeleteMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	reqBody := map[string]string{"miner_id": minerID}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/miner/delete", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleDeleteMiner(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]string
	json.NewDecoder(w.Body).Decode(&result)

	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got '%s'", result["status"])
	}

	// Verify miner is deleted
	pool.mu.RLock()
	_, exists := pool.miners[minerID]
	pool.mu.RUnlock()

	if exists {
		t.Error("Miner should be deleted")
	}
}

func TestHandleThrottleMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	reqBody := map[string]interface{}{
		"miner_id":         minerID,
		"throttle_percent": 50,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/miner/throttle", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleThrottleMiner(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)

	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", result["status"])
	}

	// Verify throttle was set
	throttle, _ := pool.GetCPUThrottle(minerID)
	if throttle != 50 {
		t.Errorf("Expected throttle 50, got %d", throttle)
	}
}

func TestHandleThrottleMinerInvalidValue(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	reqBody := map[string]interface{}{
		"miner_id":         minerID,
		"throttle_percent": 150, // Invalid value > 100
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/miner/throttle", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handleThrottleMiner(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleIndexWithAuth(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	// Test with header auth
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html" {
		t.Errorf("Expected content type 'text/html', got '%s'", contentType)
	}
}

func TestHandleIndexWithQueryToken(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	// Test with query parameter auth
	req := httptest.NewRequest(http.MethodGet, "/?token=test-token", nil)
	w := httptest.NewRecorder()

	api.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleIndexNoAuth(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	api.handleIndex(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestHandleMethodNotAllowed(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	// Test pause endpoint with wrong method
	req := httptest.NewRequest(http.MethodGet, "/api/miner/pause", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	api.handlePauseMiner(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleInvalidJSON(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	api := NewAPIServer(pool, bc, "test-token", false, "", "")

	// Test with invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/miner/pause", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	api.handlePauseMiner(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
