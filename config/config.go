// Package config provides centralized configuration management using Viper.
// It supports loading configuration from files, environment variables, and
// command-line flags with a clear hierarchy: Flags > Env > Config File > Defaults.
package config

import (
	"fmt"
	"time"

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

	return &config, nil
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
