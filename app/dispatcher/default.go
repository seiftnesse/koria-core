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
	ohm    *outbound.Manager
	router *Router
}

// NewDefaultDispatcher создает новый dispatcher
func NewDefaultDispatcher(ohm *outbound.Manager, router *Router) *DefaultDispatcher {
	return &DefaultDispatcher{
		ohm:    ohm,
		router: router,
	}
}

// Dispatch создает соединение через outbound
func (d *DefaultDispatcher) Dispatch(ctx context.Context, dest commnet.Destination) (net.Conn, error) {
	// Выбираем outbound через router
	var handler outbound.Handler

	if d.router != nil {
		tag := d.router.MatchOutbound(dest)
		if tag != "" {
			handler = d.ohm.Select(tag)
		}
	}

	// Если router не выбрал или не найден - используем default
	if handler == nil {
		handler = d.ohm.GetDefaultHandler()
		if handler == nil {
			return nil, fmt.Errorf("no default outbound handler")
		}
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
