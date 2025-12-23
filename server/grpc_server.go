// Package main implements the RedTeamCoin mining pool server components.
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

// MiningPoolServer implements the gRPC MiningPool service for miner
// communication. It handles miner registration, work distribution, solution
// submission, and heartbeats. The server extracts actual client IP addresses
// from gRPC connections for accurate tracking.
type MiningPoolServer struct {
	pb.UnimplementedMiningPoolServer             // Embedded for forward compatibility
	pool                             *MiningPool // Mining pool to coordinate
}

// NewMiningPoolServer creates a new gRPC mining pool server that wraps
// the given mining pool. The returned server implements the pb.MiningPoolServer
// interface and can be registered with a gRPC server.
func NewMiningPoolServer(pool *MiningPool) *MiningPoolServer {
	return &MiningPoolServer{
		pool: pool,
	}
}

// getClientIP extracts the actual client IP address from the gRPC context
// by examining the peer information. It handles both IPv4 and IPv6 addresses
// and returns "unknown" if the IP cannot be determined.
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

// RegisterMiner handles miner registration requests from clients.
// It extracts the actual client IP from the gRPC context and registers
// the miner with the pool using both reported and actual IP addresses.
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

// GetWork assigns a mining work unit to the requesting miner and returns
// the block parameters needed for proof-of-work computation. It retrieves
// or generates work from the pool and returns block index, previous hash,
// data, difficulty, and timestamp. Returns an error if the miner is not
// registered with the pool.
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

// SubmitWork processes a block solution submitted by a miner. It validates
// the proof-of-work, adds valid blocks to the blockchain, and returns
// acceptance status with reward information. Rejected submissions include
// an error message explaining the rejection reason. Returns nil error on
// successful processing regardless of block acceptance.
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

// Heartbeat processes periodic status updates from miners, recording their
// hash rates, CPU usage, total hashes, mining time, and GPU information.
// It returns server control flags (ShouldMine, CPUThrottlePercent) that
// allow the pool to remotely pause mining or limit CPU usage. If a miner
// has been deleted from the pool, returns Active=false to trigger client
// shutdown and self-deletion.
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
		Active:             true,
		Message:            "Heartbeat received",
		ShouldMine:         shouldMine,
		CpuThrottlePercent: int32(cpuThrottle),
	}, nil
}

// StopMining processes a graceful shutdown request from a miner. It marks
// the miner as inactive in the pool and returns the total number of blocks
// the miner successfully mined during its session. This is called when
// miners shut down normally (not when deleted by the server).
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
