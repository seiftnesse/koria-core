package dispatcher

import (
	"context"
	"fmt"
	commnet "koria-core/common/net"
	"koria-core/app/proxyman/outbound"
	"net"
)

// DefaultDispatcher стандартный dispatcher
type DefaultDispatcher struct {
	ohm *outbound.Manager
}

// NewDefaultDispatcher создает новый dispatcher
func NewDefaultDispatcher(ohm *outbound.Manager) *DefaultDispatcher {
	return &DefaultDispatcher{
		ohm: ohm,
	}
}

// Dispatch создает соединение через outbound
func (d *DefaultDispatcher) Dispatch(ctx context.Context, dest commnet.Destination) (net.Conn, error) {
	// Используем дефолтный outbound
	handler := d.ohm.GetDefaultHandler()
	if handler == nil {
		return nil, fmt.Errorf("no default outbound handler")
	}

	return handler.Dial(ctx, dest)
}

// DispatchWithTag создает соединение через конкретный outbound по тегу
func (d *DefaultDispatcher) DispatchWithTag(ctx context.Context, dest commnet.Destination, tag string) (net.Conn, error) {
	handler := d.ohm.Select(tag)
	if handler == nil {
		return nil, fmt.Errorf("outbound handler not found: %s", tag)
	}

	return handler.Dial(ctx, dest)
}
