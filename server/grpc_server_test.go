package main

import (
	"context"
	"testing"

	pb "redteamcoin/proto"

	"google.golang.org/grpc/peer"
)

func TestNewMiningPoolServer(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	if server == nil {
		t.Fatal("NewMiningPoolServer returned nil")
	}

	if server.pool != pool {
		t.Error("Server pool reference mismatch")
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		expected string
	}{
		{"IPv4 with port", "192.168.1.100:12345", "192.168.1.100"},
		{"IPv6 with port", "[::1]:12345", "::1"},
		{"IPv4 localhost", "127.0.0.1:8080", "127.0.0.1"},
		{"No port", "192.168.1.100", "192.168.1.100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context with peer information
			p := &peer.Peer{
				Addr: &testAddr{addr: tt.addr},
			}
			ctx := peer.NewContext(context.Background(), p)

			result := getClientIP(ctx)
			if result != tt.expected {
				t.Errorf("Expected IP %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetClientIPNoPeer(t *testing.T) {
	ctx := context.Background()
	result := getClientIP(ctx)
	if result != "unknown" {
		t.Errorf("Expected 'unknown', got %s", result)
	}
}

func TestRegisterMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	req := &pb.MinerInfo{
		MinerId:   "test-miner-1",
		IpAddress: "192.168.1.100",
		Hostname:  "test-host",
	}

	p := &peer.Peer{
		Addr: &testAddr{addr: "192.168.1.100:12345"},
	}
	ctx := peer.NewContext(context.Background(), p)

	resp, err := server.RegisterMiner(ctx, req)
	if err != nil {
		t.Fatalf("RegisterMiner failed: %v", err)
	}

	if !resp.Success {
		t.Error("Registration should be successful")
	}

	if resp.MinerId != req.MinerId {
		t.Errorf("Expected miner ID %s, got %s", req.MinerId, resp.MinerId)
	}

	if resp.Message != "Miner registered successfully" {
		t.Errorf("Unexpected message: %s", resp.Message)
	}
}

func TestRegisterMinerDuplicate(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	req := &pb.MinerInfo{
		MinerId:   "test-miner-1",
		IpAddress: "192.168.1.100",
		Hostname:  "test-host",
	}

	p := &peer.Peer{
		Addr: &testAddr{addr: "192.168.1.100:12345"},
	}
	ctx := peer.NewContext(context.Background(), p)

	// Register first time
	resp1, _ := server.RegisterMiner(ctx, req)
	if !resp1.Success {
		t.Fatal("First registration should succeed")
	}

	// Register second time (update)
	resp2, _ := server.RegisterMiner(ctx, req)
	if !resp2.Success {
		t.Error("Re-registration should succeed (update)")
	}
}

func TestGetWork(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	req := &pb.WorkRequest{
		MinerId: minerID,
	}

	resp, err := server.GetWork(context.Background(), req)
	if err != nil {
		t.Fatalf("GetWork failed: %v", err)
	}

	if resp.BlockIndex != 1 {
		t.Errorf("Expected block index 1, got %d", resp.BlockIndex)
	}

	if resp.Difficulty != 4 {
		t.Errorf("Expected difficulty 4, got %d", resp.Difficulty)
	}

	if resp.PreviousHash == "" {
		t.Error("Previous hash should not be empty")
	}

	if resp.Data == "" {
		t.Error("Block data should not be empty")
	}
}

func TestGetWorkUnregisteredMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	req := &pb.WorkRequest{
		MinerId: "unregistered-miner",
	}

	_, err := server.GetWork(context.Background(), req)
	if err == nil {
		t.Error("GetWork should fail for unregistered miner")
	}
}

func TestSubmitWork(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	// Get work
	workReq := &pb.WorkRequest{MinerId: minerID}
	workResp, _ := server.GetWork(context.Background(), workReq)

	// Mine the block
	block := &Block{
		Index:        workResp.BlockIndex,
		Timestamp:    workResp.Timestamp,
		Data:         workResp.Data,
		PreviousHash: workResp.PreviousHash,
		Nonce:        0,
	}

	for {
		block.Hash = calculateHash(block)
		if len(block.Hash) >= int(bc.Difficulty) && block.Hash[:bc.Difficulty] == "0000" {
			break
		}
		block.Nonce++
	}

	// Submit work
	submitReq := &pb.WorkSubmission{
		MinerId:    minerID,
		BlockIndex: block.Index,
		Nonce:      block.Nonce,
		Hash:       block.Hash,
	}

	resp, err := server.SubmitWork(context.Background(), submitReq)
	if err != nil {
		t.Fatalf("SubmitWork failed: %v", err)
	}

	if !resp.Accepted {
		t.Errorf("Work should be accepted: %s", resp.Message)
	}

	if resp.Reward != 50 {
		t.Errorf("Expected reward 50, got %d", resp.Reward)
	}
}

func TestSubmitWorkInvalid(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	// Submit invalid work (no pending work)
	submitReq := &pb.WorkSubmission{
		MinerId:    minerID,
		BlockIndex: 1,
		Nonce:      0,
		Hash:       "invalid_hash",
	}

	resp, err := server.SubmitWork(context.Background(), submitReq)
	if err != nil {
		t.Fatalf("SubmitWork returned error: %v", err)
	}

	if resp.Accepted {
		t.Error("Invalid work should not be accepted")
	}
}

func TestHeartbeat(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	req := &pb.MinerStatus{
		MinerId:           minerID,
		HashRate:          1000000,
		CpuUsagePercent:   75.5,
		TotalHashes:       5000000,
		MiningTimeSeconds: 30,
		GpuDevices:        []*pb.GPUDevice{},
		GpuHashRate:       0,
		GpuEnabled:        false,
		HybridMode:        false,
	}

	resp, err := server.Heartbeat(context.Background(), req)
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	if !resp.Active {
		t.Error("Heartbeat should be active")
	}

	if !resp.ShouldMine {
		t.Error("Miner should be mining by default")
	}

	if resp.CpuThrottlePercent != 0 {
		t.Errorf("Expected no throttling (0), got %d", resp.CpuThrottlePercent)
	}
}

func TestHeartbeatWithGPU(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	gpuDevices := []*pb.GPUDevice{
		{
			Id:           0,
			Name:         "NVIDIA RTX 3080",
			Type:         "CUDA",
			Memory:       10737418240,
			ComputeUnits: 68,
			Available:    true,
		},
	}

	req := &pb.MinerStatus{
		MinerId:           minerID,
		HashRate:          1000000,
		CpuUsagePercent:   50.0,
		TotalHashes:       5000000,
		MiningTimeSeconds: 30,
		GpuDevices:        gpuDevices,
		GpuHashRate:       5000000,
		GpuEnabled:        true,
		HybridMode:        true,
	}

	resp, err := server.Heartbeat(context.Background(), req)
	if err != nil {
		t.Fatalf("Heartbeat with GPU failed: %v", err)
	}

	if !resp.Active {
		t.Error("Heartbeat should be active")
	}

	// Verify GPU stats were recorded
	pool.mu.RLock()
	miner := pool.miners[minerID]
	pool.mu.RUnlock()

	if !miner.GPUEnabled {
		t.Error("GPU should be enabled")
	}

	if !miner.HybridMode {
		t.Error("Hybrid mode should be enabled")
	}

	if len(miner.GPUDevices) != 1 {
		t.Errorf("Expected 1 GPU device, got %d", len(miner.GPUDevices))
	}
}

func TestHeartbeatUnregisteredMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	req := &pb.MinerStatus{
		MinerId:           "unregistered-miner",
		HashRate:          1000000,
		CpuUsagePercent:   75.5,
		TotalHashes:       5000000,
		MiningTimeSeconds: 30,
		GpuDevices:        []*pb.GPUDevice{},
		GpuHashRate:       0,
		GpuEnabled:        false,
		HybridMode:        false,
	}

	resp, err := server.Heartbeat(context.Background(), req)
	if err != nil {
		t.Fatalf("Heartbeat returned error: %v", err)
	}

	if resp.Active {
		t.Error("Heartbeat for unregistered miner should return inactive")
	}

	if resp.ShouldMine {
		t.Error("Unregistered miner should not be mining")
	}
}

func TestHeartbeatPausedMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")
	pool.PauseMiner(minerID)

	req := &pb.MinerStatus{
		MinerId:           minerID,
		HashRate:          1000000,
		CpuUsagePercent:   75.5,
		TotalHashes:       5000000,
		MiningTimeSeconds: 30,
		GpuDevices:        []*pb.GPUDevice{},
		GpuHashRate:       0,
		GpuEnabled:        false,
		HybridMode:        false,
	}

	resp, err := server.Heartbeat(context.Background(), req)
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	if !resp.Active {
		t.Error("Heartbeat should be active")
	}

	if resp.ShouldMine {
		t.Error("Paused miner should not be mining")
	}
}

func TestHeartbeatThrottledMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")
	pool.SetCPUThrottle(minerID, 50)

	req := &pb.MinerStatus{
		MinerId:           minerID,
		HashRate:          1000000,
		CpuUsagePercent:   75.5,
		TotalHashes:       5000000,
		MiningTimeSeconds: 30,
		GpuDevices:        []*pb.GPUDevice{},
		GpuHashRate:       0,
		GpuEnabled:        false,
		HybridMode:        false,
	}

	resp, err := server.Heartbeat(context.Background(), req)
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	if resp.CpuThrottlePercent != 50 {
		t.Errorf("Expected throttle 50%%, got %d%%", resp.CpuThrottlePercent)
	}
}

func TestHeartbeatDeletedMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")
	pool.DeleteMiner(minerID)

	req := &pb.MinerStatus{
		MinerId:           minerID,
		HashRate:          1000000,
		CpuUsagePercent:   75.5,
		TotalHashes:       5000000,
		MiningTimeSeconds: 30,
		GpuDevices:        []*pb.GPUDevice{},
		GpuHashRate:       0,
		GpuEnabled:        false,
		HybridMode:        false,
	}

	resp, err := server.Heartbeat(context.Background(), req)
	if err != nil {
		t.Fatalf("Heartbeat returned error: %v", err)
	}

	if resp.Active {
		t.Error("Deleted miner should return inactive")
	}

	if resp.ShouldMine {
		t.Error("Deleted miner should not be mining")
	}

	if resp.Message != "Miner has been deleted from the pool" {
		t.Errorf("Unexpected message: %s", resp.Message)
	}
}

