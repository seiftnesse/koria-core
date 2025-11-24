package steganography

import (
	"encoding/binary"
	"fmt"
	c2s "koria-core/protocol/minecraft/packets/c2s"
	"math"
)

// Decoder декодирует фреймы из Minecraft пакетов
type Decoder struct{}

// NewDecoder создает новый декодер
func NewDecoder() *Decoder {
	return &Decoder{}
}

// DecodeFrame декодирует фрейм из PlayerMovePacket
func (d *Decoder) DecodeFrame(pkt *c2s.PlayerMovePacket) (*Frame, error) {
	// Извлекаем данные из координат
	encodedData := make([]byte, 17)

	// X coordinate -> первые 4 байта
	xData := d.decodeDataFromDouble(pkt.X)
	binary.BigEndian.PutUint32(encodedData[0:4], xData)

	// Y coordinate -> следующие 4 байта
	yData := d.decodeDataFromDouble(pkt.Y)
	binary.BigEndian.PutUint32(encodedData[4:8], yData)

	// Z coordinate -> следующие 4 байта
	zData := d.decodeDataFromDouble(pkt.Z)
	binary.BigEndian.PutUint32(encodedData[8:12], zData)

	// Yaw -> следующие 2 байта
	yawData := d.decodeDataFromFloat(pkt.Yaw)
	binary.BigEndian.PutUint16(encodedData[12:14], yawData)

	// Pitch -> следующие 2 байта
	pitchData := d.decodeDataFromFloat(pkt.Pitch)
	binary.BigEndian.PutUint16(encodedData[14:16], pitchData)

	// Flags -> последний байт
	encodedData[16] = pkt.Flags

	// Парсим заголовок фрейма
	if len(encodedData) < HeaderSize {
		return nil, fmt.Errorf("not enough data for frame header")
	}

	frame := &Frame{
		StreamID: binary.BigEndian.Uint16(encodedData[0:2]),
		Sequence: binary.BigEndian.Uint16(encodedData[2:4]),
		Flags:    encodedData[4],
	}

	dataLen := binary.BigEndian.Uint16(encodedData[5:7])
	frame.Length = dataLen

	// DEBUG: Логируем декодированные данные
	// log.Printf("[DEBUG DECODER] encodedData (17 bytes): %v", encodedData)
	// log.Printf("[DEBUG DECODER] StreamID=%d, Seq=%d, Flags=0x%02X, DataLen=%d",
	// 	frame.StreamID, frame.Sequence, frame.Flags, dataLen)

	// Извлекаем данные
	if dataLen > 0 {
		if HeaderSize+int(dataLen) > len(encodedData) {
			return nil, fmt.Errorf("frame data length exceeds available data: %d > %d",
				HeaderSize+int(dataLen), len(encodedData))
		}

		frame.Data = make([]byte, dataLen)
		copy(frame.Data, encodedData[HeaderSize:HeaderSize+int(dataLen)])
	}

	return frame, nil
}

// decodeDataFromDouble извлекает данные из младших 32 бит мантиссы double
func (d *Decoder) decodeDataFromDouble(value float64) uint32 {
	bits := math.Float64bits(value)
	// Возвращаем младшие 32 бита
	return uint32(bits & 0x00000000FFFFFFFF)
}

// decodeDataFromFloat извлекает данные из младших 16 бит float
func (d *Decoder) decodeDataFromFloat(value float32) uint16 {
	bits := math.Float32bits(value)
	// Возвращаем младшие 16 бит
	return uint16(bits & 0x0000FFFF)
}

// DecodeFrameFromCustomPayload декодирует фрейм из CustomPayloadPacket
func (d *Decoder) DecodeFrameFromCustomPayload(pkt *c2s.CustomPayloadPacket) (*Frame, error) {
	if len(pkt.Data) < HeaderSize {
		return nil, fmt.Errorf("payload too small for frame header")
	}

	frame := &Frame{
		StreamID: binary.BigEndian.Uint16(pkt.Data[0:2]),
		Sequence: binary.BigEndian.Uint16(pkt.Data[2:4]),
		Flags:    pkt.Data[4],
	}

	dataLen := binary.BigEndian.Uint16(pkt.Data[5:7])
	frame.Length = dataLen

	if dataLen > 0 {
		if HeaderSize+int(dataLen) > len(pkt.Data) {
			return nil, fmt.Errorf("frame data length exceeds payload: %d > %d",
				HeaderSize+int(dataLen), len(pkt.Data))
		}

		frame.Data = make([]byte, dataLen)
		copy(frame.Data, pkt.Data[HeaderSize:HeaderSize+int(dataLen)])
	}

	return frame, nil
}
