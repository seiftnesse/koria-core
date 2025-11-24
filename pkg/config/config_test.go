package config

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.ListenAddress == "" {
		t.Error("Default listen address should not be empty")
	}

	if cfg.UpstreamAddr == "" {
		t.Error("Default upstream address should not be empty")
	}

	if cfg.Timeout <= 0 {
		t.Error("Default timeout should be positive")
	}

	if cfg.MinecraftPort == 0 {
		t.Error("Default Minecraft port should not be zero")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty listen address",
			config: &Config{
				ListenAddress: "",
				UpstreamAddr:  "localhost:8081",
				Timeout:       30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "empty upstream address",
			config: &Config{
				ListenAddress: "localhost:8080",
				UpstreamAddr:  "",
				Timeout:       30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "negative timeout",
			config: &Config{
				ListenAddress: "localhost:8080",
				UpstreamAddr:  "localhost:8081",
				Timeout:       -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: &Config{
				ListenAddress: "localhost:8080",
				UpstreamAddr:  "localhost:8081",
				Timeout:       0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create temporary file
	tmpfile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.Close()

	// Save config
	originalConfig := DefaultConfig()
	originalConfig.ListenAddress = "0.0.0.0:9999"
	originalConfig.Timeout = 45 * time.Second

	err = SaveConfig(originalConfig, tmpfile.Name())
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Load config
	loadedConfig, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Compare
	if loadedConfig.ListenAddress != originalConfig.ListenAddress {
		t.Errorf("ListenAddress mismatch: got %s, want %s", loadedConfig.ListenAddress, originalConfig.ListenAddress)
	}

	if loadedConfig.UpstreamAddr != originalConfig.UpstreamAddr {
		t.Errorf("UpstreamAddr mismatch: got %s, want %s", loadedConfig.UpstreamAddr, originalConfig.UpstreamAddr)
	}

	if loadedConfig.Timeout != originalConfig.Timeout {
		t.Errorf("Timeout mismatch: got %s, want %s", loadedConfig.Timeout, originalConfig.Timeout)
	}
}

func TestConfigJSONMarshaling(t *testing.T) {
	cfg := &Config{
		ListenAddress:   "127.0.0.1:8080",
		UpstreamAddr:    "127.0.0.1:8081",
		Timeout:         30 * time.Second,
		MinecraftServer: "localhost",
		MinecraftPort:   25565,
		LogLevel:        "info",
	}

	// Marshal
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var loaded Config
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Compare
	if loaded.Timeout != cfg.Timeout {
		t.Errorf("Timeout mismatch after marshal/unmarshal: got %s, want %s", loaded.Timeout, cfg.Timeout)
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	_, err := LoadConfig("/tmp/nonexistent-config-file.json")
	if err == nil {
		t.Error("Expected error when loading non-existent config")
	}
}

func TestLoadConfigInvalidJSON(t *testing.T) {
	// Create temporary file with invalid JSON
	tmpfile, err := os.CreateTemp("", "invalid-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	tmpfile.WriteString("{ invalid json }")
	tmpfile.Close()

	_, err = LoadConfig(tmpfile.Name())
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}
}

func TestTimeoutParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{"seconds", `{"timeout":"30s","listen_address":"a","upstream_address":"b"}`, 30 * time.Second},
		{"minutes", `{"timeout":"5m","listen_address":"a","upstream_address":"b"}`, 5 * time.Minute},
		{"hours", `{"timeout":"1h","listen_address":"a","upstream_address":"b"}`, 1 * time.Hour},
		{"combined", `{"timeout":"1h30m","listen_address":"a","upstream_address":"b"}`, 90 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			err := json.Unmarshal([]byte(tt.input), &cfg)
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if cfg.Timeout != tt.expected {
				t.Errorf("Timeout mismatch: got %s, want %s", cfg.Timeout, tt.expected)
			}
		})
	}
}
