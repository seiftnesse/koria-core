package c2s

import (
	"encoding/binary"
	"io"
	"koria-core/protocol/minecraft"
)

// PlayerActionPacket - действия игрока (копание блока, дроп предмета и т.д.)
// Небольшой пакет для мелких данных (~12 байт)
type PlayerActionPacket struct {
	Action    int32   // Тип действия (enum)
	X, Y, Z   int32   // Позиция блока
	Direction uint8   // Направление
	Sequence  int32   // Порядковый номер
}

const (
	ActionStartDestroyBlock = iota
	ActionAbortDestroyBlock
	ActionStopDestroyBlock
	ActionDropAllItems
	ActionDropItem
	ActionReleaseUseItem
	ActionSwapItemWithOffhand
)

func (p *PlayerActionPacket) PacketID() minecraft.PacketType {
	return minecraft.PacketTypePlayerAction
}

func (p *PlayerActionPacket) Encode(w io.Writer) error {
	// Action
	if err := minecraft.WriteVarInt(w, p.Action); err != nil {
		return err
	}

	// Block Position (encoded as long)
	// Format: ((x & 0x3FFFFFF) << 38) | ((z & 0x3FFFFFF) << 12) | (y & 0xFFF)
	pos := ((int64(p.X) & 0x3FFFFFF) << 38) | ((int64(p.Z) & 0x3FFFFFF) << 12) | (int64(p.Y) & 0xFFF)
	if err := binary.Write(w, binary.BigEndian, pos); err != nil {
		return err
	}

	// Direction
	if _, err := w.Write([]byte{p.Direction}); err != nil {
		return err
	}

	// Sequence
	if err := minecraft.WriteVarInt(w, p.Sequence); err != nil {
		return err
	}

	return nil
}

func (p *PlayerActionPacket) Decode(r io.Reader) error {
	var err error

	// Action
	p.Action, err = minecraft.ReadVarInt(r)
	if err != nil {
		return err
	}

	// Block Position
	var pos int64
	if err := binary.Read(r, binary.BigEndian, &pos); err != nil {
		return err
	}

	// Decode position
	p.X = int32(pos >> 38)
	p.Y = int32(pos & 0xFFF)
	p.Z = int32((pos >> 12) & 0x3FFFFFF)

	// Direction
	dirBuf := make([]byte, 1)
	if _, err := io.ReadFull(r, dirBuf); err != nil {
		return err
	}
	p.Direction = dirBuf[0]

	// Sequence
	p.Sequence, err = minecraft.ReadVarInt(r)
	return err
}

// HandSwingPacket - взмах рукой (очень маленький пакет, 1 байт)
type HandSwingPacket struct {
	Hand uint8 // 0 = main hand, 1 = offhand
}

func (p *HandSwingPacket) PacketID() minecraft.PacketType {
	return minecraft.PacketTypeHandSwing
}

func (p *HandSwingPacket) Encode(w io.Writer) error {
	if err := minecraft.WriteVarInt(w, int32(p.Hand)); err != nil {
		return err
	}
	return nil
}

func (p *HandSwingPacket) Decode(r io.Reader) error {
	hand, err := minecraft.ReadVarInt(r)
	if err != nil {
		return err
	}
	p.Hand = uint8(hand)
	return nil
}
