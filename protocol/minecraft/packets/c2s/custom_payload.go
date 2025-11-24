package c2s

import (
	"io"
	"koria-core/protocol/minecraft"
)

const (
	// MaxCustomPayloadSize максимальный размер custom payload
	MaxCustomPayloadSize = 32767 // 32KB
)

// CustomPayloadPacket представляет пакет с произвольными данными
// Используется для передачи больших блоков данных (до 32KB)
type CustomPayloadPacket struct {
	Channel string // Идентификатор канала (например, "minecraft:brand")
	Data    []byte // Произвольные данные
}

func (p *CustomPayloadPacket) PacketID() minecraft.PacketType {
	return minecraft.PacketTypeCustomPayload
}

func (p *CustomPayloadPacket) Encode(w io.Writer) error {
	// Записываем идентификатор канала
	if err := minecraft.WriteString(w, p.Channel, 32767); err != nil {
		return err
	}

	// Записываем данные
	_, err := w.Write(p.Data)
	return err
}

func (p *CustomPayloadPacket) Decode(r io.Reader) error {
	// Читаем идентификатор канала
	channel, err := minecraft.ReadString(r, 32767)
	if err != nil {
		return err
	}
	p.Channel = channel

	// Читаем остальные данные целиком
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	p.Data = data

	return nil
}

// Size возвращает размер пакета
func (p *CustomPayloadPacket) Size() int {
	return minecraft.VarIntSize(int32(len(p.Channel))) + len(p.Channel) + len(p.Data)
}
