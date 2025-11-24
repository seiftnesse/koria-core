package multiplexer

import (
	"io"
	"koria-core/protocol/steganography"
	"koria-core/stats"
	"net"
	"sync"
	"time"
)

// Stream представляет виртуальный поток внутри мультиплексированного соединения
// Реализует интерфейс net.Conn для совместимости со стандартными сетевыми операциями
type Stream struct {
	id       uint16
	mux      *Multiplexer
	sequence uint16

	// Буферы для чтения и записи
	readBuf  chan []byte
	writeCh  chan *steganography.Frame

	// Канал для ожидания SYN-ACK при открытии потока
	synAckCh chan struct{}

	// Канал закрытия
	closeCh chan struct{}

	// Дедлайны для операций чтения/записи
	readDeadline  time.Time
	writeDeadline time.Time

	// Состояние потока
	state      StreamState
	stateMu    sync.RWMutex

	mu        sync.Mutex
	closeOnce sync.Once
}

// StreamState представляет состояние потока
type StreamState int

const (
	StreamStateIdle StreamState = iota
	StreamStateSYN      // Открытие потока (SYN отправлен)
	StreamStateOpen     // Поток активен
	StreamStateClosing  // Закрывается (FIN отправлен)
	StreamStateClosed   // Закрыт
)

// newStream создает новый виртуальный поток
func newStream(id uint16, mux *Multiplexer) *Stream {
	return &Stream{
		id:       id,
		mux:      mux,
		readBuf:  make(chan []byte, 256),
		writeCh:  make(chan *steganography.Frame, 256),
		synAckCh: make(chan struct{}, 1),
		closeCh:  make(chan struct{}),
		state:    StreamStateIdle,
	}
}

// Read читает данные из потока (реализация io.Reader)
func (s *Stream) Read(p []byte) (int, error) {
	select {
	case data := <-s.readBuf:
		n := copy(p, data)
		// TODO: если data больше p, нужно буферизовать остаток
		if n < len(data) {
			// Возвращаем остаток обратно в буфер
			remainder := data[n:]
			select {
			case s.readBuf <- remainder:
			default:
				// Буфер полон, теряем данные (не должно происходить при правильном использовании)
			}
		}
		stats.Global().AddBytesReceived(uint64(n))
		return n, nil
	case <-s.closeCh:
		return 0, io.EOF
	case <-s.getReadDeadline():
		return 0, &timeoutError{}
	}
}

// Write записывает данные в поток (реализация io.Writer)
func (s *Stream) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем состояние
	s.stateMu.RLock()
	if s.state == StreamStateClosed || s.state == StreamStateClosing {
		s.stateMu.RUnlock()
		return 0, io.ErrClosedPipe
	}
	s.stateMu.RUnlock()

	written := 0

	// Разбиваем данные на chunks по размеру, который может вместить выбранный тип пакета
	for written < len(p) {
		// Определяем размер следующего chunk
		remaining := len(p) - written
		chunkSize := s.mux.selector.GetMaxPayload(s.mux.selector.SelectPacketType(remaining))

		if chunkSize > remaining {
			chunkSize = remaining
		}

		chunk := p[written : written+chunkSize]

		// Создаем фрейм
		frame := &steganography.Frame{
			StreamID: s.id,
			Sequence: s.sequence,
			Flags:    0, // Обычные данные
			Length:   uint16(len(chunk)),
			Data:     chunk,
		}
		s.sequence++

		// Отправляем фрейм через мультиплексор
		if err := s.mux.sendFrame(frame); err != nil {
			return written, err
		}

		written += chunkSize
	}

	stats.Global().AddBytesSent(uint64(written))
	return written, nil
}

// Close закрывает поток (реализация io.Closer)
func (s *Stream) Close() error {
	s.closeOnce.Do(func() {
		// Изменяем состояние
		s.stateMu.Lock()
		s.state = StreamStateClosing
		s.stateMu.Unlock()

		// Отправляем FIN фрейм
		finFrame := &steganography.Frame{
			StreamID: s.id,
			Sequence: s.sequence,
			Flags:    steganography.FlagFIN,
			Length:   0,
			Data:     nil,
		}
		s.mux.sendFrame(finFrame)

		// Закрываем канал
		close(s.closeCh)

		// Изменяем состояние
		s.stateMu.Lock()
		s.state = StreamStateClosed
		s.stateMu.Unlock()

		// Удаляем из мультиплексора
		s.mux.closeStream(s.id)
	})

	return nil
}

// handleFrame обрабатывает входящий фрейм
func (s *Stream) handleFrame(frame *steganography.Frame) {
	// SYN-ACK
	if frame.HasFlag(steganography.FlagACK) && frame.HasFlag(steganography.FlagSYN) {
		s.stateMu.Lock()
		s.state = StreamStateOpen
		s.stateMu.Unlock()

		select {
		case s.synAckCh <- struct{}{}:
		default:
		}
		return
	}

	// FIN - закрытие потока
	if frame.HasFlag(steganography.FlagFIN) {
		s.Close()
		return
	}

	// RST - сброс потока
	if frame.HasFlag(steganography.FlagRST) {
		s.Close()
		return
	}

	// DATA - обычные данные
	if frame.Length > 0 && len(frame.Data) > 0 {
		// Копируем данные (важно для избежания race conditions)
		data := make([]byte, len(frame.Data))
		copy(data, frame.Data)

		select {
		case s.readBuf <- data:
		case <-s.closeCh:
		case <-time.After(5 * time.Second):
			// Таймаут - сбрасываем данные
		}
	}
}

// Реализация net.Conn интерфейса

// LocalAddr возвращает локальный адрес
func (s *Stream) LocalAddr() net.Addr {
	return s.mux.conn.LocalAddr()
}

// RemoteAddr возвращает удаленный адрес
func (s *Stream) RemoteAddr() net.Addr {
	return s.mux.conn.RemoteAddr()
}

// SetDeadline устанавливает дедлайн для чтения и записи
func (s *Stream) SetDeadline(t time.Time) error {
	s.SetReadDeadline(t)
	s.SetWriteDeadline(t)
	return nil
}

// SetReadDeadline устанавливает дедлайн для чтения
func (s *Stream) SetReadDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.readDeadline = t
	return nil
}

// SetWriteDeadline устанавливает дедлайн для записи
func (s *Stream) SetWriteDeadline(t time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeDeadline = t
	return nil
}

// getReadDeadline возвращает канал, который закроется при истечении дедлайна
func (s *Stream) getReadDeadline() <-chan time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.readDeadline.IsZero() {
		return nil
	}

	return time.After(time.Until(s.readDeadline))
}

// ID возвращает идентификатор потока
func (s *Stream) ID() uint16 {
	return s.id
}

// State возвращает текущее состояние потока
func (s *Stream) State() StreamState {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	return s.state
}

// timeoutError представляет ошибку таймаута
type timeoutError struct{}

func (e *timeoutError) Error() string {
	return "i/o timeout"
}

func (e *timeoutError) Timeout() bool {
	return true
}

func (e *timeoutError) Temporary() bool {
	return true
}
