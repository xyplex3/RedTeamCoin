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
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"redteamcoin/config"
	"redteamcoin/logger"
	pb "redteamcoin/proto"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	configPath string
	logLevel   string
	logFormat  string
	quiet      bool
	verbose    bool
)

func init() {
	flag.StringVar(&configPath, "config", "", "Path to configuration file")
	flag.StringVar(&logLevel, "log-level", "", "Log level (debug, info, warn, error)")
	flag.StringVar(&logFormat, "log-format", "", "Log format (text, color, json)")
	flag.BoolVar(&quiet, "quiet", false, "Quiet mode (errors only)")
	flag.BoolVar(&verbose, "verbose", false, "Verbose mode (enable debug)")
}

// generateAuthToken generates a cryptographically secure random 32-byte
// authentication token encoded as a hexadecimal string. The function
// terminates the program if random number generation fails.
func generateAuthToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		logger.Get().Error("failed to generate auth token", "error", err)
		os.Exit(1)
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
	flag.Parse()

	fmt.Println("=== RedTeamCoin Mining Pool Server ===")
	fmt.Println()

	cfg, err := config.LoadServerConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Apply CLI flag overrides to logging config
	if logLevel != "" {
		cfg.Logging.Level = logLevel
	}
	if logFormat != "" {
		cfg.Logging.Format = logFormat
	}
	if quiet {
		cfg.Logging.Quiet = true
	}
	if verbose {
		cfg.Logging.Verbose = true
	}

	// Initialize application logger
	logger.Set(logger.NewFromServerConfig(cfg))
	logger.Get().Info("starting RedTeamCoin mining pool server",
		"grpc_port", cfg.Network.GRPCPort,
		"api_port", cfg.Network.APIPort,
		"tls_enabled", cfg.TLS.Enabled,
		"difficulty", cfg.Mining.Difficulty)

	authToken := getAuthToken()

	checkTLSCertificates(cfg)

	blockchain, pool := initializeBlockchainAndPool(cfg)
	initializePoolLogger(pool, blockchain, cfg)

	// Create errgroup for coordinated server startup and shutdown
	ctx := context.Background()
	g, gCtx := errgroup.WithContext(ctx)

	// Start gRPC server in errgroup
	g.Go(func() error {
		return startGRPCServer(gCtx, pool, cfg.Network.GRPCPort, cfg.TLS.Enabled, cfg.TLS.CertFile, cfg.TLS.KeyFile)
	})

	// Start API server in errgroup
	api := NewAPIServer(gCtx, pool, blockchain, authToken, cfg.TLS.Enabled, cfg.TLS.CertFile, cfg.TLS.KeyFile)
	protocol, port, redirectPort := determineServerPorts(cfg)

	g.Go(func() error {
		return api.Start(port, redirectPort)
	})

	displayServerInfo(cfg, authToken, protocol, port)

	// Setup config watcher (non-blocking, uses its own context lifecycle)
	setupConfigWatcher(cfg)

	// Wait for all servers to complete or for an error
	if err := g.Wait(); err != nil {
		logger.Get().Error("server error", "error", err)
		os.Exit(1)
	}
}

func checkTLSCertificates(cfg *config.ServerConfig) {
	if !cfg.TLS.Enabled {
		logger.Get().Debug("TLS disabled, skipping certificate check")
		return
	}

	certExists := fileExists(cfg.TLS.CertFile)
	keyExists := fileExists(cfg.TLS.KeyFile)

	if certExists && keyExists {
		logger.Get().Info("TLS certificates verified",
			"cert_file", cfg.TLS.CertFile,
			"key_file", cfg.TLS.KeyFile)
		return
	}

	logger.Get().Error("TLS is enabled but certificates not found",
		"cert_file", cfg.TLS.CertFile,
		"cert_exists", certExists,
		"key_file", cfg.TLS.KeyFile,
		"key_exists", keyExists)

	// Still use fmt for user instructions (UI output)
	fmt.Printf("\nGenerate certificates by running:\n")
	fmt.Printf("  ./generate_certs.sh\n")
	fmt.Printf("\nOr disable TLS in server-config.yaml\n")
	os.Exit(1)
}

