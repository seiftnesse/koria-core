package common

import (
	"io"
	"koria-core/protocol/minecraft"
)

// HandshakePacket - первый пакет, который отправляет клиент
// Определяет намерение (Status, Login)
type HandshakePacket struct {
	ProtocolVersion int32  // Версия протокола Minecraft (например, 765 для 1.20.4)
	ServerAddress   string // Адрес сервера
	ServerPort      uint16 // Порт сервера
	NextState       int32  // Следующее состояние: 1 = Status, 2 = Login
}

func (p *HandshakePacket) PacketID() minecraft.PacketType {
	return minecraft.PacketTypeHandshake
}

func (p *HandshakePacket) Encode(w io.Writer) error {
	// Protocol Version
	if err := minecraft.WriteVarInt(w, p.ProtocolVersion); err != nil {
		return err
	}

	// Server Address
	if err := minecraft.WriteString(w, p.ServerAddress, 255); err != nil {
		return err
	}

	// Server Port
	portBytes := []byte{byte(p.ServerPort >> 8), byte(p.ServerPort)}
	if _, err := w.Write(portBytes); err != nil {
		return err
	}

	// Next State
	if err := minecraft.WriteVarInt(w, p.NextState); err != nil {
		return err
	}

	return nil
}

func (p *HandshakePacket) Decode(r io.Reader) error {
	var err error

	// Protocol Version
	p.ProtocolVersion, err = minecraft.ReadVarInt(r)
	if err != nil {
		return err
	}

	// Server Address
	p.ServerAddress, err = minecraft.ReadString(r, 255)
	if err != nil {
		return err
	}

	// Server Port
	portBytes := make([]byte, 2)
	if _, err := io.ReadFull(r, portBytes); err != nil {
		return err
	}
	p.ServerPort = uint16(portBytes[0])<<8 | uint16(portBytes[1])

	// Next State
	p.NextState, err = minecraft.ReadVarInt(r)
	if err != nil {
		return err
	}

	return nil
}
