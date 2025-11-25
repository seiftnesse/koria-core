package v2

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config главная конфигурация
type Config struct {
	Log       *LogConfig       `json:"log,omitempty"`
	Inbounds  []InboundConfig  `json:"inbounds"`
	Outbounds []OutboundConfig `json:"outbounds"`
	Routing   *RoutingConfig   `json:"routing,omitempty"`
}

// LogConfig конфигурация логирования
type LogConfig struct {
	Level string `json:"level"` // "debug", "info", "warning", "error"
}

// InboundConfig конфигурация inbound
type InboundConfig struct {
	Tag      string                 `json:"tag"`
	Protocol string                 `json:"protocol"` // "http", "socks", "koria"
	Listen   string                 `json:"listen"`   // "127.0.0.1:8080"
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// OutboundConfig конфигурация outbound
type OutboundConfig struct {
	Tag      string                 `json:"tag"`
	Protocol string                 `json:"protocol"` // "freedom", "koria"
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// RoutingConfig конфигурация маршрутизации
type RoutingConfig struct {
	DomainStrategy string        `json:"domainStrategy,omitempty"` // "AsIs", "IPIfNonMatch", "IPOnDemand"
	Rules          []RoutingRule `json:"rules"`
}

// RoutingRule правило маршрутизации
type RoutingRule struct {
	Type        string   `json:"type,omitempty"`        // "field"
	Domain      []string `json:"domain,omitempty"`      // Domain matching
	IP          []string `json:"ip,omitempty"`          // IP CIDR matching
	Port        string   `json:"port,omitempty"`        // Port matching
	Network     string   `json:"network,omitempty"`     // "tcp", "udp"
	Protocol    []string `json:"protocol,omitempty"`    // Protocol matching
	OutboundTag string   `json:"outboundTag"`           // Target outbound tag
}

// KoriaInboundSettings настройки Koria inbound
type KoriaInboundSettings struct {
	Clients []ClientConfig `json:"clients"`
}

// KoriaOutboundSettings настройки Koria outbound
type KoriaOutboundSettings struct {
	Address string       `json:"address"`
	Port    int          `json:"port"`
	UserID  string       `json:"userId"`
}

// ClientConfig конфигурация клиента для inbound
type ClientConfig struct {
	ID    string `json:"id"`
	Email string `json:"email,omitempty"`
	Level int    `json:"level,omitempty"`
}

// LoadConfig загружает конфигурацию из файла
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

// SaveConfig сохраняет конфигурацию в файл
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