func initializeBlockchainAndPool(cfg *config.ServerConfig) (*Blockchain, *MiningPool) {
	fmt.Printf("Initializing blockchain with difficulty: %d\n", cfg.Mining.Difficulty)
	blockchain := NewBlockchain(cfg.Mining.Difficulty)
	fmt.Println("Initializing mining pool...")
	pool := NewMiningPool(blockchain)
	return blockchain, pool
}

func initializePoolLogger(pool *MiningPool, blockchain *Blockchain, cfg *config.ServerConfig) {
	exePath, err := os.Executable()
	if err != nil {
		logger.Get().Error("failed to get executable path", "error", err)
		os.Exit(1)
	}
	exeDir := filepath.Dir(exePath)
	logFile := filepath.Join(exeDir, cfg.Logging.FilePath)
	poolLogger := NewPoolLogger(pool, blockchain, logFile, cfg.Logging.UpdateInterval)
	pool.SetLogger(poolLogger)
	poolLogger.Start()
	logger.Get().Info("pool logger initialized",
		"file_path", logFile,
		"update_interval", cfg.Logging.UpdateInterval)
}

func determineServerPorts(cfg *config.ServerConfig) (protocol string, port, redirectPort int) {
	protocol = "http"
	port = cfg.Network.HTTPPort
	redirectPort = 0

	if cfg.TLS.Enabled {
		protocol = "https"
		port = cfg.Network.APIPort
		redirectPort = cfg.Network.HTTPPort
	}
	return
}

func displayServerInfo(cfg *config.ServerConfig, authToken, protocol string, port int) {
	networkIPs := getNetworkIPs()

	fmt.Printf("\nServer started successfully!\n")
	displayGRPCInfo(cfg, networkIPs)
	displayDashboardInfo(cfg, authToken, protocol, port, networkIPs)
	displayAuthInfo(authToken)
	displayTLSInfo(cfg)
	fmt.Printf("================================\n")
	fmt.Println("\nWaiting for miners to connect...")
	fmt.Println()
}

func displayGRPCInfo(cfg *config.ServerConfig, networkIPs []string) {
	fmt.Printf("- gRPC Server: Port %d (all network interfaces)\n", cfg.Network.GRPCPort)
	fmt.Printf("  Local access: localhost:%d or 127.0.0.1:%d\n", cfg.Network.GRPCPort, cfg.Network.GRPCPort)
	if len(networkIPs) > 0 {
		fmt.Printf("  Network access from: ")
		for i, ip := range networkIPs {
			if i > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%s:%d", ip, cfg.Network.GRPCPort)
		}
		fmt.Printf("\n")
	}
}

func displayDashboardInfo(cfg *config.ServerConfig, authToken, protocol string, port int, networkIPs []string) {
	fmt.Printf("- Web Dashboard: %s://localhost:%d?token=%s\n", protocol, port, authToken)
	if len(networkIPs) > 0 {
		for _, ip := range networkIPs {
			fmt.Printf("                 %s://%s:%d?token=%s\n", protocol, ip, port, authToken)
		}
	}
	if cfg.TLS.Enabled {
		fmt.Printf("- HTTP Redirect: http://localhost:%d (redirects to HTTPS)\n", cfg.Network.HTTPPort)
	}
}

func displayAuthInfo(authToken string) {
	fmt.Printf("\n=== API Authentication Token ===\n")
	fmt.Printf("Token: %s\n", authToken)
	fmt.Printf("\nUse this token in the Authorization header:\n")
	fmt.Printf("  Authorization: Bearer %s\n", authToken)
	fmt.Printf("\nOr access the dashboard with the token in the URL (shown above)\n")
}

func displayTLSInfo(cfg *config.ServerConfig) {
	if cfg.TLS.Enabled {
		fmt.Printf("\n=== TLS/HTTPS Configuration ===\n")
		fmt.Printf("TLS Enabled: Yes\n")
		fmt.Printf("Note: Using self-signed certificate. Browsers will show a warning.\n")
		fmt.Printf("To accept: Click 'Advanced' -> 'Proceed to localhost or the server IP address'.\n")
	} else {
		fmt.Printf("\nWARNING: TLS is disabled. Enable with: export RTC_SERVER_TLS_ENABLED=true\n")
	}
}

