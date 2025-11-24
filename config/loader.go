package config

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"os"
)

// ServerConfigFile - конфигурация сервера из файла
type ServerConfigFile struct {
	ListenAddr string   `json:"listen_addr"`
	Users      []User   `json:"users"`
	LogLevel   string   `json:"log_level"`
	MaxStreams int      `json:"max_streams"`
}

// ClientConfigFile - конфигурация клиента из файла
type ClientConfigFile struct {
	ServerAddr string `json:"server_addr"`
	ServerPort int    `json:"server_port"`
	UserID     string `json:"user_id"`
	LogLevel   string `json:"log_level"`
}

// ProxyConfigFile - конфигурация proxy из файла
type ProxyConfigFile struct {
	Mode       string `json:"mode"`        // "tcp", "http", "socks5"
	ListenAddr string `json:"listen_addr"`
	TargetAddr string `json:"target_addr"` // для TCP режима

	// Koria server connection
	ServerAddr string `json:"server_addr"`
	ServerPort int    `json:"server_port"`
	UserID     string `json:"user_id"`

	LogLevel   string `json:"log_level"`
}

// DefaultServerConfigFile возвращает конфигурацию сервера по умолчанию
func DefaultServerConfigFile() *ServerConfigFile {
	return &ServerConfigFile{
		ListenAddr: "0.0.0.0:25565",
		Users: []User{
			{
				ID:    uuid.New(),
				Email: "admin@koria.local",
				Level: 0,
			},
		},
		LogLevel:   "info",
		MaxStreams: 1000,
	}
}

// DefaultClientConfigFile возвращает конфигурацию клиента по умолчанию
func DefaultClientConfigFile() *ClientConfigFile {
	return &ClientConfigFile{
		ServerAddr: "127.0.0.1",
		ServerPort: 25565,
		UserID:     uuid.New().String(),
		LogLevel:   "info",
	}
}

// DefaultProxyConfigFile возвращает конфигурацию proxy по умолчанию
func DefaultProxyConfigFile() *ProxyConfigFile {
	return &ProxyConfigFile{
		Mode:       "tcp",
		ListenAddr: "127.0.0.1:8080",
		TargetAddr: "httpbin.org:80",
		ServerAddr: "127.0.0.1",
		ServerPort: 25565,
		UserID:     uuid.New().String(),
		LogLevel:   "info",
	}
}

// LoadServerConfig загружает конфигурацию сервера из файла
func LoadServerConfig(filename string) (*ServerConfigFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg ServerConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// LoadClientConfig загружает конфигурацию клиента из файла
func LoadClientConfig(filename string) (*ClientConfigFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg ClientConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// LoadProxyConfig загружает конфигурацию proxy из файла
func LoadProxyConfig(filename string) (*ProxyConfigFile, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg ProxyConfigFile
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// SaveServerConfig сохраняет конфигурацию сервера в файл
func SaveServerConfig(filename string, cfg *ServerConfigFile) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// SaveClientConfig сохраняет конфигурацию клиента в файл
func SaveClientConfig(filename string, cfg *ClientConfigFile) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

// SaveProxyConfig сохраняет конфигурацию proxy в файл
func SaveProxyConfig(filename string, cfg *ProxyConfigFile) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
