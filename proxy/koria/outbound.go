package koria

import (
	"context"
	"fmt"
	commnet "koria-core/common/net"
	"koria-core/transport"
	"log"
	"net"
)

// Handler представляет Koria outbound (через Koria протокол)
type Handler struct {
	tag    string
	client *transport.Client
}

// NewHandler создает новый Koria outbound handler
func NewHandler(tag string, client *transport.Client) *Handler {
	return &Handler{
		tag:    tag,
		client: client,
	}
}

// Tag возвращает тег обработчика
func (h *Handler) Tag() string {
	return h.tag
}

// Dial создает соединение через Koria
func (h *Handler) Dial(ctx context.Context, dest commnet.Destination) (net.Conn, error) {
	log.Printf("[Koria Outbound:%s] Opening stream for %s", h.tag, dest.String())

	// Открываем виртуальный поток
	stream, err := h.client.DialStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}

	// Отправляем информацию о destination серверу
	// Формат: "CONNECT host:port\n" (совместимо с http_proxy)
	destStr := fmt.Sprintf("CONNECT %s\n", dest.NetAddr())
	if _, err := stream.Write([]byte(destStr)); err != nil {
		stream.Close()
		return nil, fmt.Errorf("failed to send destination: %w", err)
	}

	// Ждем подтверждение от сервера
	buf := make([]byte, 3)
	if _, err := stream.Read(buf); err != nil {
		stream.Close()
		return nil, fmt.Errorf("failed to read server response: %w", err)
	}

	if string(buf) != "OK\n" {
		stream.Close()
		return nil, fmt.Errorf("server rejected connection")
	}

	log.Printf("[Koria Outbound:%s] Stream opened for %s", h.tag, dest.String())
	return stream, nil
}
