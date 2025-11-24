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
	"log"
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
	log.Printf("[DEBUG SERVE] Serve() started, waiting for connections...")
	for {
		log.Printf("[DEBUG SERVE] Calling Accept()...")
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("[DEBUG SERVE] Accept() error: %v", err)
			select {
			case <-s.closeCh:
				return nil
			default:
				return fmt.Errorf("accept connection: %w", err)
			}
		}

		log.Printf("[DEBUG SERVE] Accepted connection from %s, starting handleConnection...", conn.RemoteAddr())
		// Обрабатываем соединение в отдельной горутине
		go s.handleConnection(conn)
	}
}

// handleConnection обрабатывает входящее TCP соединение
func (s *Server) handleConnection(conn net.Conn) {
	log.Printf("[DEBUG HANDLE] handleConnection started for %s", conn.RemoteAddr())

	// Включаем TCP keep-alive для предотвращения обрыва соединения
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	stats.Global().IncrementConnections()
	defer func() {
		log.Printf("[DEBUG HANDLE] handleConnection exiting for %s", conn.RemoteAddr())
		stats.Global().DecrementConnections()
		conn.Close()
	}()

	// 1. Читаем и проверяем Handshake
	log.Printf("[DEBUG HANDLE] Reading handshake from %s...", conn.RemoteAddr())
	handshake, err := s.readHandshake(conn)
	if err != nil {
		log.Printf("[DEBUG HANDLE] Handshake error from %s: %v", conn.RemoteAddr(), err)
		stats.Global().IncrementConnectionErrors()
		// log error
		return
	}
	log.Printf("[DEBUG HANDLE] Handshake OK from %s, NextState=%d", conn.RemoteAddr(), handshake.NextState)

	// Проверяем, что клиент хочет войти (NextState = 2)
	if handshake.NextState != 2 {
		log.Printf("[DEBUG HANDLE] NextState != 2, ignoring status request")
		// Это status запрос, не login - игнорируем
		return
	}

	// 2. Читаем LoginStart и валидируем UUID
	log.Printf("[DEBUG HANDLE] Reading and validating login from %s...", conn.RemoteAddr())
	user, err := s.readAndValidateLogin(conn)
	if err != nil {
		log.Printf("[DEBUG HANDLE] Login validation failed for %s: %v", conn.RemoteAddr(), err)
		// Отправляем disconnect
		disconnect := &s2c.LoginDisconnectPacket{
			Reason: fmt.Sprintf(`{"text":"Authentication failed: %s"}`, err.Error()),
		}
		minecraft.WritePacket(conn, disconnect)
		stats.Global().IncrementFailedConnections()
		stats.Global().IncrementConnectionErrors()
		return
	}
	log.Printf("[DEBUG HANDLE] Login OK for %s, user: %s", conn.RemoteAddr(), user.Email)

	// 3. Отправляем LoginSuccess
	log.Printf("[DEBUG HANDLE] Sending LoginSuccess to %s...", conn.RemoteAddr())
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
		log.Printf("[DEBUG HANDLE] Failed to write LoginSuccess to %s: %v", conn.RemoteAddr(), err)
		return
	}
	log.Printf("[DEBUG HANDLE] LoginSuccess sent to %s", conn.RemoteAddr())

	// 4. Создаем мультиплексор для этого соединения
	log.Printf("[DEBUG HANDLE] Creating multiplexer for %s...", conn.RemoteAddr())
	mux := multiplexer.NewMultiplexer(conn)

	// DEBUG
	log.Printf("[DEBUG] Created multiplexer for %s", conn.RemoteAddr())

	// Регистрируем мультиплексор
	connKey := conn.RemoteAddr().String()
	s.muxesMu.Lock()
	s.muxes[connKey] = mux
	s.muxesMu.Unlock()

	log.Printf("[DEBUG] Registered multiplexer, waiting for close...")

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
		log.Printf("[DEBUG] Multiplexer closed for %s", connKey)
	case <-s.closeCh:
		log.Printf("[DEBUG] Server shutting down")
	}

	log.Printf("[DEBUG] handleConnection exiting for %s", connKey)
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
