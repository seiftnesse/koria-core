package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"koria-core/stats"
	"time"
)

func main() {
	watch := flag.Bool("watch", false, "Непрерывно отображать статистику")
	interval := flag.Int("interval", 1, "Интервал обновления в секундах (для watch режима)")
	jsonOutput := flag.Bool("json", false, "Вывод в JSON формате")
	flag.Parse()

	if *watch {
		watchStats(*interval, *jsonOutput)
	} else {
		printStats(*jsonOutput)
	}
}

func printStats(asJSON bool) {
	snapshot := stats.Global().GetSnapshot()

	if asJSON {
		data, _ := json.MarshalIndent(snapshot, "", "  ")
		fmt.Println(string(data))
		return
	}

	// Красивый текстовый вывод
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║            Koria-Core - Статистика протокола             ║")
	fmt.Println("╠══════════════════════════════════════════════════════════╣")

	// Соединения
	fmt.Println("║ 📡 Соединения:                                          ║")
	fmt.Printf("║   Всего:         %-10d                            ║\n", snapshot.TotalConnections)
	fmt.Printf("║   Активных:      %-10d                            ║\n", snapshot.ActiveConnections)
	fmt.Printf("║   Ошибок:        %-10d                            ║\n", snapshot.FailedConnections)
	fmt.Println("║                                                          ║")

	// Виртуальные потоки
	fmt.Println("║ 🔀 Виртуальные потоки:                                  ║")
	fmt.Printf("║   Всего:         %-10d                            ║\n", snapshot.TotalStreams)
	fmt.Printf("║   Активных:      %-10d                            ║\n", snapshot.ActiveStreams)
	fmt.Printf("║   Закрыто:       %-10d                            ║\n", snapshot.ClosedStreams)
	fmt.Println("║                                                          ║")

	// Трафик
	fmt.Println("║ 📊 Трафик:                                              ║")
	fmt.Printf("║   Отправлено:    %-10s                            ║\n", formatBytes(snapshot.BytesSent))
	fmt.Printf("║   Получено:      %-10s                            ║\n", formatBytes(snapshot.BytesReceived))
	fmt.Printf("║   Пакетов (TX):  %-10d                            ║\n", snapshot.PacketsSent)
	fmt.Printf("║   Пакетов (RX):  %-10d                            ║\n", snapshot.PacketsReceived)
	fmt.Println("║                                                          ║")

	// Ошибки
	fmt.Println("║ ⚠️  Ошибки:                                              ║")
	fmt.Printf("║   Всего:         %-10d                            ║\n", snapshot.TotalErrors)
	fmt.Printf("║   Соединений:    %-10d                            ║\n", snapshot.ConnectionErrors)
	fmt.Printf("║   Потоков:       %-10d                            ║\n", snapshot.StreamErrors)
	fmt.Printf("║   Пакетов:       %-10d                            ║\n", snapshot.PacketErrors)
	fmt.Println("║                                                          ║")

	// Время работы
	fmt.Println("║ ⏱️  Время:                                               ║")
	fmt.Printf("║   Uptime:        %-10s                            ║\n", formatDuration(snapshot.Uptime))
	fmt.Printf("║   Последняя активность: %s              ║\n", snapshot.LastActivity.Format("15:04:05"))

	// Типы пакетов
	if len(snapshot.PacketTypes) > 0 {
		fmt.Println("║                                                          ║")
		fmt.Println("║ 📦 Типы пакетов:                                        ║")
		for pktType, count := range snapshot.PacketTypes {
			fmt.Printf("║   %-20s %-10d                   ║\n", pktType, count)
		}
	}

	fmt.Println("╚══════════════════════════════════════════════════════════╝")
}

func watchStats(interval int, asJSON bool) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		// Очистка экрана (работает на Linux/Mac)
		if !asJSON {
			fmt.Print("\033[H\033[2J")
		}

		printStats(asJSON)

		<-ticker.C
	}
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}
