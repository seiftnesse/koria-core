package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"koria-core/transport"
	"log"
	"time"
)

func main() {
	// Параметры командной строки
	uuidStr := flag.String("uuid", "", "UUID пользователя для аутентификации")
	serverAddr := flag.String("server", "127.0.0.1", "Адрес сервера")
	serverPort := flag.Int("port", 25565, "Порт сервера")
	streams := flag.Int("streams", 5, "Количество виртуальных потоков")
	flag.Parse()

	if *uuidStr == "" {
		log.Fatal("UUID обязателен! Используйте: -uuid <uuid>")
	}

	// Парсим UUID
	userID, err := uuid.Parse(*uuidStr)
	if err != nil {
		log.Fatalf("Invalid UUID: %v", err)
	}

	// Конфигурация клиента
	clientConfig := &transport.ClientConfig{
		ServerAddr: *serverAddr,
		ServerPort: *serverPort,
		UserID:     userID,
	}

	// Подключаемся к серверу
	ctx := context.Background()
	client, err := transport.Dial(ctx, clientConfig)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	log.Println("Connected to server successfully!")
	log.Printf("Client authenticated with UUID: %s", userID)

	// Открываем множество виртуальных потоков
	log.Printf("Opening %d virtual streams through ONE TCP connection...", *streams)

	for i := 0; i < *streams; i++ {
		streamID := i
		go func() {
			// Открываем виртуальный поток
			stream, err := client.DialStream(ctx)
			if err != nil {
				log.Printf("Failed to open stream %d: %v", streamID, err)
				return
			}
			defer stream.Close()

			log.Printf("✓ Opened virtual stream #%d", streamID)

			// Отправляем данные
			message := fmt.Sprintf("Hello from stream #%d!", streamID)
			if _, err := stream.Write([]byte(message)); err != nil {
				log.Printf("Stream %d write error: %v", streamID, err)
				return
			}

			// Читаем ответ
			buf := make([]byte, 1024)
			n, err := stream.Read(buf)
			if err != nil {
				log.Printf("Stream %d read error: %v", streamID, err)
				return
			}

			log.Printf("✓ Stream #%d received response: %s", streamID, string(buf[:n]))
		}()
	}

	// Ждем завершения работы потоков
	time.Sleep(5 * time.Second)

	log.Printf("Active streams: %d", client.StreamCount())
	log.Println("Test completed!")
}
