package transport

import (
	"fmt"
	"koria-core/config"
	"koria-core/protocol/minecraft"
	c2s "koria-core/protocol/minecraft/packets/c2s"
	"koria-core/protocol/minecraft/packets/common"
	s2c "koria-core/protocol/minecraft/packets/s2c"
	"koria-core/protocol/multiplexer"
	"koria-core/stats"
	"net"
	"sync"
	"time"
)

// Server представляет сервер протокола
type Server struct {
	listener  net.Listener
	validator *config.UserValidator

	// Активные мультиплексоры (одно TCP соединение = один мультиплексор)
	muxes   map[string]*multiplexer.Multiplexer
	muxesMu sync.RWMutex

	closeCh chan struct{}
}

// ServerConfig конфигурация сервера
type ServerConfig struct {
	ListenAddr string        // Адрес для прослушивания (например, "0.0.0.0:25565")
	Users      []config.User // Список пользователей
}

// Listen создает и запускает сервер
func Listen(cfg *ServerConfig) (*Server, error) {
	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("listen TCP: %w", err)
	}

	server := &Server{
		listener:  listener,
		validator: config.NewUserValidator(cfg.Users),
		muxes:     make(map[string]*multiplexer.Multiplexer),
		closeCh:   make(chan struct{}),
	}

	return server, nil
}

// Serve начинает принимать соединения
func (s *Server) Serve() error {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.closeCh:
				return nil
			default:
				return fmt.Errorf("accept connection: %w", err)
			}
		}

		// Обрабатываем соединение в отдельной горутине
		go s.handleConnection(conn)
	}
}

// handleConnection обрабатывает входящее TCP соединение
func (s *Server) handleConnection(conn net.Conn) {
	// Оптимизируем TCP параметры для высокой производительности
	// Это критично для снижения CPU при высоких нагрузках
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetNoDelay(true)                        // Отключаем Nagle
		tcpConn.SetKeepAlive(true)                      // Keep-alive
		tcpConn.SetKeepAlivePeriod(30 * time.Second)    // Период
		tcpConn.SetReadBuffer(512 * 1024)               // 512KB read buffer
		tcpConn.SetWriteBuffer(512 * 1024)              // 512KB write buffer
	}

	stats.Global().IncrementConnections()
	defer func() {
		stats.Global().DecrementConnections()
		conn.Close()
	}()

	// 1. Читаем и проверяем Handshake
	handshake, err := s.readHandshake(conn)
	if err != nil {
		stats.Global().IncrementConnectionErrors()
		// log error
		return
	}

	// Проверяем, что клиент хочет войти (NextState = 2)
	if handshake.NextState != 2 {
		// Это status запрос, не login - игнорируем
		return
	}

	// 2. Читаем LoginStart и валидируем UUID
	user, err := s.readAndValidateLogin(conn)
	if err != nil {
		// Отправляем disconnect
		disconnect := &s2c.LoginDisconnectPacket{
			Reason: fmt.Sprintf(`{"text":"Authentication failed: %s"}`, err.Error()),
		}
		minecraft.WritePacket(conn, disconnect)
		stats.Global().IncrementFailedConnections()
		stats.Global().IncrementConnectionErrors()
		return
	}

	// 3. Отправляем LoginSuccess
	// Minecraft protocol ограничивает имя пользователя 16 символами
	username := user.Email
	if len(username) > 16 {
		username = username[:16]
	}
	success := &s2c.LoginSuccessPacket{
		UUID:       user.ID,
		Username:   username,
		Properties: nil,
	}

	if err := minecraft.WritePacket(conn, success); err != nil {
		return
	}

	// 4. Создаем мультиплексор для этого соединения
	mux := multiplexer.NewMultiplexer(conn)

	// DEBUG

	// Регистрируем мультиплексор
	connKey := conn.RemoteAddr().String()
	s.muxesMu.Lock()
	s.muxes[connKey] = mux
	s.muxesMu.Unlock()


	// Очистка при закрытии
	defer func() {
		s.muxesMu.Lock()
		delete(s.muxes, connKey)
		s.muxesMu.Unlock()
		mux.Close()
	}()

	// 5. Принимаем виртуальные потоки и обрабатываем их
	// Это зависит от вашей логики проксирования
	// Например, каждый виртуальный поток можно проксировать к целевому серверу

	// Ждем пока соединение не закроется
	// Либо клиент отключится (мультиплексор закроется)
	// Либо сервер остановится
	select {
	case <-mux.CloseCh():
	case <-s.closeCh:
	}

}

// readHandshake читает и парсит handshake пакет
func (s *Server) readHandshake(conn net.Conn) (*common.HandshakePacket, error) {
	var handshake common.HandshakePacket
	if err := minecraft.ReadPacket(conn, &handshake); err != nil {
		return nil, fmt.Errorf("read handshake: %w", err)
	}

	return &handshake, nil
}

// readAndValidateLogin читает LoginStart и валидирует UUID пользователя
func (s *Server) readAndValidateLogin(conn net.Conn) (*config.User, error) {
	var loginStart c2s.LoginStartPacket
	if err := minecraft.ReadPacket(conn, &loginStart); err != nil {
		return nil, fmt.Errorf("read login start: %w", err)
	}

	// Валидируем UUID
	user, valid := s.validator.Validate(loginStart.UUID)
	if !valid {
		return nil, fmt.Errorf("invalid user UUID: %s", loginStart.UUID)
	}

	return user, nil
}

// AcceptStream ждет новый виртуальный поток от любого подключенного клиента
// В реальной реализации это нужно доработать для управления потоками от разных клиентов
func (s *Server) AcceptStream() (net.Conn, error) {
	// Простая реализация: берем первый доступный мультиплексор
	s.muxesMu.RLock()
	var mux *multiplexer.Multiplexer
	for _, m := range s.muxes {
		mux = m
		break
	}
	s.muxesMu.RUnlock()

	if mux == nil {
		return nil, fmt.Errorf("no active connections")
	}

	return mux.AcceptStream()
}

// Close закрывает сервер
func (s *Server) Close() error {
	close(s.closeCh)

	// Закрываем все мультиплексоры
	s.muxesMu.Lock()
	for _, mux := range s.muxes {
		mux.Close()
	}
	s.muxesMu.Unlock()

	return s.listener.Close()
}

// ConnectionCount возвращает количество активных TCP соединений
func (s *Server) ConnectionCount() int {
	s.muxesMu.RLock()
	defer s.muxesMu.RUnlock()
	return len(s.muxes)
}

// Addr возвращает адрес на котором слушает сервер
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}
