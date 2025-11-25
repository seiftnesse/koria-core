package net

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Network тип сети
type Network string

const (
	// TCP сеть
	TCP Network = "tcp"
	// UDP сеть
	UDP Network = "udp"
)

// Destination представляет пункт назначения
type Destination struct {
	Network Network
	Address string
	Port    uint16
}

// String возвращает строковое представление
func (d Destination) String() string {
	return fmt.Sprintf("%s:%s:%d", d.Network, d.Address, d.Port)
}

// NetAddr возвращает net.Addr
func (d Destination) NetAddr() string {
	return net.JoinHostPort(d.Address, strconv.Itoa(int(d.Port)))
}

// TCPDestination создает TCP назначение
func TCPDestination(host string, port uint16) Destination {
	return Destination{
		Network: TCP,
		Address: host,
		Port:    port,
	}
}

// UDPDestination создает UDP назначение
func UDPDestination(host string, port uint16) Destination {
	return Destination{
		Network: UDP,
		Address: host,
		Port:    port,
	}
}

// ParseDestination парсит строку в Destination
func ParseDestination(dest string) (Destination, error) {
	parts := strings.Split(dest, ":")
	if len(parts) < 2 {
		return Destination{}, fmt.Errorf("invalid destination format: %s", dest)
	}

	network := TCP
	host := parts[0]
	portStr := parts[1]

	if len(parts) == 3 {
		network = Network(parts[0])
		host = parts[1]
		portStr = parts[2]
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return Destination{}, fmt.Errorf("invalid port: %w", err)
	}

	return Destination{
		Network: network,
		Address: host,
		Port:    uint16(port),
	}, nil
}
