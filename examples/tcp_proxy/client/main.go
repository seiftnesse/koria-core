package main

import (
	"context"
	"flag"
	"github.com/google/uuid"
	"io"
	"koria-core/transport"
	"log"
	"net"
	"sync"
)

// ProxyClient - клиент, который слушает локальный порт
// и проксирует соединения через Koria к удаленному серверу
func main() {
	listenAddr := flag.String("listen", "127.0.0.1:8080", "Локальный адрес для прослушивания")
	serverAddr := flag.String("server", "127.0.0.1", "Адрес Koria сервера")
	serverPort := flag.Int("port", 25565, "Порт Koria сервера")
	uuidStr := flag.String("uuid", "", "UUID для аутентификации (ОБЯЗАТЕЛЬНО)")
	flag.Parse()

	if *uuidStr == "" {
		log.Fatal("UUID обязателен! Используйте: -uuid <uuid>")
	}

	// Парсим UUID
	userID, err := uuid.Parse(*uuidStr)
	if err != nil {
		log.Fatalf("Invalid UUID: %v", err)
	}

	log.Printf("═══════════════════════════════════════════════════════════")
	log.Printf("  Koria TCP Proxy Client")
	log.Printf("═══════════════════════════════════════════════════════════")
	log.Printf("Local listening: %s", *listenAddr)
	log.Printf("Koria server: %s:%d", *serverAddr, *serverPort)
	log.Printf("UUID: %s", userID)
	log.Printf("═══════════════════════════════════════════════════════════")

	// Конфигурация клиента
	clientConfig := &transport.ClientConfig{
		ServerAddr: *serverAddr,
		ServerPort: *serverPort,
		UserID:     userID,
	}

	// Подключаемся к Koria серверу
	ctx := context.Background()
	koriaClient, err := transport.Dial(ctx, clientConfig)
	if err != nil {
		log.Fatalf("Failed to connect to Koria server: %v", err)
	}
	defer koriaClient.Close()

	log.Println("✓ Connected to Koria server successfully!")
	log.Printf("✓ Authenticated with UUID: %s", userID)

	// Слушаем локальный порт
	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", *listenAddr, err)
	}
	defer listener.Close()

	log.Printf("✓ Listening on %s", *listenAddr)
	log.Println("Ready to accept connections! Configure your browser to use this proxy.")
	log.Println("")
	log.Println("Example: curl -x http://127.0.0.1:8080 http://httpbin.org/ip")
	log.Println("")

	// Принимаем локальные соединения
	for {
		localConn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		log.Printf("✓ Accepted local connection from %s", localConn.RemoteAddr())

		// Проксируем в отдельной горутине
		go proxyConnection(ctx, localConn, koriaClient)
	}
}

// proxyConnection проксирует локальное соединение через Koria
func proxyConnection(ctx context.Context, localConn net.Conn, koriaClient *transport.Client) {
	defer localConn.Close()

	// Открываем виртуальный поток через Koria
	koriaStream, err := koriaClient.DialStream(ctx)
	if err != nil {
		log.Printf("Failed to open Koria stream: %v", err)
		return
	}
	defer koriaStream.Close()

	log.Printf("✓ Opened virtual stream through Koria (Minecraft protocol)")

	// Двунаправленное копирование данных
	var wg sync.WaitGroup
	wg.Add(2)

	// Локальное соединение -> Koria stream
	go func() {
		defer wg.Done()
		written, err := io.Copy(koriaStream, localConn)
		if err != nil && err != io.EOF {
			log.Printf("Error copying local -> koria: %v", err)
		}
		log.Printf("Local -> Koria: %d bytes", written)
		koriaStream.Close()
	}()

	// Koria stream -> Локальное соединение
	go func() {
		defer wg.Done()
		written, err := io.Copy(localConn, koriaStream)
		if err != nil && err != io.EOF {
			log.Printf("Error copying koria -> local: %v", err)
		}
		log.Printf("Koria -> Local: %d bytes", written)
		localConn.Close()
	}()

	wg.Wait()
	log.Printf("✓ Proxy session completed")
}
