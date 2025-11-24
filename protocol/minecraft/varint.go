package minecraft

import (
	"fmt"
	"io"
)

const (
	// MaxVarIntLength максимальная длина VarInt в байтах
	MaxVarIntLength = 5
	// MaxVarLongLength максимальная длина VarLong в байтах
	MaxVarLongLength = 10
)

// ReadVarInt читает VarInt из reader
// VarInt - это переменная длина целого числа, используемая в Minecraft протоколе
// Каждый байт использует 7 бит для данных и 1 бит (MSB) для продолжения
func ReadVarInt(r io.Reader) (int32, error) {
	var value int32
	var position uint
	buf := make([]byte, 1)

	for {
		n, err := r.Read(buf)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, io.ErrUnexpectedEOF
		}

		// Извлекаем 7 бит данных
		value |= int32(buf[0]&0x7F) << position

		// Проверяем бит продолжения (MSB)
		if buf[0]&0x80 == 0 {
			break
		}

		position += 7
		if position >= 32 {
			return 0, fmt.Errorf("VarInt too big")
		}
	}

	return value, nil
}

// WriteVarInt записывает VarInt в writer
func WriteVarInt(w io.Writer, value int32) error {
	buf := make([]byte, 0, MaxVarIntLength)

	for {
		// Если остались только данные без продолжения
		if value&^0x7F == 0 {
			buf = append(buf, byte(value))
			break
		}

		// Записываем 7 бит данных + устанавливаем бит продолжения
		buf = append(buf, byte(value&0x7F|0x80))
		value >>= 7
	}

	_, err := w.Write(buf)
	return err
}

// VarIntSize возвращает размер VarInt в байтах
func VarIntSize(value int32) int {
	size := 0
	for {
		if value&^0x7F == 0 {
			size++
			break
		}
		size++
		value >>= 7
	}
	return size
}

// ReadVarLong читает VarLong из reader
func ReadVarLong(r io.Reader) (int64, error) {
	var value int64
	var position uint
	buf := make([]byte, 1)

	for {
		n, err := r.Read(buf)
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, io.ErrUnexpectedEOF
		}

		value |= int64(buf[0]&0x7F) << position

		if buf[0]&0x80 == 0 {
			break
		}

		position += 7
		if position >= 64 {
			return 0, fmt.Errorf("VarLong too big")
		}
	}

	return value, nil
}

// WriteVarLong записывает VarLong в writer
func WriteVarLong(w io.Writer, value int64) error {
	buf := make([]byte, 0, MaxVarLongLength)

	for {
		if value&^0x7F == 0 {
			buf = append(buf, byte(value))
			break
		}

		buf = append(buf, byte(value&0x7F|0x80))
		value >>= 7
	}

	_, err := w.Write(buf)
	return err
}

// VarLongSize возвращает размер VarLong в байтах
func VarLongSize(value int64) int {
	size := 0
	for {
		if value&^0x7F == 0 {
			size++
			break
		}
		size++
		value >>= 7
	}
	return size
}
