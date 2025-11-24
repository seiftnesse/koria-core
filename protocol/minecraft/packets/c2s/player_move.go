package c2s

import (
	"io"
	"koria-core/protocol/minecraft"
)

// PlayerMovePacket представляет пакет движения игрока (MOVE_PLAYER_POS_ROT)
// Это самый важный пакет для стеганографии - содержит координаты и углы поворота
type PlayerMovePacket struct {
	X     float64 // Позиция X
	Y     float64 // Позиция Y
	Z     float64 // Позиция Z
	Yaw   float32 // Поворот горизонтальный (0-360)
	Pitch float32 // Поворот вертикальный (-90 - 90)
	Flags uint8   // Флаги: bit 0 = onGround, bit 1 = horizontalCollision
}

func (p *PlayerMovePacket) PacketID() minecraft.PacketType {
	return minecraft.PacketTypePlayerMove
}

func (p *PlayerMovePacket) Encode(w io.Writer) error {
	// Записываем координаты (3 x double = 24 байта)
	if err := minecraft.WriteDouble(w, p.X); err != nil {
		return err
	}
	if err := minecraft.WriteDouble(w, p.Y); err != nil {
		return err
	}
	if err := minecraft.WriteDouble(w, p.Z); err != nil {
		return err
	}

	// Записываем углы поворота (2 x float = 8 байт)
	if err := minecraft.WriteFloat(w, p.Yaw); err != nil {
		return err
	}
	if err := minecraft.WriteFloat(w, p.Pitch); err != nil {
		return err
	}

	// Записываем флаги (1 байт)
	_, err := w.Write([]byte{p.Flags})
	return err
}

func (p *PlayerMovePacket) Decode(r io.Reader) error {
	var err error

	// Читаем координаты
	p.X, err = minecraft.ReadDouble(r)
	if err != nil {
		return err
	}
	p.Y, err = minecraft.ReadDouble(r)
	if err != nil {
		return err
	}
	p.Z, err = minecraft.ReadDouble(r)
	if err != nil {
		return err
	}

	// Читаем углы
	p.Yaw, err = minecraft.ReadFloat(r)
	if err != nil {
		return err
	}
	p.Pitch, err = minecraft.ReadFloat(r)
	if err != nil {
		return err
	}

	// Читаем флаги
	buf := make([]byte, 1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	p.Flags = buf[0]

	return nil
}

// Size возвращает размер пакета в байтах
func (p *PlayerMovePacket) Size() int {
	return 8 + 8 + 8 + 4 + 4 + 1 // 33 байта
}

// PlayerPositionPacket - только позиция (без углов)
type PlayerPositionPacket struct {
	X     float64
	Y     float64
	Z     float64
	Flags uint8
}

func (p *PlayerPositionPacket) PacketID() minecraft.PacketType {
	return minecraft.PacketTypePlayerPosition
}

func (p *PlayerPositionPacket) Encode(w io.Writer) error {
	if err := minecraft.WriteDouble(w, p.X); err != nil {
		return err
	}
	if err := minecraft.WriteDouble(w, p.Y); err != nil {
		return err
	}
	if err := minecraft.WriteDouble(w, p.Z); err != nil {
		return err
	}
	_, err := w.Write([]byte{p.Flags})
	return err
}

func (p *PlayerPositionPacket) Decode(r io.Reader) error {
	var err error
	p.X, err = minecraft.ReadDouble(r)
	if err != nil {
		return err
	}
	p.Y, err = minecraft.ReadDouble(r)
	if err != nil {
		return err
	}
	p.Z, err = minecraft.ReadDouble(r)
	if err != nil {
		return err
	}

	buf := make([]byte, 1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	p.Flags = buf[0]
	return nil
}

// PlayerRotationPacket - только углы поворота (без позиции)
type PlayerRotationPacket struct {
	Yaw   float32
	Pitch float32
	Flags uint8
}

func (p *PlayerRotationPacket) PacketID() minecraft.PacketType {
	return minecraft.PacketTypePlayerRotation
}

func (p *PlayerRotationPacket) Encode(w io.Writer) error {
	if err := minecraft.WriteFloat(w, p.Yaw); err != nil {
		return err
	}
	if err := minecraft.WriteFloat(w, p.Pitch); err != nil {
		return err
	}
	_, err := w.Write([]byte{p.Flags})
	return err
}

func (p *PlayerRotationPacket) Decode(r io.Reader) error {
	var err error
	p.Yaw, err = minecraft.ReadFloat(r)
	if err != nil {
		return err
	}
	p.Pitch, err = minecraft.ReadFloat(r)
	if err != nil {
		return err
	}

	buf := make([]byte, 1)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	p.Flags = buf[0]
	return nil
}
