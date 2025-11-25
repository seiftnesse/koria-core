package http

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"koria-core/app/dispatcher"
	commnet "koria-core/common/net"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// Server представляет HTTP proxy сервер
type Server struct {
	tag        string
	listen     string
	listener   net.Listener
	dispatcher dispatcher.Interface
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewServer создает новый HTTP proxy сервер
func NewServer(tag string, listen string, d dispatcher.Interface) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		tag:        tag,
		listen:     listen,
		dispatcher: d,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Tag возвращает тег сервера
func (s *Server) Tag() string {
	return s.tag
}

// Start запускает сервер
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.listen)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.listen, err)
	}
	s.listener = listener

	log.Printf("[HTTP Inbound:%s] Listening on %s", s.tag, s.listen)

	go s.acceptLoop()
	return nil
}

// Close закрывает сервер
func (s *Server) Close() error {
	s.cancel()
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// GetRandomInboundProxy возвращает адрес прокси (не используется для HTTP)
func (s *Server) GetRandomInboundProxy() (*net.TCPAddr, error) {
	return nil, fmt.Errorf("not implemented")
}

// acceptLoop принимает входящие соединения
func (s *Server) acceptLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				log.Printf("[HTTP Inbound:%s] Accept error: %v", s.tag, err)
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

// handleConnection обрабатывает одно HTTP соединение
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("[HTTP Inbound:%s] Failed to read request: %v", s.tag, err)
		return
	}

	log.Printf("[HTTP Inbound:%s] %s %s %s", s.tag, req.Method, req.Host, req.Proto)

	if req.Method == "CONNECT" {
		s.handleCONNECT(conn, req)
	} else {
		s.handleHTTP(conn, reader, req)
	}
}

// handleCONNECT обрабатывает HTTPS туннелинг
func (s *Server) handleCONNECT(conn net.Conn, req *http.Request) {
	// Парсим хост и порт
	host, portStr, err := net.SplitHostPort(req.Host)
	if err != nil {
		log.Printf("[HTTP Inbound:%s] Invalid host: %v", s.tag, err)
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		log.Printf("[HTTP Inbound:%s] Invalid port: %v", s.tag, err)
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	// Создаем destination
	dest := commnet.TCPDestination(host, uint16(port))

	// Диспатчим через outbound
	outConn, err := s.dispatcher.Dispatch(s.ctx, dest)
	if err != nil {
		log.Printf("[HTTP Inbound:%s] Failed to dispatch: %v", s.tag, err)
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer outConn.Close()

	// Отправляем успешный ответ
	conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	log.Printf("[HTTP Inbound:%s] HTTPS tunnel established to %s", s.tag, req.Host)

	// Туннелирование
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(outConn, conn)
		outConn.Close()
	}()

	go func() {
		defer wg.Done()
		io.Copy(conn, outConn)
		conn.Close()
	}()

	wg.Wait()
	log.Printf("[HTTP Inbound:%s] HTTPS tunnel closed for %s", s.tag, req.Host)
}

// handleHTTP обрабатывает обычный HTTP запрос
func (s *Server) handleHTTP(conn net.Conn, reader *bufio.Reader, req *http.Request) {
	// Определяем хост и порт
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}

	if !strings.Contains(host, ":") {
		host = host + ":80"
	}

	h, portStr, err := net.SplitHostPort(host)
	if err != nil {
		log.Printf("[HTTP Inbound:%s] Invalid host: %v", s.tag, err)
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		log.Printf("[HTTP Inbound:%s] Invalid port: %v", s.tag, err)
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
		return
	}

	// Создаем destination
	dest := commnet.TCPDestination(h, uint16(port))

	// Диспатчим через outbound
	outConn, err := s.dispatcher.Dispatch(s.ctx, dest)
	if err != nil {
		log.Printf("[HTTP Inbound:%s] Failed to dispatch: %v", s.tag, err)
		conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer outConn.Close()

	// Отправляем запрос
	if err := req.Write(outConn); err != nil {
		log.Printf("[HTTP Inbound:%s] Failed to write request: %v", s.tag, err)
		return
	}

	// Копируем ответ
	io.Copy(conn, outConn)
	log.Printf("[HTTP Inbound:%s] HTTP request completed for %s", s.tag, req.Host)
}
