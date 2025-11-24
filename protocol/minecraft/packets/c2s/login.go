package c2s

import (
	"github.com/google/uuid"
	"io"
	"koria-core/protocol/minecraft"
)

// LoginStartPacket - пакет начала авторизации
// Username содержит UUID пользователя в hex формате (для нашего протокола)
type LoginStartPacket struct {
	Username string    // Username или UUID в hex формате
	UUID     uuid.UUID // UUID игрока
}

func (p *LoginStartPacket) PacketID() minecraft.PacketType {
	return minecraft.PacketTypeLoginStart
}

func (p *LoginStartPacket) Encode(w io.Writer) error {
	// Username
	if err := minecraft.WriteString(w, p.Username, 16); err != nil {
		return err
	}

	// UUID (16 байт)
	uuidBytes, err := p.UUID.MarshalBinary()
	if err != nil {
		return err
	}

	_, err = w.Write(uuidBytes)
	return err
}

func (p *LoginStartPacket) Decode(r io.Reader) error {
	var err error

	// Username
	p.Username, err = minecraft.ReadString(r, 16)
	if err != nil {
		return err
	}

	// UUID
	uuidBytes := make([]byte, 16)
	if _, err := io.ReadFull(r, uuidBytes); err != nil {
		return err
	}

	p.UUID, err = uuid.FromBytes(uuidBytes)
	return err
}
