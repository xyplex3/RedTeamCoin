package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestClientConfigDefaults verifies that default client configuration values are correct.
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
	if cfg.Mining.GPUEnabled != true {
		t.Error("Expected GPU enabled by default")
	}
	if cfg.Mining.HybridMode != false {
		t.Error("Expected hybrid mode disabled by default")
	}
	if cfg.Mining.AutoDelete != true {
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
}

// TestServerConfigDefaults verifies that default server configuration values are correct.
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
	if cfg.TLS.Enabled != false {
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

// TestClientConfigFromFile tests loading client configuration from a YAML file.
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
	if cfg.Mining.GPUEnabled != false {
		t.Error("Expected GPU disabled")
	}
	if cfg.Mining.HybridMode != true {
		t.Error("Expected hybrid mode enabled")
	}
	if cfg.GPU.NonceRange != 1000000000 {
		t.Errorf("Expected nonce range 1000000000, got %d", cfg.GPU.NonceRange)
	}
	if cfg.Network.HeartbeatInterval != 60*time.Second {
		t.Errorf("Expected heartbeat interval 60s, got %v", cfg.Network.HeartbeatInterval)
	}
}

// TestServerConfigFromFile tests loading server configuration from a YAML file.
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

// TestClientConfigEnvironmentOverride tests that environment variables override config file values.
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
	if cfg.Mining.GPUEnabled != false {
		t.Error("Expected GPU disabled from env")
	}
	if cfg.GPU.NonceRange != 999999999 {
		t.Errorf("Expected nonce range 999999999 from env, got %d", cfg.GPU.NonceRange)
	}
}

// TestServerConfigEnvironmentOverride tests that environment variables override config file values.
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

// TestClientConfigValidation tests validation logic for client configuration.
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

// TestServerConfigValidation tests validation logic for server configuration.
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

// TestInvalidYAML tests handling of malformed YAML configuration.
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

// TestInvalidConfigValues tests that invalid values in config file trigger validation errors.
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

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}
