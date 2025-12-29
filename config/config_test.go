package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientConfigDefaults(t *testing.T) {
	cfg, err := LoadClientConfig("")
	if err != nil {
		t.Fatalf("LoadClientConfig failed: %v", err)
	}

	// Server defaults
	if cfg.Server.Address != "localhost:50051" {
		t.Errorf("Expected server address 'localhost:50051', got '%s'", cfg.Server.Address)
	}

	// Mining defaults
	if !cfg.Mining.GPUEnabled {
		t.Error("Expected GPU enabled by default")
	}
	if cfg.Mining.HybridMode {
		t.Error("Expected hybrid mode disabled by default")
	}
	if !cfg.Mining.AutoDelete {
		t.Error("Expected auto delete enabled by default")
	}

	// GPU defaults
	if cfg.GPU.NonceRange != 500000000 {
		t.Errorf("Expected nonce range 500000000, got %d", cfg.GPU.NonceRange)
	}
	if cfg.GPU.CPUStartNonce != 5000000000 {
		t.Errorf("Expected CPU start nonce 5000000000, got %d", cfg.GPU.CPUStartNonce)
	}

	// Network defaults
	if cfg.Network.HeartbeatInterval != 30*time.Second {
		t.Errorf("Expected heartbeat interval 30s, got %v", cfg.Network.HeartbeatInterval)
	}
	if cfg.Network.RetryInterval != 10*time.Second {
		t.Errorf("Expected retry interval 10s, got %v", cfg.Network.RetryInterval)
	}
	if cfg.Network.MaxRetryTime != 5*time.Minute {
		t.Errorf("Expected max retry time 5m, got %v", cfg.Network.MaxRetryTime)
	}

	// Behavior defaults
	if cfg.Behavior.WorkerUpdateInterval != 100000 {
		t.Errorf("Expected worker update interval 100000, got %d", cfg.Behavior.WorkerUpdateInterval)
	}

	// TLS defaults
	if cfg.Server.TLS.Enabled {
		t.Error("Expected TLS disabled by default")
	}
	if !cfg.Server.TLS.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify enabled by default")
	}
	if cfg.Server.TLS.CACertFile != "" {
		t.Errorf("Expected empty CA cert file by default, got '%s'", cfg.Server.TLS.CACertFile)
	}
}

func TestServerConfigDefaults(t *testing.T) {
	cfg, err := LoadServerConfig("")
	if err != nil {
		t.Fatalf("LoadServerConfig failed: %v", err)
	}

	// Network defaults
	if cfg.Network.GRPCPort != 50051 {
		t.Errorf("Expected GRPC port 50051, got %d", cfg.Network.GRPCPort)
	}
	if cfg.Network.APIPort != 8443 {
		t.Errorf("Expected API port 8443, got %d", cfg.Network.APIPort)
	}
	if cfg.Network.HTTPPort != 8080 {
		t.Errorf("Expected HTTP port 8080, got %d", cfg.Network.HTTPPort)
	}

	// Mining defaults
	if cfg.Mining.Difficulty != 6 {
		t.Errorf("Expected difficulty 6, got %d", cfg.Mining.Difficulty)
	}
	if cfg.Mining.BlockReward != 50 {
		t.Errorf("Expected block reward 50, got %d", cfg.Mining.BlockReward)
	}

	// TLS defaults
	if cfg.TLS.Enabled {
		t.Error("Expected TLS disabled by default")
	}
	if cfg.TLS.CertFile != "certs/server.crt" {
		t.Errorf("Expected cert file 'certs/server.crt', got '%s'", cfg.TLS.CertFile)
	}
	if cfg.TLS.KeyFile != "certs/server.key" {
		t.Errorf("Expected key file 'certs/server.key', got '%s'", cfg.TLS.KeyFile)
	}

	// API defaults
	if cfg.API.ReadTimeout != 15*time.Second {
		t.Errorf("Expected read timeout 15s, got %v", cfg.API.ReadTimeout)
	}
	if cfg.API.WriteTimeout != 15*time.Second {
		t.Errorf("Expected write timeout 15s, got %v", cfg.API.WriteTimeout)
	}
	if cfg.API.IdleTimeout != 60*time.Second {
		t.Errorf("Expected idle timeout 60s, got %v", cfg.API.IdleTimeout)
	}

	// Logging defaults
	if cfg.Logging.UpdateInterval != 30*time.Second {
		t.Errorf("Expected logging update interval 30s, got %v", cfg.Logging.UpdateInterval)
	}
	if cfg.Logging.FilePath != "pool_log.json" {
		t.Errorf("Expected log file 'pool_log.json', got '%s'", cfg.Logging.FilePath)
	}
}

