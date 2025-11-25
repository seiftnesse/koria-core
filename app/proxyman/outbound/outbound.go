package outbound

import (
	"context"
	commnet "koria-core/common/net"
	"net"
)

// Handler интерфейс для обработки исходящих соединений
type Handler interface {
	// Tag возвращает уникальный тег обработчика
	Tag() string

	// Dial создает исходящее соединение
	Dial(ctx context.Context, dest commnet.Destination) (net.Conn, error)
}

// Manager управляет исходящими обработчиками
type Manager struct {
	defaultHandler Handler
	taggedHandlers map[string]Handler
}

// NewManager создает новый менеджер
func NewManager() *Manager {
	return &Manager{
		taggedHandlers: make(map[string]Handler),
	}
}

// AddHandler добавляет обработчик
func (m *Manager) AddHandler(ctx context.Context, handler Handler) error {
	tag := handler.Tag()
	m.taggedHandlers[tag] = handler
	return nil
}

// RemoveHandler удаляет обработчик
func (m *Manager) RemoveHandler(ctx context.Context, tag string) error {
	delete(m.taggedHandlers, tag)
	return nil
}

// GetHandler возвращает обработчик по тегу
func (m *Manager) GetHandler(tag string) Handler {
	return m.taggedHandlers[tag]
}

// GetDefaultHandler возвращает обработчик по умолчанию
func (m *Manager) GetDefaultHandler() Handler {
	return m.defaultHandler
}

// SetDefaultHandler устанавливает обработчик по умолчанию
func (m *Manager) SetDefaultHandler(handler Handler) {
	m.defaultHandler = handler
}

// Select выбирает обработчик по тегу или возвращает дефолтный
func (m *Manager) Select(tag string) Handler {
	if tag == "" {
		return m.defaultHandler
	}
	if handler := m.taggedHandlers[tag]; handler != nil {
		return handler
	}
	return m.defaultHandler
}
