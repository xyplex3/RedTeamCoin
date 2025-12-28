// Package config provides centralized configuration management using Viper.
// It supports loading configuration from files, environment variables, and
// command-line flags with a clear hierarchy: Flags > Env > Config File > Defaults.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// ClientConfig contains all configuration options for the mining client.
type ClientConfig struct {
	Server   ServerConnection `mapstructure:"server"`
	Mining   MiningConfig     `mapstructure:"mining"`
	GPU      GPUConfig        `mapstructure:"gpu"`
	Network  NetworkConfig    `mapstructure:"network"`
	Behavior BehaviorConfig   `mapstructure:"behavior"`
}

// ServerConnection defines pool server connection settings.
type ServerConnection struct {
	Address string `mapstructure:"address"`
}

// MiningConfig defines mining behavior and performance settings.
type MiningConfig struct {
	GPUEnabled bool `mapstructure:"gpu_enabled"`
	HybridMode bool `mapstructure:"hybrid_mode"`
	AutoDelete bool `mapstructure:"auto_delete"`
}

// GPUConfig defines GPU-specific mining parameters.
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

// BehaviorConfig defines client operational behavior.
type BehaviorConfig struct {
	WorkerUpdateInterval int64 `mapstructure:"worker_update_interval"`
}

// ServerConfig contains all configuration options for the pool server.
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

// ServerMining defines pool mining parameters.
type ServerMining struct {
	Difficulty  int `mapstructure:"difficulty"`
	BlockReward int `mapstructure:"block_reward"`
}

// TLSConfig defines TLS/HTTPS settings.
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// APIConfig defines API server behavior.
type APIConfig struct {
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
}

// LoggingConfig defines logging behavior.
type LoggingConfig struct {
	UpdateInterval time.Duration `mapstructure:"update_interval"`
	FilePath       string        `mapstructure:"file_path"`
}

// Validate checks if the client configuration is valid and returns an error if not.
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

	return nil
}

// Validate checks if the server configuration is valid and returns an error if not.
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
	if c.Logging.UpdateInterval < time.Second {
		return fmt.Errorf("logging.update_interval too short (minimum 1s), got %v", c.Logging.UpdateInterval)
	}
	if c.Logging.FilePath == "" {
		return fmt.Errorf("logging.file_path cannot be empty")
	}
	return nil
}

// LoadClientConfig loads client configuration from file, environment, and defaults.
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

// WatchServerConfig sets up file watching for the server configuration and
// calls the provided callback function whenever the config file changes.
// The callback receives the newly loaded configuration.
//
// This function blocks and should be run in a goroutine.
func WatchServerConfig(configPath string, callback func(*ServerConfig)) error {
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
		fmt.Printf("Config file changed: %s\n", e.Name)

		var newConfig ServerConfig
		if err := v.Unmarshal(&newConfig); err != nil {
			fmt.Printf("Error unmarshaling config on reload: %v\n", err)
			return
		}

		if err := newConfig.Validate(); err != nil {
			fmt.Printf("Invalid configuration after reload: %v\n", err)
			return
		}

		callback(&newConfig)
	})

	// Block forever (this function should be run in a goroutine)
	select {}
}

func setClientDefaults(v *viper.Viper) {
	v.SetDefault("server.address", "localhost:50051")
	v.SetDefault("mining.gpu_enabled", true)
	v.SetDefault("mining.hybrid_mode", false)
	v.SetDefault("mining.auto_delete", true)
	v.SetDefault("gpu.nonce_range", 500000000)
	v.SetDefault("gpu.cpu_start_nonce", 5000000000)
	v.SetDefault("network.heartbeat_interval", 30*time.Second)
	v.SetDefault("network.retry_interval", 10*time.Second)
	v.SetDefault("network.max_retry_time", 5*time.Minute)
	v.SetDefault("behavior.worker_update_interval", 100000)
}

func setServerDefaults(v *viper.Viper) {
	v.SetDefault("network.grpc_port", 50051)
	v.SetDefault("network.api_port", 8443)
	v.SetDefault("network.http_port", 8080)
	v.SetDefault("mining.difficulty", 6)
	v.SetDefault("mining.block_reward", 50)
	v.SetDefault("tls.enabled", false)
	v.SetDefault("tls.cert_file", "certs/server.crt")
	v.SetDefault("tls.key_file", "certs/server.key")
	v.SetDefault("api.read_timeout", 15*time.Second)
	v.SetDefault("api.write_timeout", 15*time.Second)
	v.SetDefault("api.idle_timeout", 60*time.Second)
	v.SetDefault("logging.update_interval", 30*time.Second)
	v.SetDefault("logging.file_path", "pool_log.json")
}
