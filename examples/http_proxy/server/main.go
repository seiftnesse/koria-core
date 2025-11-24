package main

import (
	"bufio"
	"flag"
	"github.com/google/uuid"
	"io"
	"koria-core/config"
	"koria-core/transport"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HTTPProxyServer - серверная часть HTTP/HTTPS proxy
// Принимает виртуальные потоки от клиента и проксирует к целевым серверам
func main() {
	listenAddr := flag.String("listen", "0.0.0.0:25565", "Адрес для прослушивания")
	flag.Parse()

	// Создаем UUID для пользователя
	userID := uuid.New()
	log.Printf("═══════════════════════════════════════════════════════════")
	log.Printf("  Koria HTTP/HTTPS Proxy Server")
	log.Printf("═══════════════════════════════════════════════════════════")
	log.Printf("Server UUID: %s", userID)
	log.Printf("Используйте этот UUID для подключения клиента!")
	log.Printf("Listening: %s", *listenAddr)
	log.Printf("═══════════════════════════════════════════════════════════")

	// Конфигурация сервера
	serverConfig := &transport.ServerConfig{
		ListenAddr: *listenAddr,
		Users: []config.User{
			{
				ID:    userID,
				Email: "proxy@koria.local",
				Level: 0,
			},
		},
	}

	// Создаем сервер
	server, err := transport.Listen(serverConfig)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	defer server.Close()

	log.Println("✓ Server started successfully")
	log.Println("✓ Ready to accept HTTP/HTTPS connections")
	log.Println("")

	// Запускаем приём TCP соединений в фоне
	go func() {
		if err := server.Serve(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Обрабатываем входящие виртуальные потоки
	for {
		stream, err := server.AcceptStream()
		if err != nil {
			// Игнорируем ошибку "no active connections" - это нормально когда клиент еще не подключился
			if err.Error() != "no active connections" {
				log.Printf("Accept stream error: %v", err)
			}
			// Небольшая задержка чтобы не спамить CPU
			time.Sleep(100 * time.Millisecond)
			continue
		}

		log.Printf("✓ Accepted virtual stream from %s", stream.RemoteAddr())

		// Обрабатываем в отдельной горутине
		go handleProxyStream(stream)
	}
}

// handleProxyStream обрабатывает виртуальный поток от клиента
func handleProxyStream(clientStream net.Conn) {
	defer clientStream.Close()

	// Читаем первую строку - команду от клиента
	reader := bufio.NewReader(clientStream)
	cmdLine, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Failed to read command: %v", err)
		return
	}

	cmdLine = strings.TrimSpace(cmdLine)
	parts := strings.Split(cmdLine, " ")
	if len(parts) < 2 {
		log.Printf("Invalid command: %s", cmdLine)
		return
	}

	command := parts[0]
	target := parts[1]

	switch command {
	case "CONNECT":
		handleHTTPSConnect(clientStream, reader, target)
	case "HTTP":
		if len(parts) < 4 {
			log.Printf("Invalid HTTP command")
			return
		}
		method := parts[1]
		host := parts[2]
		path := parts[3]
		handleHTTPRequest(clientStream, reader, method, host, path)
	default:
		log.Printf("Unknown command: %s", command)
	}
}

// handleHTTPSConnect обрабатывает HTTPS туннелинг
func handleHTTPSConnect(clientStream net.Conn, reader *bufio.Reader, targetHost string) {
	log.Printf("→ CONNECT %s", targetHost)

	// Подключаемся к целевому серверу
	targetConn, err := net.Dial("tcp", targetHost)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", targetHost, err)
		clientStream.Write([]byte("ERROR\n"))
		return
	}
	defer targetConn.Close()

	// Отправляем успешный ответ клиенту
	clientStream.Write([]byte("OK\n"))
	log.Printf("✓ Connected to %s", targetHost)

	// Двунаправленное копирование
	var wg sync.WaitGroup
	wg.Add(2)

	// Client -> Target
	go func() {
		defer wg.Done()
		written, _ := io.Copy(targetConn, clientStream)
		log.Printf("Client -> %s: %d bytes", targetHost, written)
		targetConn.Close()
	}()

	// Target -> Client
	go func() {
		defer wg.Done()
		written, _ := io.Copy(clientStream, targetConn)
		log.Printf("%s -> Client: %d bytes", targetHost, written)
		clientStream.Close()
	}()

	wg.Wait()
	log.Printf("✓ HTTPS tunnel closed for %s", targetHost)
}

// handleHTTPRequest обрабатывает обычный HTTP запрос
func handleHTTPRequest(clientStream net.Conn, reader *bufio.Reader, method, host, path string) {
	log.Printf("→ HTTP %s %s%s", method, host, path)

	// Определяем порт
	targetAddr := host
	if !strings.Contains(host, ":") {
		targetAddr = host + ":80"
	}

	// Подключаемся к целевому серверу
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("Failed to connect to %s: %v", targetAddr, err)
		clientStream.Write([]byte("ERROR\n"))
		return
	}
	defer targetConn.Close()

	// Отправляем OK клиенту
	clientStream.Write([]byte("OK\n"))

	// Читаем оригинальный запрос от клиента
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("Failed to read request: %v", err)
		return
	}

	// Отправляем запрос целевому серверу
	if err := req.Write(targetConn); err != nil {
		log.Printf("Failed to forward request: %v", err)
		return
	}

	// Копируем ответ обратно клиенту
	written, _ := io.Copy(clientStream, targetConn)
	log.Printf("✓ HTTP %s %s: %d bytes", method, host, written)
}
