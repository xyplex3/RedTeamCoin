// Package config provides centralized configuration management using Viper.
// It supports loading configuration from files, environment variables, and
// command-line flags with a clear hierarchy: Flags > Env > Config File > Defaults.
package config

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Default client configuration values.
const (
	DefaultClientServerAddress         = "localhost:50051"
	DefaultClientGPUEnabled            = true
	DefaultClientHybridMode            = false
	DefaultClientAutoDelete            = true
	DefaultClientGPUNonceRange         = 500000000
	DefaultClientGPUCPUStartNonce      = 5000000000
	DefaultClientHeartbeatInterval     = 30 * time.Second
	DefaultClientRetryInterval         = 10 * time.Second
	DefaultClientMaxRetryTime          = 5 * time.Minute
	DefaultClientWorkerUpdateInterval  = 100000
	DefaultClientLoggingLevel          = "info"
	DefaultClientLoggingFormat         = "color"
	DefaultClientLoggingQuiet          = false
	DefaultClientLoggingVerbose        = false
	DefaultClientTLSEnabled            = false
	DefaultClientTLSInsecureSkipVerify = true
)

// Default server configuration values.
const (
	DefaultServerGRPCPort              = 50051
	DefaultServerAPIPort               = 8443
	DefaultServerHTTPPort              = 8080
	DefaultServerMiningDifficulty      = 6
	DefaultServerBlockReward           = 50
	DefaultServerTLSEnabled            = false
	DefaultServerTLSCertFile           = "certs/server.crt"
	DefaultServerTLSKeyFile            = "certs/server.key"
	DefaultServerAPIReadTimeout        = 15 * time.Second
	DefaultServerAPIWriteTimeout       = 15 * time.Second
	DefaultServerAPIIdleTimeout        = 60 * time.Second
	DefaultServerLoggingUpdateInterval = 30 * time.Second
	DefaultServerLoggingFilePath       = "pool_log.json"
	DefaultServerLoggingLevel          = "info"
	DefaultServerLoggingFormat         = "color"
	DefaultServerLoggingQuiet          = false
	DefaultServerLoggingVerbose        = false
)

type ClientConfig struct {
	Server   ServerConnection    `mapstructure:"server"`
	Mining   MiningConfig        `mapstructure:"mining"`
	GPU      GPUConfig           `mapstructure:"gpu"`
	Network  NetworkConfig       `mapstructure:"network"`
	Behavior BehaviorConfig      `mapstructure:"behavior"`
	Logging  ClientLoggingConfig `mapstructure:"logging"`
}

type ServerConnection struct {
	Address string          `mapstructure:"address"`
	TLS     ClientTLSConfig `mapstructure:"tls"`
}

// ClientTLSConfig defines TLS settings for client gRPC connections.
//
// When Enabled is true, the client uses TLS to connect to the server.
// InsecureSkipVerify disables certificate validation (insecure for production).
// CACertFile specifies a custom CA certificate for server validation.
//
// Security note: Setting InsecureSkipVerify to true disables certificate
// validation and makes connections vulnerable to man-in-the-middle attacks.
// Only use in development or with additional security controls.
type ClientTLSConfig struct {
	Enabled            bool   `mapstructure:"enabled"`
	InsecureSkipVerify bool   `mapstructure:"insecure_skip_verify"`
	CACertFile         string `mapstructure:"ca_cert_file"`
}

type MiningConfig struct {
	GPUEnabled bool `mapstructure:"gpu_enabled"`
	HybridMode bool `mapstructure:"hybrid_mode"`
	AutoDelete bool `mapstructure:"auto_delete"`
}

type GPUConfig struct {
	NonceRange    int64 `mapstructure:"nonce_range"`
	CPUStartNonce int64 `mapstructure:"cpu_start_nonce"`
}

