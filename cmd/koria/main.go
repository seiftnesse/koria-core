package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"koria-core/app/dispatcher"
	"koria-core/app/proxyman/inbound"
	"koria-core/app/proxyman/outbound"
	"koria-core/config"
	v2config "koria-core/config/v2"
	"koria-core/proxy/freedom"
	"koria-core/proxy/http"
	koriaproxy "koria-core/proxy/koria"
	"koria-core/proxy/socks"
	"koria-core/transport"
)

const banner = `
╔═══════════════════════════════════════════════════════════════╗
║                    KORIA-CORE v0.1.0                         ║
║           Stealthy Network Tunneling System                  ║
╚═══════════════════════════════════════════════════════════════╝
`

func main() {
	configFile := flag.String("config", "", "Configuration file path (JSON)")
	version := flag.Bool("version", false, "Show version")
	flag.Parse()

	if *version {
		fmt.Println("Koria-Core v0.1.0")
		return
	}

	if *configFile == "" {
		log.Fatal("Usage: koria -config <config.json>")
	}

	fmt.Print(banner)
	log.Printf("Loading configuration from: %s", *configFile)

	// Загружаем конфигурацию
	cfg, err := v2config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Создаем инстанс
	instance, err := NewInstance(cfg)
	if err != nil {
		log.Fatalf("Failed to create instance: %v", err)
	}

	// Запускаем
	if err := instance.Start(); err != nil {
		log.Fatalf("Failed to start instance: %v", err)
	}

	log.Println("✓ Koria-Core started successfully")
	log.Println("Press Ctrl+C to stop")

	// Ожидаем сигнала завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	log.Println("\nShutting down...")
	instance.Close()
	log.Println("✓ Shutdown complete")
}

// Instance представляет запущенный инстанс Koria
type Instance struct {
	ihm *inbound.Manager
	ohm *outbound.Manager
	d   dispatcher.Interface
}

// NewInstance создает новый инстанс из конфигурации
func NewInstance(cfg *v2config.Config) (*Instance, error) {
	instance := &Instance{
		ihm: inbound.NewManager(),
		ohm: outbound.NewManager(),
	}

	// Создаем dispatcher
	instance.d = dispatcher.NewDefaultDispatcher(instance.ohm)

	// Инициализируем outbounds
	if err := instance.initOutbounds(cfg.Outbounds); err != nil {
		return nil, fmt.Errorf("init outbounds: %w", err)
	}

	// Инициализируем inbounds
	if err := instance.initInbounds(cfg.Inbounds); err != nil {
		return nil, fmt.Errorf("init inbounds: %w", err)
	}

	return instance, nil
}

// initOutbounds инициализирует outbound handlers
func (i *Instance) initOutbounds(configs []v2config.OutboundConfig) error {
	ctx := context.Background()

	for idx, cfg := range configs {
		log.Printf("Initializing outbound [%d]: %s (%s)", idx, cfg.Tag, cfg.Protocol)

		var handler outbound.Handler
		var err error

		switch cfg.Protocol {
		case "freedom":
			handler = freedom.NewHandler(cfg.Tag)

		case "koria":
			handler, err = i.createKoriaOutbound(cfg)
			if err != nil {
				return fmt.Errorf("create koria outbound: %w", err)
			}

		default:
			return fmt.Errorf("unsupported outbound protocol: %s", cfg.Protocol)
		}

		if err := i.ohm.AddHandler(ctx, handler); err != nil {
			return fmt.Errorf("add outbound handler: %w", err)
		}

		// Первый outbound становится дефолтным
		if idx == 0 {
			i.ohm.SetDefaultHandler(handler)
			log.Printf("  → Set as default outbound")
		}
	}

	return nil
}

// createKoriaOutbound создает Koria outbound handler
func (i *Instance) createKoriaOutbound(cfg v2config.OutboundConfig) (outbound.Handler, error) {
	// Парсим settings
	settingsJSON, err := jsonMarshal(cfg.Settings)
	if err != nil {
		return nil, fmt.Errorf("marshal settings: %w", err)
	}

	var settings v2config.KoriaOutboundSettings
	if err := jsonUnmarshal(settingsJSON, &settings); err != nil {
		return nil, fmt.Errorf("unmarshal koria settings: %w", err)
	}

	// Парсим UUID
	userID, err := uuid.Parse(settings.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}

	// Создаем transport client
	clientConfig := &transport.ClientConfig{
		ServerAddr: settings.Address,
		ServerPort: settings.Port,
		UserID:     userID,
	}

	log.Printf("  → Connecting to %s:%d (UUID: %s)", settings.Address, settings.Port, userID)

	client, err := transport.Dial(context.Background(), clientConfig)
	if err != nil {
		return nil, fmt.Errorf("dial koria server: %w", err)
	}

	log.Printf("  ✓ Connected to Koria server")

	return koriaproxy.NewHandler(cfg.Tag, client), nil
}

// initInbounds инициализирует inbound handlers
func (i *Instance) initInbounds(configs []v2config.InboundConfig) error {
	ctx := context.Background()

	for idx, cfg := range configs {
		log.Printf("Initializing inbound [%d]: %s (%s) on %s", idx, cfg.Tag, cfg.Protocol, cfg.Listen)

		var handler inbound.Handler
		var err error

		switch cfg.Protocol {
		case "http":
			handler = http.NewServer(cfg.Tag, cfg.Listen, i.d)

		case "socks":
			handler = socks.NewServer(cfg.Tag, cfg.Listen, i.d)

		case "koria":
			handler, err = i.createKoriaInbound(cfg)
			if err != nil {
				return fmt.Errorf("create koria inbound: %w", err)
			}

		default:
			return fmt.Errorf("unsupported inbound protocol: %s", cfg.Protocol)
		}

		if err := i.ihm.AddHandler(ctx, handler); err != nil {
			return fmt.Errorf("add inbound handler: %w", err)
		}
	}

	return nil
}

// createKoriaInbound создает Koria inbound handler
func (i *Instance) createKoriaInbound(cfg v2config.InboundConfig) (inbound.Handler, error) {
	// Парсим settings
	settingsJSON, err := jsonMarshal(cfg.Settings)
	if err != nil {
		return nil, fmt.Errorf("marshal settings: %w", err)
	}

	var settings v2config.KoriaInboundSettings
	if err := jsonUnmarshal(settingsJSON, &settings); err != nil {
		return nil, fmt.Errorf("unmarshal koria settings: %w", err)
	}

	// Конвертируем клиентов
	users := make([]config.User, len(settings.Clients))
	for i, client := range settings.Clients {
		userID, err := uuid.Parse(client.ID)
		if err != nil {
			return nil, fmt.Errorf("parse client id: %w", err)
		}

		users[i] = config.User{
			ID:    userID,
			Email: client.Email,
			Level: client.Level,
		}

		log.Printf("  → Client [%d]: %s (%s)", i, userID, client.Email)
	}

	return koriaproxy.NewServer(cfg.Tag, cfg.Listen, users, i.d)
}

// Start запускает инстанс
func (i *Instance) Start() error {
	// Inbounds уже запущены при добавлении через Start()
	return nil
}

// Close закрывает инстанс
func (i *Instance) Close() error {
	return i.ihm.Close()
}

// Вспомогательные функции для работы с JSON
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
