package minecraft

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// PacketID represents different Minecraft packet types
type PacketID byte

const (
	// Common Minecraft packet IDs for camouflage
	PacketHandshake      PacketID = 0x00
	PacketStatusRequest  PacketID = 0x00
	PacketStatusResponse PacketID = 0x00
	PacketPing           PacketID = 0x01
	PacketPong           PacketID = 0x01
	PacketKeepAlive      PacketID = 0x21
	PacketCustomPayload  PacketID = 0x17 // For embedding our data
)

// Packet represents a Minecraft protocol packet
type Packet struct {
	ID   PacketID
	Data []byte
}

// EncodePacket encodes data into a fake Minecraft packet
func EncodePacket(packetID PacketID, data []byte) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Minecraft packets start with a VarInt length followed by packet ID and data
	packetData := append([]byte{byte(packetID)}, data...)
	length := len(packetData)

	// Write VarInt length
	if err := writeVarInt(buf, int32(length)); err != nil {
		return nil, fmt.Errorf("failed to write packet length: %w", err)
	}

	// Write packet data
	if _, err := buf.Write(packetData); err != nil {
		return nil, fmt.Errorf("failed to write packet data: %w", err)
	}

	return buf.Bytes(), nil
}

// DecodePacket decodes a fake Minecraft packet back to original data
func DecodePacket(reader io.Reader) (*Packet, error) {
	// Read VarInt length
	length, err := readVarInt(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read packet length: %w", err)
	}

	if length <= 0 || length > 1024*1024*2 { // Max 2MB packet size
		return nil, fmt.Errorf("invalid packet length: %d", length)
	}

	// Read packet ID
	packetIDBuf := make([]byte, 1)
	if _, err := io.ReadFull(reader, packetIDBuf); err != nil {
		return nil, fmt.Errorf("failed to read packet ID: %w", err)
	}

	// Read packet data
	data := make([]byte, length-1)
	if _, err := io.ReadFull(reader, data); err != nil {
		return nil, fmt.Errorf("failed to read packet data: %w", err)
	}

	return &Packet{
		ID:   PacketID(packetIDBuf[0]),
		Data: data,
	}, nil
}

// writeVarInt writes a VarInt (Minecraft's variable-length integer format)
func writeVarInt(w io.Writer, value int32) error {
	for {
		temp := byte(value & 0x7F)
		value >>= 7
		if value != 0 {
			temp |= 0x80
		}
		if err := binary.Write(w, binary.BigEndian, temp); err != nil {
			return err
		}
		if value == 0 {
			break
		}
	}
	return nil
}

// readVarInt reads a VarInt from the reader
func readVarInt(r io.Reader) (int32, error) {
	var result int32
	var position uint

	for {
		var currentByte byte
		if err := binary.Read(r, binary.BigEndian, &currentByte); err != nil {
			return 0, err
		}

		result |= int32(currentByte&0x7F) << position

		if (currentByte & 0x80) == 0 {
			break
		}

		position += 7
		if position >= 32 {
			return 0, fmt.Errorf("VarInt is too big")
		}
	}

	return result, nil
}

// WriteString writes a Minecraft string (VarInt length + UTF-8 data)
func WriteString(w io.Writer, s string) error {
	data := []byte(s)
	if err := writeVarInt(w, int32(len(data))); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// ReadString reads a Minecraft string
func ReadString(r io.Reader) (string, error) {
	length, err := readVarInt(r)
	if err != nil {
		return "", err
	}

	if length < 0 || length > 32767 {
		return "", fmt.Errorf("invalid string length: %d", length)
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return "", err
	}

	return string(data), nil
}

// CreateHandshakePacket creates a fake Minecraft handshake packet for camouflage
func CreateHandshakePacket(serverAddress string, serverPort uint16) ([]byte, error) {
	buf := new(bytes.Buffer)

	// Protocol version (e.g., 760 for 1.19.2)
	if err := writeVarInt(buf, 760); err != nil {
		return nil, err
	}

	// Server address
	if err := WriteString(buf, serverAddress); err != nil {
		return nil, err
	}

	// Server port
	if err := binary.Write(buf, binary.BigEndian, serverPort); err != nil {
		return nil, err
	}

	// Next state (1 = status, 2 = login)
	if err := writeVarInt(buf, 1); err != nil {
		return nil, err
	}

	return EncodePacket(PacketHandshake, buf.Bytes())
}

// CreateKeepAlivePacket creates a fake keep-alive packet for maintaining connection appearance
func CreateKeepAlivePacket(id int64) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, id); err != nil {
		return nil, err
	}
	return EncodePacket(PacketKeepAlive, buf.Bytes())
}