// NetworkConfig defines network timing and retry behavior.
type NetworkConfig struct {
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	RetryInterval     time.Duration `mapstructure:"retry_interval"`
	MaxRetryTime      time.Duration `mapstructure:"max_retry_time"`
}

type BehaviorConfig struct {
	WorkerUpdateInterval int64 `mapstructure:"worker_update_interval"`
}

type ClientLoggingConfig struct {
	Level   string `mapstructure:"level"`   // debug, info, warn, error
	Format  string `mapstructure:"format"`  // text, color, json
	Quiet   bool   `mapstructure:"quiet"`   // suppress all but errors
	Verbose bool   `mapstructure:"verbose"` // enable debug logs
}

type ServerConfig struct {
	Network ServerNetwork `mapstructure:"network"`
	Mining  ServerMining  `mapstructure:"mining"`
	TLS     TLSConfig     `mapstructure:"tls"`
	API     APIConfig     `mapstructure:"api"`
	Logging LoggingConfig `mapstructure:"logging"`
}

// ServerNetwork defines server listening addresses and ports.
type ServerNetwork struct {
	GRPCPort int `mapstructure:"grpc_port"`
	APIPort  int `mapstructure:"api_port"`
	HTTPPort int `mapstructure:"http_port"`
}

type ServerMining struct {
	Difficulty  int32 `mapstructure:"difficulty"`
	BlockReward int   `mapstructure:"block_reward"`
}

type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

type APIConfig struct {
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

type LoggingConfig struct {
	// PoolLogger fields (existing - for JSON event analytics)
	UpdateInterval time.Duration `mapstructure:"update_interval"`
	FilePath       string        `mapstructure:"file_path"`

	// Application logging fields (new - for operational logs)
	Level   string `mapstructure:"level"`   // debug, info, warn, error
	Format  string `mapstructure:"format"`  // text, color, json
	Quiet   bool   `mapstructure:"quiet"`   // suppress all but errors
	Verbose bool   `mapstructure:"verbose"` // enable debug logs
}

func (c *ClientConfig) Validate() error {
	if c.Server.Address == "" {
		return fmt.Errorf("server address cannot be empty")
	}

	if c.GPU.NonceRange <= 0 {
		return fmt.Errorf("nonce_range must be positive, got %d", c.GPU.NonceRange)
	}

	if c.GPU.CPUStartNonce < 0 {
		return fmt.Errorf("cpu_start_nonce cannot be negative, got %d", c.GPU.CPUStartNonce)
	}

	if c.Network.HeartbeatInterval < time.Second {
		return fmt.Errorf("heartbeat_interval too short (minimum 1s), got %v", c.Network.HeartbeatInterval)
	}

	if c.Network.RetryInterval < time.Second {
		return fmt.Errorf("retry_interval too short (minimum 1s), got %v", c.Network.RetryInterval)
	}

	if c.Network.MaxRetryTime < c.Network.RetryInterval {
		return fmt.Errorf("max_retry_time (%v) must be >= retry_interval (%v)", c.Network.MaxRetryTime, c.Network.RetryInterval)
	}

	if c.Behavior.WorkerUpdateInterval <= 0 {
		return fmt.Errorf("worker_update_interval must be positive, got %d", c.Behavior.WorkerUpdateInterval)
	}

	// Validate logging configuration
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "warning": true, "error": true}
	if c.Logging.Level != "" && !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid logging.level: %q (must be debug, info, warn, or error)", c.Logging.Level)
	}

	validFormats := map[string]bool{"text": true, "color": true, "json": true}
	if c.Logging.Format != "" && !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid logging.format: %q (must be text, color, or json)", c.Logging.Format)
	}

	// TLS configuration is validated at connection time
	// CA certificate file existence checking happens in createClientTLSConfig

	return nil
}

func (c *ServerConfig) Validate() error {
	if err := c.validatePorts(); err != nil {
		return err
	}
	if err := c.validateMiningConfig(); err != nil {
		return err
	}
	if err := c.validateTLSConfig(); err != nil {
		return err
	}
	if err := c.validateAPIConfig(); err != nil {
		return err
	}
	return c.validateLoggingConfig()
}