func setupConfigWatcher(cfg *config.ServerConfig) {
	// Start config file watcher for hot-reload (non-blocking)
	// Monitors the config file for changes and reports detected changes.
	// Note: Application logging settings (level, format, quiet, verbose) are
	// hot-reloaded automatically. Other settings require a server restart:
	//   - Network ports (grpc_port, api_port, http_port)
	//   - TLS settings (enabled, cert_file, key_file)
	//   - Mining parameters (difficulty, block_reward)
	//   - PoolLogger configuration (update_interval, file_path)
	configWatcherLogger := slog.Default()
	if err := config.WatchServerConfig(context.Background(), configPath, func(newCfg *config.ServerConfig) {
		handleConfigChange(cfg, newCfg)
	}, configWatcherLogger); err != nil {
		logger.Get().Warn("failed to start config watcher", "error", err)
	} else {
		logger.Get().Info("config file watcher started", "config_path", configPath)
	}
}

func handleConfigChange(cfg, newCfg *config.ServerConfig) {
	fmt.Println("\n=== Configuration File Changed ===")
	changed := false

	changed = checkPortChanges(cfg, newCfg) || changed
	changed = checkMiningChanges(cfg, newCfg) || changed
	changed = checkTLSChanges(cfg, newCfg) || changed
	changed = checkLoggingChanges(cfg, newCfg) || changed

	if changed {
		fmt.Println("To apply changes, restart the server with: systemctl restart rtc-server")
	} else {
		fmt.Println("No relevant configuration changes detected")
	}

	fmt.Println("=====================================")
	fmt.Println()
}

func checkPortChanges(cfg, newCfg *config.ServerConfig) bool {
	changed := false
	if newCfg.Network.GRPCPort != cfg.Network.GRPCPort {
		fmt.Printf("GRPC port changed: %d -> %d (requires restart)\n", cfg.Network.GRPCPort, newCfg.Network.GRPCPort)
		changed = true
	}
	if newCfg.Network.APIPort != cfg.Network.APIPort {
		fmt.Printf("API port changed: %d -> %d (requires restart)\n", cfg.Network.APIPort, newCfg.Network.APIPort)
		changed = true
	}
	if newCfg.Network.HTTPPort != cfg.Network.HTTPPort {
		fmt.Printf("HTTP port changed: %d -> %d (requires restart)\n", cfg.Network.HTTPPort, newCfg.Network.HTTPPort)
		changed = true
	}
	return changed
}

func checkMiningChanges(cfg, newCfg *config.ServerConfig) bool {
	changed := false
	if newCfg.Mining.Difficulty != cfg.Mining.Difficulty {
		fmt.Printf("Mining difficulty changed: %d -> %d (requires restart)\n", cfg.Mining.Difficulty, newCfg.Mining.Difficulty)
		changed = true
	}
	if newCfg.Mining.BlockReward != cfg.Mining.BlockReward {
		fmt.Printf("Block reward changed: %d -> %d (requires restart)\n", cfg.Mining.BlockReward, newCfg.Mining.BlockReward)
		changed = true
	}
	return changed
}

func checkTLSChanges(cfg, newCfg *config.ServerConfig) bool {
	changed := false
	if newCfg.TLS.Enabled != cfg.TLS.Enabled {
		fmt.Printf("TLS enabled changed: %v -> %v (requires restart)\n", cfg.TLS.Enabled, newCfg.TLS.Enabled)
		changed = true
	}
	if newCfg.TLS.CertFile != cfg.TLS.CertFile {
		fmt.Printf("TLS certificate file path changed: %s -> %s (requires restart)\n", cfg.TLS.CertFile, newCfg.TLS.CertFile)
		changed = true
	}
	if newCfg.TLS.KeyFile != cfg.TLS.KeyFile {
		fmt.Printf("TLS key file path changed: %s -> %s (requires restart)\n", cfg.TLS.KeyFile, newCfg.TLS.KeyFile)
		changed = true
	}
	return changed
}

