package stats

import (
	"sync"
	"sync/atomic"
	"time"
)

// Stats собирает статистику работы протокола
type Stats struct {
	// Соединения
	TotalConnections    atomic.Uint64
	ActiveConnections   atomic.Uint64
	FailedConnections   atomic.Uint64

	// Потоки
	TotalStreams        atomic.Uint64
	ActiveStreams       atomic.Uint64
	ClosedStreams       atomic.Uint64

	// Трафик
	BytesSent           atomic.Uint64
	BytesReceived       atomic.Uint64
	PacketsSent         atomic.Uint64
	PacketsReceived     atomic.Uint64

	// Ошибки
	TotalErrors         atomic.Uint64
	ConnectionErrors    atomic.Uint64
	StreamErrors        atomic.Uint64
	PacketErrors        atomic.Uint64

	// Время
	StartTime           time.Time
	LastActivity        atomic.Value // time.Time

	// Детальная статистика по типам пакетов
	packetTypesMu       sync.RWMutex
	packetTypes         map[string]uint64
}

// NewStats создает новый экземпляр статистики
func NewStats() *Stats {
	s := &Stats{
		StartTime:   time.Now(),
		packetTypes: make(map[string]uint64),
	}
	s.LastActivity.Store(time.Now())
	return s
}

// Connection tracking
func (s *Stats) IncrementConnections() {
	s.TotalConnections.Add(1)
	s.ActiveConnections.Add(1)
	s.updateActivity()
}

func (s *Stats) DecrementConnections() {
	s.ActiveConnections.Add(^uint64(0)) // Subtract 1
}

func (s *Stats) IncrementFailedConnections() {
	s.FailedConnections.Add(1)
}

// Stream tracking
func (s *Stats) IncrementStreams() {
	s.TotalStreams.Add(1)
	s.ActiveStreams.Add(1)
	s.updateActivity()
}

func (s *Stats) DecrementStreams() {
	s.ActiveStreams.Add(^uint64(0)) // Subtract 1
	s.ClosedStreams.Add(1)
}

// Traffic tracking
func (s *Stats) AddBytesSent(n uint64) {
	s.BytesSent.Add(n)
	s.updateActivity()
}

func (s *Stats) AddBytesReceived(n uint64) {
	s.BytesReceived.Add(n)
	s.updateActivity()
}

func (s *Stats) IncrementPacketsSent(packetType string) {
	s.PacketsSent.Add(1)
	s.updateActivity()

	s.packetTypesMu.Lock()
	s.packetTypes[packetType]++
	s.packetTypesMu.Unlock()
}

func (s *Stats) IncrementPacketsReceived() {
	s.PacketsReceived.Add(1)
	s.updateActivity()
}

// Error tracking
func (s *Stats) IncrementErrors() {
	s.TotalErrors.Add(1)
}

func (s *Stats) IncrementConnectionErrors() {
	s.ConnectionErrors.Add(1)
	s.TotalErrors.Add(1)
}

func (s *Stats) IncrementStreamErrors() {
	s.StreamErrors.Add(1)
	s.TotalErrors.Add(1)
}

func (s *Stats) IncrementPacketErrors() {
	s.PacketErrors.Add(1)
	s.TotalErrors.Add(1)
}

// updateActivity обновляет время последней активности
func (s *Stats) updateActivity() {
	s.LastActivity.Store(time.Now())
}

// Snapshot возвращает снимок текущей статистики
type Snapshot struct {
	// Connections
	TotalConnections  uint64
	ActiveConnections uint64
	FailedConnections uint64

	// Streams
	TotalStreams   uint64
	ActiveStreams  uint64
	ClosedStreams  uint64

	// Traffic
	BytesSent       uint64
	BytesReceived   uint64
	PacketsSent     uint64
	PacketsReceived uint64

	// Errors
	TotalErrors      uint64
	ConnectionErrors uint64
	StreamErrors     uint64
	PacketErrors     uint64

	// Time
	Uptime       time.Duration
	LastActivity time.Time

	// Packet types
	PacketTypes map[string]uint64
}

// GetSnapshot возвращает снимок текущей статистики
func (s *Stats) GetSnapshot() Snapshot {
	s.packetTypesMu.RLock()
	packetTypesCopy := make(map[string]uint64, len(s.packetTypes))
	for k, v := range s.packetTypes {
		packetTypesCopy[k] = v
	}
	s.packetTypesMu.RUnlock()

	lastActivity := s.LastActivity.Load().(time.Time)

	return Snapshot{
		TotalConnections:  s.TotalConnections.Load(),
		ActiveConnections: s.ActiveConnections.Load(),
		FailedConnections: s.FailedConnections.Load(),

		TotalStreams:  s.TotalStreams.Load(),
		ActiveStreams: s.ActiveStreams.Load(),
		ClosedStreams: s.ClosedStreams.Load(),

		BytesSent:       s.BytesSent.Load(),
		BytesReceived:   s.BytesReceived.Load(),
		PacketsSent:     s.PacketsSent.Load(),
		PacketsReceived: s.PacketsReceived.Load(),

		TotalErrors:      s.TotalErrors.Load(),
		ConnectionErrors: s.ConnectionErrors.Load(),
		StreamErrors:     s.StreamErrors.Load(),
		PacketErrors:     s.PacketErrors.Load(),

		Uptime:       time.Since(s.StartTime),
		LastActivity: lastActivity,

		PacketTypes: packetTypesCopy,
	}
}

// Reset сбрасывает статистику
func (s *Stats) Reset() {
	s.TotalConnections.Store(0)
	s.ActiveConnections.Store(0)
	s.FailedConnections.Store(0)

	s.TotalStreams.Store(0)
	s.ActiveStreams.Store(0)
	s.ClosedStreams.Store(0)

	s.BytesSent.Store(0)
	s.BytesReceived.Store(0)
	s.PacketsSent.Store(0)
	s.PacketsReceived.Store(0)

	s.TotalErrors.Store(0)
	s.ConnectionErrors.Store(0)
	s.StreamErrors.Store(0)
	s.PacketErrors.Store(0)

	s.StartTime = time.Now()
	s.LastActivity.Store(time.Now())

	s.packetTypesMu.Lock()
	s.packetTypes = make(map[string]uint64)
	s.packetTypesMu.Unlock()
}

// Global instance
var globalStats = NewStats()

// Global returns the global stats instance
func Global() *Stats {
	return globalStats
}
