package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	pb "redteamcoin/proto"
)

const (
	grpcPort        = 50051
	apiPort         = 8443 // HTTPS port (8080 for HTTP fallback)
	httpPort        = 8080 // HTTP redirect port (when TLS is enabled)
	difficulty      = 6
	defaultCertFile = "certs/server.crt"
	defaultKeyFile  = "certs/server.key"
)

// generateAuthToken generates a secure random authentication token
func generateAuthToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Failed to generate auth token: %v", err)
	}
	return hex.EncodeToString(bytes)
}

// getAuthToken returns auth token from environment or generates a new one
func getAuthToken() string {
	// Check if token is provided via environment variable
	token := os.Getenv("RTC_AUTH_TOKEN")
	if token != "" {
		return token
	}

	// Generate a new token
	return generateAuthToken()
}

// getTLSConfig returns TLS configuration from environment
func getTLSConfig() (bool, string, string) {
	useTLS := os.Getenv("RTC_USE_TLS") == "true"
	certFile := os.Getenv("RTC_CERT_FILE")
	keyFile := os.Getenv("RTC_KEY_FILE")

	// Use defaults if not specified
	if certFile == "" {
		certFile = defaultCertFile
	}
	if keyFile == "" {
		keyFile = defaultKeyFile
	}

	return useTLS, certFile, keyFile
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func main() {
	fmt.Println("=== RedTeamCoin Mining Pool Server ===")
	fmt.Println()

	// Get or generate authentication token
	authToken := getAuthToken()

	// Get TLS configuration
	useTLS, certFile, keyFile := getTLSConfig()

	// Validate TLS certificates if TLS is enabled
	if useTLS {
		if !fileExists(certFile) || !fileExists(keyFile) {
			fmt.Printf("ERROR: TLS is enabled but certificates not found!\n")
			fmt.Printf("  Certificate: %s (exists: %v)\n", certFile, fileExists(certFile))
			fmt.Printf("  Private Key: %s (exists: %v)\n", keyFile, fileExists(keyFile))
			fmt.Printf("\nGenerate certificates by running:\n")
			fmt.Printf("  ./generate_certs.sh\n")
			fmt.Printf("\nOr disable TLS by unsetting RTC_USE_TLS\n")
			os.Exit(1)
		}
	}

	// Initialize blockchain
	fmt.Printf("Initializing blockchain with difficulty: %d\n", difficulty)
	blockchain := NewBlockchain(difficulty)

	// Initialize mining pool
	fmt.Println("Initializing mining pool...")
	pool := NewMiningPool(blockchain)

	// Initialize and start logger
	// Get executable path and create log file in the same directory
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	exeDir := filepath.Dir(exePath)
	logFile := filepath.Join(exeDir, "pool_log.json")
	updateInterval := 30 * time.Second // Update log every 30 seconds
	logger := NewPoolLogger(pool, blockchain, logFile, updateInterval)
	pool.SetLogger(logger)
	logger.Start()

	// Start gRPC server
	go startGRPCServer(pool)

	// Start API server
	api := NewAPIServer(pool, blockchain, authToken, useTLS, certFile, keyFile)

	// Determine protocol and port
	protocol := "http"
	port := apiPort
	redirectPort := 0

	if useTLS {
		protocol = "https"
		redirectPort = httpPort
	} else {
		// When not using TLS, use port 8080
		port = httpPort
	}

	fmt.Printf("\nServer started successfully!\n")
	fmt.Printf("- gRPC Server: localhost:%d\n", grpcPort)
	fmt.Printf("- Web Dashboard: %s://localhost:%d?token=%s\n", protocol, port, authToken)

	if useTLS {
		fmt.Printf("- HTTP Redirect: http://localhost:%d (redirects to HTTPS)\n", httpPort)
	}

	fmt.Printf("\n=== API Authentication Token ===\n")
	fmt.Printf("Token: %s\n", authToken)
	fmt.Printf("\nUse this token in the Authorization header:\n")
	fmt.Printf("  Authorization: Bearer %s\n", authToken)
	fmt.Printf("\nOr access the dashboard with the token in the URL (shown above)\n")

	if useTLS {
		fmt.Printf("\n=== TLS/HTTPS Configuration ===\n")
		fmt.Printf("TLS Enabled: Yes\n")
		fmt.Printf("Note: Using self-signed certificate. Browsers will show a warning.\n")
		fmt.Printf("To accept: Click 'Advanced' -> 'Proceed to localhost'\n")
	} else {
		fmt.Printf("\nWARNING: TLS is disabled. Enable with: export RTC_USE_TLS=true\n")
	}

	fmt.Printf("================================\n")
	fmt.Println("\nWaiting for miners to connect...")
	fmt.Println()

	if err := api.Start(port, redirectPort); err != nil {
		log.Fatalf("Failed to start API server: %v", err)
	}
}

func startGRPCServer(pool *MiningPool) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		log.Fatalf("Failed to listen on port %d: %v", grpcPort, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterMiningPoolServer(grpcServer, NewMiningPoolServer(pool))

	fmt.Printf("gRPC server listening on :%d\n", grpcPort)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve gRPC: %v", err)
	}
}
