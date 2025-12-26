// Package main implements the RedTeamCoin mining pool server.
//
// The server coordinates cryptocurrency mining operations by distributing work
// to connected miners, validating submitted blocks, and maintaining the
// blockchain. It provides both gRPC and REST API interfaces, along with a
// web dashboard for monitoring pool operations.
//
// The server supports TLS/HTTPS for secure communications and can be configured
// via environment variables for production deployments.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	pb "redteamcoin/proto"

	"google.golang.org/grpc"
)

const (
	grpcPort        = 50051              // Default port for gRPC mining protocol
	apiPort         = 8443               // HTTPS port for web dashboard and REST API
	httpPort        = 8080               // HTTP port for redirects when TLS enabled
	difficulty      = 6                  // Mining difficulty (leading zeros required)
	defaultCertFile = "certs/server.crt" // Default TLS certificate file path
	defaultKeyFile  = "certs/server.key" // Default TLS private key file path
)

// generateAuthToken generates a cryptographically secure random 32-byte
// authentication token encoded as a hexadecimal string. The function
// terminates the program if random number generation fails.
func generateAuthToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Failed to generate auth token: %v", err)
	}
	return hex.EncodeToString(bytes)
}

// getAuthToken returns the authentication token from the RTC_AUTH_TOKEN
// environment variable, or generates a new cryptographically secure token
// if the variable is not set.
func getAuthToken() string {
	// Check if token is provided via environment variable
	token := os.Getenv("RTC_AUTH_TOKEN")
	if token != "" {
		return token
	}

	// Generate a new token
	return generateAuthToken()
}

// getTLSConfig returns TLS configuration from environment variables.
// It reads RTC_USE_TLS, RTC_CERT_FILE, and RTC_KEY_FILE, returning
// whether TLS is enabled and the certificate/key file paths.
// Default paths are used if environment variables are not set.
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

// fileExists reports whether the file at the given path exists and is
// accessible. It returns false for directories or if any error occurs
// during the stat operation.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getNetworkIPs returns a list of non-loopback IPv4 addresses for all active
// network interfaces on the system. This is useful for displaying connection
// information to users on multi-homed systems.
func getNetworkIPs() []string {
	var ips []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		// Skip down or loopback interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Only include IPv4 addresses
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue
			}

			ips = append(ips, ip.String())
		}
	}

	return ips
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
	api := NewAPIServer(context.Background(), pool, blockchain, authToken, useTLS, certFile, keyFile)

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

	// Get network interface IPs
	networkIPs := getNetworkIPs()

	fmt.Printf("\nServer started successfully!\n")
	fmt.Printf("- gRPC Server: Port %d (all network interfaces)\n", grpcPort)
	fmt.Printf("  Local access: localhost:%d or 127.0.0.1:%d\n", grpcPort, grpcPort)
	if len(networkIPs) > 0 {
		fmt.Printf("  Network access from: ")
		for i, ip := range networkIPs {
			if i > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%s:%d", ip, grpcPort)
		}
		fmt.Printf("\n")
	}
	fmt.Printf("- Web Dashboard: %s://localhost:%d?token=%s\n", protocol, port, authToken)
	if len(networkIPs) > 0 {
		for _, ip := range networkIPs {
			fmt.Printf("                 %s://%s:%d?token=%s\n", protocol, ip, port, authToken)
		}
	}
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
		fmt.Printf("To accept: Click 'Advanced' -> 'Proceed to localhost or the server IP address. '\n")
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

// startGRPCServer starts the gRPC server for miner connections.
// It creates listeners for both IPv4 and IPv6 to accept connections from
// all network interfaces. If IPv6 is unavailable, it silently continues
// with IPv4 only. This function blocks until the server stops or encounters
// a fatal error.
func startGRPCServer(pool *MiningPool) {
	grpcServer := grpc.NewServer()
	pb.RegisterMiningPoolServer(grpcServer, NewMiningPoolServer(pool))

	// Try to create IPv4 listener
	ipv4Addr := fmt.Sprintf("0.0.0.0:%d", grpcPort)
	lis4, err := net.Listen("tcp4", ipv4Addr)
	if err != nil {
		log.Fatalf("Failed to listen on IPv4 port %d: %v", grpcPort, err)
	}
	fmt.Printf("gRPC server listening on %s (IPv4)\n", lis4.Addr().String())

	// Try to create IPv6 listener (silently skip if unavailable)
	ipv6Addr := fmt.Sprintf("[::]:%d", grpcPort)
	lis6, err := net.Listen("tcp6", ipv6Addr)
	if err == nil {
		fmt.Printf("gRPC server listening on %s (IPv6)\n", lis6.Addr().String())
		// Serve IPv6 in a separate goroutine
		go func() {
			if err := grpcServer.Serve(lis6); err != nil {
				log.Fatalf("Failed to serve gRPC on IPv6: %v", err)
			}
		}()
	}

	// Serve IPv4 in the main goroutine
	if err := grpcServer.Serve(lis4); err != nil {
		log.Fatalf("Failed to serve gRPC on IPv4: %v", err)
	}
}
