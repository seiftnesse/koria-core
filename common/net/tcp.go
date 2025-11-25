package net

import (
	"net"
	"time"
)

// OptimizeTCPConn применяет оптимальные TCP параметры для высокой производительности
func OptimizeTCPConn(conn net.Conn) error {
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return nil // Не TCP соединение
	}

	// Отключаем алгоритм Nagle для низкой latency
	if err := tcpConn.SetNoDelay(true); err != nil {
		return err
	}

	// Включаем TCP keep-alive для предотвращения timeout
	if err := tcpConn.SetKeepAlive(true); err != nil {
		return err
	}

	// Keep-alive каждые 30 секунд
	if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil {
		return err
	}

	// Увеличиваем буферы отправки и получения (512KB)
	// Это критично для высокопроизводительных соединений
	if err := tcpConn.SetReadBuffer(512 * 1024); err != nil {
		return err
	}

	if err := tcpConn.SetWriteBuffer(512 * 1024); err != nil {
		return err
	}

	return nil
}

// SetTCPDeadlines устанавливает deadlines для TCP соединения
func SetTCPDeadlines(conn net.Conn, readTimeout, writeTimeout time.Duration) error {
	if readTimeout > 0 {
		if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			return err
		}
	}

	if writeTimeout > 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			return err
		}
	}

	return nil
}
