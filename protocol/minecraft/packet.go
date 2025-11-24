package minecraft

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// PacketType определяет тип пакета
type PacketType int32

const (
	// Handshake packets
	PacketTypeHandshake PacketType = 0x00

	// Login packets
	PacketTypeLoginStart   PacketType = 0x00
	PacketTypeLoginSuccess PacketType = 0x02

	// Play packets (C2S)
	PacketTypePlayerMove      PacketType = 0x1A // MOVE_PLAYER_POS_ROT
	PacketTypePlayerPosition  PacketType = 0x17 // MOVE_PLAYER_POS
	PacketTypePlayerRotation  PacketType = 0x19 // MOVE_PLAYER_ROT
	PacketTypePlayerAction    PacketType = 0x24 // PLAYER_ACTION
	PacketTypeHandSwing       PacketType = 0x36 // SWING
	PacketTypeChatMessage     PacketType = 0x07 // CHAT
	PacketTypeCustomPayload   PacketType = 0x12 // CUSTOM_PAYLOAD
	PacketTypeUpdateSelectedSlot PacketType = 0x2E // SET_CARRIED_ITEM
)

// NetworkPhase определяет фазу протокола
type NetworkPhase int

const (
	PhaseHandshaking NetworkPhase = iota
	PhaseStatus
	PhaseLogin
	PhasePlay
)

// Packet базовый интерфейс для всех пакетов
type Packet interface {
	// PacketID возвращает ID пакета
	PacketID() PacketType

	// Encode кодирует пакет в writer
	Encode(w io.Writer) error

	// Decode декодирует пакет из reader
	Decode(r io.Reader) error
}

// PacketReader интерфейс для чтения типизированных пакетов
type PacketReader interface {
	ReadTypedPacket(r io.Reader, packetID PacketType) (Packet, error)
}

// ReadPacket читает пакет из соединения
// Формат: [VarInt: длина] [VarInt: packet ID] [данные]
// Возвращает PacketType и данные для дальнейшей обработки
func ReadPacketRaw(r io.Reader) (PacketType, []byte, error) {
	// Читаем длину пакета
	length, err := ReadVarInt(r)
	if err != nil {
		return 0, nil, fmt.Errorf("read packet length: %w", err)
	}

	if length <= 0 || length > 2097151 { // 2^21-1 max packet size
		return 0, nil, fmt.Errorf("invalid packet length: %d", length)
	}

	// Читаем данные пакета
	packetData := make([]byte, length)
	if _, err := io.ReadFull(r, packetData); err != nil {
		return 0, nil, fmt.Errorf("read packet data: %w", err)
	}

	// Создаем буфер для чтения
	buf := bytes.NewReader(packetData)

	// Читаем packet ID
	packetID, err := ReadVarInt(buf)
	if err != nil {
		return 0, nil, fmt.Errorf("read packet ID: %w", err)
	}

	// Возвращаем оставшиеся данные
	remainingData := make([]byte, buf.Len())
	buf.Read(remainingData)

	return PacketType(packetID), remainingData, nil
}

// ReadPacket читает и декодирует пакет используя предоставленный пакет
func ReadPacket(r io.Reader, packet Packet) error {
	packetID, data, err := ReadPacketRaw(r)
	if err != nil {
		return err
	}

	if packetID != packet.PacketID() {
		return fmt.Errorf("unexpected packet ID: got 0x%02X, expected 0x%02X", packetID, packet.PacketID())
	}

	// Декодируем пакет
	buf := bytes.NewReader(data)
	if err := packet.Decode(buf); err != nil {
		return fmt.Errorf("decode packet: %w", err)
	}

	return nil
}

// WritePacket записывает пакет в соединение
func WritePacket(w io.Writer, packet Packet) error {
	// Кодируем пакет в буфер
	var buf bytes.Buffer

	// Записываем packet ID
	if err := WriteVarInt(&buf, int32(packet.PacketID())); err != nil {
		return fmt.Errorf("write packet ID: %w", err)
	}

	// Записываем данные пакета
	if err := packet.Encode(&buf); err != nil {
		return fmt.Errorf("encode packet: %w", err)
	}

	// Записываем длину и данные
	packetData := buf.Bytes()
	if err := WriteVarInt(w, int32(len(packetData))); err != nil {
		return fmt.Errorf("write packet length: %w", err)
	}

	if _, err := w.Write(packetData); err != nil {
		return fmt.Errorf("write packet data: %w", err)
	}

	return nil
}

// DecodePacket декодирует пакет из данных
func DecodePacket(packet Packet, data []byte) error {
	buf := bytes.NewReader(data)
	return packet.Decode(buf)
}

// Utility functions

// ReadString читает строку из reader
func ReadString(r io.Reader, maxLength int) (string, error) {
	length, err := ReadVarInt(r)
	if err != nil {
		return "", err
	}

	if length < 0 || length > int32(maxLength) {
		return "", fmt.Errorf("string length out of range: %d", length)
	}

	if length == 0 {
		return "", nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

// WriteString записывает строку в writer
func WriteString(w io.Writer, s string, maxLength int) error {
	if len(s) > maxLength {
		return fmt.Errorf("string too long: %d > %d", len(s), maxLength)
	}

	if err := WriteVarInt(w, int32(len(s))); err != nil {
		return err
	}

	_, err := w.Write([]byte(s))
	return err
}

// ReadUUID читает UUID (16 байт)
func ReadUUID(r io.Reader) ([16]byte, error) {
	var uuid [16]byte
	if _, err := io.ReadFull(r, uuid[:]); err != nil {
		return uuid, err
	}
	return uuid, nil
}

// WriteUUID записывает UUID
func WriteUUID(w io.Writer, uuid [16]byte) error {
	_, err := w.Write(uuid[:])
	return err
}

// ReadDouble читает double (float64)
func ReadDouble(r io.Reader) (float64, error) {
	var bits uint64
	if err := binary.Read(r, binary.BigEndian, &bits); err != nil {
		return 0, err
	}
	return float64frombits(bits), nil
}

// WriteDouble записывает double
func WriteDouble(w io.Writer, val float64) error {
	bits := float64tobits(val)
	return binary.Write(w, binary.BigEndian, bits)
}

// ReadFloat читает float (float32)
func ReadFloat(r io.Reader) (float32, error) {
	var bits uint32
	if err := binary.Read(r, binary.BigEndian, &bits); err != nil {
		return 0, err
	}
	return float32frombits(bits), nil
}

// WriteFloat записывает float
func WriteFloat(w io.Writer, val float32) error {
	bits := float32tobits(val)
	return binary.Write(w, binary.BigEndian, bits)
}

// Helper functions for float conversions
func float64tobits(f float64) uint64 {
	return math.Float64bits(f)
}

func float64frombits(bits uint64) float64 {
	return math.Float64frombits(bits)
}

func float32tobits(f float32) uint32 {
	return math.Float32bits(f)
}

func float32frombits(bits uint32) float32 {
	return math.Float32frombits(bits)
}