func TestStopMining(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	minerID := "test-miner-1"
	pool.RegisterMiner(minerID, "192.168.1.100", "test-host", "192.168.1.100")

	// Simulate some blocks mined
	pool.mu.Lock()
	pool.miners[minerID].BlocksMined = 5
	pool.mu.Unlock()

	req := &pb.MinerInfo{
		MinerId: minerID,
	}

	resp, err := server.StopMining(context.Background(), req)
	if err != nil {
		t.Fatalf("StopMining returned error: %v", err)
	}

	if !resp.Success {
		t.Error("StopMining should succeed")
	}

	if resp.TotalBlocksMined != 5 {
		t.Errorf("Expected 5 blocks mined, got %d", resp.TotalBlocksMined)
	}
}

func TestStopMiningUnregisteredMiner(t *testing.T) {
	bc := NewBlockchain(4)
	pool := NewMiningPool(bc)
	server := NewMiningPoolServer(pool)

	req := &pb.MinerInfo{
		MinerId: "unregistered-miner",
	}

	resp, err := server.StopMining(context.Background(), req)
	if err != nil {
		t.Fatalf("StopMining returned error: %v", err)
	}

	if resp.Success {
		t.Error("StopMining should fail for unregistered miner")
	}
}

// Helper type for testing peer addresses
type testAddr struct {
	addr string
}

func (t *testAddr) Network() string {
	return "tcp"
}

func (t *testAddr) String() string {
	return t.addr
}
