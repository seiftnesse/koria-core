package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config represents the koria-core configuration
type Config struct {
	// Server settings
	ListenAddress string        `json:"listen_address"`
	UpstreamAddr  string        `json:"upstream_address"`
	Timeout       time.Duration `json:"timeout"`
	
	// Minecraft camouflage settings
	MinecraftServer string `json:"minecraft_server"`
	MinecraftPort   uint16 `json:"minecraft_port"`
	
	// Logging
	LogLevel string `json:"log_level"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		ListenAddress:   "127.0.0.1:8080",
		UpstreamAddr:    "127.0.0.1:8081",
		Timeout:         30 * time.Second,
		MinecraftServer: "localhost",
		MinecraftPort:   25565,
		LogLevel:        "info",
	}
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to a JSON file
func SaveConfig(config *Config, filename string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ListenAddress == "" {
		return fmt.Errorf("listen_address is required")
	}
	if c.UpstreamAddr == "" {
		return fmt.Errorf("upstream_address is required")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}

// UnmarshalJSON implements custom JSON unmarshaling for duration
func (c *Config) UnmarshalJSON(data []byte) error {
	type Alias Config
	aux := &struct {
		Timeout string `json:"timeout"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.Timeout != "" {
		duration, err := time.ParseDuration(aux.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout format: %w", err)
		}
		c.Timeout = duration
	}

	return nil
}

// MarshalJSON implements custom JSON marshaling for duration
func (c *Config) MarshalJSON() ([]byte, error) {
	type Alias Config
	return json.Marshal(&struct {
		Timeout string `json:"timeout"`
		*Alias
	}{
		Timeout: c.Timeout.String(),
		Alias:   (*Alias)(c),
	})
}
