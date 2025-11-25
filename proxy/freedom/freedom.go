package freedom

import (
	"context"
	"fmt"
	commnet "koria-core/common/net"
	"log"
	"net"
)

// Handler представляет Freedom outbound (прямое соединение)
type Handler struct {
	tag string
}

// NewHandler создает новый Freedom handler
func NewHandler(tag string) *Handler {
	return &Handler{
		tag: tag,
	}
}

// Tag возвращает тег обработчика
func (h *Handler) Tag() string {
	return h.tag
}

// Dial создает прямое соединение к назначению
func (h *Handler) Dial(ctx context.Context, dest commnet.Destination) (net.Conn, error) {
	log.Printf("[Freedom Outbound:%s] Dialing %s", h.tag, dest.String())

	var d net.Dialer
	conn, err := d.DialContext(ctx, string(dest.Network), dest.NetAddr())
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", dest.String(), err)
	}

	log.Printf("[Freedom Outbound:%s] Connected to %s", h.tag, dest.String())
	return conn, nil
}