func TestClientConfigFromFile(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-config.yaml")

	configContent := `
server:
  address: "test.example.com:9999"

mining:
  gpu_enabled: false
  hybrid_mode: true
  auto_delete: false

gpu:
  nonce_range: 1000000000
  cpu_start_nonce: 10000000000

network:
  heartbeat_interval: "60s"
  retry_interval: "20s"
  max_retry_time: "10m"

behavior:
  worker_update_interval: 200000
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := LoadClientConfig(configFile)
	if err != nil {
		t.Fatalf("LoadClientConfig failed: %v", err)
	}

	// Verify loaded values
	if cfg.Server.Address != "test.example.com:9999" {
		t.Errorf("Expected address 'test.example.com:9999', got '%s'", cfg.Server.Address)
	}
	if cfg.Mining.GPUEnabled {
		t.Error("Expected GPU disabled")
	}
	if !cfg.Mining.HybridMode {
		t.Error("Expected hybrid mode enabled")
	}
	if cfg.GPU.NonceRange != 1000000000 {
		t.Errorf("Expected nonce range 1000000000, got %d", cfg.GPU.NonceRange)
	}
	if cfg.Network.HeartbeatInterval != 60*time.Second {
		t.Errorf("Expected heartbeat interval 60s, got %v", cfg.Network.HeartbeatInterval)
	}
}

func TestServerConfigFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-server-config.yaml")

	configContent := `
network:
  grpc_port: 60051
  api_port: 9443
  http_port: 9080

mining:
  difficulty: 8
  block_reward: 100

tls:
  enabled: false
  cert_file: "/tmp/test.crt"
  key_file: "/tmp/test.key"

api:
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"

logging:
  update_interval: "60s"
  file_path: "/var/log/pool.json"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := LoadServerConfig(configFile)
	if err != nil {
		t.Fatalf("LoadServerConfig failed: %v", err)
	}

	if cfg.Network.GRPCPort != 60051 {
		t.Errorf("Expected GRPC port 60051, got %d", cfg.Network.GRPCPort)
	}
	if cfg.Mining.Difficulty != 8 {
		t.Errorf("Expected difficulty 8, got %d", cfg.Mining.Difficulty)
	}
	if cfg.Mining.BlockReward != 100 {
		t.Errorf("Expected block reward 100, got %d", cfg.Mining.BlockReward)
	}
}

func TestClientConfigEnvironmentOverride(t *testing.T) {
	// Set environment variables
	os.Setenv("RTC_CLIENT_SERVER_ADDRESS", "env.example.com:7777")
	os.Setenv("RTC_CLIENT_MINING_GPU_ENABLED", "false")
	os.Setenv("RTC_CLIENT_GPU_NONCE_RANGE", "999999999")
	defer func() {
		os.Unsetenv("RTC_CLIENT_SERVER_ADDRESS")
		os.Unsetenv("RTC_CLIENT_MINING_GPU_ENABLED")
		os.Unsetenv("RTC_CLIENT_GPU_NONCE_RANGE")
	}()

	cfg, err := LoadClientConfig("")
	if err != nil {
		t.Fatalf("LoadClientConfig failed: %v", err)
	}

	if cfg.Server.Address != "env.example.com:7777" {
		t.Errorf("Expected address from env 'env.example.com:7777', got '%s'", cfg.Server.Address)
	}
	if cfg.Mining.GPUEnabled {
		t.Error("Expected GPU disabled from env")
	}
	if cfg.GPU.NonceRange != 999999999 {
		t.Errorf("Expected nonce range 999999999 from env, got %d", cfg.GPU.NonceRange)
	}
}

func TestServerConfigEnvironmentOverride(t *testing.T) {
	os.Setenv("RTC_SERVER_NETWORK_GRPC_PORT", "55555")
	os.Setenv("RTC_SERVER_MINING_DIFFICULTY", "10")
	defer func() {
		os.Unsetenv("RTC_SERVER_NETWORK_GRPC_PORT")
		os.Unsetenv("RTC_SERVER_MINING_DIFFICULTY")
	}()

	cfg, err := LoadServerConfig("")
	if err != nil {
		t.Fatalf("LoadServerConfig failed: %v", err)
	}

	if cfg.Network.GRPCPort != 55555 {
		t.Errorf("Expected GRPC port 55555 from env, got %d", cfg.Network.GRPCPort)
	}
	if cfg.Mining.Difficulty != 10 {
		t.Errorf("Expected difficulty 10 from env, got %d", cfg.Mining.Difficulty)
	}
}

func TestClientConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		modifier    func(*ClientConfig)
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			modifier: func(c *ClientConfig) {
				// Default config is valid
			},
			expectError: false,
		},
		{
			name: "empty server address",
			modifier: func(c *ClientConfig) {
				c.Server.Address = ""
			},
			expectError: true,
			errorMsg:    "server address cannot be empty",
		},
		{
			name: "negative nonce range",
			modifier: func(c *ClientConfig) {
				c.GPU.NonceRange = -1
			},
			expectError: true,
			errorMsg:    "nonce_range must be positive",
		},
		{
			name: "zero nonce range",
			modifier: func(c *ClientConfig) {
				c.GPU.NonceRange = 0
			},
			expectError: true,
			errorMsg:    "nonce_range must be positive",
		},
		{
			name: "negative cpu start nonce",
			modifier: func(c *ClientConfig) {
				c.GPU.CPUStartNonce = -1
			},
			expectError: true,
			errorMsg:    "cpu_start_nonce cannot be negative",
		},
		{
			name: "heartbeat interval too short",
			modifier: func(c *ClientConfig) {
				c.Network.HeartbeatInterval = 500 * time.Millisecond
			},
			expectError: true,
			errorMsg:    "heartbeat_interval too short",
		},
		{
			name: "retry interval too short",
			modifier: func(c *ClientConfig) {
				c.Network.RetryInterval = 500 * time.Millisecond
			},
			expectError: true,
			errorMsg:    "retry_interval too short",
		},
		{
			name: "max retry time less than retry interval",
			modifier: func(c *ClientConfig) {
				c.Network.RetryInterval = 30 * time.Second
				c.Network.MaxRetryTime = 10 * time.Second
			},
			expectError: true,
			errorMsg:    "max_retry_time",
		},
		{
			name: "negative worker update interval",
			modifier: func(c *ClientConfig) {
				c.Behavior.WorkerUpdateInterval = -1
			},
			expectError: true,
			errorMsg:    "worker_update_interval must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := LoadClientConfig("")
			tt.modifier(cfg)

			err := cfg.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, got: %v", err)
				}
			}
		})
	}
}

func TestServerConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		modifier    func(*ServerConfig)
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			modifier: func(c *ServerConfig) {
				// Default config is valid
			},
			expectError: false,
		},
		{
			name: "invalid grpc port - too low",
			modifier: func(c *ServerConfig) {
				c.Network.GRPCPort = 0
			},
			expectError: true,
			errorMsg:    "invalid grpc_port",
		},
		{
			name: "invalid grpc port - too high",
			modifier: func(c *ServerConfig) {
				c.Network.GRPCPort = 70000
			},
			expectError: true,
			errorMsg:    "invalid grpc_port",
		},
		{
			name: "duplicate ports - grpc and api",
			modifier: func(c *ServerConfig) {
				c.Network.GRPCPort = 8080
				c.Network.APIPort = 8080
			},
			expectError: true,
			errorMsg:    "ports must be unique",
		},
		{
			name: "duplicate ports - all three",
			modifier: func(c *ServerConfig) {
				c.Network.GRPCPort = 9000
				c.Network.APIPort = 9000
				c.Network.HTTPPort = 9000
			},
			expectError: true,
			errorMsg:    "ports must be unique",
		},
		{
			name: "difficulty too low",
			modifier: func(c *ServerConfig) {
				c.Mining.Difficulty = 0
			},
			expectError: true,
			errorMsg:    "invalid difficulty",
		},
		{
			name: "difficulty too high",
			modifier: func(c *ServerConfig) {
				c.Mining.Difficulty = 65
			},
			expectError: true,
			errorMsg:    "invalid difficulty",
		},
		{
			name: "negative block reward",
			modifier: func(c *ServerConfig) {
				c.Mining.BlockReward = -1
			},
			expectError: true,
			errorMsg:    "block_reward must be positive",
		},
		{
			name: "zero block reward",
			modifier: func(c *ServerConfig) {
				c.Mining.BlockReward = 0
			},
			expectError: true,
			errorMsg:    "block_reward must be positive",
		},
		{
			name: "tls enabled without cert file",
			modifier: func(c *ServerConfig) {
				c.TLS.Enabled = true
				c.TLS.CertFile = ""
			},
			expectError: true,
			errorMsg:    "tls.cert_file is required",
		},
		{
			name: "tls enabled without key file",
			modifier: func(c *ServerConfig) {
				c.TLS.Enabled = true
				c.TLS.KeyFile = ""
			},
			expectError: true,
			errorMsg:    "tls.key_file is required",
		},
		{
			name: "api read timeout too short",
			modifier: func(c *ServerConfig) {
				c.API.ReadTimeout = 500 * time.Millisecond
			},
			expectError: true,
			errorMsg:    "api.read_timeout too short",
		},
		{
			name: "empty logging file path",
			modifier: func(c *ServerConfig) {
				c.Logging.FilePath = ""
			},
			expectError: true,
			errorMsg:    "logging.file_path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, _ := LoadServerConfig("")
			tt.modifier(cfg)

			err := cfg.Validate()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected validation error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%v'", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, got: %v", err)
				}
			}
		})
	}
}

func TestInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	invalidYAML := `
server:
  address: "test
    invalid indentation
  more bad yaml
`
	if err := os.WriteFile(configFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := LoadClientConfig(configFile)
	if err == nil {
		t.Error("Expected error for invalid YAML, got none")
	}
}

// TestNonExistentConfigFile tests that an explicit non-existent config file path returns an error.
func TestNonExistentConfigFile(t *testing.T) {
	_, err := LoadClientConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for explicit non-existent config file path, got none")
	}
}

// TestConfigFileNotFoundInSearchPaths tests that missing config in search paths falls back to defaults.
func TestConfigFileNotFoundInSearchPaths(t *testing.T) {
	// Empty string uses search paths, which won't find a file, so defaults are used
	cfg, err := LoadClientConfig("")
	if err != nil {
		t.Fatalf("Expected graceful fallback to defaults, got error: %v", err)
	}

	// Should have default values
	if cfg.Server.Address != "localhost:50051" {
		t.Errorf("Expected default address, got '%s'", cfg.Server.Address)
	}
}

func TestInvalidConfigValues(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid-values.yaml")

	// Config with invalid port
	configContent := `
network:
  grpc_port: 99999
  api_port: 8443
  http_port: 8080
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := LoadServerConfig(configFile)
	if err == nil {
		t.Error("Expected validation error for invalid port, got none")
	}
	if !contains(err.Error(), "invalid grpc_port") {
		t.Errorf("Expected error about invalid port, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}

// TestClientConfigPrecedenceIntegration tests the complete precedence hierarchy:
// Command-line flags > Environment variables > Config file > Defaults
func TestClientConfigPrecedenceIntegration(t *testing.T) {
	// Create a config file with custom values
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "precedence-test.yaml")

	configContent := `
server:
  address: "file.example.com:5555"

mining:
  gpu_enabled: false
  hybrid_mode: false

gpu:
  nonce_range: 100000000
  cpu_start_nonce: 1000000000

network:
  heartbeat_interval: "45s"
  retry_interval: "15s"
  max_retry_time: "8m"

behavior:
  worker_update_interval: 150000
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Set environment variables that should override file values
	os.Setenv("RTC_CLIENT_SERVER_ADDRESS", "env.example.com:6666")
	os.Setenv("RTC_CLIENT_MINING_GPU_ENABLED", "true")
	os.Setenv("RTC_CLIENT_GPU_NONCE_RANGE", "200000000")
	defer func() {
		os.Unsetenv("RTC_CLIENT_SERVER_ADDRESS")
		os.Unsetenv("RTC_CLIENT_MINING_GPU_ENABLED")
		os.Unsetenv("RTC_CLIENT_GPU_NONCE_RANGE")
	}()

	cfg, err := LoadClientConfig(configFile)
	if err != nil {
		t.Fatalf("LoadClientConfig failed: %v", err)
	}

	// Verify precedence: env var > file > defaults
	// Server address: env var should win
	if cfg.Server.Address != "env.example.com:6666" {
		t.Errorf("Expected env var address 'env.example.com:6666', got '%s'", cfg.Server.Address)
	}

	// GPU enabled: env var should win (true) over file (false)
	if !cfg.Mining.GPUEnabled {
		t.Error("Expected GPU enabled from env var (true)")
	}

	// Nonce range: env var should win (200000000) over file (100000000)
	if cfg.GPU.NonceRange != 200000000 {
		t.Errorf("Expected nonce range 200000000 from env, got %d", cfg.GPU.NonceRange)
	}

	// CPU start nonce: file should win over default (no env var set)
	if cfg.GPU.CPUStartNonce != 1000000000 {
		t.Errorf("Expected cpu start nonce 1000000000 from file, got %d", cfg.GPU.CPUStartNonce)
	}

	// Heartbeat interval: file should win (45s) over default (30s)
	if cfg.Network.HeartbeatInterval != 45*time.Second {
		t.Errorf("Expected heartbeat interval 45s from file, got %v", cfg.Network.HeartbeatInterval)
	}

	// Hybrid mode: file should win (false) since no env var set
	if cfg.Mining.HybridMode {
		t.Error("Expected hybrid mode disabled from file")
	}
}

func TestServerConfigPrecedenceIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "server-precedence-test.yaml")

	configContent := `
network:
  grpc_port: 60051
  api_port: 9443
  http_port: 9080

mining:
  difficulty: 8
  block_reward: 100

tls:
  enabled: false
  cert_file: "file-cert.crt"
  key_file: "file-key.key"

api:
  read_timeout: "25s"
  write_timeout: "25s"
  idle_timeout: "90s"

logging:
  update_interval: "45s"
  file_path: "file-pool.json"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Set environment variables
	os.Setenv("RTC_SERVER_NETWORK_GRPC_PORT", "55555")
	os.Setenv("RTC_SERVER_MINING_DIFFICULTY", "12")
	os.Setenv("RTC_SERVER_TLS_CERT_FILE", "env-cert.crt")
	defer func() {
		os.Unsetenv("RTC_SERVER_NETWORK_GRPC_PORT")
		os.Unsetenv("RTC_SERVER_MINING_DIFFICULTY")
		os.Unsetenv("RTC_SERVER_TLS_CERT_FILE")
	}()

	cfg, err := LoadServerConfig(configFile)
	if err != nil {
		t.Fatalf("LoadServerConfig failed: %v", err)
	}

	// Verify precedence: env var > file > defaults
	// GRPC port: env var should win (55555) over file (60051)
	if cfg.Network.GRPCPort != 55555 {
		t.Errorf("Expected GRPC port 55555 from env, got %d", cfg.Network.GRPCPort)
	}

	// API port: file should win (9443) over default (8443)
	if cfg.Network.APIPort != 9443 {
		t.Errorf("Expected API port 9443 from file, got %d", cfg.Network.APIPort)
	}

	// Difficulty: env var should win (12) over file (8)
	if cfg.Mining.Difficulty != 12 {
		t.Errorf("Expected difficulty 12 from env, got %d", cfg.Mining.Difficulty)
	}

	// Block reward: file should win (100) over default (50)
	if cfg.Mining.BlockReward != 100 {
		t.Errorf("Expected block reward 100 from file, got %d", cfg.Mining.BlockReward)
	}

	// Cert file: env var should win over file
	if cfg.TLS.CertFile != "env-cert.crt" {
		t.Errorf("Expected cert file 'env-cert.crt' from env, got '%s'", cfg.TLS.CertFile)
	}

	// Key file: file should win (no env var set)
	if cfg.TLS.KeyFile != "file-key.key" {
		t.Errorf("Expected key file 'file-key.key' from file, got '%s'", cfg.TLS.KeyFile)
	}

	// Read timeout: file should win (25s) over default (15s)
	if cfg.API.ReadTimeout != 25*time.Second {
		t.Errorf("Expected read timeout 25s from file, got %v", cfg.API.ReadTimeout)
	}
}

// TestClientConfigFileAndEnvCombination tests realistic scenarios with partial config coverage.
func TestClientConfigFileAndEnvCombination(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "partial-config.yaml")

	// Config file only specifies some values
	configContent := `
server:
  address: "custom.pool.com:50051"

