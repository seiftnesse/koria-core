package c2s

import (
	"encoding/binary"
	"fmt"
	"io"
	"koria-core/protocol/minecraft"
)

// StatusRequestPacket - запрос статуса сервера (Server List Ping)
// Packet ID: 0x00 (Status state)
type StatusRequestPacket struct {
	// Пустой пакет
}

// PacketID возвращает ID пакета
func (p *StatusRequestPacket) PacketID() minecraft.PacketType {
	return 0x00
}

// Encode кодирует пакет
func (p *StatusRequestPacket) Encode(w io.Writer) error {
	// Пустой пакет, ничего не пишем
	return nil
}

// Decode декодирует пакет
func (p *StatusRequestPacket) Decode(reader io.Reader) error {
	// Пустой пакет, ничего не декодируем
	return nil
}

// PingRequestPacket - ping запрос
// Packet ID: 0x01 (Status state)
type PingRequestPacket struct {
	Payload int64
}

// PacketID возвращает ID пакета
func (p *PingRequestPacket) PacketID() minecraft.PacketType {
	return 0x01
}

// Encode кодирует пакет
func (p *PingRequestPacket) Encode(w io.Writer) error {
	// Long (8 bytes, big-endian)
	return binary.Write(w, binary.BigEndian, p.Payload)
}

// Decode декодирует пакет
func (p *PingRequestPacket) Decode(reader io.Reader) error {
	// Read 8 bytes (long)
	var buf [8]byte
	if _, err := io.ReadFull(reader, buf[:]); err != nil {
		return fmt.Errorf("failed to read long: %w", err)
	}

	p.Payload = int64(binary.BigEndian.Uint64(buf[:]))
	return nil
}
