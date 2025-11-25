package dispatcher

import (
	"context"
	commnet "koria-core/common/net"
	"net"
)

// Interface интерфейс dispatcher
type Interface interface {
	// Dispatch создает соединение к destination через соответствующий outbound
	Dispatch(ctx context.Context, dest commnet.Destination) (net.Conn, error)
}
