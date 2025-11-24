package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

// TestProxyIntegration tests the basic proxy functionality
func TestProxyIntegration(t *testing.T) {
	// Start an echo server (upstream)
	echoListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start echo server: %v", err)
	}
	defer echoListener.Close()

	echoAddr := echoListener.Addr().String()

	// Echo server goroutine
	go func() {
		conn, err := echoListener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Echo back everything received
		io.Copy(conn, conn)
	}()

	// Start proxy server
	proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxyListener.Close()

	proxyAddr := proxyListener.Addr().String()
	handler := NewHandler(echoAddr, 5*time.Second)

	// Proxy server goroutine
	go func() {
		conn, err := proxyListener.Accept()
		if err != nil {
			return
		}
		ctx := context.Background()
		handler.HandleConnection(ctx, conn)
	}()

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	// Connect client to proxy
	clientConn, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("Failed to connect to proxy: %v", err)
	}
	defer clientConn.Close()

	// Test message
	testMsg := []byte("Hello, Minecraft!")

	// Write to proxy
	_, err = clientConn.Write(testMsg)
	if err != nil {
		t.Fatalf("Failed to write to proxy: %v", err)
	}

	// Read response
	response := make([]byte, len(testMsg))
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := io.ReadFull(clientConn, response)
	if err != nil {
		t.Fatalf("Failed to read from proxy: %v", err)
	}

	if n != len(testMsg) {
		t.Errorf("Expected to read %d bytes, got %d", len(testMsg), n)
	}

	if !bytes.Equal(response, testMsg) {
		t.Errorf("Response mismatch: got %q, want %q", response, testMsg)
	}
}

// TestStreamWrapper tests the stream wrapper functionality
func TestStreamWrapper(t *testing.T) {
	// Create a pipe for testing
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	wrapper := NewStreamWrapper(client)

	// Test data
	testData := []byte("Test data for stream wrapper")

	// Write in goroutine
	errChan := make(chan error, 1)
	go func() {
		n, err := wrapper.Write(testData)
		if err != nil {
			errChan <- err
			return
		}
		// Note: n is the encoded packet size, not the raw data size
		if n <= len(testData) {
			errChan <- fmt.Errorf("Write encoded size should be larger: got %d, raw data %d", n, len(testData))
			return
		}
		errChan <- nil
	}()

	// Read on server side
	readBuf := make([]byte, 1024)
	n, err := server.Read(readBuf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	// Wait for write to complete
	if err := <-errChan; err != nil {
		t.Fatal(err)
	}

	// The read data should be encoded as Minecraft packet
	if n <= len(testData) {
		t.Errorf("Expected encoded data to be larger than raw data")
	}

	// Verify it starts with a VarInt length prefix
	if readBuf[0] == 0 {
		t.Error("Expected non-zero VarInt length prefix")
	}
}

// TestHandlerTimeout tests connection timeout
func TestHandlerTimeout(t *testing.T) {
	// Try to connect to a non-existent server
	handler := NewHandler("192.0.2.1:9999", 1*time.Second) // Using TEST-NET-1 (should be unreachable)

	// Create a dummy connection
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	ctx := context.Background()

	// This should timeout
	errChan := make(chan error, 1)
	go func() {
		errChan <- handler.HandleConnection(ctx, client)
	}()

	select {
	case err := <-errChan:
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Error("Test timed out waiting for handler to fail")
	}
}
