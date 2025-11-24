package main

import (
	"github.com/google/uuid"
	"io"
	"koria-core/config"
	"koria-core/transport"
	"log"
)

func main() {
	// Создаем UUID для тестового пользователя
	userID := uuid.New()
	log.Printf("Server UUID: %s", userID)
	log.Printf("Используйте этот UUID для подключения клиента!")

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
	log.Println("Waiting for connections...")

	// Обрабатываем входящие виртуальные потоки
	for {
		stream, err := server.AcceptStream()
		if err != nil {
			log.Printf("Accept stream error: %v", err)
			continue
		}

		log.Printf("Accepted new virtual stream from %s", stream.RemoteAddr())

		// Обрабатываем поток в отдельной горутине (Echo server)
		go func() {
			defer stream.Close()

			buf := make([]byte, 4096)
			for {
				n, err := stream.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Printf("Stream read error: %v", err)
					}
					return
				}

				log.Printf("Received %d bytes: %s", n, string(buf[:n]))

				// Отправляем обратно (echo)
				if _, err := stream.Write(buf[:n]); err != nil {
					log.Printf("Stream write error: %v", err)
					return
				}
			}
		}()
	}
}
