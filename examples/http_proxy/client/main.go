package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"io"
	"koria-core/transport"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
)

// HTTPProxyClient - HTTP/HTTPS proxy клиент с поддержкой CONNECT
func main() {
	listenAddr := flag.String("listen", "127.0.0.1:8080", "Локальный адрес для прослушивания")
	serverAddr := flag.String("server", "127.0.0.1", "Адрес Koria сервера")
	serverPort := flag.Int("port", 25565, "Порт Koria сервера")
	uuidStr := flag.String("uuid", "", "UUID для аутентификации (ОБЯЗАТЕЛЬНО)")
	flag.Parse()

	if *uuidStr == "" {
		log.Fatal("UUID обязателен! Используйте: -uuid <uuid>")
	}

	userID, err := uuid.Parse(*uuidStr)
	if err != nil {
		log.Fatalf("Invalid UUID: %v", err)
	}

	log.Printf("═══════════════════════════════════════════════════════════")
	log.Printf("  Koria HTTP/HTTPS Proxy Client")
	log.Printf("═══════════════════════════════════════════════════════════")
	log.Printf("Local listening: %s", *listenAddr)
	log.Printf("Koria server: %s:%d", *serverAddr, *serverPort)
	log.Printf("UUID: %s", userID)
	log.Printf("═══════════════════════════════════════════════════════════")
	log.Println("")
	log.Printf("Connecting to Koria server %s:%d...", *serverAddr, *serverPort)

	// Подключаемся к Koria серверу
	ctx := context.Background()
	clientConfig := &transport.ClientConfig{
		ServerAddr: *serverAddr,
		ServerPort: *serverPort,
		UserID:     userID,
	}

	koriaClient, err := transport.Dial(ctx, clientConfig)
	if err != nil {
		log.Printf("✗ Connection failed!")
		log.Printf("✗ Error: %v", err)
		log.Printf("")
		log.Printf("Possible reasons:")
		log.Printf("  1. Server is not running on %s:%d", *serverAddr, *serverPort)
		log.Printf("  2. Firewall is blocking port %d", *serverPort)
		log.Printf("  3. Wrong server address")
		log.Printf("  4. UUID mismatch: %s", userID)
		log.Fatalf("")
	}
	defer koriaClient.Close()

	log.Println("✓ Connected to Koria server successfully!")
	log.Println("✓ HTTP and HTTPS (CONNECT) proxy ready")
	log.Println("")
	log.Println("Configure your browser:")
	log.Printf("  HTTP Proxy: 127.0.0.1:%s", strings.Split(*listenAddr, ":")[1])
	log.Println("")

	// Слушаем локальный порт
	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer listener.Close()

	// Принимаем соединения
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		go handleHTTPConnection(ctx, conn, koriaClient)
	}
}

// handleHTTPConnection обрабатывает HTTP/HTTPS запрос
func handleHTTPConnection(ctx context.Context, clientConn net.Conn, koriaClient *transport.Client) {
	defer clientConn.Close()

	// Читаем первую строку запроса
	reader := bufio.NewReader(clientConn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("Failed to read request: %v", err)
		return
	}

	log.Printf("✓ %s %s %s", req.Method, req.Host, req.Proto)

	// Обрабатываем CONNECT (для HTTPS)
	if req.Method == "CONNECT" {
		handleHTTPSConnect(ctx, clientConn, koriaClient, req.Host)
		return
	}

	// Обрабатываем обычный HTTP
	handleHTTPRequest(ctx, clientConn, koriaClient, req)
}

// handleHTTPSConnect обрабатывает HTTPS туннелинг через CONNECT
func handleHTTPSConnect(ctx context.Context, clientConn net.Conn, koriaClient *transport.Client, targetHost string) {
	// Открываем виртуальный поток через Koria
	koriaStream, err := koriaClient.DialStream(ctx)
	if err != nil {
		log.Printf("Failed to open Koria stream: %v", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer koriaStream.Close()

	// Отправляем информацию о целевом хосте серверу
	// Формат: CONNECT <host>\n
	fmt.Fprintf(koriaStream, "CONNECT %s\n", targetHost)

	// Читаем ответ от сервера (используем bufio для чтения полной строки)
	streamReader := bufio.NewReader(koriaStream)
	response, err := streamReader.ReadString('\n')
	if err != nil || !strings.HasPrefix(response, "OK") {
		log.Printf("Server connection failed: %v", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	// Отправляем успешный ответ клиенту
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	log.Printf("✓ HTTPS tunnel established to %s", targetHost)

	// Начинаем туннелирование данных
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Server
	go func() {
		defer wg.Done()
		io.Copy(koriaStream, clientConn)
		koriaStream.Close()
	}()

	// Server -> Client (используем streamReader для чтения, чтобы не потерять буферизованные данные)
	go func() {
		defer wg.Done()
		io.Copy(clientConn, streamReader)
		clientConn.Close()
	}()

	wg.Wait()
	log.Printf("✓ HTTPS tunnel closed for %s", targetHost)
}

// handleHTTPRequest обрабатывает обычный HTTP запрос
func handleHTTPRequest(ctx context.Context, clientConn net.Conn, koriaClient *transport.Client, req *http.Request) {
	// Открываем виртуальный поток
	koriaStream, err := koriaClient.DialStream(ctx)
	if err != nil {
		log.Printf("Failed to open Koria stream: %v", err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer koriaStream.Close()

	// Отправляем информацию о запросе серверу
	// Формат: HTTP <method> <host> <path>\n
	fmt.Fprintf(koriaStream, "HTTP %s %s %s\n", req.Method, req.Host, req.RequestURI)

	// Читаем подтверждение (используем bufio для чтения полной строки)
	streamReader := bufio.NewReader(koriaStream)
	response, err := streamReader.ReadString('\n')
	if err != nil || !strings.HasPrefix(response, "OK") {
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}

	// Отправляем оригинальный запрос
	if err := req.Write(koriaStream); err != nil {
		log.Printf("Failed to forward request: %v", err)
		return
	}

	// Копируем ответ обратно клиенту (используем streamReader для чтения)
	io.Copy(clientConn, streamReader)
	log.Printf("✓ HTTP request completed for %s", req.Host)
}
