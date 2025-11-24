package c2s

import (
	"io"
	"koria-core/protocol/minecraft"
	"time"
)

// ChatMessagePacket - пакет отправки сообщения в чат
// Можем использовать для передачи средних блоков данных (~256 байт)
type ChatMessagePacket struct {
	Message   string    // Сообщение (до 256 символов)
	Timestamp time.Time // Временная метка
	Salt      int64     // Соль для подписи
	Signature []byte    // Подпись сообщения (опционально)
}

func (p *ChatMessagePacket) PacketID() minecraft.PacketType {
	return minecraft.PacketTypeChatMessage
}

func (p *ChatMessagePacket) Encode(w io.Writer) error {
	// Message
	if err := minecraft.WriteString(w, p.Message, 256); err != nil {
		return err
	}

	// Timestamp (milliseconds since epoch)
	timestampMs := p.Timestamp.UnixMilli()
	if err := minecraft.WriteVarLong(w, timestampMs); err != nil {
		return err
	}

	// Salt
	if err := minecraft.WriteVarLong(w, p.Salt); err != nil {
		return err
	}

	// Signature (опционально)
	hasSignature := len(p.Signature) > 0
	if err := minecraft.WriteVarInt(w, boolToInt(hasSignature)); err != nil {
		return err
	}

	if hasSignature {
		if err := minecraft.WriteVarInt(w, int32(len(p.Signature))); err != nil {
			return err
		}
		if _, err := w.Write(p.Signature); err != nil {
			return err
		}
	}

	return nil
}

func (p *ChatMessagePacket) Decode(r io.Reader) error {
	var err error

	// Message
	p.Message, err = minecraft.ReadString(r, 256)
	if err != nil {
		return err
	}

	// Timestamp
	timestampMs, err := minecraft.ReadVarLong(r)
	if err != nil {
		return err
	}
	p.Timestamp = time.UnixMilli(timestampMs)

	// Salt
	p.Salt, err = minecraft.ReadVarLong(r)
	if err != nil {
		return err
	}

	// Signature
	hasSignature, err := minecraft.ReadVarInt(r)
	if err != nil {
		return err
	}

	if hasSignature != 0 {
		sigLen, err := minecraft.ReadVarInt(r)
		if err != nil {
			return err
		}

		p.Signature = make([]byte, sigLen)
		if _, err := io.ReadFull(r, p.Signature); err != nil {
			return err
		}
	}

	return nil
}

func boolToInt(b bool) int32 {
	if b {
		return 1
	}
	return 0
}