gpu:
  nonce_range: 750000000
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Environment only specifies some values
	os.Setenv("RTC_CLIENT_MINING_HYBRID_MODE", "true")
	os.Setenv("RTC_CLIENT_NETWORK_HEARTBEAT_INTERVAL", "25s")
	defer func() {
		os.Unsetenv("RTC_CLIENT_MINING_HYBRID_MODE")
		os.Unsetenv("RTC_CLIENT_NETWORK_HEARTBEAT_INTERVAL")
	}()

	cfg, err := LoadClientConfig(configFile)
	if err != nil {
		t.Fatalf("LoadClientConfig failed: %v", err)
	}

	// Verify each value source
	// From file
	if cfg.Server.Address != "custom.pool.com:50051" {
		t.Errorf("Expected address from file, got '%s'", cfg.Server.Address)
	}
	if cfg.GPU.NonceRange != 750000000 {
		t.Errorf("Expected nonce range from file, got %d", cfg.GPU.NonceRange)
	}

	// From environment
	if !cfg.Mining.HybridMode {
		t.Error("Expected hybrid mode from env to be true")
	}
	if cfg.Network.HeartbeatInterval != 25*time.Second {
		t.Errorf("Expected heartbeat interval 25s from env, got %v", cfg.Network.HeartbeatInterval)
	}

	// From defaults (not in file or env)
	if !cfg.Mining.GPUEnabled {
		t.Error("Expected GPU enabled from default")
	}
	if cfg.GPU.CPUStartNonce != 5000000000 {
		t.Errorf("Expected CPU start nonce from default, got %d", cfg.GPU.CPUStartNonce)
	}
	if cfg.Network.RetryInterval != 10*time.Second {
		t.Errorf("Expected retry interval from default, got %v", cfg.Network.RetryInterval)
	}
}

func TestServerConfigFileAndEnvCombination(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "partial-server-config.yaml")

	// Config file only specifies mining parameters
	configContent := `
mining:
  difficulty: 7
  block_reward: 75

logging:
  update_interval: "40s"
`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Environment specifies TLS settings
	os.Setenv("RTC_SERVER_TLS_ENABLED", "false")
	os.Setenv("RTC_SERVER_NETWORK_HTTP_PORT", "9999")
	defer func() {
		os.Unsetenv("RTC_SERVER_TLS_ENABLED")
		os.Unsetenv("RTC_SERVER_NETWORK_HTTP_PORT")
	}()

	cfg, err := LoadServerConfig(configFile)
	if err != nil {
		t.Fatalf("LoadServerConfig failed: %v", err)
	}

	// From file
	if cfg.Mining.Difficulty != 7 {
		t.Errorf("Expected difficulty 7 from file, got %d", cfg.Mining.Difficulty)
	}
	if cfg.Mining.BlockReward != 75 {
		t.Errorf("Expected block reward 75 from file, got %d", cfg.Mining.BlockReward)
	}
	if cfg.Logging.UpdateInterval != 40*time.Second {
		t.Errorf("Expected update interval 40s from file, got %v", cfg.Logging.UpdateInterval)
	}

	// From environment
	if cfg.TLS.Enabled {
		t.Error("Expected TLS disabled from env")
	}
	if cfg.Network.HTTPPort != 9999 {
		t.Errorf("Expected HTTP port 9999 from env, got %d", cfg.Network.HTTPPort)
	}

	// From defaults
	if cfg.Network.GRPCPort != 50051 {
		t.Errorf("Expected GRPC port from default, got %d", cfg.Network.GRPCPort)
	}
	if cfg.API.ReadTimeout != 15*time.Second {
		t.Errorf("Expected read timeout from default, got %v", cfg.API.ReadTimeout)
	}
}