func (c *ServerConfig) validatePorts() error {
	if c.Network.GRPCPort < 1 || c.Network.GRPCPort > 65535 {
		return fmt.Errorf("invalid grpc_port: %d (must be 1-65535)", c.Network.GRPCPort)
	}
	if c.Network.APIPort < 1 || c.Network.APIPort > 65535 {
		return fmt.Errorf("invalid api_port: %d (must be 1-65535)", c.Network.APIPort)
	}
	if c.Network.HTTPPort < 1 || c.Network.HTTPPort > 65535 {
		return fmt.Errorf("invalid http_port: %d (must be 1-65535)", c.Network.HTTPPort)
	}
	if c.Network.GRPCPort == c.Network.APIPort || c.Network.GRPCPort == c.Network.HTTPPort || c.Network.APIPort == c.Network.HTTPPort {
		return fmt.Errorf("ports must be unique: grpc=%d, api=%d, http=%d", c.Network.GRPCPort, c.Network.APIPort, c.Network.HTTPPort)
	}
	return nil
}

func (c *ServerConfig) validateMiningConfig() error {
	if c.Mining.Difficulty < 1 || c.Mining.Difficulty > 64 {
		return fmt.Errorf("invalid difficulty: %d (must be 1-64)", c.Mining.Difficulty)
	}
	if c.Mining.BlockReward <= 0 {
		return fmt.Errorf("block_reward must be positive, got %d", c.Mining.BlockReward)
	}
	return nil
}

func (c *ServerConfig) validateTLSConfig() error {
	if !c.TLS.Enabled {
		return nil
	}
	if c.TLS.CertFile == "" {
		return fmt.Errorf("tls.cert_file is required when tls.enabled is true")
	}
	if c.TLS.KeyFile == "" {
		return fmt.Errorf("tls.key_file is required when tls.enabled is true")
	}
	return nil
}

func (c *ServerConfig) validateAPIConfig() error {
	if c.API.ReadTimeout < time.Second {
		return fmt.Errorf("api.read_timeout too short (minimum 1s), got %v", c.API.ReadTimeout)
	}
	if c.API.WriteTimeout < time.Second {
		return fmt.Errorf("api.write_timeout too short (minimum 1s), got %v", c.API.WriteTimeout)
	}
	if c.API.IdleTimeout < time.Second {
		return fmt.Errorf("api.idle_timeout too short (minimum 1s), got %v", c.API.IdleTimeout)
	}
	return nil
}

func (c *ServerConfig) validateLoggingConfig() error {
	// Validate PoolLogger fields
	if c.Logging.UpdateInterval < time.Second {
		return fmt.Errorf("logging.update_interval too short (minimum 1s), got %v", c.Logging.UpdateInterval)
	}
	if c.Logging.FilePath == "" {
		return fmt.Errorf("logging.file_path cannot be empty")
	}

	// Validate application logging fields
	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "warning": true, "error": true}
	if c.Logging.Level != "" && !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid logging.level: %q (must be debug, info, warn, or error)", c.Logging.Level)
	}

	validFormats := map[string]bool{"text": true, "color": true, "json": true}
	if c.Logging.Format != "" && !validFormats[c.Logging.Format] {
		return fmt.Errorf("invalid logging.format: %q (must be text, color, or json)", c.Logging.Format)
	}

	return nil
}

