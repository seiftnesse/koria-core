package socks

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	commio "koria-core/common/io"
	commnet "koria-core/common/net"
	"koria-core/app/dispatcher"
	"log"
	"net"
	"sync"
)

// SOCKS5 constants
const (
	socks5Version = 0x05
	noAuth        = 0x00
	connectCmd    = 0x01
	ipv4Address   = 0x01
	domainAddress = 0x03
	ipv6Address   = 0x04
)

// Server представляет SOCKS5 сервер
type Server struct {
	tag        string
	listen     string
	listener   net.Listener
	dispatcher dispatcher.Interface
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewServer создает новый SOCKS5 сервер
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

	log.Printf("[SOCKS5 Inbound:%s] Listening on %s", s.tag, s.listen)

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

// GetRandomInboundProxy возвращает адрес прокси (не используется для SOCKS5)
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
				log.Printf("[SOCKS5 Inbound:%s] Accept error: %v", s.tag, err)
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

// handleConnection обрабатывает одно SOCKS5 соединение
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Handshake
	if err := s.handshake(conn); err != nil {
		log.Printf("[SOCKS5 Inbound:%s] Handshake failed: %v", s.tag, err)
		return
	}

	// Read request
	dest, err := s.readRequest(conn)
	if err != nil {
		log.Printf("[SOCKS5 Inbound:%s] Read request failed: %v", s.tag, err)
		s.sendReply(conn, 0x01) // General failure
		return
	}

	log.Printf("[SOCKS5 Inbound:%s] CONNECT %s", s.tag, dest.String())

	// Dispatch
	outConn, err := s.dispatcher.Dispatch(s.ctx, dest)
	if err != nil {
		log.Printf("[SOCKS5 Inbound:%s] Failed to dispatch: %v", s.tag, err)
		s.sendReply(conn, 0x04) // Host unreachable
		return
	}
	defer outConn.Close()

	// Send success reply
	s.sendReply(conn, 0x00) // Success

	log.Printf("[SOCKS5 Inbound:%s] Tunnel established to %s", s.tag, dest.String())

	// Tunnel с оптимизированным копированием
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		commio.Copy(outConn, conn)
		outConn.Close()
	}()

	go func() {
		defer wg.Done()
		commio.Copy(conn, outConn)
		conn.Close()
	}()

	wg.Wait()
	// Логируем только при debug
	// log.Printf("[SOCKS5 Inbound:%s] Tunnel closed for %s", s.tag, dest.String())
}

// handshake выполняет SOCKS5 handshake
func (s *Server) handshake(conn net.Conn) error {
	// Read version and methods
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	version := buf[0]
	nMethods := buf[1]

	if version != socks5Version {
		return fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	// Read methods
	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	// Send no auth method
	_, err := conn.Write([]byte{socks5Version, noAuth})
	return err
}

// readRequest читает SOCKS5 запрос
func (s *Server) readRequest(conn net.Conn) (commnet.Destination, error) {
	// Read header
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return commnet.Destination{}, err
	}

	version := buf[0]
	cmd := buf[1]
	// reserved := buf[2]
	addrType := buf[3]

	if version != socks5Version {
		return commnet.Destination{}, fmt.Errorf("unsupported version: %d", version)
	}

	if cmd != connectCmd {
		return commnet.Destination{}, fmt.Errorf("unsupported command: %d", cmd)
	}

	var host string

	switch addrType {
	case ipv4Address:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return commnet.Destination{}, err
		}
		host = net.IP(addr).String()

	case domainAddress:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return commnet.Destination{}, err
		}
		domainLen := lenBuf[0]
		domain := make([]byte, domainLen)
		if _, err := io.ReadFull(conn, domain); err != nil {
			return commnet.Destination{}, err
		}
		host = string(domain)

	case ipv6Address:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(conn, addr); err != nil {
			return commnet.Destination{}, err
		}
		host = net.IP(addr).String()

	default:
		return commnet.Destination{}, fmt.Errorf("unsupported address type: %d", addrType)
	}

	// Read port
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return commnet.Destination{}, err
	}
	port := binary.BigEndian.Uint16(portBuf)

	return commnet.TCPDestination(host, port), nil
}

// sendReply отправляет SOCKS5 ответ
func (s *Server) sendReply(conn net.Conn, rep byte) error {
	// Version, Reply, Reserved, Address Type, BND.ADDR, BND.PORT
	reply := []byte{
		socks5Version,
		rep,
		0x00,
		ipv4Address,
		0, 0, 0, 0, // 0.0.0.0
		0, 0, // Port 0
	}
	_, err := conn.Write(reply)
	return err
}
