package transport

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"koria-core/protocol/minecraft"
	c2s "koria-core/protocol/minecraft/packets/c2s"
	"koria-core/protocol/minecraft/packets/common"
	s2c "koria-core/protocol/minecraft/packets/s2c"
	"koria-core/protocol/multiplexer"
	"koria-core/stats"
	"log"
	"net"
	"time"
)

// Client представляет клиента протокола
type Client struct {
	config *ClientConfig
	mux    *multiplexer.Multiplexer
}

// ClientConfig конфигурация клиента
type ClientConfig struct {
	ServerAddr string    // Адрес сервера
	ServerPort int       // Порт сервера
	UserID     uuid.UUID // UUID пользователя для аутентификации
	Flow       string    // Flow type (опционально)
}

// Dial подключается к серверу и выполняет Minecraft handshake с UUID аутентификацией
func Dial(ctx context.Context, config *ClientConfig) (*Client, error) {
	// 1. Устанавливаем TCP соединение
	addr := fmt.Sprintf("%s:%d", config.ServerAddr, config.ServerPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		stats.Global().IncrementConnectionErrors()
		return nil, fmt.Errorf("dial TCP: %w", err)
	}

	// Включаем TCP keep-alive для предотвращения обрыва соединения
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	// 2. Выполняем Minecraft handshake
	if err := performHandshake(conn, config); err != nil {
		conn.Close()
		stats.Global().IncrementConnectionErrors()
		return nil, fmt.Errorf("handshake: %w", err)
	}

	// 3. Выполняем login с UUID аутентификацией
	if err := performLogin(conn, config.UserID); err != nil {
		conn.Close()
		stats.Global().IncrementFailedConnections()
		stats.Global().IncrementConnectionErrors()
		return nil, fmt.Errorf("login: %w", err)
	}

	// 4. Создаем мультиплексор для управления виртуальными потоками
	mux := multiplexer.NewMultiplexer(conn)
	stats.Global().IncrementConnections()

	client := &Client{
		config: config,
		mux:    mux,
	}

	return client, nil
}

// DialStream открывает новый виртуальный поток через существующее соединение
// Возвращает net.Conn совместимый объект
func (c *Client) DialStream(ctx context.Context) (net.Conn, error) {
	return c.mux.OpenStream(ctx)
}

// Close закрывает клиента и все виртуальные потоки
func (c *Client) Close() error {
	stats.Global().DecrementConnections()
	return c.mux.Close()
}

// StreamCount возвращает количество активных виртуальных потоков
func (c *Client) StreamCount() int {
	return c.mux.StreamCount()
}

// performHandshake выполняет Minecraft handshake фазу
func performHandshake(conn net.Conn, config *ClientConfig) error {
	handshake := &common.HandshakePacket{
		ProtocolVersion: 765, // Minecraft 1.20.4
		ServerAddress:   config.ServerAddr,
		ServerPort:      uint16(config.ServerPort),
		NextState:       2, // 2 = LOGIN state
	}

	if err := minecraft.WritePacket(conn, handshake); err != nil {
		return fmt.Errorf("write handshake packet: %w", err)
	}

	return nil
}

// performLogin выполняет login фазу с UUID аутентификацией
func performLogin(conn net.Conn, userID uuid.UUID) error {
	// Username используем короткий (max 16 символов)
	// UUID для аутентификации передается в отдельном поле
	username := "koria"

	loginStart := &c2s.LoginStartPacket{
		Username: username,
		UUID:     userID,
	}

	log.Printf("[DEBUG CLIENT] Sending LoginStart with UUID %s", userID)
	if err := minecraft.WritePacket(conn, loginStart); err != nil {
		return fmt.Errorf("write login start packet: %w", err)
	}

	// Ждем ответ от сервера (LoginSuccess или LoginDisconnect)
	log.Printf("[DEBUG CLIENT] Waiting for login response...")
	packetID, data, err := minecraft.ReadPacketRaw(conn)
	if err != nil {
		log.Printf("[DEBUG CLIENT] ReadPacketRaw error: %v", err)
		return fmt.Errorf("read login response: %w", err)
	}
	log.Printf("[DEBUG CLIENT] Received packet 0x%02X", packetID)

	switch packetID {
	case minecraft.PacketTypeLoginSuccess:
		// Успешная аутентификация
		return nil

	case 0x00: // LOGIN_DISCONNECT
		var disconnect s2c.LoginDisconnectPacket
		if err := minecraft.DecodePacket(&disconnect, data); err != nil {
			return fmt.Errorf("decode disconnect packet: %w", err)
		}
		return fmt.Errorf("login rejected: %s", disconnect.Reason)

	default:
		return fmt.Errorf("unexpected packet type: 0x%02X", packetID)
	}
}
