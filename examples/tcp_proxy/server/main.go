package main

import (
	"flag"
	"github.com/google/uuid"
	"io"
	"koria-core/config"
	"koria-core/transport"
	"log"
	"net"
	"sync"
	"time"
)

// ProxyServer - сервер, который принимает виртуальные потоки от клиентов
// и проксирует их к реальным целевым серверам
func main() {
	listenAddr := flag.String("listen", "0.0.0.0:25565", "Адрес для прослушивания")
	targetAddr := flag.String("target", "httpbin.org:80", "Целевой адрес для проксирования")
	flag.Parse()

	// Создаем UUID для пользователя
	userID := uuid.New()
	log.Printf("═══════════════════════════════════════════════════════════")
	log.Printf("  Koria TCP Proxy Server")
	log.Printf("═══════════════════════════════════════════════════════════")
	log.Printf("Server UUID: %s", userID)
	log.Printf("Используйте этот UUID для подключения клиента!")
	log.Printf("Listening: %s", *listenAddr)
	log.Printf("Target: %s", *targetAddr)
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

	log.Println("Server started successfully. Waiting for connections...")

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

		// Проксируем в отдельной горутине
		go proxyToTarget(stream, *targetAddr)
	}
}

// proxyToTarget проксирует виртуальный поток к целевому серверу
func proxyToTarget(clientStream net.Conn, targetAddr string) {
	defer clientStream.Close()

	// Подключаемся к целевому серверу
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("Failed to connect to target %s: %v", targetAddr, err)
		return
	}
	defer targetConn.Close()

	log.Printf("✓ Connected to target %s", targetAddr)

	// Двунаправленное копирование данных
	var wg sync.WaitGroup
	wg.Add(2)

	// Клиент -> Целевой сервер
	go func() {
		defer wg.Done()
		written, err := io.Copy(targetConn, clientStream)
		if err != nil && err != io.EOF {
			log.Printf("Error copying client -> target: %v", err)
		}
		log.Printf("Client -> Target: %d bytes", written)
		targetConn.Close() // Закрываем запись
	}()

	// Целевой сервер -> Клиент
	go func() {
		defer wg.Done()
		written, err := io.Copy(clientStream, targetConn)
		if err != nil && err != io.EOF {
			log.Printf("Error copying target -> client: %v", err)
		}
		log.Printf("Target -> Client: %d bytes", written)
		clientStream.Close() // Закрываем запись
	}()

	wg.Wait()
	log.Printf("✓ Proxy session completed for %s", targetAddr)
}
