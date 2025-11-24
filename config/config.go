package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config основная конфигурация приложения
type Config struct {
	Server  ServerConfig  `json:"server,omitempty"`
	Client  ClientConfig  `json:"client,omitempty"`
	TUN     TUNConfig     `json:"tun,omitempty"`
	Proxy   ProxyConfig   `json:"proxy,omitempty"`
	Routing RoutingConfig `json:"routing,omitempty"`
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	Listen   string          `json:"listen"`   // Адрес для прослушивания (например, "0.0.0.0:25565")
	Protocol string          `json:"protocol"` // "minecraft"
	Settings ServerSettings  `json:"settings"`
}

// ServerSettings настройки сервера
type ServerSettings struct {
	Clients    []User `json:"clients"`              // Список пользователей
	Decryption string `json:"decryption,omitempty"` // "none" для нашего протокола
	Fallbacks  []string `json:"fallbacks,omitempty"` // Fallback адреса
}

// ClientConfig конфигурация клиента
type ClientConfig struct {
	Server   ServerInfo   `json:"server"`
	Protocol string       `json:"protocol"` // "minecraft"
	Settings ClientSettings `json:"settings"`
}

// ServerInfo информация о сервере
type ServerInfo struct {
	Address string `json:"address"` // Адрес сервера
	Port    int    `json:"port"`    // Порт сервера
}

// ClientSettings настройки клиента
type ClientSettings struct {
	UserID string `json:"user_id"` // UUID пользователя
	Flow   string `json:"flow,omitempty"` // Flow type
}

// TUNConfig конфигурация TUN mode
type TUNConfig struct {
	Enabled   bool     `json:"enabled"`
	Interface string   `json:"interface"` // Имя интерфейса (например, "tun0")
	IP        string   `json:"ip"`        // IP адрес интерфейса
	Subnet    string   `json:"subnet"`    // Подсеть
	Gateway   string   `json:"gateway"`   // Gateway
	MTU       int      `json:"mtu"`       // MTU (обычно 1500)
	DNS       []string `json:"dns"`       // DNS серверы
}

// ProxyConfig конфигурация System Proxy
type ProxyConfig struct {
	Enabled   bool           `json:"enabled"`
	SetSystem bool           `json:"set_system"` // Устанавливать ли системные настройки прокси
	HTTP      HTTPProxyConfig `json:"http"`
	SOCKS5    SOCKS5ProxyConfig `json:"socks5"`
	PAC       PACConfig      `json:"pac"`
}

// HTTPProxyConfig конфигурация HTTP proxy
type HTTPProxyConfig struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"` // По умолчанию 8080
}

// SOCKS5ProxyConfig конфигурация SOCKS5 proxy
type SOCKS5ProxyConfig struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"` // По умолчанию 1080
}

// PACConfig конфигурация PAC файла
type PACConfig struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"` // По умолчанию 8090
}

// RoutingConfig конфигурация routing
type RoutingConfig struct {
	Rules []RoutingRule `json:"rules"`
}

// RoutingRule правило routing
type RoutingRule struct {
	Type    string `json:"type"`    // "domain", "ip", "geoip", "default"
	Pattern string `json:"pattern,omitempty"` // Паттерн для domain
	Subnet  string `json:"subnet,omitempty"`  // Подсеть для IP
	Country string `json:"country,omitempty"` // Код страны для geoip
	Action  string `json:"action"`  // "proxy", "direct", "block"
}

// LoadConfig загружает конфигурацию из JSON файла
func LoadConfig(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &config, nil
}

// SaveConfig сохраняет конфигурацию в JSON файл
func SaveConfig(filename string, config *Config) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create config file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	return nil
}

// DefaultServerConfig возвращает конфигурацию сервера по умолчанию
func DefaultServerConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Listen:   "0.0.0.0:25565",
			Protocol: "minecraft",
			Settings: ServerSettings{
				Clients:    []User{},
				Decryption: "none",
			},
		},
	}
}

// DefaultClientConfig возвращает конфигурацию клиента по умолчанию
func DefaultClientConfig() *Config {
	return &Config{
		Client: ClientConfig{
			Server: ServerInfo{
				Address: "server.example.com",
				Port:    25565,
			},
			Protocol: "minecraft",
			Settings: ClientSettings{
				UserID: "00000000-0000-0000-0000-000000000000",
				Flow:   "",
			},
		},
		TUN: TUNConfig{
			Enabled:   false,
			Interface: "tun0",
			IP:        "10.0.0.2",
			Subnet:    "10.0.0.0/24",
			Gateway:   "10.0.0.1",
			MTU:       1500,
			DNS:       []string{"1.1.1.1", "8.8.8.8"},
		},
		Proxy: ProxyConfig{
			Enabled:   true,
			SetSystem: false,
			HTTP: HTTPProxyConfig{
				Enabled: true,
				Port:    8080,
			},
			SOCKS5: SOCKS5ProxyConfig{
				Enabled: true,
				Port:    1080,
			},
			PAC: PACConfig{
				Enabled: true,
				Port:    8090,
			},
		},
		Routing: RoutingConfig{
			Rules: []RoutingRule{
				{Type: "domain", Pattern: ".*\\.ru", Action: "direct"},
				{Type: "domain", Pattern: ".*\\.google\\.com", Action: "proxy"},
				{Type: "default", Action: "proxy"},
			},
		},
	}
}