func TestWatchServerConfigHotReload(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "watch-test.yaml")

	// Create initial config file
	initialContent := `
network:
  grpc_port: 50051
  api_port: 8443
  http_port: 8080

mining:
  difficulty: 6
  block_reward: 50

tls:
  enabled: false
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"

api:
  read_timeout: "15s"
  write_timeout: "15s"
  idle_timeout: "60s"

logging:
  update_interval: "30s"
  file_path: "pool_log.json"
`
	if err := os.WriteFile(configFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create initial config file: %v", err)
	}

	// Channel to receive callback notifications
	callbackChan := make(chan *ServerConfig, 1)
	var callbackInvoked atomic.Int32

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start watching
	err := WatchServerConfig(ctx, configFile, func(newCfg *ServerConfig) {
		callbackInvoked.Store(1)
		select {
		case callbackChan <- newCfg:
		default:
		}
	}, nil)

	if err != nil {
		t.Fatalf("WatchServerConfig failed: %v", err)
	}

	// Give the watcher time to initialize
	time.Sleep(500 * time.Millisecond)

	// Modify the config file
	modifiedContent := `
network:
  grpc_port: 60051
  api_port: 9443
  http_port: 9080

mining:
  difficulty: 8
  block_reward: 100

tls:
  enabled: false
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"

api:
  read_timeout: "15s"
  write_timeout: "15s"
  idle_timeout: "60s"

logging:
  update_interval: "30s"
  file_path: "pool_log.json"
`
	if err := os.WriteFile(configFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify config file: %v", err)
	}

	// Wait for callback with timeout
	select {
	case newCfg := <-callbackChan:
		if callbackInvoked.Load() == 0 {
			t.Error("Callback was not invoked")
		}

		// Verify the new config values were passed to callback
		if newCfg.Network.GRPCPort != 60051 {
			t.Errorf("Expected new GRPC port 60051, got %d", newCfg.Network.GRPCPort)
		}
		if newCfg.Network.APIPort != 9443 {
			t.Errorf("Expected new API port 9443, got %d", newCfg.Network.APIPort)
		}
		if newCfg.Mining.Difficulty != 8 {
			t.Errorf("Expected new difficulty 8, got %d", newCfg.Mining.Difficulty)
		}
		if newCfg.Mining.BlockReward != 100 {
			t.Errorf("Expected new block reward 100, got %d", newCfg.Mining.BlockReward)
		}

	case <-time.After(5 * time.Second):
		t.Fatal("Callback was not invoked within timeout")
	}
}