// LoadClientConfig loads client configuration from file, environment, and defaults.
//
// Configuration sources are applied in the following precedence order (highest to lowest):
//  1. Command-line flags (handled by caller, not by this function)
//  2. Environment variables (RTC_CLIENT_* prefix, e.g., RTC_CLIENT_SERVER_ADDRESS)
//  3. Configuration file (client-config.yaml or specified path)
//  4. Default values (built-in sensible defaults)
//
// Environment Variable Naming:
// Environment variables use the prefix RTC_CLIENT_ followed by the nested config key
// with dots replaced by underscores. Examples:
//   - server.address        → RTC_CLIENT_SERVER_ADDRESS
//   - mining.gpu_enabled    → RTC_CLIENT_MINING_GPU_ENABLED
//   - gpu.nonce_range       → RTC_CLIENT_GPU_NONCE_RANGE
//   - network.retry_interval → RTC_CLIENT_NETWORK_RETRY_INTERVAL
//
// Configuration File Search Paths:
// If configPath is empty, the function searches for "client-config.yaml" in:
//  1. Current directory (.)
//  2. User config directory (~/.rtc)
//  3. System config directory (/etc/rtc)
//
// If no config file is found in the search paths, defaults are used without error.
// If configPath is specified but the file doesn't exist or can't be read, an error is returned.
//
// Validation:
// The loaded configuration is validated before being returned. Invalid values
// (e.g., empty server address, negative timeouts) will cause an error to be returned.
func LoadClientConfig(configPath string) (*ClientConfig, error) {
	v := viper.New()

	setClientDefaults(v)

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("client-config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.rtc")
		v.AddConfigPath("/etc/rtc")
	}

	v.SetEnvPrefix("RTC_CLIENT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config ClientConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// LoadServerConfig loads server configuration from file, environment, and defaults.
//
// Configuration sources are applied in the following precedence order (highest to lowest):
//  1. Command-line flags (handled by caller, not by this function)
//  2. Environment variables (RTC_SERVER_* prefix, e.g., RTC_SERVER_NETWORK_GRPC_PORT)
//  3. Configuration file (server-config.yaml or specified path)
//  4. Default values (built-in sensible defaults)
//
// Environment Variable Naming:
// Environment variables use the prefix RTC_SERVER_ followed by the nested config key
// with dots replaced by underscores. Examples:
//   - network.grpc_port     → RTC_SERVER_NETWORK_GRPC_PORT
//   - mining.difficulty     → RTC_SERVER_MINING_DIFFICULTY
//   - tls.enabled           → RTC_SERVER_TLS_ENABLED
//   - tls.cert_file         → RTC_SERVER_TLS_CERT_FILE
//   - api.read_timeout      → RTC_SERVER_API_READ_TIMEOUT
//   - logging.file_path     → RTC_SERVER_LOGGING_FILE_PATH
//
// Configuration File Search Paths:
// If configPath is empty, the function searches for "server-config.yaml" in:
//  1. Current directory (.)
//  2. User config directory (~/.rtc)
//  3. System config directory (/etc/rtc)
//
// If no config file is found in the search paths, defaults are used without error.
// If configPath is specified but the file doesn't exist or can't be read, an error is returned.
//
// Validation:
// The loaded configuration is validated before being returned. Invalid values
// (e.g., out-of-range ports, invalid difficulty, missing TLS files when enabled)
// will cause an error to be returned.
func LoadServerConfig(configPath string) (*ServerConfig, error) {
	v := viper.New()

	setServerDefaults(v)

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("server-config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.rtc")
		v.AddConfigPath("/etc/rtc")
	}

	v.SetEnvPrefix("RTC_SERVER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config ServerConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// WatchServerConfig starts a background goroutine that watches the server
// configuration file and calls the callback when changes are detected.
// The watcher stops when the context is cancelled. Returns immediately after
// starting the watcher, or an error if initial config read fails.
// If logger is nil, logging is disabled.
func WatchServerConfig(ctx context.Context, configPath string, callback func(*ServerConfig), logger *slog.Logger) error {
	v := viper.New()

	setServerDefaults(v)

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("server-config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.rtc")
		v.AddConfigPath("/etc/rtc")
	}

	v.SetEnvPrefix("RTC_SERVER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Initial read
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Set up file watching
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		if logger != nil {
			logger.Info("configuration file changed",
				"file", e.Name,
				"operation", e.Op.String())
		}

		var newConfig ServerConfig
		if err := v.Unmarshal(&newConfig); err != nil {
			if logger != nil {
				logger.Error("failed to unmarshal config on reload",
					"error", err,
					"file", e.Name)
			}
			return
		}

		if err := newConfig.Validate(); err != nil {
			if logger != nil {
				logger.Error("invalid configuration after reload",
					"error", err,
					"file", e.Name)
			}
			return
		}

		if logger != nil {
			logger.Info("configuration reloaded successfully",
				"file", e.Name)
		}

		callback(&newConfig)
	})

	// Start goroutine to block until context cancellation
	go func() {
		<-ctx.Done()
		if logger != nil {
			logger.Debug("config watcher stopped",
				"reason", "context cancelled")
		}
	}()

	return nil
}

