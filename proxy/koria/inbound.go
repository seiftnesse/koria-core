package koria

import (
	"context"
	"fmt"
	"io"
	commio "koria-core/common/io"
	commnet "koria-core/common/net"
	"koria-core/app/dispatcher"
	"koria-core/config"
	"koria-core/transport"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

// Server представляет Koria inbound (принимает соединения по Koria протоколу)
type Server struct {
	tag        string
	server     *transport.Server
	dispatcher dispatcher.Interface
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewServer создает новый Koria inbound сервер
func NewServer(tag string, listen string, users []config.User, d dispatcher.Interface) (*Server, error) {
	serverConfig := &transport.ServerConfig{
		ListenAddr: listen,
		Users:      users,
	}

	server, err := transport.Listen(serverConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport server: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		tag:        tag,
		server:     server,
		dispatcher: d,
		ctx:        ctx,
		cancel:     cancel,
	}

	return s, nil
}

// Tag возвращает тег сервера
func (s *Server) Tag() string {
	return s.tag
}

// Start запускает сервер
func (s *Server) Start() error {
	log.Printf("[Koria Inbound:%s] Listening on %s", s.tag, s.server.Addr())

	// Запускаем приём TCP соединений
	go func() {
		if err := s.server.Serve(); err != nil {
			log.Printf("[Koria Inbound:%s] Serve error: %v", s.tag, err)
		}
	}()

	// Запускаем обработку виртуальных потоков
	go s.acceptLoop()

	return nil
}

// Close закрывает сервер
func (s *Server) Close() error {
	s.cancel()
	return s.server.Close()
}

// GetRandomInboundProxy возвращает адрес прокси (не используется)
func (s *Server) GetRandomInboundProxy() (*net.TCPAddr, error) {
	return nil, fmt.Errorf("not implemented")
}

// acceptLoop принимает виртуальные потоки
func (s *Server) acceptLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		stream, err := s.server.AcceptStream()
		if err != nil {
			// Игнорируем "no active connections" - это нормально
			if err.Error() != "no active connections" {
				log.Printf("[Koria Inbound:%s] Accept stream error: %v", s.tag, err)
			}
			continue
		}

		log.Printf("[Koria Inbound:%s] Accepted virtual stream", s.tag)
		go s.handleStream(stream)
	}
}

// handleStream обрабатывает виртуальный поток
func (s *Server) handleStream(stream net.Conn) {
	defer stream.Close()

	// Читаем destination от клиента
	// Формат: "CONNECT host:port\n"
	buf := make([]byte, 1024)
	n, err := stream.Read(buf)
	if err != nil {
		log.Printf("[Koria Inbound:%s] Failed to read destination: %v", s.tag, err)
		return
	}

	line := string(buf[:n])

	// Парсим команду
	if !strings.HasPrefix(line, "CONNECT ") {
		log.Printf("[Koria Inbound:%s] Invalid command: %s", s.tag, line[:min(len(line), 50)])
		return
	}

	// Извлекаем host:port
	parts := strings.Fields(line)
	if len(parts) < 2 {
		log.Printf("[Koria Inbound:%s] Invalid CONNECT command", s.tag)
		return
	}

	targetAddr := parts[1]
	log.Printf("[Koria Inbound:%s] CONNECT request to %s", s.tag, targetAddr)

	// Парсим host и port
	host, portStr, err := net.SplitHostPort(targetAddr)
	if err != nil {
		log.Printf("[Koria Inbound:%s] Invalid target address: %v", s.tag, err)
		stream.Write([]byte("ERR\n"))
		return
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		log.Printf("[Koria Inbound:%s] Invalid port: %v", s.tag, err)
		stream.Write([]byte("ERR\n"))
		return
	}

	// Создаем destination
	dest := commnet.TCPDestination(host, uint16(port))

	// Dispatch через outbound
	outConn, err := s.dispatcher.Dispatch(s.ctx, dest)
	if err != nil {
		log.Printf("[Koria Inbound:%s] Failed to dispatch: %v", s.tag, err)
		stream.Write([]byte("ERR\n"))
		return
	}
	defer outConn.Close()

	// Отправляем OK клиенту
	if _, err := stream.Write([]byte("OK\n")); err != nil {
		log.Printf("[Koria Inbound:%s] Failed to send OK: %v", s.tag, err)
		return
	}

	log.Printf("[Koria Inbound:%s] Tunnel established to %s", s.tag, targetAddr)

	// Туннелирование данных с оптимизацией
	var wg sync.WaitGroup
	wg.Add(2)

	// Stream -> Target
	go func() {
		defer wg.Done()
		commio.Copy(outConn, stream)
		outConn.Close()
	}()

	// Target -> Stream
	go func() {
		defer wg.Done()
		commio.Copy(stream, outConn)
		stream.Close()
	}()

	wg.Wait()
	// Логируем только при debug
	// log.Printf("[Koria Inbound:%s] Tunnel closed for %s", s.tag, targetAddr)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleTransparent обрабатывает поток как transparent proxy
func (s *Server) handleTransparent(stream net.Conn, dest commnet.Destination) {
	defer stream.Close()

	// Dispatch через outbound
	outConn, err := s.dispatcher.Dispatch(s.ctx, dest)
	if err != nil {
		log.Printf("[Koria Inbound:%s] Failed to dispatch: %v", s.tag, err)
		return
	}
	defer outConn.Close()

	log.Printf("[Koria Inbound:%s] Tunnel established to %s", s.tag, dest.String())

	// Tunnel
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(outConn, stream)
		outConn.Close()
	}()

	go func() {
		defer wg.Done()
		io.Copy(stream, outConn)
		stream.Close()
	}()

	wg.Wait()
	log.Printf("[Koria Inbound:%s] Tunnel closed for %s", s.tag, dest.String())
}