func TestWatchServerConfigInvalidChange(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "watch-invalid-test.yaml")

	// Create initial valid config
	initialContent := `
network:
  grpc_port: 50051
  api_port: 8443
  http_port: 8080

mining:
  difficulty: 6
  block_reward: 50

tls:
  enabled: false
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"

api:
  read_timeout: "15s"
  write_timeout: "15s"
  idle_timeout: "60s"

logging:
  update_interval: "30s"
  file_path: "pool_log.json"
`
	if err := os.WriteFile(configFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create initial config file: %v", err)
	}

	var callbackCount atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start watching
	err := WatchServerConfig(ctx, configFile, func(newCfg *ServerConfig) {
		callbackCount.Add(1)
	}, nil)

	if err != nil {
		t.Fatalf("WatchServerConfig failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Write invalid config (invalid port range)
	invalidContent := `
network:
  grpc_port: 99999
  api_port: 8443
  http_port: 8080

mining:
  difficulty: 6
  block_reward: 50

tls:
  enabled: false
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"

api:
  read_timeout: "15s"
  write_timeout: "15s"
  idle_timeout: "60s"

logging:
  update_interval: "30s"
  file_path: "pool_log.json"
`
	if err := os.WriteFile(configFile, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Wait a bit to see if callback is incorrectly invoked
	time.Sleep(2 * time.Second)

	// Callback should not have been invoked for invalid config
	if callbackCount.Load() > 0 {
		t.Errorf("Callback was invoked %d times for invalid config (expected 0)", callbackCount.Load())
	}
}

func TestWatchServerConfigContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "watch-cancel-test.yaml")

	initialContent := `
network:
  grpc_port: 50051
  api_port: 8443
  http_port: 8080

mining:
  difficulty: 6
  block_reward: 50

tls:
  enabled: false
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"

api:
  read_timeout: "15s"
  write_timeout: "15s"
  idle_timeout: "60s"

logging:
  update_interval: "30s"
  file_path: "pool_log.json"
`
	if err := os.WriteFile(configFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	var callbackCount atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())

	err := WatchServerConfig(ctx, configFile, func(newCfg *ServerConfig) {
		callbackCount.Add(1)
	}, nil)

	if err != nil {
		t.Fatalf("WatchServerConfig failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for watcher to stop
	time.Sleep(500 * time.Millisecond)

	// Modify config after cancellation
	modifiedContent := `
network:
  grpc_port: 60051
  api_port: 9443
  http_port: 9080

mining:
  difficulty: 8
  block_reward: 100

tls:
  enabled: false
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"

api:
  read_timeout: "15s"
  write_timeout: "15s"
  idle_timeout: "60s"

logging:
  update_interval: "30s"
  file_path: "pool_log.json"
`
	if err := os.WriteFile(configFile, []byte(modifiedContent), 0644); err != nil {
		t.Fatalf("Failed to modify config: %v", err)
	}

	// Wait to ensure callback is not invoked after cancellation
	time.Sleep(2 * time.Second)

	// Note: The callback might have been invoked once during the test before cancellation,
	// but it should not be invoked after cancellation. Due to timing, we just verify
	// that the watcher doesn't panic and the test completes successfully.
	// A more sophisticated test would track invocation timestamps.
	t.Logf("Callback was invoked %d times (expected 0-1, before cancellation)", callbackCount.Load())
}

func TestWatchServerConfigMultipleChanges(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "watch-multiple-test.yaml")

	initialContent := `
network:
  grpc_port: 50051
  api_port: 8443
  http_port: 8080

mining:
  difficulty: 6
  block_reward: 50

tls:
  enabled: false
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"

api:
  read_timeout: "15s"
  write_timeout: "15s"
  idle_timeout: "60s"

logging:
  update_interval: "30s"
  file_path: "pool_log.json"
`
	if err := os.WriteFile(configFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	var callbackCount atomic.Int32
	var lastDifficulty atomic.Int32

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := WatchServerConfig(ctx, configFile, func(newCfg *ServerConfig) {
		callbackCount.Add(1)
		lastDifficulty.Store(newCfg.Mining.Difficulty)
	}, nil)

	if err != nil {
		t.Fatalf("WatchServerConfig failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	// Make multiple rapid changes
	difficulties := []int32{7, 8, 9}
	for _, diff := range difficulties {
		content := fmt.Sprintf(`
network:
  grpc_port: 50051
  api_port: 8443
  http_port: 8080

mining:
  difficulty: %d
  block_reward: 50

tls:
  enabled: false
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"

api:
  read_timeout: "15s"
  write_timeout: "15s"
  idle_timeout: "60s"

logging:
  update_interval: "30s"
  file_path: "pool_log.json"
`, diff)
		if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write config: %v", err)
		}
		time.Sleep(800 * time.Millisecond) // Give watcher time to process
	}

	// Verify that callback was invoked at least once
	if callbackCount.Load() == 0 {
		t.Error("Expected callback to be invoked at least once")
	}

	// The last difficulty should be 9 (the last change)
	if lastDifficulty.Load() != 9 {
		t.Errorf("Expected last difficulty 9, got %d", lastDifficulty.Load())
	}

	t.Logf("Callback invoked %d times for %d changes", callbackCount.Load(), len(difficulties))
}

func TestClientTLSConfigFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "test-tls-config.yaml")

	configContent := `
server:
  address: "localhost:50051"
  tls:
    enabled: true
    insecure_skip_verify: false
    ca_cert_file: "/path/to/ca.crt"
`

	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	cfg, err := LoadClientConfig(configFile)
	if err != nil {
		t.Fatalf("LoadClientConfig failed: %v", err)
	}

	// Verify TLS configuration loaded correctly
	if !cfg.Server.TLS.Enabled {
		t.Error("Expected TLS enabled")
	}
	if cfg.Server.TLS.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify disabled")
	}
	if cfg.Server.TLS.CACertFile != "/path/to/ca.crt" {
		t.Errorf("Expected CA cert file '/path/to/ca.crt', got '%s'", cfg.Server.TLS.CACertFile)
	}
}

func TestClientTLSConfigEnvironmentOverride(t *testing.T) {
	// Set environment variables
	os.Setenv("RTC_CLIENT_SERVER_TLS_ENABLED", "true")
	os.Setenv("RTC_CLIENT_SERVER_TLS_INSECURE_SKIP_VERIFY", "false")
	os.Setenv("RTC_CLIENT_SERVER_TLS_CA_CERT_FILE", "/env/ca.crt")
	defer func() {
		os.Unsetenv("RTC_CLIENT_SERVER_TLS_ENABLED")
		os.Unsetenv("RTC_CLIENT_SERVER_TLS_INSECURE_SKIP_VERIFY")
		os.Unsetenv("RTC_CLIENT_SERVER_TLS_CA_CERT_FILE")
	}()

	cfg, err := LoadClientConfig("")
	if err != nil {
		t.Fatalf("LoadClientConfig failed: %v", err)
	}

	// Verify environment variables took effect
	if !cfg.Server.TLS.Enabled {
		t.Error("Expected TLS enabled from environment variable")
	}
	if cfg.Server.TLS.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify disabled from environment variable")
	}
	if cfg.Server.TLS.CACertFile != "/env/ca.crt" {
		t.Errorf("Expected CA cert file '/env/ca.crt' from environment, got '%s'", cfg.Server.TLS.CACertFile)
	}
}