func setClientDefaults(v *viper.Viper) {
	v.SetDefault("server.address", DefaultClientServerAddress)
	v.SetDefault("mining.gpu_enabled", DefaultClientGPUEnabled)
	v.SetDefault("mining.hybrid_mode", DefaultClientHybridMode)
	v.SetDefault("mining.auto_delete", DefaultClientAutoDelete)
	v.SetDefault("gpu.nonce_range", DefaultClientGPUNonceRange)
	v.SetDefault("gpu.cpu_start_nonce", DefaultClientGPUCPUStartNonce)
	v.SetDefault("network.heartbeat_interval", DefaultClientHeartbeatInterval)
	v.SetDefault("network.retry_interval", DefaultClientRetryInterval)
	v.SetDefault("network.max_retry_time", DefaultClientMaxRetryTime)
	v.SetDefault("behavior.worker_update_interval", DefaultClientWorkerUpdateInterval)
	v.SetDefault("logging.level", DefaultClientLoggingLevel)
	v.SetDefault("logging.format", DefaultClientLoggingFormat)
	v.SetDefault("logging.quiet", DefaultClientLoggingQuiet)
	v.SetDefault("logging.verbose", DefaultClientLoggingVerbose)
	v.SetDefault("server.tls.enabled", DefaultClientTLSEnabled)
	v.SetDefault("server.tls.insecure_skip_verify", DefaultClientTLSInsecureSkipVerify)
	v.SetDefault("server.tls.ca_cert_file", "")
}

func setServerDefaults(v *viper.Viper) {
	v.SetDefault("network.grpc_port", DefaultServerGRPCPort)
	v.SetDefault("network.api_port", DefaultServerAPIPort)
	v.SetDefault("network.http_port", DefaultServerHTTPPort)
	v.SetDefault("mining.difficulty", DefaultServerMiningDifficulty)
	v.SetDefault("mining.block_reward", DefaultServerBlockReward)
	v.SetDefault("tls.enabled", DefaultServerTLSEnabled)
	v.SetDefault("tls.cert_file", DefaultServerTLSCertFile)
	v.SetDefault("tls.key_file", DefaultServerTLSKeyFile)
	v.SetDefault("api.read_timeout", DefaultServerAPIReadTimeout)
	v.SetDefault("api.write_timeout", DefaultServerAPIWriteTimeout)
	v.SetDefault("api.idle_timeout", DefaultServerAPIIdleTimeout)
	// PoolLogger defaults (existing)
	v.SetDefault("logging.update_interval", DefaultServerLoggingUpdateInterval)
	v.SetDefault("logging.file_path", DefaultServerLoggingFilePath)
	// Application logging defaults (new)
	v.SetDefault("logging.level", DefaultServerLoggingLevel)
	v.SetDefault("logging.format", DefaultServerLoggingFormat)
	v.SetDefault("logging.quiet", DefaultServerLoggingQuiet)
	v.SetDefault("logging.verbose", DefaultServerLoggingVerbose)
}
