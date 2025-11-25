package multiplexer

import (
	"context"
	"fmt"
	"io"
	"koria-core/protocol/minecraft"
	c2s "koria-core/protocol/minecraft/packets/c2s"
	"koria-core/protocol/steganography"
	"koria-core/stats"
	"log"
	"net"
	"sync"
	"time"
)

// Multiplexer управляет множественными виртуальными потоками через одно TCP соединение
// Это ключевой компонент для решения проблемы блокировки ТСПУ
type Multiplexer struct {
	conn net.Conn // Базовое TCP соединение

	// Управление потоками
	streams   map[uint16]*Stream
	streamsMu sync.RWMutex

	// Следующий доступный ID потока
	nextStreamID uint16
	nextIDMu     sync.Mutex

	// Каналы для новых входящих потоков и закрытия
	acceptCh chan *Stream
	closeCh  chan struct{}

	// Компоненты стеганографии
	encoder  *steganography.Encoder
	decoder  *steganography.Decoder
	selector *steganography.PacketSelector

	// Мьютекс для защиты записи в TCP соединение
	// КРИТИЧНО: без этого пакеты от разных горутин перемешиваются!
	writeMu sync.Mutex

	// Состояние
	closed   bool
	closedMu sync.RWMutex
}

// NewMultiplexer создает новый мультиплексор
func NewMultiplexer(conn net.Conn) *Multiplexer {
	mux := &Multiplexer{
		conn:     conn,
		streams:  make(map[uint16]*Stream),
		acceptCh: make(chan *Stream, 256),
		closeCh:  make(chan struct{}),
		encoder:  steganography.NewEncoder(),
		decoder:  steganography.NewDecoder(),
		selector: steganography.NewPacketSelector(),
	}

	// Запускаем горутину для чтения пакетов
	go mux.readLoop()

	return mux
}

// OpenStream открывает новый виртуальный поток (используется клиентом)
func (m *Multiplexer) OpenStream(ctx context.Context) (*Stream, error) {
	m.closedMu.RLock()
	if m.closed {
		m.closedMu.RUnlock()
		return nil, io.ErrClosedPipe
	}
	m.closedMu.RUnlock()

	// Получаем следующий доступный ID
	m.nextIDMu.Lock()
	streamID := m.nextStreamID
	m.nextStreamID++
	if m.nextStreamID == 0 {
		m.nextStreamID = 1 // 0 зарезервирован для control frames
	}
	m.nextIDMu.Unlock()

	// Создаем поток
	stream := newStream(streamID, m)
	stream.state = StreamStateSYN

	// Регистрируем в карте
	m.streamsMu.Lock()
	m.streams[streamID] = stream
	m.streamsMu.Unlock()

	// Отправляем SYN фрейм
	synFrame := &steganography.Frame{
		StreamID: streamID,
		Sequence: 0,
		Flags:    steganography.FlagSYN,
		Length:   0,
		Data:     nil,
	}

	if err := m.sendFrame(synFrame); err != nil {
		m.closeStream(streamID)
		return nil, fmt.Errorf("send SYN: %w", err)
	}

	// Ждем SYN-ACK с таймаутом
	select {
	case <-stream.synAckCh:
		stats.Global().IncrementStreams()
		return stream, nil
	case <-ctx.Done():
		m.closeStream(streamID)
		stats.Global().IncrementStreamErrors()
		return nil, ctx.Err()
	case <-time.After(10 * time.Second):
		m.closeStream(streamID)
		stats.Global().IncrementStreamErrors()
		return nil, fmt.Errorf("timeout waiting for SYN-ACK")
	}
}

// AcceptStream ждет входящий виртуальный поток (используется сервером)
func (m *Multiplexer) AcceptStream() (*Stream, error) {
	select {
	case stream := <-m.acceptCh:
		return stream, nil
	case <-m.closeCh:
		return nil, io.ErrClosedPipe
	}
}

// readLoop читает пакеты из TCP соединения и демультиплексирует их
func (m *Multiplexer) readLoop() {
	defer func() {
		log.Printf("[Multiplexer] readLoop exiting, closing multiplexer")
		m.Close()
	}()

	for {
		select {
		case <-m.closeCh:
			log.Printf("[Multiplexer] Close channel signaled, exiting readLoop")
			return
		default:
		}

		// Читаем Minecraft пакет
		packetID, data, err := minecraft.ReadPacketRaw(m.conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("[Multiplexer] Error reading packet: %v", err)
			} else {
				log.Printf("[Multiplexer] Connection closed (EOF)")
			}
			return
		}

		// Декодируем фрейм из пакета в зависимости от типа
		var frame *steganography.Frame

		switch packetID {
		case minecraft.PacketTypePlayerMove:
			var pkt c2s.PlayerMovePacket
			if err := minecraft.DecodePacket(&pkt, data); err != nil {
				log.Printf("[Multiplexer] Error decoding PlayerMove packet: %v", err)
				continue
			}
			frame, err = m.decoder.DecodeFrame(&pkt)
		case minecraft.PacketTypeCustomPayload:
			var pkt c2s.CustomPayloadPacket
			if err := minecraft.DecodePacket(&pkt, data); err != nil {
				log.Printf("[Multiplexer] Error decoding CustomPayload packet: %v", err)
				continue
			}
			frame, err = m.decoder.DecodeFrameFromCustomPayload(&pkt)
		default:
			// Неизвестный тип пакета, пропускаем
			log.Printf("[Multiplexer] Unknown packet type: 0x%02X, skipping", packetID)
			continue
		}

		if err != nil {
			log.Printf("[Multiplexer] Error decoding frame: %v", err)
			continue
		}

		// Обрабатываем фрейм
		m.handleFrame(frame)
	}
}

