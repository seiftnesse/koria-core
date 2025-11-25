package inbound

import (
	"context"
	"net"
)

// Handler интерфейс для обработки входящих соединений
type Handler interface {
	// Tag возвращает уникальный тег обработчика
	Tag() string

	// Start запускает обработчик
	Start() error

	// Close закрывает обработчик
	Close() error

	// GetRandomInboundProxy возвращает прокси для исходящих соединений (для inbound)
	GetRandomInboundProxy() (*net.TCPAddr, error)
}

// Manager управляет входящими обработчиками
type Manager struct {
	handlers map[string]Handler
}

// NewManager создает новый менеджер
func NewManager() *Manager {
	return &Manager{
		handlers: make(map[string]Handler),
	}
}

// AddHandler добавляет обработчик
func (m *Manager) AddHandler(ctx context.Context, handler Handler) error {
	m.handlers[handler.Tag()] = handler
	return handler.Start()
}

// RemoveHandler удаляет обработчик
func (m *Manager) RemoveHandler(ctx context.Context, tag string) error {
	handler, ok := m.handlers[tag]
	if !ok {
		return nil
	}

	delete(m.handlers, tag)
	return handler.Close()
}

// GetHandler возвращает обработчик по тегу
func (m *Manager) GetHandler(tag string) Handler {
	return m.handlers[tag]
}

// Close закрывает все обработчики
func (m *Manager) Close() error {
	for _, handler := range m.handlers {
		handler.Close()
	}
	return nil
}
