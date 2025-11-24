package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"io"
	"koria-core/config"
	"koria-core/transport"
	"log"
	"net"
	"sync"
	"time"
)

// Простой пример использования koria-core протокола
// Демонстрирует:
// 1. Запуск сервера с UUID аутентификацией
// 2. Подключение клиента
// 3. Открытие множества виртуальных потоков через одно TCP соединение
// 4. Передача данных через стеганографию в Minecraft пакетах

func main() {
	// 1. Создаем UUID для тестового пользователя
	userID := uuid.New()
	fmt.Printf("Test User UUID: %s\n", userID)

	// 2. Запускаем сервер в отдельной горутине
	go runServer(userID)

	// Ждем, пока сервер запустится
	time.Sleep(1 * time.Second)

	// 3. Запускаем клиента
	runClient(userID)
}

func runServer(userID uuid.UUID) {
	// Конфигурация сервера
	serverConfig := &transport.ServerConfig{
		ListenAddr: "127.0.0.1:25565",
		Users: []config.User{
			{
				ID:    userID,
				Email: "test@example.com",
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

	log.Println("Server started on 127.0.0.1:25565")

	// Обрабатываем входящие виртуальные потоки
	go handleServerStreams(server)

	// Запускаем сервер
	if err := server.Serve(); err != nil {
		log.Printf("Server error: %v", err)
	}
}

func handleServerStreams(server *transport.Server) {
	for {
		// Принимаем виртуальный поток
		stream, err := server.AcceptStream()
		if err != nil {
			log.Printf("Accept stream error: %v", err)
			return
		}

		log.Printf("Accepted new virtual stream from %s", stream.RemoteAddr())

		// Обрабатываем поток в отдельной горутине
		go func(s net.Conn) {
			defer s.Close()

			// Echo server - возвращаем все данные обратно
			buf := make([]byte, 4096)
			for {
				n, err := s.Read(buf)
				if err != nil {
					// EOF - нормальное закрытие, не логируем
					if err != io.EOF {
						log.Printf("Stream read error: %v", err)
					}
					return
				}

				log.Printf("Received %d bytes: %s", n, string(buf[:n]))

				// Отправляем обратно
				if _, err := s.Write(buf[:n]); err != nil {
					// Клиент мог закрыть соединение
					return
				}
			}
		}(stream)
	}
}

func runClient(userID uuid.UUID) {
	// Конфигурация клиента
	clientConfig := &transport.ClientConfig{
		ServerAddr: "127.0.0.1",
		ServerPort: 25565,
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

	// Открываем множество виртуальных потоков (демонстрация мультиплексирования)
	numStreams := 10
	log.Printf("Opening %d virtual streams through ONE TCP connection...", numStreams)

	// Используем WaitGroup для ожидания завершения всех горутин
	var wg sync.WaitGroup
	wg.Add(numStreams)

	for i := 0; i < numStreams; i++ {
		streamID := i
		go func() {
			defer wg.Done()

			// Открываем виртуальный поток
			stream, err := client.DialStream(ctx)
			if err != nil {
				log.Printf("Failed to open stream %d: %v", streamID, err)
				return
			}
			defer stream.Close()

			log.Printf("Opened virtual stream #%d", streamID)

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

			log.Printf("Stream #%d received response: %s", streamID, string(buf[:n]))
		}()
	}

	// Ждем завершения работы всех потоков
	wg.Wait()

	log.Printf("Active streams: %d", client.StreamCount())
	log.Println("Test completed!")
}