// handleFrame обрабатывает входящий фрейм
func (m *Multiplexer) handleFrame(frame *steganography.Frame) {
	m.streamsMu.RLock()
	stream, exists := m.streams[frame.StreamID]
	m.streamsMu.RUnlock()

	if !exists {
		// Новый входящий поток (SYN пакет)
		if frame.HasFlag(steganography.FlagSYN) {
			m.handleNewStream(frame)
		}
		// Игнорируем пакеты для несуществующих потоков
		return
	}

	// Передаем фрейм в поток
	stream.handleFrame(frame)
}

// handleNewStream обрабатывает новый входящий поток
func (m *Multiplexer) handleNewStream(frame *steganography.Frame) {
	// Создаем новый поток
	stream := newStream(frame.StreamID, m)
	stream.state = StreamStateOpen

	// Регистрируем
	m.streamsMu.Lock()
	m.streams[frame.StreamID] = stream
	m.streamsMu.Unlock()

	// Отправляем SYN-ACK
	synAckFrame := &steganography.Frame{
		StreamID: frame.StreamID,
		Sequence: 0,
		Flags:    steganography.FlagSYN | steganography.FlagACK,
		Length:   0,
		Data:     nil,
	}

	if err := m.sendFrame(synAckFrame); err != nil {
		m.closeStream(frame.StreamID)
		return
	}

	// Уведомляем о новом потоке
	select {
	case m.acceptCh <- stream:
	case <-m.closeCh:
	case <-time.After(5 * time.Second):
		// Таймаут - закрываем поток
		stream.Close()
	}
}

// sendFrame отправляет фрейм через TCP соединение
func (m *Multiplexer) sendFrame(frame *steganography.Frame) error {
	m.closedMu.RLock()
	if m.closed {
		m.closedMu.RUnlock()
		log.Printf("[Multiplexer] Attempt to send frame on closed multiplexer (StreamID: %d)", frame.StreamID)
		return io.ErrClosedPipe
	}
	m.closedMu.RUnlock()

	// Выбираем тип пакета на основе размера данных
	packetType := m.selector.SelectPacketType(len(frame.Data))

	var packet minecraft.Packet
	var err error

	// Кодируем фрейм в выбранный тип пакета
	switch packetType {
	case minecraft.PacketTypePlayerMove:
		packet, err = m.encoder.EncodeFrame(frame)
	case minecraft.PacketTypeCustomPayload:
		packet, err = m.encoder.EncodeFrameInCustomPayload(frame)
	default:
		packet, err = m.encoder.EncodeFrame(frame)
	}

	if err != nil {
		log.Printf("[Multiplexer] Error encoding frame (StreamID: %d): %v", frame.StreamID, err)
		return fmt.Errorf("encode frame: %w", err)
	}

	// КРИТИЧНО: Блокируем запись чтобы пакеты не перемешивались!
	// Без этого при параллельной отправке из разных горутин
	// пакеты могут перемешаться в TCP stream
	m.writeMu.Lock()
	defer m.writeMu.Unlock()

	// Отправляем пакет
	if err := minecraft.WritePacket(m.conn, packet); err != nil {
		log.Printf("[Multiplexer] Error writing packet (StreamID: %d): %v", frame.StreamID, err)
		return fmt.Errorf("write packet: %w", err)
	}

	return nil
}

// closeStream удаляет поток из карты
func (m *Multiplexer) closeStream(streamID uint16) {
	m.streamsMu.Lock()
	stream, exists := m.streams[streamID]
	if exists {
		delete(m.streams, streamID)
		// Декрементируем только если поток был в активном состоянии
		if stream.state == StreamStateOpen {
			stats.Global().DecrementStreams()
		}
	}
	m.streamsMu.Unlock()
}

// Close закрывает мультиплексор и все потоки
func (m *Multiplexer) Close() error {
	m.closedMu.Lock()
	if m.closed {
		m.closedMu.Unlock()
		return nil
	}
	m.closed = true
	m.closedMu.Unlock()

	close(m.closeCh)

	// Закрываем все потоки
	m.streamsMu.Lock()
	for _, stream := range m.streams {
		stream.Close()
	}
	m.streams = make(map[uint16]*Stream)
	m.streamsMu.Unlock()

	// Закрываем TCP соединение
	return m.conn.Close()
}

// StreamCount возвращает количество активных потоков
func (m *Multiplexer) StreamCount() int {
	m.streamsMu.RLock()
	defer m.streamsMu.RUnlock()
	return len(m.streams)
}

// IsClosed проверяет, закрыт ли мультиплексор
func (m *Multiplexer) IsClosed() bool {
	m.closedMu.RLock()
	defer m.closedMu.RUnlock()
	return m.closed
}

// CloseCh возвращает канал который закрывается когда мультиплексор закрывается
func (m *Multiplexer) CloseCh() <-chan struct{} {
	return m.closeCh
}
