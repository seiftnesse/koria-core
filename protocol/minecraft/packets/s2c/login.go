package s2c

import (
	"github.com/google/uuid"
	"io"
	"koria-core/protocol/minecraft"
)

// LoginSuccessPacket - пакет успешной авторизации
type LoginSuccessPacket struct {
	UUID       uuid.UUID // UUID игрока
	Username   string    // Username игрока
	Properties []Property // Дополнительные свойства (текстуры и т.д.)
}

type Property struct {
	Name      string
	Value     string
	Signature string // Опционально
}

func (p *LoginSuccessPacket) PacketID() minecraft.PacketType {
	return minecraft.PacketTypeLoginSuccess
}

func (p *LoginSuccessPacket) Encode(w io.Writer) error {
	// UUID
	uuidBytes, err := p.UUID.MarshalBinary()
	if err != nil {
		return err
	}
	if _, err := w.Write(uuidBytes); err != nil {
		return err
	}

	// Username
	if err := minecraft.WriteString(w, p.Username, 16); err != nil {
		return err
	}

	// Properties count
	if err := minecraft.WriteVarInt(w, int32(len(p.Properties))); err != nil {
		return err
	}

	// Properties
	for _, prop := range p.Properties {
		if err := minecraft.WriteString(w, prop.Name, 32767); err != nil {
			return err
		}
		if err := minecraft.WriteString(w, prop.Value, 32767); err != nil {
			return err
		}

		// Has signature
		hasSignature := len(prop.Signature) > 0
		if _, err := w.Write([]byte{boolToByte(hasSignature)}); err != nil {
			return err
		}

		if hasSignature {
			if err := minecraft.WriteString(w, prop.Signature, 32767); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *LoginSuccessPacket) Decode(r io.Reader) error {
	// UUID
	uuidBytes := make([]byte, 16)
	if _, err := io.ReadFull(r, uuidBytes); err != nil {
		return err
	}
	var err error
	p.UUID, err = uuid.FromBytes(uuidBytes)
	if err != nil {
		return err
	}

	// Username
	p.Username, err = minecraft.ReadString(r, 16)
	if err != nil {
		return err
	}

	// Properties count
	propCount, err := minecraft.ReadVarInt(r)
	if err != nil {
		return err
	}

	// Properties
	p.Properties = make([]Property, propCount)
	for i := int32(0); i < propCount; i++ {
		prop := Property{}

		prop.Name, err = minecraft.ReadString(r, 32767)
		if err != nil {
			return err
		}

		prop.Value, err = minecraft.ReadString(r, 32767)
		if err != nil {
			return err
		}

		// Has signature
		hasSigByte := make([]byte, 1)
		if _, err := io.ReadFull(r, hasSigByte); err != nil {
			return err
		}

		if hasSigByte[0] != 0 {
			prop.Signature, err = minecraft.ReadString(r, 32767)
			if err != nil {
				return err
			}
		}

		p.Properties[i] = prop
	}

	return nil
}

// LoginDisconnectPacket - отключение во время логина
type LoginDisconnectPacket struct {
	Reason string // JSON формат (Chat component)
}

func (p *LoginDisconnectPacket) PacketID() minecraft.PacketType {
	return 0x00 // LOGIN_DISCONNECT
}

func (p *LoginDisconnectPacket) Encode(w io.Writer) error {
	return minecraft.WriteString(w, p.Reason, 262144)
}

func (p *LoginDisconnectPacket) Decode(r io.Reader) error {
	var err error
	p.Reason, err = minecraft.ReadString(r, 262144)
	return err
}

func boolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}
