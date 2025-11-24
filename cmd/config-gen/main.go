package main

import (
	"flag"
	"fmt"
	"koria-core/config"
	"log"
	"os"
	"strings"
)

func main() {
	configType := flag.String("type", "server", "Тип конфигурации: server, client, proxy")
	output := flag.String("output", "", "Имя выходного файла (по умолчанию: <type>-config.json)")
	flag.Parse()

	// Определяем имя файла
	filename := *output
	if filename == "" {
		filename = *configType + "-config.json"
	}

	// Проверяем, существует ли файл
	if _, err := os.Stat(filename); err == nil {
		fmt.Printf("⚠️  Файл %s уже существует. Перезаписать? (y/n): ", filename)
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			fmt.Println("Отменено.")
			return
		}
	}

	// Генерируем конфигурацию
	var err error
	switch *configType {
	case "server":
		cfg := config.DefaultServerConfigFile()
		err = config.SaveServerConfig(filename, cfg)
		if err == nil {
			fmt.Printf("✓ Конфигурация сервера сохранена в %s\n", filename)
			fmt.Printf("\nСгенерированный UUID пользователя:\n  %s\n", cfg.Users[0].ID)
			fmt.Println("\nИспользуйте этот UUID для подключения клиента!")
		}

	case "client":
		cfg := config.DefaultClientConfigFile()
		err = config.SaveClientConfig(filename, cfg)
		if err == nil {
			fmt.Printf("✓ Конфигурация клиента сохранена в %s\n", filename)
			fmt.Printf("\nСгенерированный UUID:\n  %s\n", cfg.UserID)
			fmt.Println("\n⚠️  ВАЖНО: Замените UUID на тот, который выдал сервер!")
			fmt.Printf("Отредактируйте файл %s и укажите правильный UUID\n", filename)
		}

	case "proxy":
		cfg := config.DefaultProxyConfigFile()
		err = config.SaveProxyConfig(filename, cfg)
		if err == nil {
			fmt.Printf("✓ Конфигурация proxy сохранена в %s\n", filename)
			fmt.Printf("\nСгенерированный UUID:\n  %s\n", cfg.UserID)
			fmt.Println("\n⚠️  ВАЖНО: Замените UUID на тот, который выдал сервер!")
			fmt.Printf("Отредактируйте файл %s и укажите:\n", filename)
			fmt.Println("  - server_addr: адрес Koria сервера")
			fmt.Println("  - user_id: UUID от сервера")
			fmt.Println("  - target_addr: целевой сервер (для TCP режима)")
		}

	default:
		log.Fatalf("Неизвестный тип конфигурации: %s\nДоступные: server, client, proxy", *configType)
	}

	if err != nil {
		log.Fatalf("Ошибка при сохранении конфигурации: %v", err)
	}

	separator := strings.Repeat("─", 60)
	fmt.Println("\n" + separator)
	fmt.Println("Следующие шаги:")
	fmt.Println(separator)

	switch *configType {
	case "server":
		fmt.Printf("\n1. Отредактируйте %s при необходимости\n", filename)
		fmt.Printf("2. Запустите сервер:\n   ./koria-server -config %s\n", filename)

	case "client":
		fmt.Printf("\n1. Получите UUID от сервера\n")
		fmt.Printf("2. Отредактируйте %s и укажите:\n", filename)
		fmt.Println("   - server_addr: адрес сервера")
		fmt.Println("   - user_id: UUID от сервера")
		fmt.Printf("3. Запустите клиент:\n   ./koria-client -config %s\n", filename)

	case "proxy":
		fmt.Printf("\n1. Получите UUID от сервера\n")
		fmt.Printf("2. Отредактируйте %s\n", filename)
		fmt.Printf("3. Запустите proxy:\n   ./koria-proxy -config %s\n", filename)
	}

	fmt.Println()
}
