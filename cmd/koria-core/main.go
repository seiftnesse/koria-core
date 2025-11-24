package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/seiftnesse/koria-core/pkg/config"
	"github.com/seiftnesse/koria-core/pkg/proxy"
)

const version = "0.1.0"

func main() {
	// Command line flags
	configFile := flag.String("config", "", "Path to configuration file")
	listenAddr := flag.String("listen", "127.0.0.1:8080", "Listen address")
	upstreamAddr := flag.String("upstream", "", "Upstream server address")
	showVersion := flag.Bool("version", false, "Show version information")
	generateConfig := flag.String("generate-config", "", "Generate default config file")
	
	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("koria-core version %s\n", version)
		fmt.Println("Custom xray-core implementation with Minecraft packet camouflage")
		os.Exit(0)
	}

	// Generate config
	if *generateConfig != "" {
		cfg := config.DefaultConfig()
		if err := config.SaveConfig(cfg, *generateConfig); err != nil {
			log.Fatalf("Failed to generate config: %v", err)
		}
		fmt.Printf("Generated default configuration at %s\n", *generateConfig)
		os.Exit(0)
	}

	// Load configuration
	var cfg *config.Config
	var err error

	if *configFile != "" {
		cfg, err = config.LoadConfig(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		log.Printf("Loaded configuration from %s", *configFile)
	} else {
		cfg = config.DefaultConfig()
		log.Println("Using default configuration")
	}

	// Override with command line flags
	if *listenAddr != "127.0.0.1:8080" {
		cfg.ListenAddress = *listenAddr
	}
	if *upstreamAddr != "" {
		cfg.UpstreamAddr = *upstreamAddr
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Print startup banner
	printBanner(cfg)

	// Create and start proxy server
	server, err := proxy.NewServer(cfg.ListenAddress, cfg.UpstreamAddr, cfg.Timeout)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		if err := server.Stop(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		os.Exit(0)
	}()

	// Start server
	log.Printf("Starting koria-core proxy server...")
	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func printBanner(cfg *config.Config) {
	banner := `
╦╔═╔═╗╦═╗╦╔═╗  ╔═╗╔═╗╦═╗╔═╗
╠╩╗║ ║╠╦╝║╠═╣  ║  ║ ║╠╦╝║╣ 
╩ ╩╚═╝╩╚═╩╩ ╩  ╚═╝╚═╝╩╚═╚═╝
Custom Xray-core with Minecraft Packet Camouflage
Version: %s
`
	fmt.Printf(banner, version)
	fmt.Println("Configuration:")
	fmt.Printf("  Listen Address:   %s\n", cfg.ListenAddress)
	fmt.Printf("  Upstream Address: %s\n", cfg.UpstreamAddr)
	fmt.Printf("  Timeout:          %s\n", cfg.Timeout)
	fmt.Printf("  Minecraft Server: %s:%d\n", cfg.MinecraftServer, cfg.MinecraftPort)
	fmt.Println()
}
