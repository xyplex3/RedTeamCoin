//go:build tools
// +build tools

// Package main provides a configuration validation tool for RedTeamCoin.
// It validates client and server configuration files for correctness.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"redteamcoin/config"
)

func main() {
	clientConfig := flag.String("client", "", "Path to client config file (default: search paths)")
	serverConfig := flag.String("server", "", "Path to server config file (default: search paths)")
	all := flag.Bool("all", false, "Validate all config files in search paths")
	flag.Parse()

	exitCode := 0

	if *all || (*clientConfig == "" && *serverConfig == "") {
		// Validate both configs
		fmt.Println("Validating RedTeamCoin Configuration Files")
		fmt.Println("==========================================")
		fmt.Println()

		if !validateClientConfig(*clientConfig) {
			exitCode = 1
		}
		fmt.Println()

		if !validateServerConfig(*serverConfig) {
			exitCode = 1
		}
	} else {
		if *clientConfig != "" {
			if !validateClientConfig(*clientConfig) {
				exitCode = 1
			}
		}

		if *serverConfig != "" {
			if !validateServerConfig(*serverConfig) {
				exitCode = 1
			}
		}
	}

	os.Exit(exitCode)
}

func validateClientConfig(configPath string) bool {
	fmt.Println("Client Configuration")
	fmt.Println("--------------------")

	if configPath == "" {
		// Search for config file
		configPath = findConfigFile("client-config.yaml")
		if configPath == "" {
			fmt.Println("Status: ⚠️  No config file found (will use defaults)")
			fmt.Println("Search paths:")
			fmt.Println("  - ./client-config.yaml")
			fmt.Println("  - ~/.rtc/client-config.yaml")
			fmt.Println("  - /etc/rtc/client-config.yaml")
			fmt.Println()
			fmt.Println("Run 'make init-client-config' to create a config file")
			return true // Not an error - defaults are valid
		}
	}

	fmt.Printf("File: %s\n", configPath)

	cfg, err := config.LoadClientConfig(configPath)
	if err != nil {
		fmt.Printf("Status: ❌ INVALID\n")
		fmt.Printf("Error: %v\n", err)
		return false
	}

	fmt.Println("Status: ✅ VALID")
	fmt.Println()
	fmt.Println("Loaded Configuration:")
	fmt.Printf("  Server Address:       %s\n", cfg.Server.Address)
	fmt.Printf("  GPU Enabled:          %t\n", cfg.Mining.GPUEnabled)
	fmt.Printf("  Hybrid Mode:          %t\n", cfg.Mining.HybridMode)
	fmt.Printf("  Auto Delete:          %t\n", cfg.Mining.AutoDelete)
	fmt.Printf("  GPU Nonce Range:      %d\n", cfg.GPU.NonceRange)
	fmt.Printf("  CPU Start Nonce:      %d\n", cfg.GPU.CPUStartNonce)
	fmt.Printf("  Heartbeat Interval:   %v\n", cfg.Network.HeartbeatInterval)
	fmt.Printf("  Retry Interval:       %v\n", cfg.Network.RetryInterval)
	fmt.Printf("  Max Retry Time:       %v\n", cfg.Network.MaxRetryTime)
	fmt.Printf("  Worker Update Interval: %d\n", cfg.Behavior.WorkerUpdateInterval)

	return true
}

func validateServerConfig(configPath string) bool {
	fmt.Println("Server Configuration")
	fmt.Println("--------------------")

	if configPath == "" {
		// Search for config file
		configPath = findConfigFile("server-config.yaml")
		if configPath == "" {
			fmt.Println("Status: ⚠️  No config file found (will use defaults)")
			fmt.Println("Search paths:")
			fmt.Println("  - ./server-config.yaml")
			fmt.Println("  - ~/.rtc/server-config.yaml")
			fmt.Println("  - /etc/rtc/server-config.yaml")
			fmt.Println()
			fmt.Println("Run 'make init-server-config' to create a config file")
			return true // Not an error - defaults are valid
		}
	}

	fmt.Printf("File: %s\n", configPath)

	cfg, err := config.LoadServerConfig(configPath)
	if err != nil {
		fmt.Printf("Status: ❌ INVALID\n")
		fmt.Printf("Error: %v\n", err)
		return false
	}

	fmt.Println("Status: ✅ VALID")
	fmt.Println()
	fmt.Println("Loaded Configuration:")
	fmt.Printf("  GRPC Port:            %d\n", cfg.Network.GRPCPort)
	fmt.Printf("  API Port:             %d\n", cfg.Network.APIPort)
	fmt.Printf("  HTTP Port:            %d\n", cfg.Network.HTTPPort)
	fmt.Printf("  Mining Difficulty:    %d\n", cfg.Mining.Difficulty)
	fmt.Printf("  Block Reward:         %d\n", cfg.Mining.BlockReward)
	fmt.Printf("  TLS Enabled:          %t\n", cfg.TLS.Enabled)
	if cfg.TLS.Enabled {
		fmt.Printf("  TLS Cert File:        %s\n", cfg.TLS.CertFile)
		fmt.Printf("  TLS Key File:         %s\n", cfg.TLS.KeyFile)
	}
	fmt.Printf("  API Read Timeout:     %v\n", cfg.API.ReadTimeout)
	fmt.Printf("  API Write Timeout:    %v\n", cfg.API.WriteTimeout)
	fmt.Printf("  API Idle Timeout:     %v\n", cfg.API.IdleTimeout)
	fmt.Printf("  Logging Update Interval: %v\n", cfg.Logging.UpdateInterval)
	fmt.Printf("  Logging File Path:    %s\n", cfg.Logging.FilePath)

	return true
}

func findConfigFile(filename string) string {
	searchPaths := []string{
		filepath.Join(".", filename),
		filepath.Join(os.Getenv("HOME"), ".rtc", filename),
		filepath.Join("/etc/rtc", filename),
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