func checkLoggingChanges(cfg, newCfg *config.ServerConfig) bool {
	restartRequired := false

	// Check PoolLogger settings (require restart)
	if newCfg.Logging.UpdateInterval != cfg.Logging.UpdateInterval {
		fmt.Printf("Logging update interval changed: %v -> %v (requires restart)\n", cfg.Logging.UpdateInterval, newCfg.Logging.UpdateInterval)
		restartRequired = true
	}
	if newCfg.Logging.FilePath != cfg.Logging.FilePath {
		fmt.Printf("Log file path changed: %s -> %s (requires restart)\n", cfg.Logging.FilePath, newCfg.Logging.FilePath)
		restartRequired = true
	}

	// Check application logging settings (hot-reloadable!)
	if newCfg.Logging.Level != cfg.Logging.Level ||
		newCfg.Logging.Format != cfg.Logging.Format ||
		newCfg.Logging.Quiet != cfg.Logging.Quiet ||
		newCfg.Logging.Verbose != cfg.Logging.Verbose {

		fmt.Printf("Application logging changed: level=%s format=%s quiet=%v verbose=%v (hot-reloading...)\n",
			newCfg.Logging.Level, newCfg.Logging.Format, newCfg.Logging.Quiet, newCfg.Logging.Verbose)

		// Hot-reload the logger with new settings
		logger.Set(logger.NewFromServerConfig(newCfg))
		logger.Get().Info("logging configuration reloaded",
			"level", newCfg.Logging.Level,
			"format", newCfg.Logging.Format,
			"quiet", newCfg.Logging.Quiet,
			"verbose", newCfg.Logging.Verbose)

		// Update the config reference
		cfg.Logging.Level = newCfg.Logging.Level
		cfg.Logging.Format = newCfg.Logging.Format
		cfg.Logging.Quiet = newCfg.Logging.Quiet
		cfg.Logging.Verbose = newCfg.Logging.Verbose

		// No restart required for application logging changes
	}

	return restartRequired
}

// startGRPCServer starts the gRPC server for miner connections.
// It creates listeners for both IPv4 and IPv6 to accept connections from
// all network interfaces. If IPv6 is unavailable, it silently continues
// with IPv4 only. This function blocks until the server stops, the context
// is cancelled, or an error occurs. If TLS is enabled, the server will use
// the provided certificate and key files for secure communication.
func startGRPCServer(ctx context.Context, pool *MiningPool, grpcPort int, useTLS bool, certFile, keyFile string) error {
	var grpcServer *grpc.Server
	if useTLS {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			logger.Get().Error("failed to load TLS credentials for gRPC",
				"cert_file", certFile,
				"key_file", keyFile,
				"error", err)
			return fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		grpcServer = grpc.NewServer(grpc.Creds(creds))
		logger.Get().Info("gRPC server configured with TLS",
			"cert_file", certFile,
			"key_file", keyFile)
	} else {
		grpcServer = grpc.NewServer() // nosemgrep: go.grpc.security.grpc-server-insecure-connection.grpc-server-insecure-connection
		logger.Get().Warn("gRPC server running without TLS encryption")
	}
	pb.RegisterMiningPoolServer(grpcServer, NewMiningPoolServer(pool))

	// Try to create IPv4 listener
	ipv4Addr := fmt.Sprintf("0.0.0.0:%d", grpcPort)
	lis4, err := net.Listen("tcp4", ipv4Addr)
	if err != nil {
		logger.Get().Error("failed to listen on IPv4 port",
			"port", grpcPort,
			"address", ipv4Addr,
			"error", err)
		return fmt.Errorf("failed to listen on IPv4 port %d: %w", grpcPort, err)
	}
	logger.Get().Info("gRPC server listening on IPv4",
		"address", lis4.Addr().String())

	// Try to create IPv6 listener (silently skip if unavailable)
	ipv6Addr := fmt.Sprintf("[::]:%d", grpcPort)
	lis6, err := net.Listen("tcp6", ipv6Addr)
	if err == nil {
		logger.Get().Info("gRPC server listening on IPv6",
			"address", lis6.Addr().String())
		// Serve IPv6 in a separate goroutine
		go func() {
			if err := grpcServer.Serve(lis6); err != nil {
				logger.Get().Error("failed to serve gRPC on IPv6", "error", err)
			}
		}()
	} else {
		logger.Get().Debug("IPv6 listener unavailable, using IPv4 only", "error", err)
	}

	// Handle graceful shutdown when context is cancelled
	go func() {
		<-ctx.Done()
		logger.Get().Info("shutting down gRPC server")
		grpcServer.GracefulStop()
	}()

	// Serve IPv4 in the main goroutine (this blocks)
	if err := grpcServer.Serve(lis4); err != nil {
		logger.Get().Error("failed to serve gRPC on IPv4", "error", err)
		return fmt.Errorf("gRPC server error: %w", err)
	}

	return nil
}
