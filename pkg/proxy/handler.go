package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/seiftnesse/koria-core/pkg/minecraft"
)

// Handler manages proxy connections with Minecraft packet camouflage
type Handler struct {
	upstreamAddr string
	timeout      time.Duration
}

// NewHandler creates a new proxy handler
func NewHandler(upstreamAddr string, timeout time.Duration) *Handler {
	return &Handler{
		upstreamAddr: upstreamAddr,
		timeout:      timeout,
	}
}

// HandleConnection handles a client connection with Minecraft packet wrapping
func (h *Handler) HandleConnection(ctx context.Context, clientConn net.Conn) error {
	defer clientConn.Close()

	log.Printf("New connection from %s", clientConn.RemoteAddr())

	// Connect to upstream server
	upstreamConn, err := net.DialTimeout("tcp", h.upstreamAddr, h.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to upstream: %w", err)
	}
	defer upstreamConn.Close()

	log.Printf("Connected to upstream %s", h.upstreamAddr)

	// Note: In a real implementation, you might want to send a fake handshake
	// to establish Minecraft protocol appearance for deep packet inspection.
	// For now, we rely on the packet wrapping itself for camouflage.

	// Bidirectional copy with Minecraft packet wrapping
	var wg sync.WaitGroup
	wg.Add(2)

	errChan := make(chan error, 2)

	// Client to Upstream (wrap in Minecraft packets)
	go func() {
		defer wg.Done()
		err := h.copyWithMinecraftEncoding(upstreamConn, clientConn, "client->upstream")
		if err != nil && err != io.EOF {
			errChan <- fmt.Errorf("client->upstream: %w", err)
		}
	}()

	// Upstream to Client (unwrap from Minecraft packets)
	go func() {
		defer wg.Done()
		err := h.copyWithMinecraftDecoding(clientConn, upstreamConn, "upstream->client")
		if err != nil && err != io.EOF {
			errChan <- fmt.Errorf("upstream->client: %w", err)
		}
	}()

	// Wait for both goroutines or context cancellation
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	case <-done:
		return nil
	}
}

// copyWithMinecraftEncoding copies data from src to dst, wrapping it in Minecraft packets
func (h *Handler) copyWithMinecraftEncoding(dst io.Writer, src io.Reader, direction string) error {
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := src.Read(buf)
		if n > 0 {
			// Wrap data in a Minecraft custom payload packet
			encoded, encErr := minecraft.EncodePacket(minecraft.PacketCustomPayload, buf[:n])
			if encErr != nil {
				return fmt.Errorf("encoding error: %w", encErr)
			}

			if _, writeErr := dst.Write(encoded); writeErr != nil {
				return fmt.Errorf("write error: %w", writeErr)
			}

			log.Printf("%s: forwarded %d bytes (encoded to %d bytes)", direction, n, len(encoded))
		}

		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

// copyWithMinecraftDecoding copies data from src to dst, unwrapping it from Minecraft packets
func (h *Handler) copyWithMinecraftDecoding(dst io.Writer, src io.Reader, direction string) error {
	reader := io.Reader(src)

	for {
		// Decode Minecraft packet
		packet, err := minecraft.DecodePacket(reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("decoding error: %w", err)
		}

		// Write the unwrapped data
		if len(packet.Data) > 0 {
			n, writeErr := dst.Write(packet.Data)
			if writeErr != nil {
				return fmt.Errorf("write error: %w", writeErr)
			}

			log.Printf("%s: forwarded %d bytes (decoded from packet)", direction, n)
		}
	}
}

// Server represents the proxy server
type Server struct {
	listener net.Listener
	handler  *Handler
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewServer creates a new proxy server
func NewServer(listenAddr, upstreamAddr string, timeout time.Duration) (*Server, error) {
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", listenAddr, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		listener: listener,
		handler:  NewHandler(upstreamAddr, timeout),
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Start starts the proxy server
func (s *Server) Start() error {
	log.Printf("Koria-core proxy server listening on %s", s.listener.Addr())

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return nil
			default:
				log.Printf("Accept error: %v", err)
				continue
			}
		}

		go func(c net.Conn) {
			if err := s.handler.HandleConnection(s.ctx, c); err != nil {
				log.Printf("Connection error: %v", err)
			}
		}(conn)
	}
}

// Stop stops the proxy server
func (s *Server) Stop() error {
	s.cancel()
	return s.listener.Close()
}

// StreamWrapper wraps a connection to automatically encode/decode Minecraft packets
type StreamWrapper struct {
	conn     net.Conn
	readBuf  *bytes.Buffer
	writeBuf *bytes.Buffer
}

// NewStreamWrapper creates a new stream wrapper
func NewStreamWrapper(conn net.Conn) *StreamWrapper {
	return &StreamWrapper{
		conn:     conn,
		readBuf:  new(bytes.Buffer),
		writeBuf: new(bytes.Buffer),
	}
}

// Read reads and decodes data from the underlying connection
func (sw *StreamWrapper) Read(p []byte) (n int, err error) {
	if sw.readBuf.Len() > 0 {
		return sw.readBuf.Read(p)
	}

	packet, err := minecraft.DecodePacket(sw.conn)
	if err != nil {
		return 0, err
	}

	sw.readBuf.Write(packet.Data)
	return sw.readBuf.Read(p)
}

// Write encodes and writes data to the underlying connection
func (sw *StreamWrapper) Write(p []byte) (n int, err error) {
	encoded, err := minecraft.EncodePacket(minecraft.PacketCustomPayload, p)
	if err != nil {
		return 0, err
	}

	return sw.conn.Write(encoded)
}

// Close closes the underlying connection
func (sw *StreamWrapper) Close() error {
	return sw.conn.Close()
}
