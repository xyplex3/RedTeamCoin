package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	pb "redteamcoin/proto"
	"google.golang.org/grpc/peer"
)

// MiningPoolServer implements the gRPC mining pool service
type MiningPoolServer struct {
	pb.UnimplementedMiningPoolServer
	pool *MiningPool
}

// NewMiningPoolServer creates a new gRPC server
func NewMiningPoolServer(pool *MiningPool) *MiningPoolServer {
	return &MiningPoolServer{
		pool: pool,
	}
}

// getClientIP extracts the actual IP address from gRPC context
func getClientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "unknown"
	}

	addr := p.Addr.String()
	// Extract IP from "IP:port" format
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		// If splitting fails, return the whole address
		return addr
	}

	// Remove IPv6 brackets if present
	host = strings.Trim(host, "[]")
	return host
}

// RegisterMiner handles miner registration
func (s *MiningPoolServer) RegisterMiner(ctx context.Context, req *pb.MinerInfo) (*pb.RegistrationResponse, error) {
	// Get actual IP from gRPC connection
	actualIP := getClientIP(ctx)

	err := s.pool.RegisterMiner(req.MinerId, req.IpAddress, req.Hostname, actualIP)
	if err != nil {
		return &pb.RegistrationResponse{
			Success: false,
			Message: err.Error(),
			MinerId: "",
		}, err
	}

	return &pb.RegistrationResponse{
		Success: true,
		Message: "Miner registered successfully",
		MinerId: req.MinerId,
	}, nil
}

// GetWork provides mining work to a miner
func (s *MiningPoolServer) GetWork(ctx context.Context, req *pb.WorkRequest) (*pb.WorkResponse, error) {
	block, err := s.pool.GetWork(req.MinerId)
	if err != nil {
		return nil, err
	}

	return &pb.WorkResponse{
		BlockIndex:   block.Index,
		PreviousHash: block.PreviousHash,
		Data:         block.Data,
		Difficulty:   int32(s.pool.blockchain.Difficulty),
		Timestamp:    block.Timestamp,
	}, nil
}

// SubmitWork handles work submission from miners
func (s *MiningPoolServer) SubmitWork(ctx context.Context, req *pb.WorkSubmission) (*pb.SubmissionResponse, error) {
	accepted, reward, err := s.pool.SubmitWork(req.MinerId, req.BlockIndex, req.Nonce, req.Hash)

	if err != nil {
		return &pb.SubmissionResponse{
			Accepted: false,
			Message:  err.Error(),
			Reward:   0,
		}, nil
	}

	if !accepted {
		return &pb.SubmissionResponse{
			Accepted: false,
			Message:  "Block rejected",
			Reward:   0,
		}, nil
	}

	return &pb.SubmissionResponse{
		Accepted: true,
		Message:  fmt.Sprintf("Block %d accepted", req.BlockIndex),
		Reward:   reward,
	}, nil
}

// Heartbeat handles miner heartbeat messages
func (s *MiningPoolServer) Heartbeat(ctx context.Context, req *pb.MinerStatus) (*pb.HeartbeatResponse, error) {
	miningTime := time.Duration(req.MiningTimeSeconds) * time.Second

	// Convert GPU devices from protobuf to server type
	var gpuDevices []GPUDeviceInfo
	for _, dev := range req.GpuDevices {
		gpuDevices = append(gpuDevices, GPUDeviceInfo{
			ID:           int(dev.Id),
			Name:         dev.Name,
			Type:         dev.Type,
			Memory:       dev.Memory,
			ComputeUnits: int(dev.ComputeUnits),
			Available:    dev.Available,
		})
	}

	err := s.pool.UpdateHeartbeatWithGPU(
		req.MinerId,
		req.HashRate,
		req.CpuUsagePercent,
		req.TotalHashes,
		miningTime,
		gpuDevices,
		req.GpuHashRate,
		req.GpuEnabled,
		req.HybridMode,
	)
	if err != nil {
		// Check if miner was deleted (not found)
		if err.Error() == "miner not registered" {
			return &pb.HeartbeatResponse{
				Active:     false,
				Message:    "Miner has been deleted from the pool",
				ShouldMine: false,
			}, nil
		}
		return &pb.HeartbeatResponse{
			Active:     false,
			Message:    err.Error(),
			ShouldMine: false,
		}, nil
	}

	// Get the mining status for this miner
	shouldMine, err := s.pool.GetMinerStatus(req.MinerId)
	if err != nil {
		shouldMine = true // Default to mining if status check fails
	}

	// Get the CPU throttle percentage for this miner
	cpuThrottle, err := s.pool.GetCPUThrottle(req.MinerId)
	if err != nil {
		cpuThrottle = 0 // Default to no throttling if check fails
	}

	return &pb.HeartbeatResponse{
		Active:              true,
		Message:             "Heartbeat received",
		ShouldMine:          shouldMine,
		CpuThrottlePercent:  int32(cpuThrottle),
	}, nil
}

// StopMining handles miner stop requests
func (s *MiningPoolServer) StopMining(ctx context.Context, req *pb.MinerInfo) (*pb.StopResponse, error) {
	blocksMined, err := s.pool.StopMiner(req.MinerId)
	if err != nil {
		return &pb.StopResponse{
			Success:          false,
			Message:          err.Error(),
			TotalBlocksMined: 0,
		}, nil
	}

	return &pb.StopResponse{
		Success:          true,
		Message:          "Miner stopped successfully",
		TotalBlocksMined: blocksMined,
	}, nil
}
